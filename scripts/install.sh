#!/bin/sh
# Recall installer — downloads the latest (or specified) release binary.
# Usage:
#   curl -sSf https://raw.githubusercontent.com/Om-Rohilla/recall/main/scripts/install.sh | sh
#   curl -sSf ... | sh -s -- --version v0.2.0

set -e

REPO="Om-Rohilla/recall"
INSTALL_DIR="/usr/local/bin"
FALLBACK_DIR="${HOME}/.local/bin"
VERSION=""

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

info()  { printf '  \033[1;34m→\033[0m %s\n' "$1"; }
ok()    { printf '  \033[1;32m✓\033[0m %s\n' "$1"; }
err()   { printf '  \033[1;31m✗\033[0m %s\n' "$1" >&2; }
die()   { err "$1"; exit 1; }

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}

# ---------------------------------------------------------------------------
# Detect OS / Arch
# ---------------------------------------------------------------------------

detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux*)  OS="linux"  ;;
        Darwin*) OS="darwin" ;;
        *)       die "Unsupported operating system: $OS (only linux and darwin are supported)" ;;
    esac

    case "$ARCH" in
        x86_64|amd64)   ARCH="amd64" ;;
        aarch64|arm64)  ARCH="arm64" ;;
        *)              die "Unsupported architecture: $ARCH (only amd64 and arm64 are supported)" ;;
    esac
}

# ---------------------------------------------------------------------------
# Resolve version (latest or explicit)
# ---------------------------------------------------------------------------

resolve_version() {
    if [ -n "$VERSION" ]; then
        return
    fi

    info "Fetching latest release version..."
    need_cmd curl

    RELEASE_URL="https://api.github.com/repos/${REPO}/releases/latest"
    VERSION="$(curl -sSf "$RELEASE_URL" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"

    if [ -z "$VERSION" ]; then
        die "Could not determine latest release version"
    fi
}

# ---------------------------------------------------------------------------
# Download, verify, install
# ---------------------------------------------------------------------------

install_binary() {
    ARCHIVE_NAME="recall_${VERSION#v}_${OS}_${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
    CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

    TMPDIR="$(mktemp -d)"
    trap 'rm -rf "$TMPDIR"' EXIT

    info "Downloading ${ARCHIVE_NAME}..."
    curl -sSfL -o "${TMPDIR}/${ARCHIVE_NAME}" "$DOWNLOAD_URL" \
        || die "Download failed — check that version ${VERSION} exists for ${OS}/${ARCH}"

    info "Downloading checksums..."
    if curl -sSfL -o "${TMPDIR}/checksums.txt" "$CHECKSUM_URL" 2>/dev/null; then
        info "Verifying checksum..."
        EXPECTED="$(grep "${ARCHIVE_NAME}" "${TMPDIR}/checksums.txt" | awk '{print $1}')"
        if [ -n "$EXPECTED" ]; then
            # Allow explicit opt-out (INSECURE) for environments where tools are unavailable
            if [ "${RECALL_SKIP_VERIFY:-0}" = "1" ]; then
                info "⚠️  RECALL_SKIP_VERIFY=1: skipping integrity check (INSECURE)"
                ACTUAL="$EXPECTED"
            elif command -v sha256sum >/dev/null 2>&1; then
                ACTUAL="$(sha256sum "${TMPDIR}/${ARCHIVE_NAME}" | awk '{print $1}')"
            elif command -v shasum >/dev/null 2>&1; then
                ACTUAL="$(shasum -a 256 "${TMPDIR}/${ARCHIVE_NAME}" | awk '{print $1}')"
            else
                die "sha256sum and shasum not found. Cannot verify binary integrity.
Install coreutils (Linux) or use a macOS system with shasum available.
To skip verification explicitly (INSECURE), set RECALL_SKIP_VERIFY=1."
            fi

            if [ "$ACTUAL" != "$EXPECTED" ]; then
                die "Checksum mismatch!\n  expected: ${EXPECTED}\n  got:      ${ACTUAL}"
            fi
            ok "Checksum verified"

            # Optional cosign verification if cosign is available
            BUNDLE_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt.bundle"
            if command -v cosign >/dev/null 2>&1; then
                if curl -sSfL -o "${TMPDIR}/checksums.txt.bundle" "$BUNDLE_URL" 2>/dev/null; then
                    cosign verify-blob \
                        --bundle "${TMPDIR}/checksums.txt.bundle" \
                        "${TMPDIR}/checksums.txt" \
                        || die "cosign signature verification failed — aborting install"
                    ok "cosign signature verified"
                fi
            fi
        fi
    else
        info "Checksums not available — skipping verification"
    fi

    info "Extracting..."
    tar -xzf "${TMPDIR}/${ARCHIVE_NAME}" -C "$TMPDIR"

    if [ ! -f "${TMPDIR}/recall" ]; then
        die "Binary 'recall' not found in archive"
    fi

    TARGET_DIR="$INSTALL_DIR"
    if [ -w "$TARGET_DIR" ]; then
        mv "${TMPDIR}/recall" "${TARGET_DIR}/recall"
    elif command -v sudo >/dev/null 2>&1; then
        info "Installing to ${TARGET_DIR} (requires sudo)..."
        sudo mv "${TMPDIR}/recall" "${TARGET_DIR}/recall"
        sudo chmod +x "${TARGET_DIR}/recall"
    else
        TARGET_DIR="$FALLBACK_DIR"
        mkdir -p "$TARGET_DIR"
        mv "${TMPDIR}/recall" "${TARGET_DIR}/recall"
        info "Installed to ${TARGET_DIR} (add to PATH if needed)"
    fi

    chmod +x "${TARGET_DIR}/recall"
    ok "Installed recall ${VERSION} to ${TARGET_DIR}/recall"
}

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------

while [ $# -gt 0 ]; do
    case "$1" in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --version=*)
            VERSION="${1#*=}"
            shift
            ;;
        -h|--help)
            printf 'Usage: install.sh [--version VERSION]\n'
            printf '\nOptions:\n'
            printf '  --version VERSION   Install a specific version (e.g. v0.1.0)\n'
            printf '  -h, --help          Show this help\n'
            exit 0
            ;;
        *)
            die "Unknown option: $1"
            ;;
    esac
done

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

printf '\n\033[1mRecall Installer\033[0m\n\n'

detect_platform
info "Detected platform: ${OS}/${ARCH}"

resolve_version
info "Version: ${VERSION}"

install_binary

printf '\n\033[1mNext steps:\033[0m\n'
printf "  Run '\033[1mrecall init\033[0m' to get started.\n\n"
