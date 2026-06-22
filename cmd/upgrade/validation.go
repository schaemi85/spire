package upgrade

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// checkGitStatus verifies the working directory is clean.
// Returns an error if there are uncommitted changes.
func checkGitStatus(force bool) error {
	// Check if .git directory exists
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository (missing .git directory)")
	}

	// Check for uncommitted changes
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// If there's any output, there are uncommitted changes
	outputStr := strings.TrimSpace(string(output))
	if len(outputStr) > 0 {
		// Count the changes
		lines := strings.Split(outputStr, "\n")
		changeCount := len(lines)

		if force {
			fmt.Printf("⚠️  WARNING: Force mode enabled - %d uncommitted change(s) detected\n", changeCount)

			// Show first few changes
			maxShow := 5
			fmt.Println("   Uncommitted changes:")
			for i, line := range lines {
				if i >= maxShow {
					fmt.Printf("   ... and %d more\n", len(lines)-maxShow)
					break
				}
				fmt.Printf("   %s\n", line)
			}
			fmt.Println()
			return nil
		}

		return fmt.Errorf("working directory is not clean - %d uncommitted change(s) found\n"+
			"   Commit or stash your changes before upgrading\n"+
			"   Use --force to bypass this check (not recommended)", changeCount)
	}

	fmt.Println("Working directory is clean")
	return nil
}
