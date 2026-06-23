#!/usr/bin/env bash
# setup-rdl-tool.sh — Download rdl-tool from GitHub Releases and optionally register with AI agents
#
# Auto-detects platform and downloads the matching binary:
#   linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/jwvolschenk/rdl_toolkit/main/scripts/setup-rdl-tool.sh | bash
#   bash scripts/setup-rdl-tool.sh
#   bash scripts/setup-rdl-tool.sh --agent Hermes
#   RDL_TOOL_VERSION=v1.0.0 bash scripts/setup-rdl-tool.sh
set -euo pipefail

# ── Config ───────────────────────────────────────────────────────────────
INSTALL_DIR="${RDL_TOOL_DIR:-$HOME/.local/bin}"
FORK_REPO="jwvolschenk/rdl_toolkit"

# ── Colors ───────────────────────────────────────────────────────────────
R='\033[0;31m' G='\033[0;32m' Y='\033[0;33m' C='\033[0;36m'
W='\033[1;37m' D='\033[0;90m' N='\033[0m'

ok()   { printf "  ${G}✓${N} %s\n" "$*"; }
skip() { printf "  ${D}–${N}  %s\n" "$*"; }
info() { printf "  ${D}│${N} %s\n" "$*"; }
warn() { printf "  ${Y}!${N} %s\n" "$*"; }
err()  { printf "  ${R}✗${N} %s\n" "$*" >&2; }
die()  { err "$@"; exit 1; }

# ── Detect platform ──────────────────────────────────────────────────────
detect_platform() {
    local os arch
    os="$(uname -s)"
    arch="$(uname -m)"

    case "$os" in
        Darwin) os="darwin" ;;
        Linux)  os="linux" ;;
        MINGW*_NT*|MSYS_NT*|CYGWIN_NT*)
            echo ""
            printf "  ${Y}Windows detected.${N} Use the PowerShell installer instead:\n\n"
            printf "    ${C}powershell -ExecutionPolicy Bypass -File scripts/setup-rdl-tool.ps1${N}\n\n"
            printf "  ${D}Or from a PowerShell terminal:${N}\n"
            printf "    ${C}irm https://raw.githubusercontent.com/${FORK_REPO}/main/scripts/setup-rdl-tool.ps1 | iex${N}\n\n"
            exit 0
            ;;
        *) die "Unsupported OS: $os" ;;
    esac
    case "$arch" in
        arm64|aarch64)        arch="arm64" ;;
        x86_64|amd64|x86-64) arch="amd64" ;;
        *) die "Unsupported architecture: $arch" ;;
    esac
    echo "${os}-${arch}"
}

# ── Map platform to artifact filename ────────────────────────────────────
platform_to_artifact() {
    local platform="$1"
    echo "rdl-tool-${platform}"
}

# ── Fetch latest version tag from releases ───────────────────────────────
fetch_latest_version() {
    local tag=""
    tag="$(curl -fsSL -A 'rdl-tool-setup' \
        "https://api.github.com/repos/${FORK_REPO}/releases/latest" 2>/dev/null \
        | grep -oE '"tag_name"\s*:\s*"[^"]*"' \
        | cut -d'"' -f4)" || true
    printf '%s' "$tag"
}

# Strip v/V prefix for display
strip_version_prefix() {
    printf '%s' "$1" | sed 's/^[vV]//'
}

