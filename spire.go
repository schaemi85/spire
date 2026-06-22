package main

import (
	"fmt"
	"os"

	"github.com/schaemi85/spire/cmd/application"
	manifestcmd "github.com/schaemi85/spire/cmd/manifest"
	plugincmd "github.com/schaemi85/spire/cmd/plugin"
	"github.com/schaemi85/spire/cmd/service"
	tmpl "github.com/schaemi85/spire/cmd/template"
	"github.com/schaemi85/spire/cmd/upgrade"
	"github.com/schaemi85/spire/cmd/utilities"

	"github.com/spf13/cobra"
)

const (
	Project   = "project"
	Maintain  = "maintain"
	Authoring = "authoring"
	Plugins   = "plugins"
)

var (
	workdir string
	nint    bool
)

func main() {
	cobra.EnableCommandSorting = false
	if err := Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "spire",
	Short: "Spire CLI",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&workdir, "workdir", "", "Directory to run CLI operations in (defaults to current directory)")
	rootCmd.PersistentFlags().BoolVar(&nint, "non-interactive", false, "Disable interactive prompts. Useful for automation and CI/CD pipelines.")

	rootCmd.AddGroup(&cobra.Group{ID: Project, Title: "Project"})
	rootCmd.AddGroup(&cobra.Group{ID: Maintain, Title: "Maintenance"})
	rootCmd.AddGroup(&cobra.Group{ID: Authoring, Title: "Template Authoring"})
	rootCmd.AddGroup(&cobra.Group{ID: Plugins, Title: "Plugins"})

	application.InitApplicationCmd.GroupID = Project
	service.ServiceCmd.GroupID = Project
	upgrade.UpgradeCmd.GroupID = Maintain
	upgrade.BackupCmd.GroupID = Maintain
	tmpl.TemplateCmd.GroupID = Authoring
	manifestcmd.ManifestCmd.GroupID = Authoring
	plugincmd.PluginCmd.GroupID = Plugins

	rootCmd.AddCommand(application.InitApplicationCmd)
	rootCmd.AddCommand(service.ServiceCmd)
	rootCmd.AddCommand(upgrade.UpgradeCmd)
	rootCmd.AddCommand(upgrade.BackupCmd)
	rootCmd.AddCommand(tmpl.TemplateCmd)
	rootCmd.AddCommand(manifestcmd.ManifestCmd)
	rootCmd.AddCommand(plugincmd.PluginCmd)
	rootCmd.AddCommand(utilities.VersionCmd)

	rootCmd.CompletionOptions.DisableDefaultCmd = false
}
