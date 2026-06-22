// Package sdk is the shared contract between the Spire CLI and Spire plugins.
//
// Plugin authors import this package, implement the [Hook] interface, and
// call [RunOrDebug] from main() — that is all that is needed.
//
// Example skeleton:
//
//	package main
//
//	import "github.com/schaemi85/spire/plugin/sdk"
//
//	type MyPlugin struct{}
//
//	func (p *MyPlugin) Execute(ctx sdk.HookContext) (sdk.HookResult, error) {
//	    // react to ctx.Hook, ctx.Slots, ctx.CurrentService …
//	    return sdk.HookResult{Success: true, Message: "done"}, nil
//	}
//
//	func main() {
//	    sdk.RunOrDebug(&MyPlugin{})
//	}
package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"

	hclog "github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
)

// quietLogger is passed to goplugin.Serve so go-plugin's own internal logging
// (e.g. the JSON "plugin address" handshake line, emitted at Trace level by the
// default server logger) does not pollute the plugin's stderr, which the Spire
// CLI forwards verbatim to the terminal. Plugin authors write user-facing output
// with fmt.Fprintf(os.Stderr, ...) or via HookResult.Message instead.
func quietLogger() hclog.Logger {
	return hclog.NewNullLogger()
}

// ─── lifecycle constants ──────────────────────────────────────────────────────

// HookName identifies a lifecycle point in the Spire CLI.
type HookName string

const (
	HookBeforeAddService HookName = "before-add-service"
	HookAfterAddService  HookName = "after-add-service"
	HookBeforeUpgrade    HookName = "before-upgrade"
	HookAfterUpgrade     HookName = "after-upgrade"
)

// AllHooks lists every supported hook in execution order.
var AllHooks = []HookName{
	HookBeforeAddService,
	HookAfterAddService,
	HookBeforeUpgrade,
	HookAfterUpgrade,
}

// ─── data types ──────────────────────────────────────────────────────────────

// HookContext is the data passed to [Hook.Execute].
type HookContext struct {
	// Hook is the lifecycle point that triggered this call.
	Hook string

	// WorkDir is the absolute path to the project root.
	WorkDir string

	// Slots contains all resolved application-level slot values.
	Slots map[string]string

	// Services is the full list of services registered in the manifest.
	Services []ServiceInfo

	// CurrentService is set only for service-scoped hooks
	// (before/after-add-service).
	CurrentService *ServiceInfo
}

// ServiceInfo is a flat representation of a manifest service.
type ServiceInfo struct {
	Name     string
	SlugName string
	Slots    map[string]string
}

// HookResult is returned by [Hook.Execute].
type HookResult struct {
	// Success must be true for the Spire operation to continue.
	// Set to false (and populate Error) to abort the current command.
	Success bool

	// Message is an optional informational line shown to the user.
	Message string

	// Error is the reason for failure when Success is false.
	Error string
}

// ─── plugin interface ─────────────────────────────────────────────────────────

// Hook is the single interface every Spire plugin must implement.
type Hook interface {
	Execute(ctx HookContext) (HookResult, error)
}

// ─── go-plugin wiring ────────────────────────────────────────────────────────

// HandshakeConfig is the shared handshake between the Spire CLI and every
// plugin binary. Mismatched values cause the plugin to refuse to start.
var HandshakeConfig = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SPIRE_PLUGIN",
	MagicCookieValue: "spire-plugin-v1",
}

// PluginMap registers "spire-plugin" as the single plugin slot.
// Both the CLI and each plugin binary must use the same map key.
var PluginMap = goplugin.PluginSet{
	"spire-plugin": &Plugin{},
}

// Plugin implements [goplugin.Plugin] and bridges the [Hook] interface over
// net/rpc.  The CLI uses the zero-value (Client side); plugin binaries set
// Impl before calling goplugin.Serve.
type Plugin struct {
	Impl Hook
}

func (p *Plugin) Server(*goplugin.MuxBroker) (interface{}, error) {
	return &hookRPCServer{impl: p.Impl}, nil
}

func (p *Plugin) Client(_ *goplugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &hookRPCClient{client: c}, nil
}

// ─── debug helpers ───────────────────────────────────────────────────────────