# ── Download binary ──────────────────────────────────────────────────────
download_binary() {
    local platform="$1" tag="$2" dest="$3"
    local artifact
    artifact="$(platform_to_artifact "$platform")"
    local tmp="/tmp/rdl-tool-setup.$$"

    info "Downloading ${artifact} ${tag}..."

    local url="https://github.com/${FORK_REPO}/releases/download/${tag}/${artifact}"
    if ! curl -fsSL -A 'rdl-tool-setup' "$url" -o "$tmp" 2>/dev/null; then
        rm -f "$tmp"
        die "Download failed. Check: https://github.com/${FORK_REPO}/releases"
    fi

    # Verify checksum
    local checksum_text expected_hash
    local checksum_url="https://github.com/${FORK_REPO}/releases/download/${tag}/checksums.sha256"
    checksum_text="$(curl -fsSL -A 'rdl-tool-setup' "$checksum_url" 2>/dev/null || true)"
    expected_hash="$(printf '%s\n' "$checksum_text" | awk "/${artifact}\$/ { print \$1 }")"
    if [ -n "$expected_hash" ]; then
        local actual_hash
        if command -v sha256sum >/dev/null 2>&1; then
            actual_hash="$(sha256sum "$tmp" | awk '{print $1}')"
        elif command -v shasum >/dev/null 2>&1; then
            actual_hash="$(shasum -a 256 "$tmp" | awk '{print $1}')"
        fi
        if [ -n "${actual_hash:-}" ] && [ "$actual_hash" != "$expected_hash" ]; then
            rm -f "$tmp"
            die "Checksum mismatch — binary may be corrupted"
        fi
        ok "Checksum verified"
    fi

    xattr -c "$tmp" 2>/dev/null || true
    mkdir -p "$(dirname "$dest")"
    mv -f "$tmp" "$dest"
    chmod +x "$dest"
    ok "Installed to ${dest}"
}

# ── Agent prompt ─────────────────────────────────────────────────────────
# Writes to >&2 so it works inside $() command substitution.
prompt_agent() {
    local agents=("Copilot" "Hermes" "Claude" "Codex" "Gemini" "Cursor" "OpenCode")

    printf "\n" >&2
    printf "  ${W}Which AI agent are you using?${N}\n" >&2
    printf "\n" >&2
    for i in "${!agents[@]}"; do
        local marker=""
        [ "$i" -eq 0 ] && marker=" ${D}(default)${N}"
        printf "    ${C}%d)${N} %s%b\n" "$((i+1))" "${agents[$i]}" "$marker" >&2
    done
    printf "\n" >&2
    printf "  Choice [1]: " >&2
    read -r choice
    choice="${choice:-1}"

    local idx=$((choice - 1))
    if [ "$idx" -ge 0 ] && [ "$idx" -lt "${#agents[@]}" ]; then
        echo "${agents[$idx]}"
    else
        echo "${agents[0]}"
    fi
}

