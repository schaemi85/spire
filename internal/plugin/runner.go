package plugin

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	hclog "github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	sdk "github.com/schaemi85/spire/plugin/sdk"
	"gopkg.in/yaml.v3"
)

const PluginsRootDir = ".spire/plugins"
const PluginSourcesDir = "templates/plugins"

// OrderFile is the optional per-project file that defines plugin execution
// order. It lives at .spire/plugins/order.yaml and maps each hook name to an
// ordered list of plugin names:
//
//	after-add-service:
//	  - create-db-schema
//	  - create-db-user
//
// Plugins listed here run first, in the given order. Any installed plugin not
// listed runs afterwards in alphabetical order. The file is optional — without
// it, all plugins run alphabetically.
const OrderFile = "order.yaml"

// loadPluginOrder reads .spire/plugins/order.yaml. It returns an empty map when
// the file is absent so callers can treat "no file" and "empty file" alike.
func loadPluginOrder() map[string][]string {
	path := filepath.Join(PluginsRootDir, OrderFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string][]string{}
	}
	var order map[string][]string
	if err := yaml.Unmarshal(data, &order); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  %s is malformed, ignoring plugin order: %v\n", path, err)
		return map[string][]string{}
	}
	return order
}

// orderNames returns names sorted by the desired execution order for a hook:
// names listed in the order spec come first (in that order), followed by any
// remaining names alphabetically. Unknown names in the spec are ignored.
func orderNames(names []string, desired []string) []string {
	present := make(map[string]bool, len(names))
	for _, n := range names {
		present[n] = true
	}

	ordered := make([]string, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, n := range desired {
		if present[n] && !seen[n] {
			ordered = append(ordered, n)
			seen[n] = true
		}
	}

	rest := make([]string, 0, len(names))
	for _, n := range names {
		if !seen[n] {
			rest = append(rest, n)
		}
	}
	sort.Strings(rest)
	return append(ordered, rest...)
}

// reattachAddr implements net.Addr for a deserialized SPIRE_REATTACH_PLUGINS entry.
type reattachAddr struct {
	network string
	addr    string
}

func (r reattachAddr) Network() string { return r.network }
func (r reattachAddr) String() string  { return r.addr }

// reattachEntry is the per-plugin JSON shape inside SPIRE_REATTACH_PLUGINS.
type reattachEntry struct {
	Protocol        string `json:"Protocol"`
	ProtocolVersion int    `json:"ProtocolVersion"`
	Pid             int    `json:"Pid"`
	Test            bool   `json:"Test"`
	Addr            struct {
		Network string `json:"Network"`
		String  string `json:"String"`
	} `json:"Addr"`
}

// loadReattachPlugins parses the SPIRE_REATTACH_PLUGINS environment variable.
// Returns nil when the variable is unset.
func loadReattachPlugins() map[string]*goplugin.ReattachConfig {
	raw := os.Getenv("SPIRE_REATTACH_PLUGINS")
	if raw == "" {
		return nil
	}
	var entries map[string]reattachEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  SPIRE_REATTACH_PLUGINS is malformed: %v\n", err)
		return nil
	}
	result := make(map[string]*goplugin.ReattachConfig, len(entries))
	for name, e := range entries {
		var addr net.Addr
		switch e.Addr.Network {
		case "tcp", "tcp4", "tcp6":
			addr, _ = net.ResolveTCPAddr(e.Addr.Network, e.Addr.String)
		default:
			addr = reattachAddr{network: e.Addr.Network, addr: e.Addr.String}
		}
		result[name] = &goplugin.ReattachConfig{
			Protocol:        goplugin.Protocol(e.Protocol),
			ProtocolVersion: e.ProtocolVersion,
			Pid:             e.Pid,
			Test:            e.Test,
			Addr:            addr,
		}
	}
	return result
}

// RunHook discovers and runs every plugin binary registered under
// .spire/plugins/<hook>/ in lexicographic order.
//
// Each plugin is launched as a subprocess via go-plugin (net/rpc).
// When SPIRE_REATTACH_PLUGINS is set (debug mode), matching plugins are
// connected to their already-running process instead of spawning a new one.
// The plugin's stderr is forwarded to the terminal; stdout is reserved for the
// RPC handshake and should not be written to directly by the plugin.
func RunHook(hook HookName, ctx HookContext) error {
	ctx.Hook = string(hook)

	dir := filepath.Join(PluginsRootDir, string(hook))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot read plugin directory %s: %w", dir, err)
	}

	reattachMap := loadReattachPlugins()

	// Index executable plugins by their logical name, then apply the order
	// defined in .spire/plugins/order.yaml (alphabetical fallback).
	byName := make(map[string]os.DirEntry)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !isExecutable(e) {
			continue
		}
		logicalName := strings.TrimSuffix(e.Name(), ".exe")
		byName[logicalName] = e
		names = append(names, logicalName)
	}

	order := loadPluginOrder()
	for _, logicalName := range orderNames(names, order[string(hook)]) {
		e := byName[logicalName]
		pluginPath := filepath.Join(dir, e.Name())

		var reattach *goplugin.ReattachConfig
		if reattachMap != nil {
			reattach = reattachMap[logicalName]
		}

		if reattach != nil {
			fmt.Printf("  Running plugin: %s (reattach PID %d)\n", e.Name(), reattach.Pid)
		} else {
			fmt.Printf("  Running plugin: %s\n", e.Name())
		}

		if err := runPlugin(pluginPath, ctx, reattach); err != nil {
			return fmt.Errorf("plugin %q failed: %w", e.Name(), err)
		}
	}
	return nil
}

