# Spire CLI — Command Reference

## Global Flags

These flags are available on every command.

| Flag | Description |
|------|-------------|
| `--workdir <path>` | Run the command as if in the given directory (defaults to current directory) |
| `--non-interactive` | Disable all interactive prompts — required values must be supplied via `--set` flags |

---

## `spire init`

Initialise a new project from a template repository or local directory.

```bash
spire init <template> [flags]
```

`<template>` can be:
- A Git URL: `https://gitlab.example.com/templates/my-app`
- A local path: `./my-local-template`

### Flags

| Flag | Description |
|------|-------------|
| `--set Key=Value` | Pre-fill a slot value; repeatable (`--set A=1 --set B=2`) |
| `--version <tag>` | Template version (git tag) to use; required in `--non-interactive` mode for git templates |

### Behaviour

1. Downloads the template at the requested version (or prompts to choose one).
2. Prompts for each slot defined in `appSlots` (skipped with `--set` overrides).
3. Renders all template files and in-place `[[ ]]` expressions.
4. Applies `pathRenames` to rename files and directories.
5. Runs `git init -b main` and creates an initial commit.

### Examples

```bash
# Interactive — prompts for version and all slots
spire init https://gitlab.example.com/templates/go-monorepo

# Local template
spire init ./my-local-template

# Non-interactive CI run
spire init https://gitlab.example.com/templates/go-monorepo \
  --non-interactive \
  --version v1.4.0 \
  --set ProjectName=payments \
  --set PackageName=github.com/acme/payments
```

---

## `spire service`

Manage services in the current Spire project.

### `spire service add`

Add a new service to an existing Spire project using the service blueprint defined in `serviceConfig` inside the manifest.

```bash
spire service add [flags]
```

### Behaviour

1. Validates the current directory is a Spire project (checks for `templates/service/`).
2. Loads the manifest and prompts for each slot in `serviceConfig.servicesSlots`.
3. Runs `before-add-service` plugins (see [plugins.md](plugins.md)).
4. Copies `templates/service/` to `services/<service-name>/`.
5. Renders files and applies service-level `pathRenames`.
6. Evaluates `postHooks` — conditionally removes paths based on slot values.
7. Runs `go work use -r services/` to register the new module.
8. Re-renders any `templateFiles` with `regenerateOnServiceChange: true`.
9. Records the new service in the manifest.
10. Runs `after-add-service` plugins.

### Examples

```bash
spire service add

# Non-interactive
spire service add --non-interactive --set ServiceName=payments --set WithDB=yes
```

---

## `spire manifest`

Manage the `.spire/manifest.yaml` file.

### `spire manifest init`

Scaffold a new `.spire/manifest.yaml` skeleton with commented examples in the current directory. Intended for template authors starting a new template from scratch.

```bash
spire manifest init [flags]
```

| Flag | Description |
|------|-------------|
| `--force` | Overwrite an existing manifest without prompting |

The generated file contains all supported top-level fields with inline documentation comments. Edit it to define your template's slots, path renames, and service configuration, then run `spire manifest validate` to check for errors.

### `spire manifest validate`

Parse and validate `.spire/manifest.yaml`, reporting all errors in a single pass.

```bash
spire manifest validate [flags]
```

| Flag | Description |
|------|-------------|
| `--file <path>` | Path to the manifest file (default: `.spire/manifest.yaml`) |

**What is checked:**

| Category | Checks |
|----------|--------|
| Required fields | `spireVersion` and `templateVersion` must be present and non-empty |
| Slot integrity | Duplicate slot keys; `DynamicValue` slots without an `expression`; invalid `validation` rule syntax |
| Go template syntax | Every `expression`, `condition`, and `pathRename.expression` field is parsed with the Spire `[[ ]]` delimiters and the full pipeline function set |
| Cross-references | `.slots.KEY` references in `pathRenames` and `postHooks` that do not match any defined slot key |

Exits with code `1` when errors are found — suitable for CI/CD pipelines.

### Examples

```bash
# Start a new template manifest
spire manifest init

# Validate after editing
spire manifest validate

# Validate a manifest at a custom path (e.g. in CI)
spire manifest validate --file path/to/manifest.yaml
```

---

## `spire template`

Template authoring commands.

### `spire template sync`

Reverse a live Spire project back into a reusable template. Slot values are replaced with their `[[ .slots.KEY ]]` expressions, generated services are removed, and the manifest is cleaned.

```bash
spire template sync --output <dir> [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--output <dir>` | Directory to write the template into (required) |

### Behaviour

1. Copies the project to `<output>`, preserving an existing `.git/` directory so template history is maintained.
2. Replaces resolved slot values back into `[[ .slots.KEY ]]` expressions throughout all files.
3. Extracts the first generated service as the reference service blueprint under `templates/service/`.
4. Reverses service slot values to expressions.
5. Reverses `pathRenames` (renames back to the original pattern).
6. Removes all generated services from the output.
7. Clears slot values and the services list from the manifest.

