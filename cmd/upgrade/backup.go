package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	commoncmds "github.com/schaemi85/spire/cmd/common"
	"github.com/schaemi85/spire/internal/tools"

	"github.com/spf13/cobra"
)

const (
	BackupLocation = ".spire/backup/"
)

func createBackup(manifest *UpgradeManifest) (string, error) {

	fmt.Println("Backup all files or folders to keep for the upgrade.")

	// Generate timestamp and backup directory name
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(BackupLocation, timestamp)

	fmt.Printf("Backup name: %s\n", backupDir)
	fmt.Println()

	// Create backup directory
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return backupDir, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Count total items to backup
	totalBackupItems := 0
	for _, r := range manifest.Application {
		totalBackupItems += len(r.Keep)
	}
	for _, r := range manifest.Services {
		totalBackupItems += len(r.Keep)
	}

	if totalBackupItems == 0 {
		fmt.Println("No items marked for backup")
		return backupDir, nil
	}

	// Backup directories
	fmt.Printf("📋 Creating Backup (%d items)...\n\n", totalBackupItems)
	backedUpCount := 0
	skippedCount := 0

	for _, r := range manifest.Application {
		if _, err := os.Stat(r.Path); err == nil {
			for _, keepPath := range r.Keep {
				gRoot := tools.GlobRoot(keepPath)
				source := filepath.Join(r.Path, keepPath)
				target := filepath.Join(backupDir, r.Path, gRoot)

				if err := tools.Copy(source, target); err != nil {
					fmt.Printf("  ⚠️  Failed to backup %s: %v\n", source, err)
					skippedCount++
					continue
				}
				fmt.Printf("  %s\n", source)
				backedUpCount++
			}
		}
	}
	// Loop over services
	entries, _ := os.ReadDir(ServicesPath)
	for _, e := range entries {
		// entry must be a directory with a go.mod file inside to be treated as a service
		if e.IsDir() {
			sPath := filepath.Join(ServicesPath, e.Name())
			if _, err := os.Stat(filepath.Join(sPath, "go.mod")); os.IsNotExist(err) {
				fmt.Printf("  ⚠️  Skipping %s, go module not found\n", e.Name())
				continue
			}
			// Loop over services path to keep
			for _, r := range manifest.Services {
				rPath := filepath.Join(sPath, r.Path)
				if _, err := os.Stat(rPath); err == nil {
					for _, keepPath := range r.Keep {
						gRoot := tools.GlobRoot(keepPath)
						source := filepath.Join(rPath, keepPath)
						target := filepath.Join(backupDir, rPath, gRoot)

						if err := tools.Copy(source, target); err != nil {
							fmt.Printf("  ⚠️  Failed to backup %s: %v\n", source, err)
							skippedCount++
							continue
						}
						fmt.Printf("  %s\n", source)
						backedUpCount++
					}
				}
			}
		}

	}

	fmt.Println()
	fmt.Printf("Backup completed: %d items backed up\n", backedUpCount)
	fmt.Printf("   Location: %s\n", backupDir)
	if skippedCount > 0 {
		fmt.Printf("   Skipped: %d items\n", skippedCount)
	}
	fmt.Println()

	return backupDir, nil
}

func previewBackup(manifest *UpgradeManifest) {
	fmt.Println("Preview: Files that would be backed up")
	fmt.Println()

	// Count total items to backup
	totalBackupItems := 0
	for _, r := range manifest.Application {
		totalBackupItems += len(r.Keep)
	}

	if totalBackupItems == 0 {
		fmt.Println("No items marked for backup")
		return
	}

	fmt.Printf("📋 Would backup %d items:\n", totalBackupItems)

	for _, r := range manifest.Application {
		if _, err := os.Stat(r.Path); err == nil {
			for _, keepPath := range r.Keep {
				source := filepath.Join(r.Path, keepPath)
				// Check if the source actually exists
				if _, err := os.Stat(source); err == nil {
					fmt.Printf("  %s\n", source)
				} else {
					fmt.Printf("  ⚠️  %s (not found, would skip)\n", source)
				}
			}
		}
	}
	// Loop over services
	entries, _ := os.ReadDir(ServicesPath)
	for _, e := range entries {
		// entry must be a directory with a go.mod file inside to be treated as a service
		if e.IsDir() {
			sPath := filepath.Join(ServicesPath, e.Name())
			if _, err := os.Stat(filepath.Join(sPath, "go.mod")); os.IsNotExist(err) {
				fmt.Printf("  ⚠️  Skipping %s, go module not found\n", e.Name())
				continue
			}
			for _, r := range manifest.Services {
				rPath := filepath.Join(sPath, r.Path)
				if _, err := os.Stat(rPath); err == nil {
					for _, keepPath := range r.Keep {
						source := filepath.Join(rPath, keepPath)
						// Check if the source actually exists
						if _, err := os.Stat(source); err == nil {
							fmt.Printf("  %s\n", source)
						} else {
							fmt.Printf("  ⚠️  %s (not found, would skip)\n", source)
						}
					}
				}
			}
		}
	}
	fmt.Println()
}

