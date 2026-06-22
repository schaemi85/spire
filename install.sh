#!/bin/sh
# Spire CLI installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/schaemi85/spire/main/install.sh | sh
#
# Options (environment variables):
#   SPIRE_VERSION   Version to install (e.g. v0.1.0). Defaults to the latest release.
#   BINDIR          Install directory. Defaults to /usr/local/bin (falls back to
#                   $HOME/.local/bin if /usr/local/bin is not writable).
#
# Examples:
#   SPIRE_VERSION=v0.0.1 sh install.sh
#   BINDIR="$HOME/bin" sh install.sh

set -eu

REPO="schaemi85/spire"
BINARY="spire"

info() { printf '\033[34m==>\033[0m %s\n' "$1"; }
err() { printf '\033[31mError:\033[0m %s\n' "$1" >&2; exit 1; }

# --- detect OS (matches GoReleaser's `title .Os`) ---
os="$(uname -s)"
case "$os" in
  Linux)  OS="Linux" ;;
  Darwin) OS="Darwin" ;;
  *) err "unsupported OS '$os' — download a release manually from https://github.com/$REPO/releases" ;;
esac

# --- detect arch (matches GoReleaser's archive name template) ---
arch="$(uname -m)"
case "$arch" in
  x86_64|amd64)   ARCH="x86_64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *) err "unsupported architecture '$arch' — download a release manually from https://github.com/$REPO/releases" ;;
esac

# --- required tools ---
if command -v curl >/dev/null 2>&1; then
  download() { curl -fsSL "$1" -o "$2"; }
  fetch() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  download() { wget -qO "$2" "$1"; }
  fetch() { wget -qO- "$1"; }
else
  err "either curl or wget is required"
fi
command -v tar >/dev/null 2>&1 || err "tar is required"

# --- resolve version ---
VERSION="${SPIRE_VERSION:-}"
if [ -z "$VERSION" ]; then
  info "Resolving latest release..."
  VERSION="$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"tag_name"' | head -n1 | sed 's/.*"tag_name": *"//;s/".*//')"
  [ -n "$VERSION" ] || err "could not determine the latest version (set SPIRE_VERSION manually)"
fi

ASSET="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$ASSET"

# --- download + extract ---
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

info "Downloading $ASSET ($VERSION)..."
download "$URL" "$TMP/$ASSET" || err "download failed: $URL"

info "Extracting..."
tar -xzf "$TMP/$ASSET" -C "$TMP"
[ -f "$TMP/$BINARY" ] || err "archive did not contain the '$BINARY' binary"
chmod +x "$TMP/$BINARY"

# --- choose install dir ---
DEST="${BINDIR:-/usr/local/bin}"
if [ ! -d "$DEST" ] || [ ! -w "$DEST" ]; then
  if [ "$DEST" = "/usr/local/bin" ] && command -v sudo >/dev/null 2>&1; then
    info "Installing to $DEST (requires sudo)..."
    sudo install -m 0755 "$TMP/$BINARY" "$DEST/$BINARY"
  else
    DEST="$HOME/.local/bin"
    mkdir -p "$DEST"
    info "Installing to $DEST..."
    install -m 0755 "$TMP/$BINARY" "$DEST/$BINARY"
  fi
else
  info "Installing to $DEST..."
  install -m 0755 "$TMP/$BINARY" "$DEST/$BINARY"
fi

info "Installed $BINARY $VERSION to $DEST/$BINARY"

case ":$PATH:" in
  *":$DEST:"*) ;;
  *) printf '\033[33mNote:\033[0m %s is not on your PATH. Add it with:\n  export PATH="%s:$PATH"\n' "$DEST" "$DEST" ;;
esac

"$DEST/$BINARY" version 2>/dev/null || true
