# Spire — Manifest Reference

The `.spire/manifest.yaml` file is the single source of truth for a Spire project. It controls slot resolution, file rendering, path renames, service generation, and upgrades.

---

## Top-Level Fields

| Field | Type | Description |
|-------|------|-------------|
| `spireVersion` | string | Spire CLI version this manifest targets (e.g. `v0.0.1`) |
| `templateVersion` | string | Current template version; updated automatically by `spire upgrade` |
| `gitRepository` | string | URL of the project's git repository (auto-detected from `git remote`) |
| `appSlots` | `[]Slot` | Application-level slots prompted during `spire init` |
| `services` | `[]Service` | Generated services (populated by `spire service add`) |
| `templateFiles` | `[]TemplateFile` | Files rendered from `templates/` into the project |
| `pathRenames` | `[]PathRename` | Path rename rules applied after rendering |
| `ignorePaths` | `[]string` | Paths excluded from Go template rendering |
| `serviceConfig` | `ServiceConfig` | Blueprint for `spire service add` |

---

## Slot

```yaml
appSlots:
  - key: ProjectName          # required — unique identifier used in expressions
    label: "Project Name"     # optional — displayed in the prompt
    description: "..."        # optional — additional help text
    type: PromptMandatory     # slot type string (see Slot Types table)
    defaultValue: ""          # optional — used when input is empty (PromptOptional)
    expression: ""            # required for DynamicValue — Go template string
    pipelines: []             # optional — list of transformation functions
    validation: ""            # optional — validation rule string
    value: ""                 # resolved value (populated after init; cleared for secrets)
```

### Slot Types

| YAML value | Name | Behaviour |
|------------|------|-----------|
| `PromptOptional` | `PromptOptional` | User is prompted; empty input accepted (uses `defaultValue`) |
| `PromptMandatory` | `PromptMandatory` | User is prompted; empty input rejected and re-prompted |
| `PromptSecret` | `PromptSecret` | User is prompted with masked input; value is **not** saved to manifest |
| `DynamicValue` | `DynamicValue` | Value is computed from `expression`; user is never prompted |

### Slot Fields

| Field | Required | Description |
|-------|----------|-------------|
| `key` | yes | Unique identifier; referenced in templates as `.slots.KEY` |
| `label` | no | Human-readable prompt label |
| `description` | no | Additional help text shown during prompting |
| `type` | yes | Slot type string (e.g. `PromptMandatory` — see Slot Types table) |
| `defaultValue` | no | Used when `PromptOptional` input is empty |
| `expression` | when `DynamicValue` | Go template expression evaluated against `ResolveContext` |
| `pipelines` | no | Ordered list of pipeline function calls applied to the value |
| `validation` | no | Single validation rule string |
| `value` | internal | Populated after resolution; do not set manually in the template |

### Validation Rules

A `validation` field contains a single rule string. The user is re-prompted until the value passes.

**String:**
- `pattern:<regex>` — must match regex
- `minLength:<n>` — at least *n* characters
- `maxLength:<n>` — at most *n* characters
- `enum:<a>,<b>,...` — must be one of the listed values
- `startsWith:<prefix>` — must begin with prefix
- `endsWith:<suffix>` — must end with suffix

**Numeric:**
- `numeric` — must be a valid integer
- `minimum:<n>` — integer >= *n*
- `maximum:<n>` — integer <= *n*
- `port` — valid TCP port (1–65535)

**Format:**
- `slug` — lowercase alphanumeric with hyphens, must start with a letter
- `email` — valid email address
- `url` — valid URL with scheme and host
- `semver` — semantic version (`v1.2.3` or `1.2.3`)

### Pipeline Functions

Pipelines listed under `pipelines:` are applied in order after the user provides a value. The same functions are available inline in `[[ ]]` template expressions.

```yaml
pipelines:
  - slugify
  - lower
```

Available functions: `slugify`, `pascalCase`, `camelCase`, `snakeCase`, `upper`, `lower`, `title`, `replace`, `trimPrefix`, `trimSuffix`, `ensureSuffix`, `split`, `join`, `contains`, `hasPrefix`, `hasSuffix`, `repeat`, `trimSpace`, `default`, `generatePassword`.

---

## Service

Services are recorded in the manifest when `spire service add` completes.

```yaml
services:
  - name: payments            # human-readable name
    slugName: payments        # slug form (used in paths)
    slots:                    # resolved service slots
      - key: ServiceName
        value: payments
      - key: WithDB
        value: "yes"
```

| Field | Description |
|-------|-------------|
| `name` | Human-readable service name |
| `slugName` | Slug form used in directory paths |
| `slots` | Resolved slot values for this service |

---

