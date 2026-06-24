#!/usr/bin/env bash
# publish-rdl-tool.sh — Build rdl-tool and publish as a GitHub Release
#
# Interactive version prompt — suggests the next semver based on the latest tag,
# then creates/pushes the tag, builds cross-platform binaries, and uploads to
# GitHub Releases.
#
# Builds for:
#   - linux-amd64, linux-arm64
#   - darwin-amd64, darwin-arm64
#   - windows-amd64
#
# Usage:
#   bash scripts/publish-rdl-tool.sh              # interactive version prompt
#   bash scripts/publish-rdl-tool.sh v1.0.0       # explicit tag (skips prompt)
#   PLATFORMS="linux-amd64 windows-amd64" bash scripts/publish-rdl-tool.sh  # subset
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
RELEASE_DIR="${REPO_DIR}/.release"
FORK_REPO="jwvolschenk/rdl_toolkit"

# All supported cross-compilation targets
ALL_TARGETS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# ── Colors ───────────────────────────────────────────────────────────────
R='\033[0;31m' G='\033[0;32m' Y='\033[0;33m' C='\033[0;36m'
W='\033[1;37m' D='\033[0;90m' N='\033[0m'

ok()   { printf "  ${G}✓${N} %s\n" "$*"; }
info() { printf "  ${D}│${N} %s\n" "$*"; }
warn() { printf "  ${Y}!${N} %s\n" "$*"; }
err()  { printf "  ${R}✗${N} %s\n" "$*" >&2; }
die()  { err "$@"; exit 1; }

# ── Ensure gh can access the target repo ─────────────────────────────────
# If the active account can't see the repo, list available accounts and
# let the user switch before we start building.
ensure_repo_access() {
    # Already good?
    gh repo view "$FORK_REPO" >/dev/null 2>&1 && return 0

    local active_user
    active_user="$(gh api user --jq '.login' 2>/dev/null || echo 'unknown')"

    echo ""
    warn "Active gh account '${active_user}' cannot access ${FORK_REPO}"
    info "This is likely a private repo owned by a different account."
    echo ""

    # List all configured accounts
    local accounts=()
    while IFS= read -r line; do
        [ -n "$line" ] && accounts+=("$line")
    done < <(gh auth status 2>&1 | grep -oE 'Logged in to github\.com account [^ ]+' | awk '{print $NF}')

    if [ ${#accounts[@]} -eq 0 ]; then
        die "No gh accounts found. Run: gh auth login"
    fi

    printf "  ${W}Available gh accounts:${N}\n"
    printf "\n"
    for i in "${!accounts[@]}"; do
        local marker=""
        [[ "${accounts[$i]}" == "$active_user" ]] && marker=" ${D}(current)${N}"
        printf "    ${C}%d)${N} %s%b\n" "$((i+1))" "${accounts[$i]}" "$marker"
    done
    printf "\n"
    printf "  ${W}Switch to which account?${N} "
    read -r choice

    [ -n "$choice" ] || die "Selection required"

    local idx=$((choice - 1))
    if [ "$idx" -lt 0 ] || [ "$idx" -ge "${#accounts[@]}" ]; then
        die "Invalid selection"
    fi

    local chosen="${accounts[$idx]}"
    if [ "$chosen" == "$active_user" ]; then
        die "Already using '${chosen}' — it cannot access ${FORK_REPO}"
    fi

    info "Switching to '${chosen}'..."
    gh auth switch --user "$chosen" >/dev/null 2>&1 || die "Failed to switch to '${chosen}'"
    ok "Switched to '${chosen}'"

    # Verify access with the new account
    if ! gh repo view "$FORK_REPO" >/dev/null 2>&1; then
        die "Account '${chosen}' still cannot access ${FORK_REPO}. Check repo permissions."
    fi
    ok "Repo accessible: ${FORK_REPO}"
}

# ── Check prerequisites ──────────────────────────────────────────────────
check_prereqs() {
    command -v go >/dev/null 2>&1     || die "Go not found. Install: https://go.dev/dl/"
    command -v git >/dev/null 2>&1    || die "git not found"
    command -v gh >/dev/null 2>&1     || die "gh CLI not found. Install: https://cli.github.com/"
    gh auth status >/dev/null 2>&1    || die "gh not authenticated. Run: gh auth login"
    ensure_repo_access
}

# ── Map Go target to artifact name ──────────────────────────────────────
# Go target format:  os/arch     (e.g. linux/amd64, darwin/arm64)
# Artifact name:     rdl-tool-{os}-{arch}[.exe]
target_to_artifact() {
    local target="$1"
    local os_part="${target%%/*}"
    local arch="${target#*/}"
    local name="rdl-tool-${os_part}-${arch}"
    if [[ "$os_part" == "windows" ]]; then
        name="${name}.exe"
    fi
    echo "$name"
}

# ── Bump semver ──────────────────────────────────────────────────────────
# Given v1.2.3, suggests: patch=v1.2.4, minor=v1.3.0, major=v2.0.0
bump_version() {
    local current="$1"
    local part="$2"
    # Strip v prefix
    local v="${current#v}"
    local major minor patch
    IFS='.' read -r major minor patch <<< "$v"
    case "$part" in
        patch) printf "v%s.%s.%s" "$major" "$minor" "$((patch + 1))" ;;
        minor) printf "v%s.%s.0" "$major" "$((minor + 1))" ;;
        major) printf "v%s.0.0" "$((major + 1))" ;;
    esac
}

