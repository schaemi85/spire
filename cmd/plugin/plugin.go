package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	commoncmds "github.com/schaemi85/spire/cmd/common"
	"github.com/schaemi85/spire/internal/manifest"
	"github.com/schaemi85/spire/internal/plugin"

	"github.com/spf13/cobra"
)

// PluginCmd is the root for all `spire plugin` subcommands.
var PluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage Spire plugins",
	Long: `Manage lifecycle plugins that extend Spire with custom behaviour.

Plugins are compiled Go (or any language) binaries placed in
  .spire/plugins/<hook-name>/

They are invoked at predefined lifecycle hook points and receive a JSON
HookContext on stdin. Supported hooks:

  before-add-service   runs before the service template is copied
  after-add-service    runs after the service is fully set up
  before-upgrade       runs before the upgrade starts
  after-upgrade        runs after the upgrade and post-upgrade hooks complete

Plugin sources can be kept in templates/plugins/<hook>/<name>/ and compiled
with 'spire plugin build'.`,
}

// pluginListCmd lists every installed plugin binary.
var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all installed plugin binaries",
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)

		plugins, err := plugin.ListPlugins()
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		if len(plugins) == 0 {
			fmt.Println("No plugins installed in .spire/plugins/")
			fmt.Println("\nTo install plugins either:")
			fmt.Println("  • Place compiled binaries in .spire/plugins/<hook-name>/")
			fmt.Println("  • Add sources to templates/plugins/<hook-name>/<name>/ and run 'spire plugin build'")
			return
		}

		fmt.Println("Installed plugins:")
		fmt.Println()
		for _, hookName := range plugin.AllHookNames {
			names, ok := plugins[hookName]
			if !ok {
				continue
			}
			fmt.Printf("  %s\n", hookName)
			for _, name := range names {
				fmt.Printf("    • %s\n", name)
			}
		}
	},
}

// pluginBuildCmd compiles plugin sources from templates/plugins/.
var pluginBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Compile plugin sources from templates/plugins/",
	Long: `Compile every plugin source directory found under templates/plugins/<hook>/<name>/
and place the resulting binary in .spire/plugins/<hook>/<name>[.exe].

Each source directory must contain a Go package with a main function and its own go.mod.`,
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)

		fmt.Println("Building plugins...")
		fmt.Println()
		built, err := plugin.BuildPlugins()
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}

		if built == 0 {
			fmt.Println("No plugin sources found in templates/plugins/")
		} else {
			fmt.Printf("\n%d plugin(s) built successfully\n", built)
		}
	},
}

// pluginRunContextFile is the optional --context flag for `spire plugin run`.
var pluginRunContextFile string

// pluginRunCmd manually triggers all plugins for a given hook.
var pluginRunCmd = &cobra.Command{
	Use:   "run <hook>",
	Short: "Manually run all plugins for a given hook",
	Long: `Run all installed plugins for the specified hook.

By default the context is built from the current project manifest. For testing,
pass --context <file> to load a HookContext from a JSON file instead — this lets
you exercise a plugin with arbitrary services, slots and WorkDir without a manifest.

Available hooks: before-add-service, after-add-service, before-upgrade, after-upgrade`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)
		hookName := plugin.HookName(args[0])

		var ctx plugin.HookContext
		if pluginRunContextFile != "" {
			loaded, err := loadContextFile(pluginRunContextFile)
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				os.Exit(1)
			}
			ctx = loaded
			fmt.Printf("Loaded context from %s\n", pluginRunContextFile)
		} else {
			ctx = buildContextFromManifest()
		}

		fmt.Printf("Running plugins for hook: %s\n\n", hookName)
		if err := plugin.RunHook(hookName, ctx); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\nDone")
	},
}

// loadContextFile reads a HookContext from a JSON file, defaulting nil maps and
// slices so plugins can safely index into them.
func loadContextFile(path string) (plugin.HookContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return plugin.HookContext{}, fmt.Errorf("cannot read context file %q: %w", path, err)
	}
	var ctx plugin.HookContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return plugin.HookContext{}, fmt.Errorf("cannot parse context file %q: %w", path, err)
	}
	if ctx.WorkDir == "" {
		ctx.WorkDir, _ = os.Getwd()
	}
	if ctx.Slots == nil {
		ctx.Slots = map[string]string{}
	}
	if ctx.Services == nil {
		ctx.Services = []plugin.ServiceInfo{}
	}
	return ctx, nil
}

