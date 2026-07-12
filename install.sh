#!/bin/sh
# Install the hera-agent-godot CLI on macOS / Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/NotNull92/hera-agent-godot/main/install.sh | sh
#
# Environment overrides:
#   HERA_VERSION   release tag to install (default: latest)
#   HERA_BIN_DIR   install directory     (default: ~/.local/bin)
#
# This installs the CLI only. The Godot addon is a separate drop-in folder —
# grab hera-agent-godot-addon.zip from the release and unzip it into your
# project root (creating <project>/addons/hera_agent_godot).
set -eu

REPO="NotNull92/hera-agent-godot"
VERSION="${HERA_VERSION:-latest}"
BIN_DIR="${HERA_BIN_DIR:-$HOME/.local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) echo "hera: unsupported architecture: $arch" >&2; exit 1 ;;
esac
case "$os" in
  linux | darwin) ;;
  *) echo "hera: unsupported OS: $os (use install.ps1 on Windows)" >&2; exit 1 ;;
esac

asset="hera-${os}-${arch}.tar.gz"
if [ "$VERSION" = "latest" ]; then
  url="https://github.com/${REPO}/releases/latest/download/${asset}"
else
  url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "hera: downloading $url"
if command -v curl >/dev/null 2>&1; then
  curl -fSL "$url" -o "$tmp/hera.tar.gz"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp/hera.tar.gz" "$url"
else
  echo "hera: need curl or wget" >&2
  exit 1
fi

tar -C "$tmp" -xzf "$tmp/hera.tar.gz"
mkdir -p "$BIN_DIR"
install -m 0755 "$tmp/hera" "$BIN_DIR/hera"
# Transitional alias for scripts that still call the long name.
ln -sf hera "$BIN_DIR/hera-agent-godot"

echo "hera: installed to $BIN_DIR/hera (alias: hera-agent-godot)"
"$BIN_DIR/hera" version >/dev/null 2>&1 && echo "hera: version $("$BIN_DIR/hera" version)"

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) echo "hera: add $BIN_DIR to your PATH, e.g. export PATH=\"$BIN_DIR:\$PATH\"" ;;
esac
