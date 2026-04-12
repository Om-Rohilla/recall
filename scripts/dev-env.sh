#!/bin/sh
# scripts/dev-env.sh — One-time developer environment setup for Recall CLI
#
# Ensures the Go toolchain is on PATH in every shell by patching the user's
# shell config files. Safe to run repeatedly (idempotent).
#
# Usage:
#   sh scripts/dev-env.sh
#   sh scripts/dev-env.sh --dry-run   (show changes without applying)

set -e

DRY_RUN=0
if [ "${1}" = "--dry-run" ]; then
  DRY_RUN=1
fi

info()  { printf '  \033[1;34m→\033[0m %s\n' "$1"; }
ok()    { printf '  \033[1;32m✓\033[0m %s\n' "$1"; }
warn()  { printf '  \033[1;33m⚠\033[0m %s\n' "$1"; }
skip()  { printf '  \033[2m–\033[0m %s\n' "$1"; }

# ---------------------------------------------------------------------------
# Locate the Go binary
# ---------------------------------------------------------------------------
find_go() {
  # Priority: already on PATH, then known install locations
  if command -v go > /dev/null 2>&1; then
    echo "$(command -v go)"
    return
  fi
  for candidate in \
    "${HOME}/go/bin/go" \
    "/usr/local/go/bin/go" \
    "/usr/lib/go/bin/go" \
    "/opt/homebrew/bin/go"; do
    if [ -x "$candidate" ]; then
      echo "$candidate"
      return
    fi
  done
  echo ""
}

GO_BIN="$(find_go)"
if [ -z "$GO_BIN" ]; then
  warn "Go toolchain not found. Install from https://go.dev/dl/"
  warn "Re-run this script after installing Go."
  exit 1
fi

GO_DIR="$(dirname "$GO_BIN")"
ok "Found Go at: $GO_BIN ($(${GO_BIN} version 2>/dev/null))"

# ---------------------------------------------------------------------------
# Build the export line we want to inject
# ---------------------------------------------------------------------------
EXPORT_LINE="export PATH=\"${GO_DIR}:\$PATH\"  # Added by recall dev-env.sh"

# ---------------------------------------------------------------------------
# Patch a shell config file (idempotent — no duplicate additions)
# ---------------------------------------------------------------------------
patch_file() {
  file="$1"
  # Skip files that don't exist yet for non-primary shells
  [ -f "$file" ] || return 0

  if grep -qF "recall dev-env.sh" "$file" 2>/dev/null; then
    skip "Already patched: $file"
    return
  fi

  if [ "$DRY_RUN" = "1" ]; then
    info "[DRY RUN] Would add to $file:"
    printf '    %s\n' "$EXPORT_LINE"
    return
  fi

  printf '\n%s\n' "$EXPORT_LINE" >> "$file"
  ok "Patched: $file"
}

# ---------------------------------------------------------------------------
# Patch all relevant shell startup files
# ---------------------------------------------------------------------------
printf '\n\033[1mRecall — Developer Environment Setup\033[0m\n\n'

patch_file "${HOME}/.bashrc"
patch_file "${HOME}/.bash_profile"
patch_file "${HOME}/.profile"
patch_file "${HOME}/.zshrc"
patch_file "${HOME}/.zprofile"
patch_file "${HOME}/.config/fish/config.fish" 2>/dev/null || true

# ---------------------------------------------------------------------------
# Also write a .envrc for direnv users (project-local, not user-global)
# ---------------------------------------------------------------------------
ENVRC=".envrc"
if [ ! -f "$ENVRC" ] || ! grep -qF "recall dev-env" "$ENVRC" 2>/dev/null; then
  if [ "$DRY_RUN" = "0" ]; then
    printf '# Added by recall dev-env.sh\nexport PATH="%s:$PATH"\n' "$GO_DIR" > "$ENVRC"
    ok "Written: .envrc (for direnv users — run: direnv allow)"
  else
    info "[DRY RUN] Would write: .envrc"
  fi
else
  skip "Already exists: .envrc"
fi

# ---------------------------------------------------------------------------
# Shellcheck install hint (optional)
# ---------------------------------------------------------------------------
printf '\n'
if ! command -v shellcheck > /dev/null 2>&1; then
  info "Optional: install shellcheck for install.sh linting:"
  info "  sudo apt-get install shellcheck    # Debian/Ubuntu"
  info "  brew install shellcheck            # macOS"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
printf '\n'
if [ "$DRY_RUN" = "0" ]; then
  ok "Done. Reload your shell to pick up the changes:"
  printf '    source ~/.bashrc  # bash\n'
  printf '    source ~/.zshrc   # zsh\n'
  printf '\n'
  info "Or start a new session. Then verify with: go version"
else
  ok "Dry run complete — no files were modified."
fi
