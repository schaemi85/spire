# Spire — Plugin System

Spire plugins extend the CLI at predefined lifecycle hook points. They are ordinary Go programs compiled to binaries; the Spire CLI communicates with them over a local net/rpc socket managed by [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin). This means plugins are fully cross-platform (including Windows), crash-safe (a panicking plugin never takes down the CLI), and straightforward to write and test in isolation.

---

## How It Works

```
spire service add
  │
  ├── [before-add-service] ← plugin binaries in .spire/plugins/before-add-service/
  │
  ├── copy template, render files, apply renames …
  │
  └── [after-add-service] ← plugin binaries in .spire/plugins/after-add-service/
        └── e.g. create-db-schema creates migrations/001_init.sql
```

1. Spire discovers every executable binary in `.spire/plugins/<hook-name>/`.
2. Each binary is launched as a subprocess via go-plugin (net/rpc).
3. The Spire CLI calls `Execute(HookContext)` over RPC.
4. The plugin returns a `HookResult`; a non-success result aborts the command.
5. Plugin stderr is forwarded to the terminal for progress logging.

---

## Supported Hooks

| Hook | When it fires |
|------|---------------|
| `before-add-service` | Before the service template is copied into `services/` |
| `after-add-service` | After the service is fully set up and recorded in the manifest |
| `before-upgrade` | Before the upgrade replaces any files (git status already checked) |
| `after-upgrade` | After post-upgrade hooks complete (render, renames, task targets) |

---

## Plugin Directory Layout

```
.spire/
  plugins/
    after-add-service/
      create-db-schema        ← compiled binary (Linux/macOS, chmod +x)
      create-db-schema.exe    ← compiled binary (Windows)
    before-upgrade/
      my-pre-upgrade-check

templates/
  plugins/
    after-add-service/
      create-db-schema/       ← plugin source (travels with the template)
        main.go
        go.mod
        go.sum
```

Source files under `templates/plugins/` are compiled to `.spire/plugins/` with `spire plugin build`.

---

## Writing a Plugin

