# Spire — Template Authoring Guide

This guide explains how to create and maintain a Spire template repository that others can use with `spire init` and `spire service add`.

---

## What Is a Template?

A Spire template is a Git repository (or local directory) that contains:

- A working application (or skeleton) with slot placeholders.
- A `.spire/manifest.yaml` that declares slots, services, path renames, and template files.
- Optionally a `templates/service/` directory as the blueprint for `spire service add`.
- Optionally a `templates/plugins/` directory with plugin source code that consumers compile with `spire plugin build`.

When a user runs `spire init <your-template>`, Spire resolves slot values (by prompting the user or applying defaults), renders all files, renames paths, and produces a ready-to-use project.

---

## Template Syntax

Spire uses custom delimiters `[[ ]]` instead of the standard `{{ }}` to avoid conflicts with other tools (Helm, GitHub Actions, Bruno, etc.) that live inside generated projects.

```
[[ .slots.ProjectName ]]         → resolved slot value
[[ .slots.ProjectName | upper ]] → slot value piped through a function
[[ range .services ]]            → iterate over generated services
[[ end ]]
```

Any text file in the template that contains at least one `[[ ]]` expression is rendered as a Go template. Files without these delimiters are copied verbatim.

---

## Slot Types

| Type constant | YAML value | Behaviour |
|---------------|-----------|-----------|
| `PromptMandatory` | `PromptMandatory` | User is prompted; empty input is rejected |
| `PromptOptional` | `PromptOptional` | User is prompted; empty input uses `defaultValue` |
| `PromptSecret` | `PromptSecret` | Masked input; value is **not** written to manifest |
| `DynamicValue` | `DynamicValue` | Computed from `expression`; never prompted |

---

## Manifest Reference

The full manifest lives at `.spire/manifest.yaml` in both the template and the generated project.

```yaml
spireVersion: v0.0.1
templateVersion: v1.0.0          # updated by spire upgrade

# Application-level slots — prompted during spire init
appSlots:
  - key: ProjectName
    label: "Project Name"
    description: "Human-readable name of the application"
    type: PromptMandatory
    validation: "minLength:2"

  - key: ProjectSlugName
    type: DynamicValue
    expression: "[[ .slots.ProjectName | slugify ]]"

  - key: PackageName
    label: "Go Module Path"
    type: PromptMandatory
    validation: "minLength:5"

  - key: HttpPort
    label: "HTTP Port"
    type: PromptOptional
    defaultValue: "8080"
    validation: "port"

  - key: DBPassword
    label: "Database Password"
    type: PromptSecret
    pipelines:
      - generatePassword:16       # auto-generate if empty

# Generated services (populated by spire service add)
services: []

# Files rendered from templates/ on init and when services change
templateFiles:
  - source: templates/devcontainer/.env.tmpl
    destination: .devcontainer/.env
    regenerateOnServiceChange: true

  - source: templates/vscode/launch.json.tmpl
    destination: .vscode/launch.json
    regenerateOnServiceChange: true

# Rename files/directories based on slot values
pathRenames:
  - pattern: myapp                 # literal string to find in paths
    expression: "[[ .slots.ProjectSlugName ]]"

# Paths to exclude from Go template rendering
ignorePaths:
  - .git
  - .spire
  - templates
  - vendor
  - go.sum

# Blueprint for spire service add
serviceConfig:
  originalPath: services/sampleservice
  servicesSlots:
    - key: ServiceName
      label: "Service Name"
      type: PromptMandatory
      validation: "slug"
    - key: WithDB
      label: "Include database?"
      type: PromptOptional
      defaultValue: "no"
      validation: "enum:yes,no"
  pathRenames:
    - pattern: sampleresource
      expression: "[[ .slots.ServiceName ]]"
  postHooks:
    - condition: '[[ if ne .slots.WithDB "yes" ]]true[[ end ]]'
      removePaths:
        - database
```

---

## Validation Rules

Slots accept a single `validation` string. The user is re-prompted until the value satisfies the rule.

### String Rules

| Rule | Example | Description |
|------|---------|-------------|
| `pattern:<regex>` | `pattern:^[a-z]+$` | Must match the regular expression |
| `minLength:<n>` | `minLength:3` | At least *n* characters |
| `maxLength:<n>` | `maxLength:64` | At most *n* characters |
| `enum:<a>,<b>,...` | `enum:yes,no` | Must be one of the listed values |
| `startsWith:<prefix>` | `startsWith:https://` | Must begin with prefix |
| `endsWith:<suffix>` | `endsWith:.git` | Must end with suffix |

### Numeric Rules

