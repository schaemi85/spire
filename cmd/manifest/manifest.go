package manifest

import (
	"fmt"
	"os"
	"strings"

	commoncmds "github.com/schaemi85/spire/cmd/common"
	"github.com/schaemi85/spire/internal/manifest"
	"github.com/schaemi85/spire/internal/metadata"

	"github.com/spf13/cobra"
)

// skeletonTemplate is written to .spire/manifest.yaml by `spire manifest init`.
// %s is replaced with the current Spire version.
const skeletonTemplate = `# Spire Manifest – Template Configuration
# Reference: https://github.com/schaemi85/spire/blob/main/docs/manifest-reference.md

spireVersion: %s
templateVersion: v0.0.1

# gitRepository: ""   # URL of the template's Git repository (populated automatically by template sync)

# Paths to skip during Go template rendering (matched by filename anywhere in the tree)
ignorePaths:
  - .git
  - .spire

# ---------------------------------------------------------------------------
# Application-level slots
# Collected from the developer when they run ` + "`spire init`" + `.
# type values: PromptOptional | PromptMandatory | PromptSecret | DynamicValue
# ---------------------------------------------------------------------------
appSlots: []
# appSlots:
#   - key: ProjectName
#     label: "Project Name"
#     description: "Human-readable name of the project"
#     type: PromptMandatory
#     defaultValue: "my-project"
#     # validation: "slug"
#
#   - key: ProjectSlugName
#     label: "Project Slug"
#     type: DynamicValue
#     expression: "[[ .slots.ProjectName | slugify ]]"

# ---------------------------------------------------------------------------
# Template files
# Rendered from the template directory into the generated project on ` + "`spire init`" + `.
# ---------------------------------------------------------------------------
# templateFiles:
#   - source: ".devcontainer/.env.tmpl"
#     destination: ".devcontainer/.env"
#     regenerateOnServiceChange: false

# ---------------------------------------------------------------------------
# Path renames
# Applied after rendering — pattern text in file/directory names is replaced
# by the evaluated expression.
# ---------------------------------------------------------------------------
# pathRenames:
#   - pattern: "sampleproject"
#     expression: "[[ .slots.ProjectSlugName ]]"

# ---------------------------------------------------------------------------
# Service configuration
# Blueprint used when the developer runs ` + "`spire service add`" + `.
# ---------------------------------------------------------------------------
# serviceConfig:
#   originalPath: "services/sampleservice"
#   servicesSlots:
#     - key: ServiceName
#       label: "Service Name"
#       type: PromptMandatory
#       validation: "slug"
#   pathRenames:
#     - pattern: "sampleservice"
#       expression: "[[ .slots.ServiceName ]]"
#   postHooks: []
`

// ManifestCmd is the parent command for all manifest subcommands.
var ManifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Manage the Spire manifest (.spire/manifest.yaml)",
}

// InitManifestCmd scaffolds a new .spire/manifest.yaml skeleton.
var InitManifestCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a new .spire/manifest.yaml in the current directory",
	Long: `Creates a .spire/manifest.yaml skeleton with commented examples.

Edit the generated file to define your template's slots, path renames,
and service configuration, then run 'spire manifest validate' to check for errors.`,
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)

		force, _ := cmd.Flags().GetBool("force")

		if _, err := os.Stat(manifest.FilePath); err == nil && !force {
			fmt.Printf("❌ %s already exists. Use --force to overwrite.\n", manifest.FilePath)
			os.Exit(1)
		}

		if err := os.MkdirAll(".spire", 0755); err != nil {
			fmt.Printf("❌ Failed to create .spire directory: %v\n", err)
			os.Exit(1)
		}

		content := fmt.Sprintf(skeletonTemplate, metadata.VERSION)
		if err := os.WriteFile(manifest.FilePath, []byte(content), 0644); err != nil {
			fmt.Printf("❌ Failed to write %s: %v\n", manifest.FilePath, err)
			os.Exit(1)
		}

		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("Created %s\n\n", manifest.FilePath)
		fmt.Println("Next steps:")
		fmt.Println("  1. Edit the manifest to define your template's slots and configuration")
		fmt.Println("  2. Run 'spire manifest validate' to check for errors")
		fmt.Println("  3. Run 'spire template sync --output <template-dir>' to generate the template")
	},
}

func init() {
	InitManifestCmd.Flags().Bool("force", false, "Overwrite existing manifest without prompting")
	ManifestCmd.AddCommand(InitManifestCmd)
	ManifestCmd.AddCommand(ValidateManifestCmd)
}
