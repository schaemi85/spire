# Spire CLI - DevContainer

A fully portable DevContainer that works out of the box with public infrastructure (Docker Hub, GitHub, Go proxy) and can be customized for any corporate environment.

## Quick start (public / open-source)

1. Copy the example env file:
   ```bash
   cp .devcontainer/.env.example .devcontainer/.env
   ```
2. Set `DEV_USER` and `DEV_EMAIL` in `.env`.
3. Open the project in VS Code and select **"Reopen in Container"**.

That's it — no tokens, no registries, no certificates needed.

## Corporate / enterprise setup

All corporate-specific settings are driven by **`.devcontainer/.env`** and the **`certs/`** folder. Nothing is hard-coded.

### Custom CA certificates (SSL interception, private CAs)

Drop your `.crt` or `.pem` files into `.devcontainer/certs/`. They are automatically installed into the container's trust store at build time.

> Certificate files are gitignored by default so they won't leak into the repository.

### Private container registry (Artifactory, ACR, ECR, …)

Set `BASE_IMAGE` in `.env` to pull the Go base image from your corporate registry:

```env
BASE_IMAGE=artifactory.company.com/golang:1.26.0-bookworm
```

To authenticate with a registry on container start:

```env
DOCKER_REGISTRY=artifactory.company.com
REGISTRY_TOKEN=<your-token>
```

### Private Go module proxy

Override the Go proxy to use a corporate Artifactory, Athens, or other Go proxy:

```env
GOPROXY_URL=https://user:token@artifactory.company.com/artifactory/api/go/go-virtual
GONOSUMDB_VALUE=company.com
GONOSUMCHECK_VALUE=company.com
```

### Old spire binary for upgrade tests

To install a previous version of spire as `old_spire`:

```env
OLD_SPIRE_MODULE=github.com/your-org/spire
OLD_SPIRE_VERSION=v1.4.0
```

## Configuration reference

| Variable | Default | Description |
|---|---|---|
| `DEV_USER` | *(required)* | Git username |
| `DEV_EMAIL` | *(required)* | Git email |
| `BASE_IMAGE` | `golang:1.26.0-bookworm` | Base Docker image |
| `DOCKER_REGISTRY` | *(empty — skip login)* | Registry for `docker login` at startup |
| `REGISTRY_TOKEN` | *(empty)* | Token for registry authentication |
| `GOPROXY` | `https://proxy.golang.org,direct` | Go module proxy URL |
| `GONOSUMDB` | *(empty)* | Comma-separated module patterns to skip the sum DB |
| `GONOSUMCHECK` | *(empty)* | Comma-separated module patterns to skip checksum verification |
| `OLD_SPIRE_MODULE` | *(empty)* | Go module path for old spire |
| `OLD_SPIRE_VERSION` | *(empty)* | Version of old spire to install |
