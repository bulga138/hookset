#!/usr/bin/env sh
# hookset installer
#
# Usage (always pin to a tagged release — never pipe from master/HEAD):
#   curl -fsSL https://github.com/bulga138/hookset/releases/latest/download/install.sh | sh
#
# Or for a specific version:
#   curl -fsSL https://github.com/bulga138/hookset/releases/download/v1.2.3/install.sh | sh
#
# Environment overrides:
#   HOOKSET_VERSION       — install a specific version tag, e.g. "1.2.3" (default: latest)
#   HOOKSET_INSTALL_DIR   — install directory (default: ~/.local/bin)
#   HOOKSET_BINARY_PATH   — skip download, copy from this local path (air-gapped installs)
#   HOOKSET_VERIFY_GPG=1  — additionally verify GPG signature (requires gpg on PATH)
#   HOOKSET_DRY_RUN=1     — print what would be done without downloading or writing files

set -eu

REPO="bulga138/hookset"
BINARY="hookset"
INSTALL_DIR="${HOOKSET_INSTALL_DIR:-$HOME/.local/bin}"
DRY_RUN="${HOOKSET_DRY_RUN:-0}"
VERIFY_GPG="${HOOKSET_VERIFY_GPG:-0}"

# ── ANSI ──────────────────────────────────────────────────────────────────────

if [ -t 1 ]; then
  G='\033[1;32m' R='\033[1;31m' B='\033[1;34m' Y='\033[1;33m' D='\033[2m' N='\033[0m'
else
  G='' R='' B='' Y='' D='' N=''
fi

info()  { printf "${B}[hookset]${N} %s\n" "$*"; }
ok()    { printf "  ${G}✓${N} %s\n" "$*"; }
warn()  { printf "  ${Y}⚠${N} %s\n" "$*" >&2; }
err()   { printf "  ${R}✗${N} %s\n" "$*" >&2; }
die()   { err "$*"; exit 1; }
dry()   { printf "  ${D}[dry-run]${N} %s\n" "$*"; }

# ── detect OS / arch ──────────────────────────────────────────────────────────

has() { command -v "$1" >/dev/null 2>&1; }

detect_os() {
  case "$(uname -s)" in
    Linux)  echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)      die "Unsupported OS: $(uname -s). Install manually: https://github.com/$REPO/releases" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)   echo "amd64" ;;
    arm64|aarch64)  echo "arm64" ;;
    *)              die "Unsupported arch: $(uname -m)" ;;
  esac
}

# ── git version check ─────────────────────────────────────────────────────────

check_git() {
  if ! has git; then
    warn "git not found — hookset requires git ≥ 2.54. Install git first."
    return
  fi
  ver="$(git --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+' | head -1)"
  maj="$(printf '%s' "$ver" | cut -d. -f1)"
  min="$(printf '%s' "$ver" | cut -d. -f2)"
  if [ "$maj" -gt 2 ] || { [ "$maj" -eq 2 ] && [ "$min" -ge 54 ]; }; then
    ok "git $ver — native config hooks supported (≥ 2.54)"
  else
    warn "git $ver is too old — hookset requires git ≥ 2.54"
    warn "Upgrade: https://git-scm.com/downloads"
  fi
}

# ── download helpers ──────────────────────────────────────────────────────────

download() {
  url="$1"; dest="$2"
  if has curl; then
    curl --fail --silent --show-error --location --retry 3 "$url" -o "$dest"
  elif has wget; then
    wget --quiet --tries=3 -O "$dest" "$url"
  else
    die "curl or wget is required to download hookset"
  fi
}

latest_version() {
  api="https://api.github.com/repos/$REPO/releases/latest"
  if has curl; then
    raw="$(curl --fail --silent --show-error --location "$api")"
  elif has wget; then
    raw="$(wget --quiet -O- "$api")"
  else
    die "curl or wget is required"
  fi
  # Extract the tag_name value, strip leading "v".
  printf '%s' "$raw" | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/' | head -1
}

# ── checksum verification ─────────────────────────────────────────────────────

verify_sha256() {
  file="$1"; expected_file="$2"
  expected="$(cat "$expected_file" | awk '{print $1}')"
  if has sha256sum; then
    actual="$(sha256sum "$file" | awk '{print $1}')"
  elif has shasum; then
    actual="$(shasum -a 256 "$file" | awk '{print $1}')"
  else
    warn "sha256sum/shasum not found — skipping checksum verification"
    return
  fi
  if [ "$actual" != "$expected" ]; then
    die "SHA-256 mismatch for $(basename "$file")\n  expected: $expected\n  got:      $actual\n  The download may be corrupt or tampered with. Aborting."
  fi
  ok "SHA-256 checksum verified"
}

