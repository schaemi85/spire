package application

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	commoncmds "github.com/schaemi85/spire/cmd/common"
	"github.com/schaemi85/spire/internal/engine"
	"github.com/schaemi85/spire/internal/manifest"
	"github.com/schaemi85/spire/internal/metadata"
	"github.com/schaemi85/spire/internal/templatesource"
	"github.com/schaemi85/spire/internal/tools"

	"github.com/spf13/cobra"
)

// PromptTemplateVersion lists available template versions and prompts the user to select one.
func PromptTemplateVersion(src templatesource.Source) (string, error) {
	fmt.Println("\n🔖 Fetching available template versions...")
	versions, err := src.ListVersions(context.Background(), 4)
	if err != nil {
		return "", fmt.Errorf("failed to list template versions: %w", err)
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no template versions found")
	}

	latest := versions[0]
	fmt.Println("\nAvailable template versions (latest 4):")
	fmt.Println()
	for i, v := range versions {
		marker := "  "
		if i == 0 {
			marker = "* "
		}
		fmt.Printf("  %s[%d] %s\n", marker, i+1, v)
	}
	fmt.Println()

	for {
		input := tools.ReadLine(fmt.Sprintf("Select a version [1-%d] (press Enter for latest: %s): ", len(versions), latest))
		if input == "" {
			fmt.Printf("   Selected template version: %s (latest)\n", latest)
			return latest, nil
		}
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(versions) {
			fmt.Printf("Please enter a number between 1 and %d\n", len(versions))
			continue
		}
		selected := versions[num-1]
		fmt.Printf("   Selected template version: %s\n", selected)
		return selected, nil
	}
}

// InitApplication creates a new application from a template.
func InitApplication(m *manifest.SpireManifest, rc *engine.ResolveContext, scaffoldingDir string) error {
	projectSlugName := m.GetSlotValue("ProjectSlugName")
	fmt.Printf("\nInitializing new application: '%s'\n", projectSlugName)
	fmt.Println(strings.Repeat("=", 60))

	cloneDir := projectSlugName
	if _, err := os.Stat(cloneDir); err == nil {
		return fmt.Errorf("target directory '%s' already exists", cloneDir)
	}

	fmt.Println("\n📂 Setting up project directory...")
	if err := tools.CopyDir(scaffoldingDir, cloneDir); err != nil {
		return fmt.Errorf("error copying template: %w", err)
	}
	fmt.Println("Template downloaded and extracted successfully")

	fmt.Println("\n🔧 Customizing application from template slots...")
	if err := engine.RenderProjectFiles(cloneDir, rc, m.IgnorePaths); err != nil {
		return fmt.Errorf("error rendering project files: %w", err)
	}
	fmt.Println("Application customized successfully")

	if len(m.TemplateFiles) > 0 {
		fmt.Println("\n⚙️  Rendering template files...")
		if err := engine.RenderTemplateFiles(scaffoldingDir, cloneDir, m.TemplateFiles, rc); err != nil {
			return fmt.Errorf("error rendering template files: %w", err)
		}
		fmt.Println("Template files rendered successfully")
	}

	if len(m.PathRenames) > 0 {
		fmt.Println("\nApplying path renames...")
		if err := engine.ApplyPathRenames(cloneDir, m.PathRenames, rc); err != nil {
			return fmt.Errorf("error applying path renames: %w", err)
		}
		fmt.Println("Path renames applied successfully")
	}

	engine.PopulateManifestFromSlots(m, rc)
	engine.ClearSecretSlotValues(m.AppSlots)

	if err := manifest.SaveManifest(m, cloneDir); err != nil {
		return fmt.Errorf("could not save manifest: %w", err)
	}

	fmt.Println("\nInitializing Git repository...")
	initGitRepo(cloneDir)

	printNextSteps(cloneDir)
	return nil
}

func initGitRepo(cloneDir string) {
	gitInitCmd := exec.Command("git", "init", "-b", "main")
	gitInitCmd.Dir = cloneDir
	gitInitCmd.Stdout = os.Stdout
	gitInitCmd.Stderr = os.Stderr
	if err := gitInitCmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not initialize new git repository: %v\n", err)
		return
	}
	fmt.Println("Git repository initialized")

	fmt.Println("\nCreating initial commit...")
	gitAddCmd := exec.Command("git", "add", ".")
	gitAddCmd.Dir = cloneDir
	gitAddCmd.Stdout = os.Stdout
	gitAddCmd.Stderr = os.Stderr
	if err := gitAddCmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not add files to git: %v\n", err)
		return
	}
	gitCommitCmd := exec.Command("git", "commit", "-m", "Initial commit")
	gitCommitCmd.Dir = cloneDir
	gitCommitCmd.Stdout = os.Stdout
	gitCommitCmd.Stderr = os.Stderr
	if err := gitCommitCmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not execute initial commit: %v\n", err)
		return
	}
	fmt.Println("Initial commit created")
}

