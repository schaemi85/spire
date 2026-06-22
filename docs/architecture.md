# Spire CLI — Architecture

## Overview

Spire is a manifest-driven scaffolding CLI written in Go. It solves three related problems:

1. **Project generation** — create a new application from a reusable template, filling in parameterised slots interactively or via flags.
2. **Project maintenance** — keep generated projects in sync with an evolving template through safe, backup-aware upgrades.
3. **Extensibility** — run custom compiled binary plugins at predefined lifecycle hook points, without modifying the CLI itself.

The tool is designed so a template author can publish a versioned Git repository and consumers can both generate fresh projects from it and upgrade their projects as the template evolves, all without losing local customisations.

---

## High-Level Architecture

```mermaid
graph TD
    User["👤 User / CI"]

    subgraph CLI["Spire CLI (cobra)"]
        Init["spire init"]
        AddSvc["spire service add"]
        TplSync["spire template sync"]
        Upgrade["spire upgrade"]
        Backup["spire backup"]
        PluginCmd["spire plugin\nlist / build / run"]
        Version["spire version"]
    end

    subgraph Core["Core Engine"]
        SlotResolver["Slot Resolver\n(prompt / compute)"]
        Renderer["Template Renderer\n([[ ]] delimiters)"]
        PathRenamer["Path Renamer"]
        PostHooks["Post-Hooks\n(conditional removals)"]
    end

    subgraph PluginEngine["Plugin Engine"]
        PluginRunner["Plugin Runner\n(go-plugin / net/rpc)"]
        PluginBins[".spire/plugins/<hook>/\ncompiled binaries"]
        PluginSrcs["templates/plugins/<hook>/<name>/\nGo source files"]
    end

    subgraph Storage["Persistence"]
        Manifest[".spire/manifest.yaml"]
        UpgradeMf[".spire/upgrade-manifest.yaml"]
        BackupDir[".spire/backup/<timestamp>/"]
    end

    subgraph Sources["Template Sources"]
        GitSrc["Git Source\n(go-git v6)"]
        LocalSrc["Local Source\n(filesystem)"]
    end

    ProjectFS["📁 Project Files"]

    User --> CLI
    Init --> SlotResolver
    Init --> GitSrc
    Init --> LocalSrc
    AddSvc --> SlotResolver
    AddSvc --> PluginRunner
    TplSync --> Renderer
    Upgrade --> GitSrc
    Upgrade --> BackupDir
    Upgrade --> PluginRunner
    PluginCmd --> PluginRunner
    PluginCmd --> PluginSrcs

    GitSrc --> ProjectFS
    LocalSrc --> ProjectFS

    SlotResolver --> Renderer
    Renderer --> PathRenamer
    PathRenamer --> PostHooks
    PostHooks --> ProjectFS

    PluginRunner --> PluginBins
    PluginSrcs -->|spire plugin build| PluginBins
    PluginBins --> ProjectFS

    Manifest --> SlotResolver
    Manifest --> Renderer
    Manifest --> PluginRunner
    UpgradeMf --> Upgrade
    Backup --> BackupDir
```

---

## Command Flow Diagrams

### `spire init` — Project Initialisation

```mermaid
sequenceDiagram
    actor User
    participant CLI as spire init
    participant Src as Template Source
    participant Slots as Slot Resolver
    participant Renderer
    participant FS as File System

    User->>CLI: spire init <template> [--set K=V]
    CLI->>Src: Download(version)
    Src-->>CLI: scaffolding directory

    CLI->>Slots: ResolveSlots(appSlots, overrides)
    loop For each slot
        alt PromptMandatory / PromptOptional / PromptSecret
            Slots->>User: prompt
            User-->>Slots: value
        else DynamicValue
            Slots->>Slots: evaluate expression
        end
        Slots->>Slots: validate + apply pipelines
    end
    Slots-->>CLI: ResolveContext

    CLI->>Renderer: RenderProjectFiles(dir, ctx)
    CLI->>Renderer: RenderTemplateFiles(templateFiles, ctx)
    CLI->>FS: ApplyPathRenames(renames, ctx)
    CLI->>FS: git init + initial commit
    CLI-->>User: project ready
```