# ── Agent Instructions ───────────────────────────────────────────────────
show_agent_instructions() {
    local agent_name="$1"
    local exe_path="$2"

    echo ""
    printf "  ${W}MCP Configuration for ${agent_name}${N}\n"
    echo ""

    case "$(echo "$agent_name" | tr '[:upper:]' '[:lower:]')" in
        hermes)
            printf "  Add the following to ~/.hermes/config.yaml :\n\n"
            printf "  ${C}mcp_servers:${N}\n"
            printf "    ${C}rdl-toolkit:${N}\n"
            printf "      ${C}command: ${exe_path}${N}\n"
            printf "      ${C}args:${N}\n"
            printf "        ${C}--mcp${N}\n"
            printf "      ${C}enabled: true${N}\n"
            ;;
        copilot)
            printf "  To register globally, add to ~/.copilot/mcp-config.json :\n\n"
            printf "  ${C}{\n    \"mcpServers\": {\n      \"rdl-toolkit\": {\n        \"command\": \"${exe_path}\",\n        \"args\": [\"--mcp\"]\n      }\n    }\n  }${N}\n\n"
            printf "  To register for a specific VS Code workspace, add to .vscode/mcp.json :\n\n"
            printf "  ${C}{\n    \"servers\": {\n      \"rdl-toolkit\": {\n        \"type\": \"stdio\",\n        \"command\": \"${exe_path}\",\n        \"args\": [\"--mcp\"]\n      }\n    }\n  }${N}\n"
            ;;
        claude)
            printf "  Add the following to ~/.claude.json :\n\n"
            printf "  ${C}{\n    \"mcpServers\": {\n      \"rdl-toolkit\": {\n        \"command\": \"${exe_path}\",\n        \"args\": [\"--mcp\"]\n      }\n    }\n  }${N}\n"
            ;;
        codex)
            printf "  Add the following to ~/.codex/config.toml :\n\n"
            printf "  ${C}[mcp_servers.rdl-toolkit]${N}\n"
            printf "  ${C}command = \"${exe_path}\"${N}\n"
            printf "  ${C}args = [\"--mcp\"]${N}\n"
            printf "  ${C}startup_timeout_sec = 30${N}\n"
            ;;
        gemini)
            printf "  Add the following to ~/.gemini/settings.json :\n\n"
            printf "  ${C}{\n    \"mcpServers\": {\n      \"rdl-toolkit\": {\n        \"command\": \"${exe_path}\",\n        \"args\": [\"--mcp\"]\n      }\n    }\n  }${N}\n"
            ;;
        cursor)
            printf "  Add the following to ~/.cursor/mcp.json :\n\n"
            printf "  ${C}{\n    \"mcpServers\": {\n      \"rdl-toolkit\": {\n        \"command\": \"${exe_path}\",\n        \"args\": [\"--mcp\"]\n      }\n    }\n  }${N}\n"
            ;;
        opencode)
            printf "  Add the following to ~/.config/opencode/opencode.jsonc (mcp section):\n\n"
            printf "  ${C}\"rdl-toolkit\": {\n  \"command\": \"${exe_path}\",\n  \"args\": [\"--mcp\"],\n  \"enabled\": true\n}${N}\n"
            ;;
        *)
            err "Unknown agent: $agent_name"
            info "Available agents: Hermes, Copilot, Claude, Codex, Gemini, Cursor, OpenCode"
            ;;
    esac
    echo ""
}

# ── Main ─────────────────────────────────────────────────────────────────
main() {
    local agent=""
    while [[ $# -gt 0 ]]; do
        case $1 in
            -a|--agent)
                agent="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    echo ""
    printf "  ${W}rdl-tool setup${N}\n"
    echo ""

    # 1. Detect platform
    local platform
    platform="$(detect_platform)"
    info "Platform: ${platform}"

    # 2. Get version tag
    local tag="${RDL_TOOL_VERSION:-}"
    if [ -z "$tag" ]; then
        tag="$(fetch_latest_version)"
    fi
    if [ -z "$tag" ]; then
        die "Could not fetch latest version from https://github.com/${FORK_REPO}/releases"
    fi
    info "Version:  $(strip_version_prefix "$tag")"

    # 3. Download and install
    local dest="${INSTALL_DIR}/rdl-tool"
    download_binary "$platform" "$tag" "$dest"

    # 4. Verify installation
    if "$dest" --help >/dev/null 2>&1; then
        ok "Binary verified"
    fi

    # 5. Check PATH
    case ":$PATH:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            echo ""
            printf "  ${Y}Note:${N} Add ${INSTALL_DIR} to your PATH:\n"
            printf "    ${C}export PATH=\"${INSTALL_DIR}:\$PATH\"${N}\n"
            printf "    ${D}(add to ~/.bashrc or ~/.zshrc)${N}\n"
            ;;
    esac

    # 6. Agent Instructions
    if [ -z "$agent" ]; then
        agent="$(prompt_agent)"
    fi
    show_agent_instructions "$agent" "$dest"

    # 7. Done
    echo ""
    printf "  ${G}Done!${N} rdl-tool installed.\n"
    printf "  ${D}Binary: ${dest}${N}\n"
    printf "  ${D}CLI:    rdl-tool --help${N}\n"
    printf "  ${D}MCP:    rdl-tool --mcp${N}\n"
    echo ""
    printf "  ${D}Update:    bash scripts/setup-rdl-tool.sh${N}\n"
    printf "  ${D}Uninstall: rm ${dest}${N}\n"
    echo ""
}

main "$@"
