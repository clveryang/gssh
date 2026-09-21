#!/bin/sh
# gssh installer.
#
#   curl -fsSL https://raw.githubusercontent.com/clveryang/gssh/main/install.sh | sh
#
# Downloads a prebuilt binary from GitHub Releases into ~/.local/bin (no sudo).
# Override with GSSH_INSTALL_DIR=/usr/local/bin or GSSH_VERSION=v0.2.0.
# GSSH_BASE_URL=<mirror> moves the download only; checksums still come from
# GitHub, so a malicious mirror cannot validate its own payload.
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
upstream="https://github.com/$REPO/releases/download/$version"

# GSSH_BASE_URL only moves the *bulk download* to a mirror. Checksums are always
# fetched from GitHub, so a mirror that serves a tampered binary is caught by the
# upstream checksum rather than by its own.
base=${GSSH_BASE_URL:-$upstream}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

info "gssh $version ($os/$arch)"
fetch_to "$base/$tarball" "$tmp/$tarball" || die "download failed: $base/$tarball"

# Verification is mandatory. Skipping it silently -- because the file was
# missing or no sha tool was present -- would defeat the point of having it.
if [ "${GSSH_SKIP_CHECKSUM:-0}" = "1" ]; then
  info "WARNING: checksum verification disabled by GSSH_SKIP_CHECKSUM"
else
  fetch_to "$upstream/checksums.txt" "$tmp/checksums.txt" \
    || die "could not fetch checksums from $upstream (set GSSH_SKIP_CHECKSUM=1 to bypass, at your own risk)"

  if command -v sha256sum >/dev/null 2>&1; then
    sum=$(sha256sum "$tmp/$tarball" | cut -d' ' -f1)
  elif command -v shasum >/dev/null 2>&1; then
    sum=$(shasum -a 256 "$tmp/$tarball" | cut -d' ' -f1)
  elif command -v openssl >/dev/null 2>&1; then
    sum=$(openssl dgst -sha256 "$tmp/$tarball" | awk '{print $NF}')
  else
    die "no sha256 tool found (need sha256sum, shasum or openssl)"
  fi

  grep -q "$sum" "$tmp/checksums.txt" || die "checksum mismatch -- refusing to install"
  info "checksum ok"
fi

tar -xzf "$tmp/$tarball" -C "$tmp" || die "extract failed"
[ -f "$tmp/$BIN" ] || die "archive did not contain $BIN"

mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/$BIN" "$INSTALL_DIR/$BIN" 2>/dev/null \
  || { cp "$tmp/$BIN" "$INSTALL_DIR/$BIN" && chmod 0755 "$INSTALL_DIR/$BIN"; } \
  || die "could not write to $INSTALL_DIR (set GSSH_INSTALL_DIR)"

info "installed $INSTALL_DIR/$BIN"

# --- shell setup ------------------------------------------------------
# One command should leave gssh fully working, so the installer adds a small,
# marked block to the user's shell rc: PATH (if needed) and tab completion.
# It is idempotent, prints what it did, and GSSH_NO_MODIFY_RC=1 skips it.
shell_name=$(basename "${SHELL:-sh}")
case "$shell_name" in
  zsh)  rc="$HOME/.zshrc" ;;
  bash) rc="$HOME/.bashrc" ;;
  *)    rc="" ;;
esac

if [ "${GSSH_NO_MODIFY_RC:-0}" = "1" ] || [ -z "$rc" ]; then
  case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) printf '\n  add to your PATH:  export PATH="%s:$PATH"\n' "$INSTALL_DIR" ;;
  esac
elif grep -qs '>>> gssh >>>' "$rc"; then
  info "shell already set up ($rc)"
else
  {
    printf '\n# >>> gssh >>>\n'
    printf 'case ":$PATH:" in *":%s:"*) ;; *) export PATH="%s:$PATH" ;; esac\n' "$INSTALL_DIR" "$INSTALL_DIR"
    # An older manual setup may already source the completion; do not do it twice.
    if ! grep -qs 'gssh completion' "$rc"; then
      if [ "$shell_name" = zsh ]; then
        # compdef only exists after compinit; a bare .zshrc may never call it.
        printf '(( $+functions[compdef] )) || { autoload -Uz compinit && compinit; }\n'
      fi
      # eval, not source <(...): bash 3.2 (macOS /bin/bash) silently ignores
      # sourcing a process substitution.
      printf 'command -v gssh >/dev/null && eval "$(gssh completion %s)"\n' "$shell_name"
    fi
    printf '# <<< gssh <<<\n'
  } >> "$rc"
  info "set up PATH and tab completion in $rc"
fi

printf '\n  done. open a new terminal and run: gssh\n\n'
