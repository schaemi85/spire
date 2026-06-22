package upgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/schaemi85/spire/cmd/application"
	commoncmds "github.com/schaemi85/spire/cmd/common"
	mfst "github.com/schaemi85/spire/internal/manifest"
	"github.com/schaemi85/spire/internal/plugin"
	"github.com/schaemi85/spire/internal/templatesource"
	"github.com/schaemi85/spire/internal/tools"

	"github.com/spf13/cobra"
)

const (
	ServicesPath        = "services"
	ServiceTemplatePath = "templates/service"
)

type UpgradeStats struct {
	ManifestAppItems     int      // Number of items in manifest.Application
	ManifestServiceItems int      // Number of items in manifest.Services
	FilesProcessed       int      // Actual files copied
	FoldersProcessed     int      // Actual folders copied
	ItemsRemoved         int      // Items removed
	ItemsSkipped         int      // Items skipped
	PathsProcessed       []string // List of paths processed
	PathsSkipped         []string // List of paths skipped
	Warnings             []string // Warnings from post-upgrade hooks
}

// logOperation prints operation messages based on the operation type and dry-run mode.
func logOperation(operationType string, path string, isDir bool, dryRun bool) {
	var emoji, action, itemType string

	if isDir {
		itemType = "directory"
	} else {
		itemType = "file"
	}

	switch operationType {
	case "remove":
		emoji = "🗑️ "
		if dryRun {
			action = "Would remove"
		} else {
			action = "Removing"
		}
	case "copy":
		emoji = "📋"
		if dryRun {
			action = "Would copy"
		} else {
			action = "Copying"
		}
	case "replace":
		emoji = "📋"
		if dryRun {
			action = "Would replace"
		} else {
			action = "Copying"
		}
	default:
		return
	}

	fmt.Printf("  %s %s %s %s", emoji, action, itemType, path)
	if operationType == "copy" || operationType == "replace" {
		fmt.Print(" from template")
	}
	fmt.Println("...")
}

// processPathItem handles the common logic for removing and copying a path.
func processPathItem(targetPath, sourcePath string, dryRun bool, stats *UpgradeStats) error {
	// Check if target path exists
	pathExists := false
	if info, err := os.Stat(targetPath); err == nil {
		pathExists = true
		if dryRun {
			logOperation("remove", targetPath, info.IsDir(), dryRun)
		} else {
			logOperation("remove", targetPath, info.IsDir(), dryRun)
			if err := os.RemoveAll(targetPath); err != nil {
				return fmt.Errorf("failed to remove %s: %w", targetPath, err)
			}
			stats.ItemsRemoved++
		}
	}

	// Check if source path exists in template
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		fmt.Printf("  ⚠️  Warning: source path %s not found in template, skipping...\n", targetPath)
		stats.ItemsSkipped++
		stats.PathsSkipped = append(stats.PathsSkipped, targetPath)
		return err
	}

	// Copy from template
	if dryRun {
		if pathExists {
			logOperation("replace", targetPath, sourceInfo.IsDir(), dryRun)
		} else {
			logOperation("copy", targetPath, sourceInfo.IsDir(), dryRun)
		}
		if sourceInfo.IsDir() {
			stats.FoldersProcessed++
		} else {
			stats.FilesProcessed++
		}
	} else {
		logOperation("copy", targetPath, sourceInfo.IsDir(), dryRun)
		if err := tools.Copy(sourcePath, targetPath); err != nil {
			return fmt.Errorf("failed to copy %s from template: %w", targetPath, err)
		}
		if sourceInfo.IsDir() {
			stats.FoldersProcessed++
		} else {
			stats.FilesProcessed++
		}
		fmt.Printf("  %s updated\n", targetPath)
	}
	stats.PathsProcessed = append(stats.PathsProcessed, targetPath)
	return nil
}

// upgradeApplicationPaths processes application-level paths from the manifest.
func upgradeApplicationPaths(scaffoldingDir string, manifest *UpgradeManifest, dryRun bool, stats *UpgradeStats) error {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("Upgrading Application...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	for i, r := range manifest.Application {
		fmt.Printf("[%d/%d] Processing %s...\n", i+1, stats.ManifestAppItems, r.Path)
		sourcePath := filepath.Join(scaffoldingDir, r.Path)
		if err := processPathItem(r.Path, sourcePath, dryRun, stats); err != nil {
			return err
		}
	}
	return nil
}