# ── Interactive version prompt ───────────────────────────────────────────
# NOTE: prompts go to >&2 so that command substitution $() only captures
# the final version string on stdout.
prompt_version() {
    local latest_tag="$1"

    if [ -z "$latest_tag" ]; then
        printf "  ${W}No existing tags found.${N}\n" >&2
        printf "  Enter version tag (e.g. ${C}v1.0.0${N}): " >&2
        read -r tag
        [ -n "$tag" ] || die "Version tag required"
        echo "$tag"
        return
    fi

    local patch minor major
    patch="$(bump_version "$latest_tag" patch)"
    minor="$(bump_version "$latest_tag" minor)"
    major="$(bump_version "$latest_tag" major)"

    printf "\n" >&2
    printf "  ${W}Current version:${N} ${latest_tag}\n" >&2
    printf "\n" >&2
    printf "  ${W}Select next version:${N}\n" >&2
    printf "    ${C}1)${N} ${patch}  ${D}(patch)${N}\n" >&2
    printf "    ${C}2)${N} ${minor}  ${D}(minor)${N}\n" >&2
    printf "    ${C}3)${N} ${major}  ${D}(major)${N}\n" >&2
    printf "    ${C}4)${N} custom\n" >&2
    printf "\n" >&2
    printf "  Choice [1]: " >&2
    read -r choice
    choice="${choice:-1}"

    case "$choice" in
        1) echo "$patch" ;;
        2) echo "$minor" ;;
        3) echo "$major" ;;
        4)
            printf "  Enter version tag (e.g. ${C}v2.0.0-rc1${N}): " >&2
            read -r tag
            [ -n "$tag" ] || die "Version tag required"
            echo "$tag"
            ;;
        *) echo "$patch" ;;
    esac
}

# ── Get latest git tag ──────────────────────────────────────────────────
get_latest_tag() {
    cd "$REPO_DIR" && git describe --tags --abbrev=0 2>/dev/null || true
}

# ── Build from local source ──────────────────────────────────────────────
build_target() {
    local target="$1"
    local artifact
    artifact="$(target_to_artifact "$target")"
    local output="${REPO_DIR}/bin/${artifact}"

    info "Building ${target} → ${artifact}..."
    (cd "$REPO_DIR" && GOOS="${target%%/*}" GOARCH="${target#*/}" \
        go build -ldflags "-s -w" -o "$output" ./cmd/rdl-tool)

    if [ ! -f "$output" ]; then
        die "Build failed for ${target}"
    fi

    ok "Built: ${artifact} ($(du -h "$output" | cut -f1))"
}

# ── Package artifacts ────────────────────────────────────────────────────
package() {
    local target="$1"
    local artifact
    artifact="$(target_to_artifact "$target")"
    local src="${REPO_DIR}/bin/${artifact}"

    mkdir -p "$RELEASE_DIR"
    cp "$src" "${RELEASE_DIR}/${artifact}"
    chmod +x "${RELEASE_DIR}/${artifact}"

    ok "Packaged: ${artifact}"
}

# ── Resolve which targets to build ──────────────────────────────────────
resolve_targets() {
    if [ -n "${PLATFORMS:-}" ]; then
        echo "$PLATFORMS"
        return
    fi
    echo "${ALL_TARGETS[*]}"
}