## TemplateFile

Template files are rendered from a source `.tmpl` file in `templates/` to a destination path in the project. They are rendered during `spire init` and, if `regenerateOnServiceChange` is set, every time a service is added.

```yaml
templateFiles:
  - source: templates/devcontainer/.env.tmpl
    destination: .devcontainer/.env
    regenerateOnServiceChange: true
```

| Field | Description |
|-------|-------------|
| `source` | Path to the `.tmpl` source file (relative to project root) |
| `destination` | Output path (relative to project root) |
| `regenerateOnServiceChange` | Re-render the file when `spire service add` runs |

Template files have access to the full `ResolveContext`:

```
[[ .slots.KEY ]]          → application slot value
[[ range .services ]]     → iterate over generated services
  [[ .Name ]]             → service name
  [[ .SlugName ]]         → service slug
  [[ range .Slots ]]      → iterate over service slots
    [[ .Key ]] [[ .Value ]]
  [[ end ]]
[[ end ]]
```

---

## PathRename

Path renames are applied after file rendering. Spire finds every file and directory whose name contains `pattern` and renames it using the result of `expression`. Renames are applied deepest-first to avoid path conflicts.

```yaml
pathRenames:
  - pattern: myapp
    expression: "[[ .slots.ProjectSlugName ]]"

  - pattern: sampleresource
    expression: "[[ .slots.ServiceName ]]"
```

| Field | Description |
|-------|-------------|
| `pattern` | Literal substring to match in file/directory names |
| `expression` | Go template expression evaluated against `ResolveContext`; result replaces `pattern` in the path |

---

## IgnorePaths

A list of directory or file names excluded from Go template rendering. Use this for directories that contain `{{` or `[[` that should not be processed (e.g. Helm charts, vendored dependencies).

```yaml
ignorePaths:
  - .git
  - .spire
  - templates
  - vendor
  - go.sum
  - charts
```

---

## ServiceConfig

Defines how `spire service add` generates a new service.

```yaml
serviceConfig:
  originalPath: services/sampleservice   # blueprint directory in the project
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

| Field | Description |
|-------|-------------|
| `originalPath` | Path of the service blueprint directory (copied and renamed per service) |
| `servicesSlots` | Slot definitions prompted when adding a service |
| `pathRenames` | Path renames applied to the new service directory |
| `postHooks` | Conditional post-processing steps |

### PostHook

```yaml
postHooks:
  - condition: '[[ if ne .slots.WithDB "yes" ]]true[[ end ]]'
    removePaths:
      - database
      - migrations
    pathRenames:
      - pattern: with-db
        expression: "no-db"
```

| Field | Description |
|-------|-------------|
| `condition` | Go template expression; hook fires when it evaluates to `"true"`. Omit to always fire. |
| `removePaths` | Paths (relative to the new service directory) to delete when the hook fires |
| `pathRenames` | Additional path renames to apply when the hook fires |

---

## Upgrade Manifest (`.spire/upgrade-manifest.yaml`)

A separate manifest that controls `spire upgrade` behaviour. Spire creates a default one if it does not exist.

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

| Field | Description |
|-------|-------------|
| `application[].path` | Directory or file to remove and replace from the new template |
| `application[].keep` | Relative paths within the parent directory to back up and restore |
| `services[].path` | Service directory to remove and replace |

---

## Complete Example

```yaml
spireVersion: v0.0.1
templateVersion: v1.2.0
gitRepository: https://github.com/acme/payments

appSlots:
  - key: ProjectName
    label: "Project Name"
    type: PromptMandatory
    validation: "minLength:2"
    value: "ACME Payments"

  - key: ProjectSlugName
    type: DynamicValue
    expression: "[[ .slots.ProjectName | slugify ]]"
    value: "acme-payments"

  - key: PackageName
    label: "Go Module Path"
    type: PromptMandatory
    validation: "minLength:5"
    value: "github.com/acme/payments"

  - key: HttpPort
    label: "HTTP Port"
    type: PromptOptional
    defaultValue: "8080"
    validation: "port"
    value: "8080"

services:
  - name: ledger
    slugName: ledger
    slots:
      - key: ServiceName
        value: ledger
      - key: WithDB
        value: "yes"

templateFiles:
  - source: templates/devcontainer/.env.tmpl
    destination: .devcontainer/.env
    regenerateOnServiceChange: true
  - source: templates/vscode/launch.json.tmpl
    destination: .vscode/launch.json
    regenerateOnServiceChange: true

pathRenames:
  - pattern: myapp
    expression: "[[ .slots.ProjectSlugName ]]"

ignorePaths:
  - .git
  - .spire
  - templates
  - vendor
  - go.sum

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