### `spire service add` — Service Addition

```mermaid
sequenceDiagram
    actor User
    participant CLI as spire service add
    participant Slots as Slot Resolver
    participant Plugins as Plugin Runner
    participant Renderer
    participant FS as File System

    User->>CLI: spire service add
    CLI->>FS: Load manifest (.spire/manifest.yaml)
    CLI->>Slots: ResolveSlots(serviceConfig.servicesSlots)
    Slots->>User: prompt service slots
    User-->>Slots: values

    CLI->>Plugins: RunHook(before-add-service, HookContext)
    Plugins->>FS: execute .spire/plugins/before-add-service/*

    CLI->>FS: Copy templates/service → services/<name>
    CLI->>Renderer: RenderProjectFiles(service dir)
    CLI->>FS: ApplyPathRenames (service renames)
    CLI->>FS: Evaluate PostHooks (conditional removals)
    CLI->>FS: go work use -r services/
    CLI->>Renderer: Re-render templateFiles (regenerateOnServiceChange)
    CLI->>FS: Save updated manifest

    CLI->>Plugins: RunHook(after-add-service, HookContext)
    Plugins->>FS: execute .spire/plugins/after-add-service/*

    CLI-->>User: service added
```

### `spire upgrade` — Template Upgrade

```mermaid
sequenceDiagram
    actor User
    participant CLI as spire upgrade
    participant Src as Git Source
    participant Backup
    participant FS as File System
    participant Hooks as Post-Upgrade Hooks
    participant Plugins as Plugin Runner

    User->>CLI: spire upgrade [--dry-run]
    CLI->>FS: checkGitStatus (abort if dirty)

    CLI->>Plugins: RunHook(before-upgrade, HookContext)
    Plugins->>FS: execute .spire/plugins/before-upgrade/*

    CLI->>Src: Download(latest version)
    Src-->>CLI: new template directory

    CLI->>Backup: Back up files marked "keep"
    CLI->>FS: Remove + replace application paths
    CLI->>FS: Remove + replace service paths
    CLI->>Backup: Restore kept files

    CLI->>Hooks: runPostUpgradeHooks
    Hooks->>FS: RenderProjectFiles
    Hooks->>FS: ApplyPathRenames
    Hooks->>FS: task clean / proto:gen / build / tidy / format

    CLI->>Plugins: RunHook(after-upgrade, HookContext)
    Plugins->>FS: execute .spire/plugins/after-upgrade/*

    CLI->>FS: Update manifest templateVersion
    CLI-->>User: upgrade complete
```

### `spire template sync` — Template Authoring

```mermaid
sequenceDiagram
    actor Author
    participant CLI as spire template sync
    participant FS as File System

    Author->>CLI: spire template sync --output ../my-template
    CLI->>FS: CopyDir(project → output)
    CLI->>FS: ReplaceInFiles(slot values → [[ .slots.KEY ]])
    CLI->>FS: Copy service → templates/service/
    CLI->>FS: ReplaceInFiles(service slot values → expressions)
    CLI->>FS: Reverse PathRenames
    CLI->>FS: Remove generated services from output
    CLI->>FS: Clear slot values + services in manifest
    CLI->>FS: Save clean manifest
    CLI-->>Author: template ready to publish
```

---

## Package Map