func printNextSteps(cloneDir string) {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nApplication created successfully!")
	fmt.Printf("\n📁 Location: ./%s\n", cloneDir)
	absPath, _ := filepath.Abs(cloneDir)
	fmt.Printf("   Absolute path: %s\n", absPath)
	fmt.Println("\n📋 Next steps:")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("\n  [1] Open the project in VS Code:\n")
	fmt.Printf("      code %s\n\n", cloneDir)
	fmt.Println("  [2] Add your first service:")
	fmt.Println("      spire service add")
	fmt.Println()
	fmt.Println("  [3] Start coding!")
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()
	fmt.Println("Happy coding!")
}

var (
	semverPattern     = regexp.MustCompile(`^v?(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	commitHashPattern = regexp.MustCompile(`^[a-fA-F0-9]{7,40}$`)
)

func isCommitHash(s string) bool    { return commitHashPattern.MatchString(s) }
func isSemverPattern(s string) bool { return semverPattern.MatchString(s) }

var InitApplicationCmd = &cobra.Command{
	Use:   "init <template>",
	Short: "Initialize a new application from a scaffolding template",
	Long: `Initialize a new application from a template repository or local directory.

The template argument can be:
  - A Git URL:    spire init https://gitlab.com/org/template-repo
  - A local path: spire init ./my-local-template`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.EnsureGitIsInstalled()
		commoncmds.SwitchToWorkdir(cmd)
		nint, _ := cmd.Root().Flags().GetBool("non-interactive")

		templateArg := args[0]

		currentUser, err := user.Current()
		if err != nil {
			fmt.Printf("❌ Failed to retrieve current system user: %v\n", err)
			os.Exit(1)
		}
		devUser := strings.TrimPrefix(currentUser.Username, "BASE\\")

		var src templatesource.Source
		isGitSource := false

		if info, err := os.Stat(templateArg); err == nil && info.IsDir() {
			fmt.Printf("\n📂 Using local template: %s\n", templateArg)
			src = templatesource.NewLocalSource(templateArg)
		} else {
			fmt.Printf("\n📡 Using git template: %s\n", templateArg)
			src = templatesource.NewGitSource(templateArg)
			isGitSource = true
		}

		templateVersion, _ := cmd.Flags().GetString("version")
		if templateVersion == "" && isGitSource {
			if nint {
				fmt.Println("❌ Error: --version is required in non-interactive mode for git templates")
				os.Exit(1)
			}
			templateVersion, err = PromptTemplateVersion(src)
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Println("\nDownloading application template...")
		scaffoldingDir, err := src.Download(context.Background(), templateVersion)
		if err != nil {
			fmt.Printf("❌ Error downloading template: %v\n", err)
			os.Exit(1)
		}
		defer src.Cleanup()

		m, err := manifest.LoadManifestFrom(filepath.Join(scaffoldingDir, ".spire", "manifest.yaml"))
		if err != nil {
			fmt.Printf("❌ Error loading template manifest: %v\n", err)
			os.Exit(1)
		}
		m.SpireVersion = metadata.VERSION
		m.TemplateVersion = templateVersion

		rc := engine.NewResolveContext()
		rc.Slots["DevUser"] = devUser

		setValues, _ := cmd.Flags().GetStringArray("set")
		for _, sv := range setValues {
			parts := strings.SplitN(sv, "=", 2)
			if len(parts) == 2 {
				rc.Slots[parts[0]] = parts[1]
			}
		}

		if err := engine.ResolveSlots(m.AppSlots, rc, nint); err != nil {
			fmt.Printf("❌ %v\n", err)
			os.Exit(1)
		}

		engine.PopulateManifestFromSlots(m, rc)

		if err := InitApplication(m, rc, scaffoldingDir); err != nil {
			fmt.Printf("\n❌ Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	InitApplicationCmd.Flags().StringArray("set", nil, "Set slot values (repeatable): --set Key=Value")
	InitApplicationCmd.Flags().String("version", "", "Template version to use (required in non-interactive mode for git templates)")
}