// upgradeServicePaths processes service-level paths from the manifest.
func upgradeServicePaths(scaffoldingDir string, manifest *UpgradeManifest, dryRun bool, stats *UpgradeStats) error {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("Upgrading Services...")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	entries, err := os.ReadDir(ServicesPath)
	if err != nil {
		return fmt.Errorf("failed to read services directory: %w", err)
	}

	for _, e := range entries {
		// Entry must be a directory with a go.mod file inside to be treated as a service
		if !e.IsDir() {
			continue
		}

		sPath := filepath.Join(ServicesPath, e.Name())
		if _, err := os.Stat(filepath.Join(sPath, "go.mod")); os.IsNotExist(err) {
			fmt.Printf("  ⚠️  Skipping %s, go module not found\n", e.Name())
			continue
		}

		// Upgrade all paths for the current service
		for i, r := range manifest.Services {
			rPath := filepath.Join(sPath, r.Path)
			fmt.Printf("[%s - %d/%d] Processing %s...\n", e.Name(), i+1, stats.ManifestServiceItems, rPath)
			sourcePath := filepath.Join(scaffoldingDir, ServiceTemplatePath, r.Path)
			if err := processPathItem(rPath, sourcePath, dryRun, stats); err != nil {
				return err
			}
		}
	}
	return nil
}

func upgrade(scaffoldingDir string, manifest *UpgradeManifest, dryRun bool) (*UpgradeStats, error) {
	stats := &UpgradeStats{
		ManifestAppItems:     len(manifest.Application),
		ManifestServiceItems: len(manifest.Services),
		PathsProcessed:       make([]string, 0),
		PathsSkipped:         make([]string, 0),
	}

	// Upgrade application-level paths
	if err := upgradeApplicationPaths(scaffoldingDir, manifest, dryRun, stats); err != nil {
		return stats, err
	}

	// Upgrade service-level paths
	if err := upgradeServicePaths(scaffoldingDir, manifest, dryRun, stats); err != nil {
		return stats, err
	}

	if !dryRun {
		fmt.Println("\nAll template items updated successfully")
	}
	return stats, nil
}

// ============================================================================
// COBRA COMMANDS
// ============================================================================

var UpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade your project with the latest scaffolding template",
	Long: `Upgrades your Spire application by replacing project structure with the latest 
scaffolding template while preserving your customizations.

The upgrade is controlled by the upgrade manifest (.spire/upgrade-manifest.yaml),
which specifies:
  • Which Scaffolding version to use for the upgrade
  • Which files/folders to replace from the template
  • Which files/folders to preserve (keep) during the upgrade

Before running the upgrade:
  1. Ensure you have a clean git status (or use --force)
  2. Review/customize .spire/upgrade-manifest.yaml
  3. Consider creating a manual backup first

The upgrade process:
  1. Validates git status (skipped with --force)
  2. Downloads latest scaffolding template
  3. Creates automatic backup of files marked to 'keep'
  4. Removes and replaces files/directories per manifest
  5. Restores backed-up files to preserve customizations
  6. Runs post-upgrade hooks (go mod tidy, etc.)

Use --dry-run to preview changes without applying them.`,
	Example: `  spire upgrade
  spire upgrade --dry-run
  spire upgrade --force`,
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)

		// Get flags
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")

		if dryRun {
			fmt.Println("DRY RUN MODE - No changes will be applied")
			fmt.Println()
		}

		// Check git status unless force flag is set
		if err := checkGitStatus(force); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		manifest, err := loadUpgradeManifest()
		if err != nil {
			return
		}

		// Run before-upgrade plugins.
		if !dryRun {
			hookCtx := buildUpgradeHookContext()
			fmt.Println("\nRunning before-upgrade plugins...")
			if err := plugin.RunHook(plugin.HookBeforeUpgrade, hookCtx); err != nil {
				fmt.Printf("❌ before-upgrade plugin failed: %v\n", err)
				return
			}
		}

		// --- Build template source ---
		templateRepo, _ := cmd.Flags().GetString("template-repo")
		templateLocalPath, _ := cmd.Flags().GetString("template-local")
		var src templatesource.Source
		if templateLocalPath != "" {
			src = templatesource.NewLocalSource(templateLocalPath)
		} else {
			if templateRepo == "" {
				// Try to use the git repository from the Spire manifest.
				if spireManifest, err := mfst.LoadManifest(); err == nil && spireManifest.GitRepository != "" {
					templateRepo = spireManifest.GitRepository
				} else {
					fmt.Println("❌ Error: --template-repo is required (no git repository found in Spire manifest)")
					return
				}
			}
			src = templatesource.NewGitSource(templateRepo)
		}

		// Prompt user to select the scaffolding version to upgrade to
		TemplateVersion, err := application.PromptTemplateVersion(src)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		// Download and extract scaffolding template
		scaffoldingDir, err := src.Download(context.Background(), TemplateVersion)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
		defer src.Cleanup()

		// Create backup (or simulate in dry-run)
		var backupDir string
		if dryRun {
			previewBackup(manifest)
		} else {
			backupDir, err = createBackup(manifest)
		}
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		// Proceed with upgrade (or simulate in dry-run)
		stats, err := upgrade(scaffoldingDir, manifest, dryRun)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		if dryRun {
			printUpgradeSummary(stats, manifest, true, "")
			return
		}

		// Restore backed up files
		if err := restoreBackup(backupDir); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}

		// Run post-upgrade hooks
		warnings, err := runPostUpgradeHooks(TemplateVersion)
		if err != nil {
			fmt.Printf("❌ Error: Post-upgrade hooks failed: %v\n", err)
			return
		}
		stats.Warnings = warnings

		// Run after-upgrade plugins.
		hookCtx := buildUpgradeHookContext()
		fmt.Println("\nRunning after-upgrade plugins...")
		if err := plugin.RunHook(plugin.HookAfterUpgrade, hookCtx); err != nil {
			fmt.Printf("❌ after-upgrade plugin failed: %v\n", err)
			return
		}

		printUpgradeSummary(stats, nil, false, backupDir)

	},
}