> **Starting point:** copy `templates/plugins/after-add-service/hello-world/` and rename it. It already uses `sdk.RunOrDebug` and is ready to debug (see [Debugging a Plugin](#debugging-a-plugin)).

### 1. Create the source directory

```
templates/plugins/after-add-service/my-plugin/
├── go.mod
└── main.go
```

### 2. go.mod

```go
module my-plugin

go 1.25.1

require github.com/schaemi85/spire v<version>
```

`go mod tidy` will add `github.com/hashicorp/go-plugin` as an indirect dependency automatically. During local/template development use a `replace` directive:

```go
replace github.com/schaemi85/spire => ../../../../
```

### 3. main.go

```go
package main

import "github.com/schaemi85/spire/plugin/sdk"

// MyPlugin implements sdk.Hook.
type MyPlugin struct{}

func (p *MyPlugin) Execute(ctx sdk.HookContext) (sdk.HookResult, error) {
    switch sdk.HookName(ctx.Hook) {

    case sdk.HookAfterAddService:
        svc := ctx.CurrentService
        if svc == nil || svc.Slots["WithDB"] != "yes" {
            return sdk.HookResult{Success: true}, nil
        }
        // … do your work …
        return sdk.HookResult{Success: true, Message: "schema created"}, nil
    }

    // Silently skip hooks this plugin does not handle.
    return sdk.HookResult{Success: true}, nil
}

func main() {
    // RunOrDebug starts the RPC server when invoked by the Spire CLI, or — when
    // run with the -debug flag — serves in reattach mode for a debugger to
    // attach to (see Debugging a Plugin).
    sdk.RunOrDebug(&MyPlugin{})
}
```

That is the entire boilerplate. The two things every plugin must do:

| What | How |
|------|-----|
| Implement the interface | `func (p *MyPlugin) Execute(ctx sdk.HookContext) (sdk.HookResult, error)` |
| Start (with debug support) | `sdk.RunOrDebug(&MyPlugin{})` |

### 4. Build

```bash
# Compile all plugin sources to .spire/plugins/ in one shot
spire plugin build

# Or build one plugin manually
go build -o .spire/plugins/after-add-service/my-plugin \
    ./templates/plugins/after-add-service/my-plugin/
```

On Windows the output name must end in `.exe`:

```powershell
go build -o .spire\plugins\after-add-service\my-plugin.exe `
    .\templates\plugins\after-add-service\my-plugin\
```

---

## Debugging a Plugin

Spire plugins debug exactly like a **Terraform provider**: VS Code launches the plugin under Delve, the plugin prints a reattach line, and the Spire CLI connects to that already-running, debugger-attached process. Set breakpoints, run a real Spire command, and execution pauses at them.

`sdk.RunOrDebug` has two modes:

| Mode | How it's triggered | When |
|------|--------------------|------|
| **Production** | magic-cookie env set by the Spire CLI | Normal operation — the CLI spawns the plugin |
| **Debug** | `-debug` flag (set by your VS Code launch config) | Serves in reattach mode for a debugger to attach |

### Get the launch config for your plugin

```bash
spire plugin debug after-add-service my-plugin
```

This prints a ready-to-paste VS Code launch configuration with the correct `program` path, plus the follow-up steps below.

### Step 1 — add the launch config and press F5

Add this to `.vscode/launch.json` at the **project root**, then press **F5**:

```json
{
  "name": "Debug plugin: my-plugin",
  "type": "go",
  "request": "launch",
  "mode": "debug",
  "program": "${workspaceFolder}/templates/plugins/after-add-service/my-plugin",
  "args": ["-debug"]
}
```

`"mode": "debug"` compiles the plugin from source and runs it under Delve — **no `spire plugin build` needed**, and no manual `dlv exec`. Set breakpoints in `main.go` before continuing.

### Step 2 — copy the reattach line from the Debug Console

The plugin prints:

```
Plugin started. To attach the Spire CLI, set the SPIRE_REATTACH_PLUGINS
environment variable with the following, then run any Spire command that
triggers the hook:

	SPIRE_REATTACH_PLUGINS='{"my-plugin":{"Protocol":"netrpc","Pid":12345,...}}'
```

> Output appears in the **Debug Console**. If you prefer the integrated terminal (easier to copy from), add `"console": "integratedTerminal"` to the launch config.

### Step 3 — trigger the hook via Spire

In a terminal, set that env var and run any command that fires the hook:

```bash
SPIRE_REATTACH_PLUGINS='...' spire plugin run after-add-service
# or
SPIRE_REATTACH_PLUGINS='...' spire service add
```

The CLI connects to your running process instead of spawning a new one, `Execute()` is called, and Delve pauses at your breakpoints. Re-run the command to hit them again — no need to restart the debug session between invocations.

> **Delve CLI alternative:** if you don't use VS Code, run the plugin under a headless Delve server yourself: `dlv debug ./templates/plugins/after-add-service/my-plugin --headless --listen :2345 --api-version 2 --accept-multiclient -- -debug`, then `dlv connect :2345`. The `-debug` after `--` is what puts the plugin in reattach mode.

---

## The SDK Reference

All types and constants live in `github.com/schaemi85/spire/plugin/sdk`.

### `HookContext`

| Field | Type | Description |
|-------|------|-------------|
| `Hook` | `string` | The hook that triggered this call (compare with `sdk.HookAfterAddService` etc.) |
| `WorkDir` | `string` | Absolute path to the project root |
| `Slots` | `map[string]string` | Resolved application-level slot values |
| `Services` | `[]ServiceInfo` | All services registered in the manifest |
| `CurrentService` | `*ServiceInfo` | Set only for `before/after-add-service` |

### `ServiceInfo`

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Human-readable service name |
| `SlugName` | `string` | Slug used in directory paths |
| `Slots` | `map[string]string` | Resolved service-level slot values |

### `HookResult`

| Field | Type | Description |
|-------|------|-------------|
| `Success` | `bool` | Must be `true` for the Spire command to continue |
| `Message` | `string` | Optional info line shown to the user after the hook completes |
| `Error` | `string` | Reason for failure when `Success` is `false` |

### Hook name constants

```go
sdk.HookBeforeAddService  // "before-add-service"
sdk.HookAfterAddService   // "after-add-service"
sdk.HookBeforeUpgrade     // "before-upgrade"
sdk.HookAfterUpgrade      // "after-upgrade"
```

---

## Aborting an Operation

Return `Success: false` to abort the current Spire command. Spire will print the `Error` string and stop.

```go
if !preconditionMet {
    return sdk.HookResult{
        Success: false,
        Error:   "database is unreachable — refusing to add service",
    }, nil
}
```

`before-*` hooks are especially useful as **pre-condition guards**: they can veto an operation before any files are changed.

Return a non-nil `error` for unexpected failures (e.g. a panic-recovery). For controlled failure messages, prefer `HookResult{Success: false, Error: "..."}`.

---

## Logging from a Plugin

Write progress lines to **stderr** — Spire forwards it to the terminal:

```go
fmt.Fprintf(os.Stderr, "connecting to %s …\n", dbHost)
```

Use `HookResult.Message` for the final one-line summary shown after the hook completes:

```go
return sdk.HookResult{Success: true, Message: "migrated 3 tables"}, nil
```

Do **not** write to stdout — it is reserved for the go-plugin RPC handshake.

---

## CLI Commands

### List installed plugins

```bash
spire plugin list
```

### Build plugin sources

```bash
spire plugin build
```

### Manually trigger a hook

Load context from the current manifest and run all plugins for one hook:

```bash
spire plugin run after-add-service
```

For testing, pass `--context <file>` to supply a `HookContext` from a JSON file instead of the manifest. This lets you exercise a plugin against arbitrary services, slots and `WorkDir` without setting up a real project:

```bash
spire plugin run after-add-service --context ./testdata/ctx.json
```

```json
{
  "WorkDir": "/tmp/my-test-project",
  "Slots": { "ProjectName": "Demo App" },
  "CurrentService": {
    "Name": "payments",
    "SlugName": "payments",
    "Slots": { "WithDB": "yes" }
  }
}
```

Omitted fields default sensibly (`WorkDir` falls back to the current directory; `Slots`/`Services` to empty).

### Print the debug launch config

Print a ready-to-paste VS Code launch configuration and the reattach steps for a specific plugin (Terraform-style):

```bash
spire plugin debug after-add-service my-plugin
```

---

## Execution Order

Within a hook, plugins run **alphabetically by name** by default. To control the order, create `.spire/plugins/order.yaml` mapping each hook to an ordered list of plugin names:

```yaml
# .spire/plugins/order.yaml
after-add-service:
  - create-db-schema
  - create-db-user
  - move-proto-files
  - update-go-workspace

before-upgrade:
  - backup-db
```

- Plugins listed for a hook run **first, in the given order**.
- Any installed plugin **not** listed runs afterwards, alphabetically.
- Names that don't match an installed plugin are ignored.
- The file is optional — without it, everything runs alphabetically.

`spire plugin list` prints plugins in their effective execution order, so you can verify the result. Commit `order.yaml` to version control (it is not covered by the `.spire/plugins/` binary ignore).

---

## Bundled plugins

The template ships several ready-to-use plugins under `templates/plugins/after-add-service/`:

| Plugin | What it does |
|--------|-------------|
| `hello-world` | Minimal scaffold — copy this to start a new plugin |

All of these use `sdk.RunOrDebug`, so any of them can be debugged with `spire plugin debug after-add-service <name>` (see [Debugging a Plugin](#debugging-a-plugin)).

```bash
spire plugin build
spire service add    # all installed plugins fire automatically
```

---

## Cross-Platform Notes

| Platform | Executable detection | Build output |
|----------|---------------------|--------------|
| Linux / macOS | Executable bit (`chmod +x`) | No extension needed |
| Windows | `.exe` extension | Must end in `.exe` |

`spire plugin build` sets the executable bit on Linux/macOS and adds `.exe` on Windows automatically.

---

## Bundling Plugins with a Template

Keep plugin sources under `templates/plugins/` so they travel with the template and the generated project. After `spire init`, consumers run `spire plugin build` once to compile them for their platform.

Add `.spire/plugins/` to `.gitignore` so compiled binaries (which are platform-specific) are not committed.
