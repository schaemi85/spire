package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	commoncmds "github.com/schaemi85/spire/cmd/common"
	"github.com/schaemi85/spire/internal/engine"
	"github.com/schaemi85/spire/internal/manifest"
	"github.com/schaemi85/spire/internal/tools"

	"github.com/spf13/cobra"
)

// SyncTemplateFromProject generates (or updates) a reusable template from an
// existing Spire project by reversing resolved slot values back into expressions.
func SyncTemplateFromProject(projectDir, outputDir string) error {
	manifestPath := filepath.Join(projectDir, manifest.FilePath)
	m, err := manifest.LoadManifestFrom(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if info, err := os.Stat(outputDir); err == nil && info.IsDir() {
		fmt.Println("♻️  Output directory exists — updating template (preserving .git)...")
		entries, err := os.ReadDir(outputDir)
		if err != nil {
			return fmt.Errorf("failed to read output directory: %w", err)
		}
		for _, e := range entries {
			if e.Name() == ".git" {
				continue
			}
			if err := os.RemoveAll(filepath.Join(outputDir, e.Name())); err != nil {
				return fmt.Errorf("failed to clean %s: %w", e.Name(), err)
			}
		}
	}

	fmt.Println("📂 Copying project to template output directory...")
	if err := tools.CopyDir(projectDir, outputDir); err != nil {
		return fmt.Errorf("failed to copy project: %w", err)
	}

	// Build reverse replacements: Value → [[ .slots.KEY ]].
	var replacements []tools.Replacement
	for _, slot := range m.AppSlots {
		if slot.Value != "" {
			replacements = append(replacements, tools.Replacement{
				Placeholder: slot.Value,
				Replacement: fmt.Sprintf("[[ .slots.%s ]]", slot.Key),
			})
		}
	}
	if len(replacements) > 0 {
		fmt.Println("Reversing app-level slot values to Go template expressions...")
		if err := tools.ReplaceInFiles(outputDir, replacements, m.IgnorePaths); err != nil {
			return fmt.Errorf("failed to reverse replacements: %w", err)
		}
	}

	// Handle reference service.
	if m.ServiceConfig.OriginalPath != "" {
		refServiceDir := filepath.Join(outputDir, m.ServiceConfig.OriginalPath)
		if _, err := os.Stat(refServiceDir); err != nil {
			return fmt.Errorf("reference service not found at %q: %w", m.ServiceConfig.OriginalPath, err)
		}

		templateServiceDir := filepath.Join(outputDir, "templates", "service")
		if err := os.MkdirAll(filepath.Dir(templateServiceDir), 0755); err != nil {
			return fmt.Errorf("failed to create templates directory: %w", err)
		}

		fmt.Printf("Copying reference service %q to templates/service/...\n", m.ServiceConfig.OriginalPath)
		if err := tools.CopyDir(refServiceDir, templateServiceDir); err != nil {
			return fmt.Errorf("failed to copy reference service: %w", err)
		}

		var svcReplacements []tools.Replacement
		for _, slot := range m.ServiceConfig.ServicesSlots {
			if slot.Value != "" {
				svcReplacements = append(svcReplacements, tools.Replacement{
					Placeholder: slot.Value,
					Replacement: fmt.Sprintf("[[ .slots.%s ]]", slot.Key),
				})
			}
		}
		if len(svcReplacements) > 0 {
			fmt.Println("Reversing service-level slot values in templates/service/...")
			if err := tools.ReplaceInFiles(templateServiceDir, svcReplacements, m.IgnorePaths); err != nil {
				return fmt.Errorf("failed to reverse service slot values: %w", err)
			}
		}

		if len(m.ServiceConfig.PathRenames) > 0 {
			rc := engine.BuildResolveContextFromManifest(m)
			for _, slot := range m.ServiceConfig.ServicesSlots {
				if slot.Value != "" {
					rc.Slots[slot.Key] = slot.Value
				}
			}
			for _, pr := range m.ServiceConfig.PathRenames {
				resolved, err := engine.EvaluateExpression(pr.Expression, rc)
				if err != nil || resolved == "" {
					continue
				}
				if err := tools.RenamePathsWithPlaceholder(templateServiceDir, resolved, pr.Pattern); err != nil {
					return fmt.Errorf("failed to reverse service path rename for pattern %q: %w", pr.Pattern, err)
				}
			}
		}

		servicesDir := filepath.Join(outputDir, "services")
		if _, err := os.Stat(servicesDir); err == nil {
			fmt.Println("🗑️  Removing generated services directory...")
			_ = os.RemoveAll(servicesDir)
		}
	}

	// Reverse app-level path renames.
	if len(m.PathRenames) > 0 {
		rc := engine.BuildResolveContextFromManifest(m)
		for _, pr := range m.PathRenames {
			resolved, err := engine.EvaluateExpression(pr.Expression, rc)
			if err != nil || resolved == "" {
				continue
			}
			if err := tools.RenamePathsWithPlaceholder(outputDir, resolved, pr.Pattern); err != nil {
				return fmt.Errorf("failed to reverse path rename for pattern %q: %w", pr.Pattern, err)
			}
		}
	}

	// Clear resolved values so the template is a clean blueprint.
	for i := range m.AppSlots {
		m.AppSlots[i].Value = ""
	}
	for i := range m.ServiceConfig.ServicesSlots {
		m.ServiceConfig.ServicesSlots[i].Value = ""
	}
	m.GitRepository = ""
	m.Services = nil

	if err := manifest.SaveManifest(m, outputDir); err != nil {
		return fmt.Errorf("failed to save template manifest: %w", err)
	}

	fmt.Println("Template synced successfully at:", outputDir)
	return nil
}

var SyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Create or update a reusable template from an existing Spire project",
	Long: `Scans the current project's .spire/manifest.yaml and reverses resolved
slot values back into Go template expressions to produce a reusable template.

If the output directory does not exist, a new template is created.
If it already exists, its contents are replaced while preserving .git/ history.`,
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)

		outputDir, _ := cmd.Flags().GetString("output")
		if outputDir == "" {
			fmt.Println("❌ Error: --output is required")
			os.Exit(1)
		}

		projectDir, err := os.Getwd()
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		if _, err := os.Stat(filepath.Join(projectDir, manifest.FilePath)); err != nil {
			fmt.Printf("❌ Not a Spire project: %s not found\n", manifest.FilePath)
			fmt.Println("   Run this command from the root of a Spire-generated project.")
			os.Exit(1)
		}

		fmt.Println(strings.Repeat("=", 60))
		fmt.Println("🏗️  Syncing template from existing project")
		fmt.Println(strings.Repeat("=", 60))

		if err := SyncTemplateFromProject(projectDir, outputDir); err != nil {
			fmt.Printf("\n❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	SyncCmd.Flags().String("output", "", "Output directory for the generated template (required)")
}