func printUpgradeSummary(stats *UpgradeStats, manifest *UpgradeManifest, dryRun bool, backupDir string) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	if dryRun {
		fmt.Println("                 DRY RUN SUMMARY")
	} else {
		if len(stats.Warnings) > 0 {
			fmt.Println("         UPGRADE COMPLETED WITH WARNINGS")
		} else {
			fmt.Println("              UPGRADE COMPLETED SUCCESSFULLY")
		}
	}
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("📊 Upgrade Statistics:\n")
	fmt.Printf("   • Manifest items:           %d\n", stats.ManifestAppItems)

	if dryRun {
		fmt.Printf("   • Files to update:          %d\n", stats.FilesProcessed)
		fmt.Printf("   • Folders to update:        %d\n", stats.FoldersProcessed)
	} else {
		fmt.Printf("   • Files updated:            %d\n", stats.FilesProcessed)
		fmt.Printf("   • Folders updated:          %d\n", stats.FoldersProcessed)
		fmt.Printf("   • Items removed:            %d\n", stats.ItemsRemoved)
	}

	if stats.ItemsSkipped > 0 {
		if dryRun {
			fmt.Printf("   • Items to skip:            %d\n", stats.ItemsSkipped)
		} else {
			fmt.Printf("   • Items skipped:            %d\n", stats.ItemsSkipped)
		}
	}
	fmt.Println()

	// Show files to keep (dry run only)
	if dryRun && manifest != nil {
		totalKeepItems := 0
		for _, r := range manifest.Application {
			totalKeepItems += len(r.Keep)
		}
		if totalKeepItems > 0 {
			fmt.Printf("Files to preserve:        %d\n", totalKeepItems)
			fmt.Println()
		}
	}

	// Show backup location (actual upgrade only)
	if !dryRun && backupDir != "" {
		fmt.Printf("Backup location: %s\n", backupDir)
		fmt.Println()
	}

	if len(stats.PathsSkipped) > 0 {
		fmt.Println("⚠️  Skipped paths:")
		for _, path := range stats.PathsSkipped {
			fmt.Printf("   • %s\n", path)
		}
		fmt.Println()
	}

	if len(stats.Warnings) > 0 {
		fmt.Println("⚠️  Warnings:")
		for _, warning := range stats.Warnings {
			fmt.Printf("   • %s\n", warning)
		}
		fmt.Println()
	}

	if dryRun {
		fmt.Println("Dry run completed - no actual changes were made")
		fmt.Println()
		fmt.Println("To apply these changes, run:")
		fmt.Println("   spire upgrade")
	} else {
		fmt.Println("📋 Next step: Review changes:    git status && git diff")
	}
	fmt.Println()
}

// buildUpgradeHookContext constructs a plugin.HookContext from the current Spire manifest.
// Errors loading the manifest are silently ignored — plugins still run with whatever is available.
func buildUpgradeHookContext() plugin.HookContext {
	workDir, _ := os.Getwd()
	ctx := plugin.HookContext{
		WorkDir:  workDir,
		Slots:    map[string]string{},
		Services: []plugin.ServiceInfo{},
	}
	if m, err := mfst.LoadManifest(); err == nil {
		for _, s := range m.AppSlots {
			ctx.Slots[s.Key] = s.Value
		}
		for _, svc := range m.Services {
			svcSlots := make(map[string]string, len(svc.Slots))
			for _, s := range svc.Slots {
				svcSlots[s.Key] = s.Value
			}
			ctx.Services = append(ctx.Services, plugin.ServiceInfo{
				Name:     svc.Name,
				SlugName: svc.SlugName,
				Slots:    svcSlots,
			})
		}
	}
	return ctx
}

func init() {
	UpgradeCmd.Flags().Bool("dry-run", false, "Preview the upgrade without making changes")
	UpgradeCmd.Flags().Bool("force", false, "Force upgrade even with uncommitted changes (not recommended)")
	UpgradeCmd.Flags().String("template-repo", "", "Template repository git URL (auto-detected from Spire manifest if not set)")
	UpgradeCmd.Flags().String("template-local", "", "Use a local directory as template source instead of a git repository")
}