verify_gpg_sig() {
  file="$1"; sig="$2"
  if ! has gpg; then
    warn "gpg not found — skipping GPG signature verification (HOOKSET_VERIFY_GPG=1 was set)"
    return
  fi
  if ! gpg --verify "$sig" "$file" >/dev/null 2>&1; then
    die "GPG signature verification failed for $(basename "$file"). Aborting."
  fi
  ok "GPG signature verified"
}

# ── main ──────────────────────────────────────────────────────────────────────

printf "\n${B}hookset installer${N}\n\n"

check_git

OS="$(detect_os)"
ARCH="$(detect_arch)"

if [ "$DRY_RUN" = "1" ]; then
  VERSION="${HOOKSET_VERSION:-$(latest_version)}"
  ASSET="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
  URL="https://github.com/$REPO/releases/download/v${VERSION}/${ASSET}"
  dry "Would download:  $URL"
  dry "Would download:  $URL.sha256"
  if [ "$VERIFY_GPG" = "1" ]; then
    dry "Would download:  $URL.asc"
    dry "Would verify GPG signature"
  fi
  dry "Would install to: $INSTALL_DIR/$BINARY"
  printf "\n${D}Dry run complete — no files were written.${N}\n\n"
  exit 0
fi

mkdir -p "$INSTALL_DIR"
DEST="$INSTALL_DIR/$BINARY"
TMPDIR_INST="$(mktemp -d)"
# Ensure temp dir is cleaned up on exit (success or failure).
trap 'rm -rf "$TMPDIR_INST"' EXIT

if [ -n "${HOOKSET_BINARY_PATH:-}" ]; then
  # Air-gapped install: copy from local path, skip download.
  info "Air-gapped install from: $HOOKSET_BINARY_PATH"
  cp "$HOOKSET_BINARY_PATH" "$TMPDIR_INST/$BINARY"
else
  VERSION="${HOOKSET_VERSION:-$(latest_version)}"
  if [ -z "$VERSION" ]; then
    die "Could not determine latest version. Set HOOKSET_VERSION=x.y.z to install a specific version."
  fi

  ASSET="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
  BASE_URL="https://github.com/$REPO/releases/download/v${VERSION}"
  URL="${BASE_URL}/${ASSET}"
  SHA_URL="${URL}.sha256"

  info "Downloading hookset v${VERSION} (${OS}/${ARCH})…"

  # Download the binary archive.
  download "$URL" "$TMPDIR_INST/$ASSET"
  ok "Downloaded $ASSET"

  # Download and verify SHA-256 checksum.
  info "Verifying SHA-256 checksum…"
  download "$SHA_URL" "$TMPDIR_INST/$ASSET.sha256"
  verify_sha256 "$TMPDIR_INST/$ASSET" "$TMPDIR_INST/$ASSET.sha256"

  # Optionally verify GPG signature.
  if [ "$VERIFY_GPG" = "1" ]; then
    info "Verifying GPG signature (HOOKSET_VERIFY_GPG=1)…"
    download "${URL}.asc" "$TMPDIR_INST/$ASSET.asc"
    verify_gpg_sig "$TMPDIR_INST/$ASSET" "$TMPDIR_INST/$ASSET.asc"
  fi

  # Extract binary.
  tar -xzf "$TMPDIR_INST/$ASSET" -C "$TMPDIR_INST" "$BINARY"
fi

chmod +x "$TMPDIR_INST/$BINARY"
mv "$TMPDIR_INST/$BINARY" "$DEST"
ok "Installed → $DEST"

# ── PATH setup ────────────────────────────────────────────────────────────────

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
    if has fish; then
      fish -c "fish_add_path $INSTALL_DIR" 2>/dev/null \
        && ok "Added $INSTALL_DIR to fish PATH" \
        || printf "  ${D}·${N} Add %s to your fish PATH manually\n" "$INSTALL_DIR"
    fi
    ;;
esac

printf "\n${G}Done!${N} Run: hookset version\n"
printf "${D}Restart your shell or run: source ~/.zshrc (or ~/.bashrc)${N}\n\n"