// buildContextFromManifest constructs a HookContext from the current project
// manifest, falling back to an empty context when no manifest is present.
func buildContextFromManifest() plugin.HookContext {
	workDir, _ := os.Getwd()
	ctx := plugin.HookContext{
		WorkDir:  workDir,
		Slots:    map[string]string{},
		Services: []plugin.ServiceInfo{},
	}

	if m, err := manifest.LoadManifest(); err == nil {
		for _, s := range m.AppSlots {
			ctx.Slots[s.Key] = s.Value
		}
		for _, svc := range m.Services {
			svcInfo := plugin.ServiceInfo{
				Name:     svc.Name,
				SlugName: svc.SlugName,
				Slots:    make(map[string]string),
			}
			for _, slot := range svc.Slots {
				svcInfo.Slots[slot.Key] = slot.Value
			}
			ctx.Services = append(ctx.Services, svcInfo)
		}
	}
	return ctx
}

// pluginDebugCmd prints a ready-to-paste VS Code launch configuration and the
// follow-up steps for debugging a plugin, exactly like Terraform provider
// debugging: press F5 to run the plugin under the debugger, copy the reattach
// var it prints, then run a Spire command with that var set.
var pluginDebugCmd = &cobra.Command{
	Use:   "debug <hook> <name>",
	Short: "Print a VS Code launch config for debugging a plugin (Terraform-style)",
	Long: `Print the VS Code launch configuration and steps for debugging a plugin.

This mirrors how Terraform provider debugging works:
  1. Add the printed launch config to .vscode/launch.json and press F5 — VS Code
     compiles and runs the plugin under Delve with the -debug flag.
  2. The plugin prints a SPIRE_REATTACH_PLUGINS=... line in the Debug Console. Copy it.
  3. Run any Spire command with that env var set — the CLI connects to your
     already-running, debugger-attached plugin instead of spawning a new subprocess.

No 'spire plugin build' needed: "mode": "debug" compiles from source.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)
		hook := args[0]
		name := args[1]

		sourcePath := filepath.Join(plugin.PluginSourcesDir, hook, name)
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			fmt.Printf("⚠️  Plugin source not found: %s\n", sourcePath)
			fmt.Printf("   Expected a Go package at templates/plugins/%s/%s/\n", hook, name)
			os.Exit(1)
		}

		// VS Code resolves "program" relative to the workspace root; use forward
		// slashes so the snippet works on every platform.
		program := "${workspaceFolder}/" + filepath.ToSlash(sourcePath)

		fmt.Printf("Debug: %s/%s\n\n", hook, name)

		fmt.Println("── Step 1: add this to .vscode/launch.json, then press F5 ─────────────────")
		fmt.Println()
		fmt.Printf("    {\n")
		fmt.Printf("      \"name\": \"Debug plugin: %s\",\n", name)
		fmt.Printf("      \"type\": \"go\",\n")
		fmt.Printf("      \"request\": \"launch\",\n")
		fmt.Printf("      \"mode\": \"debug\",\n")
		fmt.Printf("      \"program\": \"%s\",\n", program)
		fmt.Printf("      \"args\": [\"-debug\"]\n")
		fmt.Printf("    }\n")
		fmt.Println()
		fmt.Println("  VS Code compiles the plugin and runs it under Delve. Set breakpoints in")
		fmt.Printf("  %s before continuing.\n", filepath.Join(sourcePath, "main.go"))
		fmt.Println()

		fmt.Println("── Step 2: copy the reattach line from the Debug Console ──────────────────")
		fmt.Println()
		fmt.Println("  The plugin prints:")
		fmt.Println()
		fmt.Println("    SPIRE_REATTACH_PLUGINS='{\"" + name + "\":{...}}'")
		fmt.Println()

		fmt.Println("── Step 3: trigger the hook via Spire ────────────────────────────────────")
		fmt.Println()
		fmt.Println("  In a terminal, set that env var and run any command that fires the hook:")
		fmt.Println()
		fmt.Printf("    SPIRE_REATTACH_PLUGINS='...' spire plugin run %s\n", hook)
		fmt.Println()
		fmt.Printf("  (or 'spire service add', 'spire upgrade', etc.)\n")
		fmt.Println()
		fmt.Println("  The CLI connects to your running process and Delve pauses at your breakpoints.")
		fmt.Println("  Re-run the command to hit them again without restarting the debug session.")
	},
}

func init() {
	pluginRunCmd.Flags().StringVar(&pluginRunContextFile, "context", "", "path to a HookContext JSON file to use instead of the manifest (for testing)")

	PluginCmd.AddCommand(pluginListCmd)
	PluginCmd.AddCommand(pluginBuildCmd)
	PluginCmd.AddCommand(pluginRunCmd)
	PluginCmd.AddCommand(pluginDebugCmd)
}
