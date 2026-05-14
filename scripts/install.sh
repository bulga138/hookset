#!/usr/bin/env sh
# hookset installer — curl -fsSL https://raw.githubusercontent.com/bulga138/hookset/master/install.sh | sh
#
# Detects OS/arch, downloads the latest release binary from GitHub Releases,
# installs to ~/.local/bin (or $HOOKSET_INSTALL_DIR), and adds to PATH.
#
# Environment overrides:
#   HOOKSET_VERSION       — install a specific version (default: latest)
#   HOOKSET_INSTALL_DIR   — install directory (default: ~/.local/bin)
#   HOOKSET_BINARY_PATH   — skip download, copy from this local path (air-gapped)

set -eu

REPO="bulga138/hookset"
BINARY="hookset"
INSTALL_DIR="${HOOKSET_INSTALL_DIR:-$HOME/.local/bin}"

# ── ANSI ─────────────────────────────────────────────────────

if [ -t 1 ]; then
  G='\033[1;32m' R='\033[1;31m' B='\033[1;34m' D='\033[2m' N='\033[0m'
else
  G='' R='' B='' D='' N=''
fi

info() { printf "${B}[hookset]${N} %s\n" "$*"; }
ok()   { printf "  ${G}✓${N} %s\n" "$*"; }
err()  { printf "  ${R}✗${N} %s\n" "$*" >&2; }
die()  { err "$*"; exit 1; }

# ── detect OS / arch ─────────────────────────────────────────

detect_os() {
  case "$(uname -s)" in
    Linux)  echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)      die "Unsupported OS: $(uname -s). Install manually from https://github.com/$REPO/releases" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)   echo "amd64" ;;
    arm64|aarch64)  echo "arm64" ;;
    *)              die "Unsupported arch: $(uname -m)" ;;
  esac
}

# ── download helpers ─────────────────────────────────────────

has() { command -v "$1" >/dev/null 2>&1; }

download() {
  url="$1"; dest="$2"
  if has curl; then
    curl -fsSL "$url" -o "$dest"
  elif has wget; then
    wget -qO "$dest" "$url"
  else
    die "curl or wget is required"
  fi
}

latest_version() {
  url="https://api.github.com/repos/$REPO/releases/latest"
  if has curl; then
    curl -fsSL "$url" | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/'
  elif has wget; then
    wget -qO- "$url" | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/'
  else
    die "curl or wget is required"
  fi
}

# ── verify git version ────────────────────────────────────────

check_git() {
  if ! has git; then
    err "Git is not installed — hookset requires git ≥ 2.54"
    return
  fi
  ver=$(git --version | grep -oE '[0-9]+\.[0-9]+' | head -1)
  maj=$(echo "$ver" | cut -d. -f1)
  min=$(echo "$ver" | cut -d. -f2)
  if [ "$maj" -gt 2 ] || { [ "$maj" -eq 2 ] && [ "$min" -ge 54 ]; }; then
    ok "Git $ver — config-based hooks supported"
  else
    err "Git $ver is too old — hookset requires git ≥ 2.54"
    err "Upgrade: https://git-scm.com/downloads"
  fi
}

# ── main ─────────────────────────────────────────────────────

printf "\n${B}hookset installer${N}\n\n"

check_git

OS=$(detect_os)
ARCH=$(detect_arch)

mkdir -p "$INSTALL_DIR"
DEST="$INSTALL_DIR/$BINARY"
TMP=$(mktemp)

if [ -n "${HOOKSET_BINARY_PATH:-}" ]; then
  # Air-gapped: copy from local path.
  info "Using local binary: $HOOKSET_BINARY_PATH"
  cp "$HOOKSET_BINARY_PATH" "$TMP"
else
  VERSION="${HOOKSET_VERSION:-$(latest_version)}"
  ASSET="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/$REPO/releases/download/v${VERSION}/${ASSET}"
  info "Downloading hookset $VERSION ($OS/$ARCH)…"
  download "$URL" "$TMP.tar.gz"
  tar -xzf "$TMP.tar.gz" -C "$(dirname "$TMP")" "$BINARY"
  mv "$(dirname "$TMP")/$BINARY" "$TMP"
  rm -f "$TMP.tar.gz"
fi

chmod +x "$TMP"
mv "$TMP" "$DEST"
ok "Installed → $DEST"

# ── PATH setup ────────────────────────────────────────────────

add_to_path() {
  shell_rc="$1"
  export_line='export PATH="$HOME/.local/bin:$PATH"'
  if [ -f "$shell_rc" ] && grep -qF "$export_line" "$shell_rc" 2>/dev/null; then
    return
  fi
  printf '\n# hookset\n%s\n' "$export_line" >> "$shell_rc"
  ok "Added ~/.local/bin to PATH in $shell_rc"
}

case "${SHELL:-}" in
  */zsh)  add_to_path "$HOME/.zshrc" ;;
  */bash) add_to_path "$HOME/.bashrc" ;;
  */fish) 
    fish -c "fish_add_path $INSTALL_DIR" 2>/dev/null \
      && ok "Added $INSTALL_DIR to fish PATH" \
      || printf "  ${D}·${N} Add %s to your fish PATH manually\n" "$INSTALL_DIR"
    ;;
esac

printf "\n${G}Done!${N} Run: hookset version\n"
printf "${D}Restart your shell or run: source ~/.zshrc (or ~/.bashrc)${N}\n\n"
