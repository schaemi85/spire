# Spire CLI

![Spire Gopher](docs/SpireGopher.png)

A Go-based scaffolding CLI that generates new applications from versioned templates and keeps them in sync as templates evolve. Templates are parameterised with named slots (`[[ .slots.KEY ]]`) and a manifest-driven approach handles slot resolution, path renames, service generation, and safe upgrades.

---

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture](docs/architecture.md) | System design, component diagrams, and data-flow sequences |
| [Command Reference](docs/commands.md) | All commands, flags, and examples |
| [Template Authoring](docs/template-authoring.md) | How to create and maintain a Spire template |
| [Manifest Reference](docs/manifest-reference.md) | Complete `.spire/manifest.yaml` field reference |
| [Plugin System](docs/plugins.md) | Extend Spire with compiled binary plugins |

---

## Installation

### macOS — Homebrew (recommended)

```bash
brew install schaemi85/tap/spire
```

To upgrade later:

```bash
brew upgrade spire
```

> Homebrew distributes Spire as a **cask**, which is macOS-only. Linux users should use the install script below.

### Linux — install script

```bash
curl -fsSL https://raw.githubusercontent.com/schaemi85/spire/main/install.sh | sh
```

By default this installs the latest release to `/usr/local/bin` (falling back to
`$HOME/.local/bin` if that isn't writable). Override with environment variables:

```bash
# Pin a version and/or choose the install directory
SPIRE_VERSION=v0.0.1 BINDIR="$HOME/.local/bin" \
  sh -c "$(curl -fsSL https://raw.githubusercontent.com/schaemi85/spire/main/install.sh)"
```

The script also works on macOS if you prefer not to use Homebrew.

### Manual download

Grab the archive for your platform from the [Releases page](https://github.com/schaemi85/spire/releases),
extract it, and move the `spire` binary onto your `PATH`.

### Verify

```bash
spire version
```

---

## Quick Start

### Generate a project from a template

```bash
spire init https://gitlab.example.com/templates/my-app-template
# or from a local directory
spire init ./local-template
```

Spire prompts for each slot defined in the template manifest, renders all files, renames paths, and runs `git init`.

### Add a service

```bash
cd my-project
spire add-service
```

Copies the service blueprint from `templates/service/`, renders it with service-specific slot values, and registers it in the manifest.

### Keep the project in sync with the template

```bash
spire upgrade           # upgrade to the latest template version
spire upgrade --dry-run # preview changes first
```

### Push changes back to the template

```bash
spire template-sync --output ../my-app-template
```

Reverses resolved slot values back into `[[ ]]` expressions and produces a clean template ready to publish.

---

## How It Works

```
Template repo (versioned git tags)
        │
        │  spire init
        ▼
Generated project (.spire/manifest.yaml)
        │
        ├── spire add-service   → adds services/
        ├── spire upgrade       → pulls latest scaffolding
        └── spire template-sync → pushes changes back to template
```

All behaviour is driven by `.spire/manifest.yaml`. Slots can be mandatory, optional, secret, or computed from expressions. Files containing `[[ ]]` are rendered as Go templates; everything else is copied verbatim.

See the [Architecture doc](docs/architecture.md) for detailed flow diagrams.

---

## Plugins

Spire supports lifecycle plugins — compiled binaries that run at predefined hook points (`before-add-service`, `after-add-service`, `before-upgrade`, `after-upgrade`). They can be written in Go or any language.

```bash
# Build plugin sources from templates/plugins/
spire plugin build

# List installed plugins
spire plugin list
```

See [docs/plugins.md](docs/plugins.md) for the full guide, including the `create-db-schema` example that automatically generates a SQL migration when adding a database-backed service.

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--workdir <path>` | Run the command in the given directory |
| `--non-interactive` | Disable prompts (for CI/CD — use `--set` to supply values) |

---

## Non-Interactive Usage (CI/CD)

```bash
spire init https://gitlab.example.com/templates/my-app \
  --non-interactive \
  --version v1.4.0 \
  --set ProjectName=payments \
  --set PackageName=github.com/acme/payments
```

---

## Build & Release

The build and release process is managed via GitHub Actions.

### CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml))

Runs on every push and pull request to `main`:

1. **Format** — Fails if any file is not `gofmt`-formatted
2. **Vet** — Runs `go vet ./...`
3. **Build** — Compiles all packages (`go build ./...`)
4. **Test** — Runs the full suite with the race detector (`go test -race ./...`)
5. **Lint** — Runs `golangci-lint` (configured in [`.golangci.yml`](.golangci.yml))

### Release ([`.github/workflows/release.yml`](.github/workflows/release.yml))

Triggered by pushing a `v*` git tag. It runs [`goreleaser`](.goreleaser.yaml), which
cross-compiles binaries (linux/windows/darwin, amd64/arm64), injects the version via
ldflags, and publishes a **GitHub Release** with the archives and generated changelog.

```bash
# Cut a release
git tag v0.0.2
git push origin v0.0.2
```

### Local builds

```bash
make           # build the binary for the current platform
make install   # build and install to /usr/local/bin
make snapshot  # build release artifacts for all platforms into dist/ (no publish)
```
