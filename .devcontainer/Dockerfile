## -------------------------------------------------------------------
## Base image: configurable registry + Go version
## Default: Docker Hub (golang:1.26.0-bookworm)
## Corporate example: artifactory.company.com/golang:1.26.0-bookworm
## -------------------------------------------------------------------
ARG BASE_IMAGE=golang:1.26.0-bookworm
FROM ${BASE_IMAGE}

# ---- User / Auth (optional, for corporate proxies) ----
ARG DEV_USER=""
ARG REGISTRY_TOKEN=""

# ---- Go proxy configuration ----
# Default: public Go proxy. Override for corporate Artifactory/Nexus.
# Example: https://user:token@artifactory.company.com/artifactory/api/go/go-virtual
ARG GOPROXY=https://proxy.golang.org,direct
ARG GONOSUMDB=""
ARG GONOSUMCHECK=""

# ---- Tool versions ----
ARG PROTOC_VERSION=32.1
ARG PROTOC_GEN_GO_VERSION=v1.36.9
ARG PROTOC_GEN_GO_GRPC_VERSION=v1.5.1
ARG AIR_VERSION=v1.63.0
ARG ATLAS_VERSION=v0.36.0
ARG TASK_VERSION=v3.45.3
ARG GOLANGCI_LINT_VERSION=v2.10.1
ARG PROTOC_RELEASE=linux-x86_64

# ---- Old spire binary for upgrade tests (optional) ----
ARG OLD_SPIRE_MODULE=""
ARG OLD_SPIRE_VERSION=""

ENV GOBIN=/go/bin
ENV GONOSUMDB=${GONOSUMDB}
ENV GONOSUMCHECK=${GONOSUMCHECK}
ENV GOPROXY=${GOPROXY}
ENV PATH=$PATH:/go/bin

# ---- Custom CA certificates (optional) ----
# Drop .crt/.pem files into certs/ for corporate SSL interception / private CAs
COPY certs/ /tmp/custom-certs/
RUN if ls /tmp/custom-certs/*.crt /tmp/custom-certs/*.pem 2>/dev/null | head -1 > /dev/null; then \
      cp /tmp/custom-certs/*.crt /tmp/custom-certs/*.pem /usr/local/share/ca-certificates/ 2>/dev/null || true; \
      update-ca-certificates; \
    fi && \
    rm -rf /tmp/custom-certs

ENV CURL_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt

# Install zsh and common tools
RUN apt-get update && apt-get install -y \
    curl git rsync fzf unzip zsh locales dos2unix xdg-utils \
    && locale-gen en_US.UTF-8

RUN curl -LSs https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${PROTOC_RELEASE}.zip -o protoc.zip \
 && unzip -qq protoc.zip -d /usr/local \
 && rm protoc.zip

# Install oh-my-zsh for vscode user
RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended --keep-zshrc

# Set ZSH as default
ENV SHELL=/bin/zsh

# Install Go tools
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION} && \
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION} && \
go install github.com/air-verse/air@${AIR_VERSION} && \
curl -sSf https://atlasgo.sh | sh -s -- -y  && \
curl -sSL https://taskfile.dev/install.sh | sh -s -- -b ${GOBIN} ${TASK_VERSION} && \
go install -v golang.org/x/tools/gopls@latest && \
go install -v github.com/go-delve/delve/cmd/dlv@latest && \
go install -v golang.org/x/tools/cmd/goimports@latest && \
go install -v honnef.co/go/tools/cmd/staticcheck@latest && \
go install github.com/goreleaser/goreleaser/v2@latest

# Setup .zshrc
RUN echo '\n# Load .env variables\nif [ -f /spire-cli/.devcontainer/.env ]; then\n  export $(grep -v "^#" /spire-cli/.devcontainer/.env | xargs)\nfi\neval "$(task --completion zsh)"\n' >> /root/.zshrc

# Add zsh-interactive-cd to plugins list in .zshrc
RUN sed -i '/^plugins=(/ s/)/ zsh-interactive-cd)/' /root/.zshrc

# Install golangci-lint
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin ${GOLANGCI_LINT_VERSION}

# Optionally install an old version of spire for upgrade tests
RUN if [ -n "${OLD_SPIRE_MODULE}" ] && [ -n "${OLD_SPIRE_VERSION}" ]; then \
      go install ${OLD_SPIRE_MODULE}@${OLD_SPIRE_VERSION} && \
      mv ${GOBIN}/spire ${GOBIN}/old_spire; \
    fi