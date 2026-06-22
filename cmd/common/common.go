package commoncmds

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// commoncmds.SwitchToWorkdir changes to the specified working directory if provided.
func SwitchToWorkdir(cmd *cobra.Command) {
	// Set working directory if --workdir is provided
	workdir, _ := cmd.Root().Flags().GetString("workdir")
	if workdir != "" {
		if err := os.Chdir(workdir); err != nil {
			fmt.Printf("❌ Error: Cannot change to directory %s: %v\n", workdir, err)
			os.Exit(1)
		}
	}
}

func EnsureGitIsInstalled() {
	// Check if 'git' command is available
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Println("❌ Error: Git is required but was not found in your PATH.")
		fmt.Println("   Please install Git from Software Center and try again.")
		fmt.Println("   Configuration instruction: https://gitlab.com/help")
		os.Exit(1)
	}
}