// ListPlugins returns all executable plugin names grouped by hook, in the same
// order they would execute (per .spire/plugins/order.yaml, alphabetical fallback).
func ListPlugins() (map[HookName][]string, error) {
	order := loadPluginOrder()
	result := make(map[HookName][]string)
	for _, hook := range AllHookNames {
		dir := filepath.Join(PluginsRootDir, string(hook))
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", dir, err)
		}
		var names []string
		for _, e := range entries {
			if !e.IsDir() && isExecutable(e) {
				names = append(names, strings.TrimSuffix(e.Name(), ".exe"))
			}
		}
		if len(names) > 0 {
			result[hook] = orderNames(names, order[string(hook)])
		}
	}
	return result, nil
}

// BuildPlugins compiles every plugin source found under
// templates/plugins/<hook>/<name>/ and writes the binary to
// .spire/plugins/<hook>/<name>[.exe].
func BuildPlugins() (int, error) {
	if _, err := os.Stat(PluginSourcesDir); os.IsNotExist(err) {
		return 0, nil
	}

	hookDirs, err := os.ReadDir(PluginSourcesDir)
	if err != nil {
		return 0, fmt.Errorf("cannot read %s: %w", PluginSourcesDir, err)
	}

	built := 0
	for _, hookEntry := range hookDirs {
		if !hookEntry.IsDir() {
			continue
		}
		hookName := hookEntry.Name()
		hookSrcDir := filepath.Join(PluginSourcesDir, hookName)

		pluginDirs, err := os.ReadDir(hookSrcDir)
		if err != nil {
			fmt.Printf("  ⚠️  Cannot read %s: %v\n", hookSrcDir, err)
			continue
		}

		for _, pluginEntry := range pluginDirs {
			if !pluginEntry.IsDir() {
				continue
			}
			name := pluginEntry.Name()
			srcDir := filepath.Join(hookSrcDir, name)

			outName := name
			if runtime.GOOS == "windows" {
				outName += ".exe"
			}
			outPath, err := filepath.Abs(filepath.Join(PluginsRootDir, hookName, outName))
			if err != nil {
				fmt.Printf("  ❌ Cannot resolve output path for %s: %v\n", name, err)
				continue
			}

			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				fmt.Printf("  ❌ Cannot create output dir for %s: %v\n", name, err)
				continue
			}

			fmt.Printf("  Building %s/%s → %s\n", hookName, name, outPath)

			// Sync the module's go.mod/go.sum with its imports before building.
			// go build (Go 1.16+) won't modify them itself, so a plugin that
			// added imports — or a distributed template pinned to a published
			// spire version whose checksum isn't in go.sum yet — would otherwise
			// fail. Tidy failures (e.g. offline) are non-fatal: the build may
			// still succeed from the module cache.
			if _, statErr := os.Stat(filepath.Join(srcDir, "go.mod")); statErr == nil {
				tidyCmd := exec.Command("go", "mod", "tidy")
				tidyCmd.Dir = srcDir
				tidyCmd.Stdout = os.Stdout
				tidyCmd.Stderr = os.Stderr
				if err := tidyCmd.Run(); err != nil {
					fmt.Printf("  ⚠️  go mod tidy failed for %s/%s (continuing): %v\n", hookName, name, err)
				}
			}

			buildCmd := exec.Command("go", "build", "-o", outPath, ".")
			buildCmd.Dir = srcDir
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				fmt.Printf("  ❌ Build failed for %s/%s: %v\n", hookName, name, err)
				continue
			}

			if runtime.GOOS != "windows" {
				_ = os.Chmod(outPath, 0o755)
			}

			fmt.Printf("  Built %s/%s\n", hookName, name)
			built++
		}
	}
	return built, nil
}

// runPlugin launches a single plugin via go-plugin (net/rpc), calls Execute,
// and interprets the result. When reattach is non-nil the CLI connects to an
// already-running plugin process instead of spawning a new subprocess.
func runPlugin(path string, ctx HookContext, reattach *goplugin.ReattachConfig) error {
	cfg := &goplugin.ClientConfig{
		HandshakeConfig: sdk.HandshakeConfig,
		Plugins:         sdk.PluginMap,
		// Suppress go-plugin's own structured logging; plugin authors should
		// use HookResult.Message for user-facing output.
		Logger: hclog.NewNullLogger(),
		// Forward the plugin's raw stderr to the terminal so authors can log
		// progress with fmt.Fprintf(os.Stderr, ...). For net/rpc plugins this
		// MUST be Stderr (consumed by go-plugin's logStderr); SyncStderr is only
		// wired up for gRPC plugins and would silently discard the output.
		Stderr: os.Stderr,
	}
	if reattach != nil {
		cfg.Reattach = reattach
	} else {
		cfg.Cmd = exec.Command(path)
	}
	client := goplugin.NewClient(cfg)
	defer client.Kill()

	rpcClient, err := client.Client()
	if err != nil {
		return fmt.Errorf("failed to connect to plugin: %w", err)
	}

	raw, err := rpcClient.Dispense("spire-plugin")
	if err != nil {
		return fmt.Errorf("failed to dispense plugin: %w", err)
	}

	p := raw.(sdk.Hook)
	result, err := p.Execute(ctx)
	if err != nil {
		return err
	}

	if result.Message != "" {
		fmt.Printf("    %s\n", result.Message)
	}
	if !result.Success {
		msg := result.Error
		if msg == "" {
			msg = "plugin reported failure"
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// isExecutable reports whether the entry is a runnable plugin binary.
// On Windows only .exe files qualify; elsewhere the executable bit must be set.
func isExecutable(e os.DirEntry) bool {
	if runtime.GOOS == "windows" {
		return filepath.Ext(e.Name()) == ".exe"
	}
	info, err := e.Info()
	if err != nil {
		return false
	}
	return info.Mode()&0o111 != 0
}
