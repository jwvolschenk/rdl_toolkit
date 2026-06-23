#!/usr/bin/env bash
# build-rdl-tool.sh — Build rdl-tool from local source and install locally
# Builds directly from the Go source tree (includes MCP server mode).
set -euo pipefail

# ── Config ───────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
INSTALL_DIR="${RDL_TOOL_DIR:-$HOME/.local/bin}"

# ── Colors ───────────────────────────────────────────────────────────────
R='\033[0;31m' G='\033[0;32m' Y='\033[0;33m' C='\033[0;36m'
W='\033[1;37m' D='\033[0;90m' N='\033[0m'

ok()   { printf "  ${G}✓${N} %s\n" "$*"; }
info() { printf "  ${D}│${N} %s\n" "$*"; }
err()  { printf "  ${R}✗${N} %s\n" "$*" >&2; }
die()  { err "$@"; exit 1; }

# ── Check prerequisites ──────────────────────────────────────────────────
check_prereqs() {
    command -v go >/dev/null 2>&1 || die "Go not found. Install: https://go.dev/dl/"
}

# ── Build from local source ──────────────────────────────────────────────
build() {
    local ldflags="-s -w"
    local output="${REPO_DIR}/bin/rdl-tool"

    info "Building rdl-tool from local source..."
    info "Source: ${REPO_DIR}"

    (cd "$REPO_DIR" && go build -ldflags "$ldflags" -o "$output" ./cmd/rdl-tool)

    if [ ! -f "$output" ]; then
        die "Build failed — binary not found at ${output}"
    fi

    ok "Build complete: $(du -h "$output" | cut -f1)"
}

# ── Install ──────────────────────────────────────────────────────────────
install_binary() {
    local binary="$1"
    local dest="${INSTALL_DIR}/rdl-tool"

    mkdir -p "$INSTALL_DIR"
    cp "$binary" "$dest"
    chmod +x "$dest"

    ok "Installed to ${dest}"
}

# ── Main ─────────────────────────────────────────────────────────────────
main() {
    echo ""
    printf "  ${W}rdl-tool build${N} ${D}from local source${N}\n"
    echo ""

    # 1. Check prerequisites
    check_prereqs

    # 2. Build from local source
    build

    # 3. Install
    echo ""
    install_binary "${REPO_DIR}/bin/rdl-tool"

    # 4. Check PATH
    case ":$PATH:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            echo ""
            printf "  ${Y}Note:${N} Add ${INSTALL_DIR} to your PATH:\n"
            printf "    ${C}export PATH=\"${INSTALL_DIR}:\$PATH\"${N}\n"
            printf "    ${D}(add to ~/.bashrc or ~/.zshrc)${N}\n"
            ;;
    esac

    # 5. Done
    echo ""
    printf "  ${G}Done!${N} rdl-tool built and installed.\n"
    printf "  ${D}Binary: ${INSTALL_DIR}/rdl-tool${N}\n"
    printf "  ${D}MCP mode: rdl-tool --mcp${N}\n"
    echo ""
}

main "$@"