# ── Publish to GitHub Releases ───────────────────────────────────────────
publish() {
    local tag="$1"
    local release_dir="$RELEASE_DIR"

    info "Publishing release ${tag} to ${FORK_REPO}..."

    # Gather all artifacts + checksums
    local artifacts=()
    for f in "${release_dir}"/rdl-tool-*; do
        [ -f "$f" ] && artifacts+=("$f")
    done
    [ -f "${release_dir}/checksums.sha256" ] && artifacts+=("${release_dir}/checksums.sha256")

    if [ ${#artifacts[@]} -eq 0 ]; then
        die "No artifacts found in ${release_dir}"
    fi

    # Create release (gh handles tag creation if it doesn't exist)
    if ! gh release view "$tag" --repo "$FORK_REPO" >/dev/null 2>&1; then
        gh release create "$tag" \
            --repo "$FORK_REPO" \
            --title "rdl-tool ${tag}" \
            --generate-notes \
            "${artifacts[@]}"
        ok "Release created: ${tag}"
    else
        gh release upload "$tag" \
            "${artifacts[@]}" \
            --clobber --repo "$FORK_REPO"
        ok "Assets uploaded to existing release: ${tag}"
    fi

    echo ""
    printf "  ${G}Release URL:${N}\n"
    printf "  ${C}https://github.com/${FORK_REPO}/releases/tag/${tag}${N}\n"
}

# ── Main ─────────────────────────────────────────────────────────────────
main() {
    echo ""
    printf "  ${W}rdl-tool publish${N} ${D}build + release (multi-platform)${N}\n"
    echo ""

    check_prereqs

    # 1. Determine version tag
    local tag="${1:-}"
    if [ -z "$tag" ]; then
        local latest_tag
        latest_tag="$(get_latest_tag)"
        tag="$(prompt_version "$latest_tag")"
    fi
    info "Version tag: ${tag}"

    # 2. Confirm before proceeding
    echo ""
    printf "  ${W}This will:${N}\n"
    printf "    ${D}1.${N} Create and push git tag ${C}${tag}${N}\n"
    printf "    ${D}2.${N} Build binaries for ${#ALL_TARGETS[@]} platforms\n"
    printf "    ${D}3.${N} Upload to https://github.com/${FORK_REPO}/releases\n"
    echo ""
    printf "  Continue? [y/N] "
    read -r confirm
    [[ "$confirm" =~ ^[Yy]$ ]] || { printf "\n  ${D}Aborted.${N}\n\n"; exit 0; }

    # 3. Create and push tag
    echo ""
    info "Creating tag ${tag}..."
    cd "$REPO_DIR"
    git tag -a "$tag" -m "Release ${tag}" 2>/dev/null || {
        warn "Tag ${tag} already exists locally"
    }
    git push origin "$tag" 2>/dev/null || {
        warn "Tag ${tag} already exists on remote"
    }
    ok "Tag ${tag} pushed"

    # 4. Clean release dir
    rm -rf "$RELEASE_DIR"
    mkdir -p "$RELEASE_DIR"

    # 5. Resolve targets
    local targets_str
    targets_str="$(resolve_targets)"
    read -ra targets <<< "$targets_str"

    echo ""
    printf "  ${W}Building ${#targets[@]} targets:${N}\n"
    for t in "${targets[@]}"; do
        printf "    ${D}•${N} %s → %s\n" "$t" "$(target_to_artifact "$t")"
    done

    # 6. Build and package each target
    local built=0 failed=0
    for target in "${targets[@]}"; do
        echo ""
        printf "  ${W}[$((built+failed))+${failed}/${#targets[@]}]${N} ${target}\n"
        if build_target "$target" && package "$target"; then
            built=$((built + 1))
        else
            failed=$((failed + 1))
            warn "FAILED: ${target} (skipping)"
        fi
    done

    # 7. Generate checksums
    (cd "$RELEASE_DIR" && sha256sum rdl-tool-* 2>/dev/null | sort > checksums.sha256)

    echo ""
    printf "  ${W}Build summary:${N} ${G}${built} succeeded${N}, ${R}${failed} failed${N}\n"

    if [ "$built" -eq 0 ]; then
        die "No targets built successfully"
    fi

    # 8. List artifacts
    echo ""
    printf "  ${W}Artifacts:${N}\n"
    for f in "${RELEASE_DIR}"/rdl-tool-*; do
        [ -f "$f" ] && printf "    ${D}•${N} %s  ${D}(%s)${N}\n" "$(basename "$f")" "$(du -h "$f" | cut -f1)"
    done

    # 9. Publish to GitHub
    echo ""
    publish "$tag"

    # 10. Done
    echo ""
    printf "  ${G}Done!${N} Release ${tag} published.\n"
    printf "  ${D}Install: curl -fsSL https://raw.githubusercontent.com/${FORK_REPO}/master/scripts/setup-rdl-tool.sh | bash${N}\n"
    echo ""
}

main "$@"
