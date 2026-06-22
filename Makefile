# Makefile for managing services

# Variables
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "dev")

# Detect OS and set variables
OS := $(shell uname -s)
ifeq ($(OS),Windows_NT)
	BIN_NAME := spire.exe
	INSTALL_PATH := /usr/local/bin/spire.exe
else
	BIN_NAME := spire
	INSTALL_PATH := /usr/local/bin/spire
endif

# Default target
.PHONY: all
all: spire

.PHONY: help
help:
	@echo "Usage:"
	@echo "  make                                  - Build the Spire CLI"
	@echo "  make install                          - Build and install to /usr/local/bin (fast, local binary)"
	@echo "  make snapshot                         - Build release artifacts for all platforms into dist/"
	@echo "  make snapshot-install                 - Build snapshot and install the current-platform binary to /usr/local/bin"
	@echo "  make help                             - Show this help message"

# Build the Spire CLI
.PHONY: spire
spire:
	@echo "Building Spire CLI version $(VERSION)..."
	@echo "Using OS: $(OS)"
	go build -ldflags "-X github.com/schaemi85/spire/internal/metadata.VERSION=$(VERSION)" -v -o $(BIN_NAME) .
	@echo "Spire CLI built successfully"
	@echo "Run 'make install' to install the CLI to $(INSTALL_PATH)"

# Install the Go CLI to /usr/local/bin
.PHONY: install
install: spire
	install -m755 $(BIN_NAME) $(INSTALL_PATH)
	@echo "Spire CLI installed to $(INSTALL_PATH)"

# Build snapshot release artifacts for all platforms into dist/
.PHONY: snapshot
snapshot:
	goreleaser release --snapshot --clean

# Build snapshot and install the current-platform binary to /usr/local/bin
# Use this to test the goreleaser-built binary locally instead of `make install`
.PHONY: snapshot-install
snapshot-install: snapshot
	@GOOS=$$(go env GOOS); GOARCH=$$(go env GOARCH); \
	BIN=$$(ls dist/spire_$${GOOS}_$${GOARCH}*/spire 2>/dev/null | head -1); \
	if [ -z "$$BIN" ]; then echo "❌ No snapshot binary found for $${GOOS}/$${GOARCH}"; exit 1; fi; \
	install -m755 "$$BIN" $(INSTALL_PATH); \
	echo "✅ Installed $$BIN → $(INSTALL_PATH)"

# Copy the latest Linux snapshot binary into the spire-app devcontainer for testing
.PHONY: deploy-devcontainer
deploy-devcontainer: snapshot
	cp dist/spire_linux_amd64_v1/spire ../spire-app/spire
	@echo "✅ Binary copied — run 'sudo cp /workspace/spire /usr/local/bin/spire' inside the DevContainer"