| Rule | Example | Description |
|------|---------|-------------|
| `numeric` | `numeric` | Must be a valid integer |
| `minimum:<n>` | `minimum:1` | Integer >= *n* |
| `maximum:<n>` | `maximum:65535` | Integer <= *n* |
| `port` | `port` | Valid TCP port (1–65535) |

### Format Rules

| Rule | Example | Description |
|------|---------|-------------|
| `slug` | `slug` | Lowercase letters, digits, hyphens; must start with a letter |
| `email` | `email` | Valid email address |
| `url` | `url` | Valid URL with scheme and host |
| `semver` | `semver` | Semantic version (`v1.2.3` or `1.2.3`) |

---

## Pipeline Functions

Pipelines post-process a slot's value after prompting. They are listed in order under `pipelines:` and can also be used inline in `[[ ]]` expressions.

| Function | Description | Example |
|----------|-------------|---------|
| `slugify` | Lowercase alphanumeric with hyphens | `My App` → `my-app` |
| `pascalCase` | Each word capitalised, no separators | `my-app` → `MyApp` |
| `camelCase` | Like pascalCase but first word lowercase | `my-app` → `myApp` |
| `snakeCase` | Words joined with underscores, lowercase | `my-app` → `my_app` |
| `upper` | All uppercase | `hello` → `HELLO` |
| `lower` | All lowercase | `HELLO` → `hello` |
| `title` | Title case | `hello world` → `Hello World` |
| `replace <old> <new>` | Replace substring | — |
| `trimPrefix <s>` | Remove leading string | — |
| `trimSuffix <s>` | Remove trailing string | — |
| `ensureSuffix <s>` | Append if not already present | — |
| `split <sep>` | Split into list | — |
| `join <sep>` | Join list into string | — |
| `contains <s>` | Boolean test | — |
| `hasPrefix <s>` | Boolean test | — |
| `hasSuffix <s>` | Boolean test | — |
| `repeat <n>` | Repeat string N times | — |
| `trimSpace` | Strip leading/trailing whitespace | — |
| `default <val>` | Use `val` if value is empty | — |
| `generatePassword` | Random alphanumeric password; pass `true` as second arg to include special characters | — |

### Inline Example

```
[[ .slots.ServiceName | pascalCase ]]Handler
[[ .slots.ProjectName | slugify | ensureSuffix "-api" ]]
[[ .slots.DBPassword | default (generatePassword 24) ]]
```

---

## Template Files (`.tmpl`)

Use `templateFiles` when a file should be regenerated from a source template rather than rendered in-place. This is useful for files that aggregate service information (e.g. a VSCode launch configuration listing all services).

```yaml
templateFiles:
  - source: templates/vscode/launch.json.tmpl
    destination: .vscode/launch.json
    regenerateOnServiceChange: true
```

- `source` is relative to the project root and **must** exist in the template.
- `destination` is the output path.
- `regenerateOnServiceChange: true` causes the file to be re-rendered every time a service is added.

### Accessing Services in a Template File

```json
// templates/vscode/launch.json.tmpl
{
  "version": "0.2.0",
  "configurations": [
    [[ range .services ]]
    {
      "name": "[[ .Name ]]",
      "type": "go",
      "request": "launch",
      "program": "${workspaceFolder}/services/[[ .SlugName ]]/cmd/main.go"
    },
    [[ end ]]
  ]
}
```

---

## Path Renames

After rendering, Spire renames file and directory paths that match a pattern. Renames are applied deepest-first (files before their parent directories).

```yaml
pathRenames:
  - pattern: sampleapp       # exact substring to match in any path component
    expression: "[[ .slots.ProjectSlugName ]]"
```

The `expression` is itself a Go template evaluated against the current `ResolveContext`.

---

## Post-Hooks

Post-hooks run after file rendering and path renames during `spire service add`. They conditionally remove paths or apply additional renames.

```yaml
postHooks:
  # Remove the database directory if WithDB is not "yes"
  - condition: '[[ if ne .slots.WithDB "yes" ]]true[[ end ]]'
    removePaths:
      - database

  # Remove the jobs directory if WithJob is not set
  - condition: '[[ if ne .slots.WithJob "yes" ]]true[[ end ]]'
    removePaths:
      - jobs
    pathRenames:
      - pattern: with-jobs
        expression: "no-jobs"
```

A hook fires when its `condition` expression evaluates to the string `"true"`. Omit `condition` to always fire.

---

## Service Blueprint (`templates/service/`)

The `serviceConfig.originalPath` directory is the blueprint copied and rendered whenever `spire service add` is run. It should contain a complete, working service skeleton using slot placeholders.