```mermaid
graph LR
    subgraph cmd
        CInit["cmd/application\ninit_application.go"]
        CSvc["cmd/service\nservice.go\nadd_service.go"]
        CTpl["cmd/template\ntemplate.go\nsync.go"]
        CMfst["cmd/manifest\nmanifest.go\nvalidate.go"]
        CUpg["cmd/upgrade\nupgrade.go\nbackup.go\nhooks.go\nmanifest.go"]
        CPlugin["cmd/plugin\nplugin.go"]
        CUtil["cmd/utilities\nversion.go"]
    end

    subgraph internal
        IMfst["internal/manifest\ntypes.go\nmanifest.go"]
        IEng["internal/engine\ncontext.go\nslots.go\ngenerate.go\npipelines.go\nvalidation.go"]
        IMeta["internal/metadata\nVERSION"]
        ISrc["internal/templatesource\ngit.go / local.go"]
        ITools["internal/tools\ntools.go copy.go\nprompting.go validation.go\nyaml.go pwd.go"]
        IPlugin["internal/plugin\ntypes.go / runner.go"]
    end

    subgraph pluginsdk["Plugin SDK (public)"]
        PSDK["plugin/sdk\nsdk.go\n(Hook interface, HookContext,\nHandshakeConfig, PluginMap)"]
    end

    CInit --> ISrc
    CInit --> IEng
    CInit --> IMfst
    CInit --> ITools
    CSvc --> IEng
    CSvc --> IMfst
    CSvc --> ITools
    CSvc --> IPlugin
    CTpl --> IEng
    CTpl --> IMfst
    CMfst --> IEng
    CMfst --> IMfst
    CUpg --> ISrc
    CUpg --> IMfst
    CUpg --> ITools
    CUpg --> IPlugin
    CPlugin --> IPlugin
    CPlugin --> IMfst
    CUtil --> IMeta
    IEng --> IMfst
    IPlugin --> PSDK
```

---

## Slot Resolution Pipeline

Each slot goes through a defined resolution pipeline before its value is stored in the `ResolveContext`.

```mermaid
flowchart LR
    A["Slot definition\nin manifest"] --> B{Type?}

    B -->|PromptMandatory| C["Prompt user\n(re-prompt on empty)"]
    B -->|PromptOptional| D["Prompt user\n(use defaultValue if empty)"]
    B -->|PromptSecret| E["Masked input\n(cleared from manifest on save)"]
    B -->|DynamicValue| F["Evaluate\nGo template expression"]

    C --> G[Validate\nagainst rule]
    D --> G
    E --> G
    F --> H

    G -->|pass| H["Apply pipelines\n(slugify, upper, lower…)"]
    G -->|fail| C

    H --> I["Store in\nResolveContext.Slots"]
```

---

## Manifest Structure

```mermaid
classDiagram
    class SpireManifest {
        SpireVersion string
        TemplateVersion string
        GitRepository string
        AppSlots []Slot
        Services []Service
        TemplateFiles []TemplateFile
        PathRenames []PathRename
        IgnorePaths []string
        ServiceConfig ServiceConfig
    }

    class Slot {
        Key string
        Label string
        Description string
        Type int
        DefaultValue string
        Expression string
        Pipelines []string
        Validation string
        Value string
    }

    class Service {
        Name string
        SlugName string
        Slots []Slot
    }

    class TemplateFile {
        Source string
        Destination string
        RegenerateOnServiceChange bool
    }

    class PathRename {
        Pattern string
        Expression string
    }

    class ServiceConfig {
        OriginalPath string
        ServicesSlots []Slot
        PathRenames []PathRename
        PostHooks []PostHook
    }

    class PostHook {
        Condition string
        RemovePaths []string
        PathRenames []PathRename
    }

    SpireManifest "1" --> "*" Slot : appSlots
    SpireManifest "1" --> "*" Service : services
    SpireManifest "1" --> "*" TemplateFile : templateFiles
    SpireManifest "1" --> "*" PathRename : pathRenames
    SpireManifest "1" --> "1" ServiceConfig : serviceConfig
    Service "1" --> "*" Slot : slots
    ServiceConfig "1" --> "*" Slot : servicesSlots
    ServiceConfig "1" --> "*" PathRename : pathRenames
    ServiceConfig "1" --> "*" PostHook : postHooks
```

---

## Template Source Abstraction

