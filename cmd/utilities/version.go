package utilities

import (
	"fmt"

	"github.com/schaemi85/spire/internal/metadata"

	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Spire CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Spire CLI", metadata.VERSION)
	},
}
