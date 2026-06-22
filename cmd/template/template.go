package template

import "github.com/spf13/cobra"

// TemplateCmd is the parent for all `spire template` subcommands.
var TemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Template authoring tools",
	Long:  `Commands for creating and maintaining reusable Spire templates.`,
}

func init() {
	TemplateCmd.AddCommand(SyncCmd)
}