```mermaid
classDiagram
    class Source {
        <<interface>>
        +ListVersions(ctx, limit) []string
        +Download(ctx, version) string
        +Cleanup()
    }

    class GitSource {
        repoURL string
        +ListVersions(ctx, limit) []string
        +Download(ctx, version) string
        +Cleanup()
    }

    class LocalSource {
        dir string
        +ListVersions(ctx, limit) error
        +Download(ctx, version) string
        +Cleanup()
    }

    Source <|.. GitSource
    Source <|.. LocalSource
```

`GitSource` performs a shallow clone (`--depth 1`) of the requested tag using the user's existing Git credentials (SSH keys, credential helpers). It lists available versions by parsing `ls-remote` tags sorted by semantic version — no full clone required.

`LocalSource` returns the directory path directly and is used with `--template-local`.

---

## File Rendering Engine

All text files containing `[[ ]]` delimiters are processed as Go templates. Binary files and paths listed in `ignorePaths` are skipped.

```mermaid
flowchart TD
    A["Walk project directory"] --> B{"Is text file?"}
    B -->|no| Z["Skip"]
    B -->|yes| C{"In ignorePaths?"}
    C -->|yes| Z
    C -->|no| D{"Contains\n[[ ]]?"}
    D -->|no| Z
    D -->|yes| E["Parse as Go template\n(delimiters: [[ ]])"]
    E --> F["Execute with\nResolveContext\n(.slots.* / .services)"]
    F --> G["Write rendered\ncontent back to file"]
```

Pipeline functions available within templates:

| Function | Description |
|----------|-------------|
| `slugify` | Lowercase alphanumeric with hyphens |
| `pascalCase` | PascalCase |
| `camelCase` | camelCase |
| `snakeCase` | snake_case |
| `upper` / `lower` / `title` | Case conversion |
| `replace` | String replacement |
| `trimPrefix` / `trimSuffix` | Trim affixes |
| `ensureSuffix` | Append if not present |
| `split` / `join` | String splitting/joining |
| `contains` / `hasPrefix` / `hasSuffix` | Predicates |
| `repeat` | Repeat string N times |
| `default` | Fallback value |
| `generatePassword` | Random alphanumeric password; optional second arg `true` includes special characters |

---

## Upgrade Safety Model

```mermaid
flowchart TD
    A["spire upgrade"] --> B["Check git status\n(fail if dirty unless --force)"]
    B --> C["Download latest template"]
    C --> D["Read upgrade-manifest.yaml\n(paths to keep / replace)"]
    D --> E["Backup files marked 'keep'\n→ .spire/backup/<timestamp>/"]
    E --> F["Remove & replace\napplication paths"]
    F --> G["Remove & replace\nservice paths"]
    G --> H["Restore backed-up files"]
    H --> I["runPostUpgradeHooks\n(render, rename, task targets)"]
    I --> J["Update templateVersion\nin manifest"]
    J --> K["Done ✓"]
```

The backup system is independent (`spire backup create/restore/list/clean`) and can be used outside of upgrades for any checkpoint workflow.

---

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Custom `[[ ]]` delimiters | Avoids conflicts with `{{ }}` used by Helm, GitHub Actions, Bruno, and other tools that may be present in generated projects |
| Manifest-driven | A single `.spire/manifest.yaml` is the source of truth for slots, services, renames, and template files — no code changes needed to customise behaviour |
| Slot pipelines | Derived values (slugs, case variants) are computed from one canonical input, reducing the number of prompts and preventing inconsistencies |
| Reversible templates | `spire template sync` re-parameterises a living project back into a template, so the template can be maintained as a real working application |
| Shallow git clone | Only the requested tag is fetched, keeping network usage minimal even for large template repositories |
| Backup before upgrade | Files the user wants to customise are snapshotted before the upgrade replaces them, then restored — merging is not required |
| Compiled binary plugins | The Go native `plugin` package is Linux/macOS only. Compiled binary subprocesses managed by [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin) (net/rpc) are fully cross-platform (including Windows), crash-safe, and independently versioned |
| Plugin sources in `templates/plugins/` | Plugin sources travel with the template and project so they can be rebuilt for any target OS with `spire plugin build`; `spire template sync` preserves them during round-trips |
