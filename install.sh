#!/bin/sh
# gssh installer.
#
#   curl -fsSL https://raw.githubusercontent.com/clveryang/gssh/main/install.sh | sh
#
# Downloads a prebuilt binary from GitHub Releases into ~/.local/bin (no sudo).
# Override with GSSH_INSTALL_DIR=/usr/local/bin, GSSH_VERSION=v0.1.0,
# or GSSH_BASE_URL=<mirror> if GitHub is slow where you are.
set -eu

REPO=clveryang/gssh
BIN=gssh
INSTALL_DIR=${GSSH_INSTALL_DIR:-$HOME/.local/bin}

info() { printf '  %s\n' "$*"; }
die()  { printf 'gssh install: %s\n' "$*" >&2; exit 1; }

# --- platform -----------------------------------------------------------
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux|darwin) ;;
  *) die "unsupported OS: $os (build from source: go install github.com/$REPO@latest)" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64)  arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) die "unsupported architecture: $arch" ;;
esac

# --- fetcher ------------------------------------------------------------
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
  fetch_to() { curl -fsSL -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
  fetch_to() { wget -qO "$2" "$1"; }
else
  die "need curl or wget"
fi

# --- version ------------------------------------------------------------
version=${GSSH_VERSION:-}
if [ -z "$version" ]; then
  version=$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
    | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
fi
[ -n "$version" ] || die "could not determine latest version; set GSSH_VERSION=vX.Y.Z"

# --- download -----------------------------------------------------------
tarball="${BIN}_${os}_${arch}.tar.gz"
# GSSH_BASE_URL points at a mirror (or a local dir served over HTTP, which is
# how the installer is tested).
base=${GSSH_BASE_URL:-"https://github.com/$REPO/releases/download/$version"}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

info "gssh $version ($os/$arch)"
fetch_to "$base/$tarball" "$tmp/$tarball" || die "download failed: $base/$tarball"

# Verify against the published checksums when we have a tool for it.
if fetch_to "$base/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
  if command -v sha256sum >/dev/null 2>&1; then
    sum=$(sha256sum "$tmp/$tarball" | cut -d' ' -f1)
  elif command -v shasum >/dev/null 2>&1; then
    sum=$(shasum -a 256 "$tmp/$tarball" | cut -d' ' -f1)
  else
    sum=
  fi
  if [ -n "$sum" ]; then
    grep -q "$sum" "$tmp/checksums.txt" || die "checksum mismatch -- refusing to install"
    info "checksum ok"
  fi
fi

tar -xzf "$tmp/$tarball" -C "$tmp" || die "extract failed"
[ -f "$tmp/$BIN" ] || die "archive did not contain $BIN"

mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/$BIN" "$INSTALL_DIR/$BIN" 2>/dev/null \
  || { cp "$tmp/$BIN" "$INSTALL_DIR/$BIN" && chmod 0755 "$INSTALL_DIR/$BIN"; } \
  || die "could not write to $INSTALL_DIR (set GSSH_INSTALL_DIR)"

info "installed $INSTALL_DIR/$BIN"

# --- PATH hint ----------------------------------------------------------
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    printf '\n  %s is not on your PATH. Add:\n\n    export PATH="%s:$PATH"\n\n' \
      "$INSTALL_DIR" "$INSTALL_DIR"
    ;;
esac

printf '\n  next: %s/%s import\n\n' "$INSTALL_DIR" "$BIN"
