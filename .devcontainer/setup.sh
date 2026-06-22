#!/bin/zsh
set -e

# Verify tools
echo "Verifying installation..."
protoc-gen-go --version
protoc-gen-go-grpc --version
air -v
atlas version
task --version

dos2unix .devcontainer/.env

git config --global core.autocrlf input
git config --global user.name  "${DEV_USER}"
git config --global user.email "${DEV_EMAIL}"

# Docker login (only if a registry is configured)
if [ -n "${DOCKER_REGISTRY}" ]; then
  echo "Logging into ${DOCKER_REGISTRY}..."
  docker login "${DOCKER_REGISTRY}"
fi

echo "Setup complete."
