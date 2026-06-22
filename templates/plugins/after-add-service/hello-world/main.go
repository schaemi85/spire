// hello-world is a minimal Spire plugin scaffold. Copy this directory, rename
// it, and replace the Execute body with your own logic.
//
// Run in production (via the Spire CLI):
//
//	spire plugin build && spire service add
//
// Debug it (Terraform-style): run `spire plugin debug after-add-service hello-world`
// for a ready-to-paste VS Code launch config. Press F5, copy the printed
// SPIRE_REATTACH_PLUGINS line, then run a Spire command with that env var set.
package main

import (
	"fmt"
	"os"

	"github.com/schaemi85/spire/plugin/sdk"
)

type helloWorld struct{}

func (p *helloWorld) Execute(ctx sdk.HookContext) (sdk.HookResult, error) {
	// Guard: only act on the hook this plugin was designed for.
	if ctx.Hook != string(sdk.HookAfterAddService) {
		return sdk.HookResult{Success: true}, nil
	}

	svc := ctx.CurrentService
	if svc == nil {
		return sdk.HookResult{Success: true, Message: "no current service in context, skipping"}, nil
	}

	// Write progress lines to stderr — Spire forwards them to the terminal.
	// Do NOT write to stdout; it is reserved for the go-plugin RPC handshake.
	fmt.Fprintf(os.Stderr, "hello-world plugin running\n")
	fmt.Fprintf(os.Stderr, "    hook:    %s\n", ctx.Hook)
	fmt.Fprintf(os.Stderr, "    service: %s  (slug: %s)\n", svc.Name, svc.SlugName)
	fmt.Fprintf(os.Stderr, "    workDir: %s\n", ctx.WorkDir)

	// Example: read a service slot value.
	if withDB := svc.Slots["WithDB"]; withDB == "yes" {
		fmt.Fprintf(os.Stderr, "    → this service requested database support\n")
	}

	// Example: read an application slot value.
	if projectName := ctx.Slots["ProjectName"]; projectName != "" {
		fmt.Fprintf(os.Stderr, "    → project name: %s\n", projectName)
	}

	// Return Success:true to let the Spire command continue.
	// Set Success:false (and populate Error) to abort it.
	return sdk.HookResult{
		Success: true,
		Message: fmt.Sprintf("hello from service %q", svc.Name),
	}, nil
}

func main() {
	// RunOrDebug starts the RPC server when invoked by Spire, or — when run with
	// the -debug flag (e.g. from a VS Code "mode": "debug" launch) — serves in
	// reattach mode for a debugger to attach to.
	sdk.RunOrDebug(&helloWorld{})
}