// RunOrDebug is the entry point for plugin binaries. It mirrors Terraform's
// provider debugging model and dispatches to one of two modes:
//
//   - Default → normal go-plugin RPC server. When launched by the Spire CLI the
//     handshake succeeds; when run directly without the magic cookie, go-plugin
//     prints "This binary is a plugin..." and exits — exactly like a Terraform
//     provider.
//
//   - -debug flag → RPC server started in reattach mode. The binary prints a
//     SPIRE_REATTACH_PLUGINS line and blocks. This is meant to be launched from
//     a VS Code "mode": "debug" launch configuration (press F5): VS Code compiles
//     and runs the plugin under Delve, the reattach line appears in the Debug
//     Console, and any Spire command run with that env var set connects to this
//     already-running, debugger-attached process instead of spawning a new one.
//     Run `spire plugin debug <hook> <name>` to print a ready-to-paste
//     launch.json for a specific plugin.
//
// Use this instead of calling goplugin.Serve directly.
func RunOrDebug(impl Hook) {
	fs := flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ExitOnError)
	debug := fs.Bool("debug", false, "run the plugin in debug mode for a debugger to attach to (Terraform-style)")
	_ = fs.Parse(os.Args[1:])

	if *debug {
		runDebugServe(impl)
		return
	}

	// go-plugin's net/rpc accept loop logs a benign "use of closed network
	// connection" line via the standard logger on every shutdown. The Spire CLI
	// forwards the plugin's stderr verbatim, so silence the stdlib logger to keep
	// that noise out of the terminal. Authors use fmt.Fprintf(os.Stderr, ...).
	log.SetOutput(io.Discard)

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         goplugin.PluginSet{"spire-plugin": &Plugin{Impl: impl}},
		Logger:          quietLogger(),
	})
}

// runDebugServe starts the plugin as an RPC server in go-plugin test mode,
// prints the SPIRE_REATTACH_PLUGINS env var, and blocks until killed. Attach a
// debugger to this process (the F5 / "mode": "debug" launch does this for you)
// and run Spire with the env var to hit breakpoints.
func runDebugServe(impl Hook) {
	reattachCh := make(chan *goplugin.ReattachConfig)

	go goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         goplugin.PluginSet{"spire-plugin": &Plugin{Impl: impl}},
		Logger:          quietLogger(),
		Test: &goplugin.ServeTestConfig{
			Context:          context.Background(),
			ReattachConfigCh: reattachCh,
		},
	})

	reattach := <-reattachCh

	name := pluginName()
	cfg := map[string]any{
		name: map[string]any{
			"Protocol":        string(reattach.Protocol),
			"ProtocolVersion": reattach.ProtocolVersion,
			"Pid":             os.Getpid(),
			"Test":            true,
			"Addr": map[string]string{
				"Network": reattach.Addr.Network(),
				"String":  reattach.Addr.String(),
			},
		},
	}
	data, _ := json.Marshal(cfg)

	fmt.Printf("Plugin started. To attach the Spire CLI, set the SPIRE_REATTACH_PLUGINS\n")
	fmt.Printf("environment variable with the following, then run any Spire command that\n")
	fmt.Printf("triggers the hook:\n\n")
	fmt.Printf("\tSPIRE_REATTACH_PLUGINS='%s'\n", data)

	select {} // block until the process is killed
}

// pluginName returns the logical name the Spire CLI uses to match this plugin
// when reattaching (the installed binary / source-directory name). It derives
// the name from the Go module path so it is stable regardless of how the binary
// was compiled — in particular when VS Code's "mode": "debug" builds to a random
// temp binary name. Falls back to the executable name when build info is absent.
func pluginName() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Path != "" && bi.Main.Path != "command-line-arguments" {
		return path.Base(bi.Main.Path)
	}
	return filepath.Base(os.Args[0])
}

// ─── net/rpc internals (not part of the public plugin API) ───────────────────

// ExecuteArgs and ExecuteReply are the net/rpc wire types for Plugin.Execute.
// They MUST be exported: net/rpc only registers a method when both of its
// argument types are exported, otherwise the plugin fails to register and the
// CLI hangs waiting for a dispense that never completes.
type ExecuteArgs struct {
	Ctx HookContext
}

type ExecuteReply struct {
	Result HookResult
	Err    string
}

// hookRPCClient is the client-side stub used by the Spire CLI.
type hookRPCClient struct {
	client *rpc.Client
}

func (r *hookRPCClient) Execute(ctx HookContext) (HookResult, error) {
	var reply ExecuteReply
	if err := r.client.Call("Plugin.Execute", &ExecuteArgs{Ctx: ctx}, &reply); err != nil {
		return HookResult{}, err
	}
	if reply.Err != "" {
		return reply.Result, errors.New(reply.Err)
	}
	return reply.Result, nil
}

// hookRPCServer is the server-side handler that runs inside the plugin binary.
type hookRPCServer struct {
	impl Hook
}

func (s *hookRPCServer) Execute(args *ExecuteArgs, reply *ExecuteReply) error {
	result, err := s.impl.Execute(args.Ctx)
	reply.Result = result
	if err != nil {
		reply.Err = err.Error()
	}
	return nil
}