### Examples

```bash
# First-time template creation
spire template sync --output ../my-template

# Update existing template (preserves .git history)
spire template sync --output ../my-template
```

---

## `spire upgrade`

Upgrade the project scaffolding to the latest template version while preserving customisations defined in `.spire/upgrade-manifest.yaml`.

```bash
spire upgrade [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--dry-run` | Preview the operations without modifying any files |
| `--force` | Proceed even if there are uncommitted git changes |
| `--template-repo <url>` | Template git URL (auto-detected from manifest `gitRepository` if not set) |
| `--template-local <dir>` | Use a local directory as the template source instead of git |

### Behaviour

1. Checks git status (aborts if dirty unless `--force`).
2. Runs `before-upgrade` plugins (see [plugins.md](plugins.md)).
3. Downloads the latest template version.
4. Reads `.spire/upgrade-manifest.yaml` to determine which paths to keep and which to replace.
5. Backs up all files marked `keep` to `.spire/backup/<timestamp>/`.
6. Removes and replaces application-level paths.
7. Removes and replaces service-level paths.
8. Restores backed-up files over the new scaffolding.
9. Runs post-upgrade hooks: re-renders templates, applies renames, runs task targets (`clean`, `proto:gen`, `build`, `tidy`, `format`).
10. Runs `after-upgrade` plugins.
11. Updates `templateVersion` in the manifest.

### Examples

```bash
# Preview what would change
spire upgrade --dry-run

# Normal upgrade
spire upgrade

# Skip git cleanliness check
spire upgrade --force

# Use a local template for testing
spire upgrade --template-local ../my-template
```

---

## `spire backup`

Manage project backups. Backups are stored under `.spire/backup/<timestamp>/` and are created automatically before each upgrade.

### `spire backup create`

Create a manual timestamped backup of all files listed in `.spire/upgrade-manifest.yaml`.

```bash
spire backup create
```

### `spire backup restore`

Restore a previous backup to its original locations.

```bash
spire backup restore --backup <timestamp>
```

| Flag | Description |
|------|-------------|
| `--backup <timestamp>` | Timestamp of the backup to restore (e.g. `20250109-143022`) |

### `spire backup list`

List all backups with size and modification time.

```bash
spire backup list
```

### `spire backup clean`

Remove old backups, keeping the most recent N.

```bash
spire backup clean [--keep N]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--keep <n>` | `2` | Number of most recent backups to keep |

### Examples

```bash
# Before a risky manual change
spire backup create

# List what's available
spire backup list

# Restore after a failed upgrade
spire backup restore --backup 20250109-143022

# Prune, keeping last 5
spire backup clean --keep 5
```

---

## `spire plugin`

Manage lifecycle plugins — compiled binaries that extend Spire at predefined hook points. Plugin binaries live in `.spire/plugins/<hook-name>/`; sources live in `templates/plugins/<hook-name>/<plugin-name>/`.

Supported hooks: `before-add-service`, `after-add-service`, `before-upgrade`, `after-upgrade`.

See [plugins.md](plugins.md) for the full authoring guide.

### `spire plugin list`

List every installed plugin binary grouped by hook.

```bash
spire plugin list
```

### `spire plugin build`

Compile every plugin source directory found under `templates/plugins/` and write the binary to `.spire/plugins/<hook>/<name>[.exe]`. Each source directory must contain a Go `main` package with its own `go.mod`.

```bash
spire plugin build
```

### `spire plugin run`

Manually trigger all plugins for a given hook, loading the current project manifest as context. Useful for testing a plugin in isolation.

```bash
spire plugin run <hook>
```

| Argument | Description |
|----------|-------------|
| `<hook>` | Hook name to trigger (e.g. `after-add-service`) |

When the `SPIRE_REATTACH_PLUGINS` environment variable is set, the CLI connects to the already-running plugin process instead of spawning a new subprocess. See `spire plugin debug` and [plugins.md](plugins.md) for details.

### `spire plugin debug`

Print step-by-step instructions for attaching a Delve debugger to a plugin in reattach mode (Terraform-style). The binary must exist — run `spire plugin build` first.

```bash
spire plugin debug <hook> <name>
```

| Argument | Description |
|----------|-------------|
| `<hook>` | Hook name (e.g. `after-add-service`) |
| `<name>` | Plugin name matching the binary in `.spire/plugins/<hook>/` |

### Examples

```bash
# After adding plugin sources to templates/plugins/
spire plugin build

# Verify what is installed
spire plugin list

# Test a plugin without running the full command
spire plugin run after-add-service

# Print Delve attach instructions for a specific plugin
spire plugin debug after-add-service my-plugin
```

---

## `spire version`

Print the CLI version string (injected at build time from the git tag).

```bash
spire version
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error (invalid flags, missing files, validation failure) |
| `2` | Pre-condition failure (dirty git state, missing Spire project) |