```
templates/service/
├── cmd/
│   └── main.go
├── internal/
│   └── sampleresource/      ← renamed by pathRenames
│       └── handler.go
├── go.mod
└── config.yaml              ← read by Spire for feature detection
```

`config.yaml` can declare feature flags that control how Spire integrates the service:

```yaml
db: true          # service uses a database
api: true         # service exposes an API
configure_as_job: false
```

---

## Bundling Plugins with a Template

Plugin source code lives in `templates/plugins/<hook>/<plugin-name>/`. Because this directory is inside `templates/`, it is:

- Included when `spire init` generates a new project.
- Preserved when `spire template sync` converts a project back to a template.
- Not rendered by the Go template engine (add `templates` to `ignorePaths` if not already there).

### Directory layout

```
templates/
  plugins/
    after-add-service/
      create-db-schema/
        main.go       ← plugin source
        go.mod        ← standalone Go module
    before-upgrade/
      my-check/
        main.go
        go.mod
```

### What consumers do after `spire init`

```bash
# Compile all plugin sources to .spire/plugins/
spire plugin build

# Verify the plugins are installed
spire plugin list
```

After building, the compiled binaries live in `.spire/plugins/` and fire automatically at the appropriate hook points.

### Platform considerations

`spire plugin build` produces a native binary for the current OS. If your team works across operating systems, add `.spire/plugins/` to `.gitignore` and document that each developer should run `spire plugin build` after cloning or after `spire upgrade`.

For the full plugin authoring guide — including the SDK contract, go-plugin RPC setup, example code, and how to abort an operation from a plugin — see [plugins.md](plugins.md).

---

## Upgrade Manifest (`.spire/upgrade-manifest.yaml`)

This file controls what happens during `spire upgrade`. It declares which paths are replaced by the template and which files the user wants to keep.

```yaml
application:
  - path: .devcontainer
    keep:
      - postgres/init-user-db.sh
      - .env

  - path: Makefile

services:
  - path: _apigw
```

- Each `path` under `application` or `services` is removed and replaced from the new template.
- Files listed under `keep` are backed up before the replacement and restored afterwards.

---

## Creating a Template from Scratch

### Step 1 — Build the Application

Create a real, working application. Name things with a placeholder that you will later substitute (e.g. `myapp`, `sampleservice`).

### Step 2 — Add the Manifest

Run `spire manifest init` to generate a commented skeleton at `.spire/manifest.yaml`:

```bash
spire manifest init
```

Then edit the file to define your `appSlots` with actual resolved values for now — `spire template sync` will replace them with expressions later. For example:

```yaml
spireVersion: v0.0.1
templateVersion: v0.1.0
appSlots:
  - key: ProjectName
    label: Project Name
    type: PromptMandatory
    value: "My Application"    # ← real value in the project
```

Run `spire manifest validate` at any point to catch errors before continuing:

```bash
spire manifest validate
```

### Step 3 — Add Slot Expressions to Files

Anywhere you want a slot value to appear, replace the hardcoded value with `[[ .slots.KEY ]]`:

```go
// Before
const AppName = "myapp"

// After
const AppName = "[[ .slots.ProjectSlugName ]]"
```

### Step 4 — Run `spire template sync`

```bash
spire template sync --output ../my-template-repo
```

This reverses the project into a clean template, replacing real values with expressions and clearing the manifest.

### Step 5 — (Optional) Add Plugins

If your template benefits from lifecycle automation, add plugin sources under `templates/plugins/`. See [Bundling Plugins with a Template](#bundling-plugins-with-a-template) and [plugins.md](plugins.md) for details.

### Step 6 — Tag and Publish

```bash
cd ../my-template-repo
git add .
git commit -m "Initial template v1.0.0"
git tag v1.0.0
git push --tags
```

### Step 7 — Test

```bash
spire init ../my-template-repo
# or
spire init https://github.com/org/my-template-repo

# If the template includes plugins
spire plugin build
```

---

## Maintaining a Template

The recommended workflow is to keep the template as a **real working project** (so it stays buildable and testable), then regenerate the template version with `template sync` whenever you make changes.

```
┌─────────────────────────────┐
│  Template repo (versioned)  │
│  spire init ──────────────► │──► Generated project
│                             │
│ ◄──────── spire template sync (from generated project)
└─────────────────────────────┘
```

This round-trip means:
- The template is always testable as a standalone application.
- Slot expressions are derived automatically — you don't hand-edit them.
- Consumers can upgrade with `spire upgrade` as you release new template versions.