func restoreBackup(backupDir string) error {
	if backupDir == "" {
		return fmt.Errorf("backup directory not specified")
	}

	// Check if backup exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		fmt.Printf("❌ Error: Backup directory not found: %s\n", backupDir)
		fmt.Println()
		return fmt.Errorf("backup not found")
	}

	fmt.Println()
	fmt.Printf("Restoring from backup: %s\n", backupDir)

	if err := tools.Copy(backupDir, "."); err != nil {
		return fmt.Errorf("failed to restore backup from %s: %w", backupDir, err)
	}

	fmt.Println("Restore completed successfully!")
	fmt.Println()
	return nil
}

// ListBackups lists all available backups.
func listBackups() error {
	fmt.Println("Available backups:")
	fmt.Println()

	// Find all backup directories
	entries, err := os.ReadDir(BackupLocation)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			backups = append(backups, entry.Name())
		}
	}

	if len(backups) == 0 {
		fmt.Println("  (no backups found)")
		fmt.Println()
		fmt.Println("💡 To create a backup:")
		fmt.Println("   spire backup create")
		return nil
	}

	// Sort backups by date (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))

	for _, backup := range backups {
		backupPath := filepath.Join(BackupLocation, backup)
		size := tools.CalculateDirSize(backupPath)

		// Get directory modification time
		info, err := os.Stat(backupPath)
		var modTime string
		if err == nil {
			modTime = info.ModTime().Format("2006-01-02 15:04:05")
		} else {
			modTime = "unknown"
		}

		fmt.Printf("  📁 %s (Size: %s, Modified: %s)\n", backup, size, modTime)
	}

	fmt.Println()
	fmt.Println("To restore a backup:")
	fmt.Println("   spire backup restore --backup=<backup-name>")

	return nil
}

// CleanOldBackups removes old backups (keeps last N).
func CleanOldBackups(keepCount int, interactive bool) error {
	fmt.Println("🗑️  Cleaning old backups...")
	fmt.Println()

	// Find all backup directories
	entries, err := os.ReadDir(BackupLocation)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			backups = append(backups, entry.Name())
		}
	}

	if len(backups) == 0 {
		fmt.Println("  (no backups found)")
		return nil
	}

	fmt.Printf("Found %d backup(s)\n", len(backups))

	if len(backups) <= keepCount {
		fmt.Printf("No cleanup needed (keeping last %d backups)\n", keepCount)
		return nil
	}

	// Sort backups by date (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(backups)))

	// Backups to delete
	toDelete := backups[keepCount:]

	fmt.Println()
	fmt.Printf("Will delete %d old backup(s):\n", len(toDelete))
	for _, backup := range toDelete {
		backupPath := filepath.Join(BackupLocation, backup)
		size := tools.CalculateDirSize(backupPath)
		fmt.Printf("  🗑️  %s (%s)\n", backup, size)
	}

	// Confirmation in interactive mode
	if interactive {
		fmt.Println()
		confirmed, err := tools.ConfirmAction("Continue?")
		if err != nil {
			fmt.Println("❌ Cleanup cancelled")
			return err
		}
		if !confirmed {
			fmt.Println("❌ Cleanup cancelled")
			return fmt.Errorf("cleanup cancelled by user")
		}
	}

	// Delete old backups
	fmt.Println()
	for _, backup := range toDelete {
		backupPath := filepath.Join(BackupLocation, backup)
		if err := os.RemoveAll(backupPath); err != nil {
			fmt.Printf("  ⚠️  Failed to delete %s: %v\n", backup, err)
			continue
		}
		fmt.Printf("  Deleted: %s\n", backup)
	}

	fmt.Println()
	fmt.Printf("Cleanup complete! Kept last %d backups\n", keepCount)

	return nil
}

var BackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup and restore operations for Spire application upgrades",
	Long: `Manage backups of your Spire application based on the upgrade manifest.

The upgrade manifest (.spire/upgrade-manifest.yaml) defines which files and 
folders should be preserved during an upgrade. If the manifest doesn't exist,
a default one is automatically created with common paths to keep.

Before running an upgrade, you should:
  1. Review the generated manifest
  2. Customize the 'keep' sections to preserve your custom files
  3. Create a backup to safely store these files

Available operations: create, restore, list, clean`,
}

var BackupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a backup of files marked for preservation in the upgrade manifest",
	Long: `Creates a timestamped backup of all files/folders marked to 'keep' in the 
upgrade manifest (.spire/upgrade-manifest.yaml).

If the manifest doesn't exist, a default one is automatically generated. 
You should review and customize it before creating backups to ensure all 
your custom configurations and files are preserved.

The backup is stored in .spire/backup/<timestamp>/ and can be restored later.`,
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)
		manifest, err := loadUpgradeManifest()
		if err != nil {
			return
		}
		if _, err := createBackup(manifest); err != nil {
			fmt.Printf("❌ Backup failed: %v\n", err)
			return
		}

	},
}

var BackupRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore files from a previous backup",
	Long:  fmt.Sprintf("Restores files from a backup created earlier.\n\nThis will copy all files from the specified backup directory (%s<timestamp>/)\nback to their original locations in your project.\n\nUse 'spire backup list' to see available backups.", BackupLocation),
	Example: `  spire backup restore --backup 20250109-143022
  spire backup restore --backup 20250109-143022 --non-interactive`,
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)
		backup, _ := cmd.Flags().GetString("backup")
		backupDir := filepath.Join(BackupLocation, backup)
		nint, _ := cmd.Root().Flags().GetBool("non-interactive")

		if backup == "" {
			fmt.Printf("❌ backup directory not specified")
			return
		}

		// Check if backup exists
		if _, err := os.Stat(backupDir); os.IsNotExist(err) {
			fmt.Printf("❌ Error: Backup directory not found: %s\n", backup)
			fmt.Println()
			fmt.Println("Available backups:")
			err2 := listBackups()
			if err2 != nil {
				fmt.Printf("  (error lispire backups: %v)\n", err2)
			}
			return
		}

		// Confirmation in interactive mode
		if !nint {
			fmt.Println("⚠️  WARNING: This will replace current files with the backup")
			fmt.Println()
			confirmed, err := tools.ConfirmAction("Are you sure you want to continue?")
			if err != nil {
				fmt.Printf("❌ Restore cancelled: %v\n", err)
				return
			}
			if !confirmed {
				fmt.Printf("❌ Restore cancelled: %v\n", err)
				return
			}
		}

		if err := restoreBackup(backupDir); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return
		}
	},
}

var BackupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available backups",
	Long: `Lists all backup directories with their timestamp, size, and modification date.

Backups are sorted from newest to oldest. Use the timestamp shown to restore
a specific backup with 'spire backup restore --backup <timestamp>'.`,
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)
		if err := listBackups(); err != nil {
			fmt.Printf("❌ Failed to list backups: %v\n", err)
			return
		}
	},
}

var BackupCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove old backups to free up disk space",
	Long: `Removes old backup directories while keeping the most recent ones.

By default, keeps the last 2 backups. You can specify a different number
with the --keep flag. In interactive mode (default), you'll be asked to
confirm before deletion.`,
	Example: `  spire backup clean
  spire backup clean --keep 3
  spire backup clean --keep 5 --non-interactive`,
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)
		keepCount, _ := cmd.Flags().GetInt("keep")
		nint, _ := cmd.Root().Flags().GetBool("non-interactive")

		if err := CleanOldBackups(keepCount, !nint); err != nil {
			fmt.Printf("❌ Cleanup failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	BackupCmd.AddCommand(BackupCreateCmd)
	BackupCmd.AddCommand(BackupRestoreCmd)
	BackupCmd.AddCommand(BackupListCmd)
	BackupCmd.AddCommand(BackupCleanCmd)

	// Flags for restore command
	BackupRestoreCmd.Flags().String("backup", "", "Backup directory to restore from (e.g., 20250109-143022)")
	if err := BackupRestoreCmd.MarkFlagRequired("backup"); err != nil {
		fmt.Printf("[WARN] Could not mark 'backup' flag as required: %v\n", err)
	}

	// Flags for clean command
	BackupCleanCmd.Flags().Int("keep", 2, "Number of recent backups to keep")
}
