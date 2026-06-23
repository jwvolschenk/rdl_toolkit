# setup-rdl-tool.ps1 — Download rdl-tool from GitHub Releases and optionally register with AI agents
#
# Auto-detects platform and downloads the matching binary.
# Defaults to Copilot agent registration.
#
# Usage:
#   irm https://raw.githubusercontent.com/jwvolschenk/rdl_toolkit/main/scripts/setup-rdl-tool.ps1 | iex
#   powershell -ExecutionPolicy Bypass -File scripts/setup-rdl-tool.ps1
#   powershell -ExecutionPolicy Bypass -File scripts/setup-rdl-tool.ps1 -Agent Hermes
#   powershell -ExecutionPolicy Bypass -File scripts/setup-rdl-tool.ps1 -Agent Copilot -Version v1.0.0

param(
    [string]$Agent = "",
    [string]$Version = "",
    [string]$InstallDir = ""
)

$ErrorActionPreference = "Stop"

# ── Config ───────────────────────────────────────────────────────────────
$ForkRepo = "jwvolschenk/rdl_toolkit"
if (-not $InstallDir) {
    $InstallDir = Join-Path $env:USERPROFILE ".local\bin"
}

# ── Helpers ──────────────────────────────────────────────────────────────
function Write-Ok($msg)   { Write-Host "  $([char]0x2713) $msg" -ForegroundColor Green }
function Write-Info($msg)  { Write-Host "  | $msg" -ForegroundColor DarkGray }
function Write-Warn($msg)  { Write-Host "  ! $msg" -ForegroundColor Yellow }
function Write-Err($msg)   { Write-Host "  X $msg" -ForegroundColor Red }

# ── Fetch latest version ─────────────────────────────────────────────────
function Get-LatestVersion {
    try {
        $resp = Invoke-RestMethod -Uri "https://api.github.com/repos/$ForkRepo/releases/latest" -UseBasicParsing
        return $resp.tag_name
    } catch {
        return $null
    }
}

# ── Download with checksum verification ──────────────────────────────────
function Install-Binary {
    param([string]$Tag, [string]$Artifact, [string]$Dest)

    $url = "https://github.com/$ForkRepo/releases/download/$Tag/$Artifact"
    $tmp = Join-Path $env:TEMP "rdl-tool-setup-$PID.exe"

    Write-Info "Downloading $Artifact $Tag..."
    try {
        Invoke-WebRequest -Uri $url -OutFile $tmp -UseBasicParsing
    } catch {
        Write-Err "Download failed. Check: https://github.com/$ForkRepo/releases"
        exit 1
    }

    # Verify checksum
    try {
        $checksumUrl = "https://github.com/$ForkRepo/releases/download/$Tag/checksums.sha256"
        $checksumText = (Invoke-WebRequest -Uri $checksumUrl -UseBasicParsing).Content
        $expectedHash = ($checksumText -split "`n" | Where-Object { $_ -match $Artifact } | ForEach-Object { ($_ -split '\s+')[0] })
        if ($expectedHash) {
            $actualHash = (Get-FileHash -Path $tmp -Algorithm SHA256).Hash.ToLower()
            if ($actualHash -ne $expectedHash.ToLower()) {
                Remove-Item $tmp -Force
                Write-Err "Checksum mismatch - binary may be corrupted"
                exit 1
            }
            Write-Ok "Checksum verified"
        }
    } catch {
        Write-Warn "Could not verify checksum (non-fatal)"
    }

    # Install
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    Copy-Item $tmp $Dest -Force
    Remove-Item $tmp -Force
    Write-Ok "Installed to $Dest"
}

# ── Agent instructions ───────────────────────────────────────────────────
function Show-AgentInstructions {
    param([string]$AgentName, [string]$ExePath)

    # Normalize path separators for config files
    $configPath = $ExePath -replace '\\', '/'

    Write-Host ""
    Write-Host "  MCP Configuration for $AgentName" -ForegroundColor White
    Write-Host ""

    switch ($AgentName.ToLower()) {
        "hermes" {
            Write-Host "  Add to ~/.hermes/config.yaml:" -ForegroundColor White
            Write-Host ""
            Write-Host "  mcp_servers:" -ForegroundColor Cyan
            Write-Host "    rdl-toolkit:" -ForegroundColor Cyan
            Write-Host "      command: $configPath" -ForegroundColor Cyan
            Write-Host "      args:" -ForegroundColor Cyan
            Write-Host "        --mcp" -ForegroundColor Cyan
            Write-Host "      enabled: true" -ForegroundColor Cyan
        }
        "copilot" {
            Write-Host "  For VS Code (user-wide), add to ~/.copilot/mcp-config.json:" -ForegroundColor White
            Write-Host ""
            $json = @"
  {
    "mcpServers": {
      "rdl-toolkit": {
        "command": "$configPath",
        "args": ["--mcp"]
      }
    }
  }
"@
            Write-Host $json -ForegroundColor Cyan
            Write-Host ""
            Write-Host "  For a specific VS Code workspace, add to .vscode/mcp.json:" -ForegroundColor White
            Write-Host ""
            $json2 = @"
  {
    "servers": {
      "rdl-toolkit": {
        "type": "stdio",
        "command": "$configPath",
        "args": ["--mcp"]
      }
    }
  }
"@
            Write-Host $json2 -ForegroundColor Cyan
        }
        "claude" {
            Write-Host "  Add to ~/.claude.json:" -ForegroundColor White
            Write-Host ""
            $json = @"
  {
    "mcpServers": {
      "rdl-toolkit": {
        "command": "$configPath",
        "args": ["--mcp"]
      }
    }
  }
"@
            Write-Host $json -ForegroundColor Cyan
        }
        "codex" {
            Write-Host "  Add to ~/.codex/config.toml:" -ForegroundColor White
            Write-Host ""
            Write-Host "  [mcp_servers.rdl-toolkit]" -ForegroundColor Cyan
            Write-Host "  command = `"$configPath`"" -ForegroundColor Cyan
            Write-Host "  args = [`"--mcp`"]" -ForegroundColor Cyan
            Write-Host "  startup_timeout_sec = 30" -ForegroundColor Cyan
        }
        "gemini" {
            Write-Host "  Add to ~/.gemini/settings.json:" -ForegroundColor White
            Write-Host ""
            $json = @"
  {
    "mcpServers": {
      "rdl-toolkit": {
        "command": "$configPath",
        "args": ["--mcp"]
      }
    }
  }
"@
            Write-Host $json -ForegroundColor Cyan
        }
        "cursor" {
            Write-Host "  Add to ~/.cursor/mcp.json:" -ForegroundColor White
            Write-Host ""
            $json = @"
  {
    "mcpServers": {
      "rdl-toolkit": {
        "command": "$configPath",
        "args": ["--mcp"]
      }
    }
  }
"@
            Write-Host $json -ForegroundColor Cyan
        }
        "opencode" {
            Write-Host "  Add to ~/.config/opencode/opencode.jsonc (mcp section):" -ForegroundColor White
            Write-Host ""
            Write-Host "  `"rdl-toolkit`": {" -ForegroundColor Cyan
            Write-Host "    `"command`": `"$configPath`"," -ForegroundColor Cyan
            Write-Host "    `"args`": [`"--mcp`"]," -ForegroundColor Cyan
            Write-Host "    `"enabled`": true" -ForegroundColor Cyan
            Write-Host "  }" -ForegroundColor Cyan
        }
        default {
            Write-Err "Unknown agent: $AgentName"
            Write-Info "Available agents: Copilot, Hermes, Claude, Codex, Gemini, Cursor, OpenCode"
        }
    }
    Write-Host ""
}

# ── Agent prompt ─────────────────────────────────────────────────────────
function Prompt-Agent {
    $agents = @("Copilot", "Hermes", "Claude", "Codex", "Gemini", "Cursor", "OpenCode")

    Write-Host ""
    Write-Host "  Which AI agent are you using?" -ForegroundColor White
    Write-Host ""
    for ($i = 0; $i -lt $agents.Count; $i++) {
        $marker = if ($i -eq 0) { " (default)" } else { "" }
        Write-Host "    $($i+1)) $($agents[$i])$marker" -ForegroundColor Cyan
    }
    Write-Host ""
    $choice = Read-Host "  Choice [1]"

    if (-not $choice) { $choice = "1" }

    $idx = 0
    if ([int]::TryParse($choice, [ref]$idx) -and $idx -ge 1 -and $idx -le $agents.Count) {
        return $agents[$idx - 1]
    }
    return $agents[0]
}

# ── Main ─────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  rdl-tool setup" -ForegroundColor White
Write-Host ""

# 1. Platform
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "amd64" }
$platform = "windows-$arch"
Write-Info "Platform: $platform"

# 2. Version
if (-not $Version) {
    $Version = Get-LatestVersion
}
if (-not $Version) {
    Write-Err "Could not fetch latest version from https://github.com/$ForkRepo/releases"
    exit 1
}
$displayVersion = $Version -replace '^[vV]', ''
Write-Info "Version:  $displayVersion"

# 3. Download
$artifact = "rdl-tool-windows-amd64.exe"
$dest = Join-Path $InstallDir "rdl-tool.exe"
Install-Binary -Tag $Version -Artifact $artifact -Dest $dest

# 4. Verify
try {
    & $dest --help 2>&1 | Out-Null
    Write-Ok "Binary verified"
} catch {
    Write-Warn "Could not verify binary (may need Visual C++ runtime)"
}

# 5. PATH
$pathParts = $env:PATH -split ";"
if ($pathParts -notcontains $InstallDir) {
    Write-Host ""
    Write-Warn "Add $InstallDir to your PATH:"
    Write-Host "    `$env:PATH += `";$InstallDir`"" -ForegroundColor Cyan
    Write-Host "    (add to your PowerShell profile for persistence)" -ForegroundColor DarkGray
}

# 6. Agent instructions
if ($Agent) {
    Show-AgentInstructions -AgentName $Agent -ExePath $dest
} else {
    $selected = Prompt-Agent
    Show-AgentInstructions -AgentName $selected -ExePath $dest
}

# 7. Done
Write-Host ""
Write-Host "  Done! rdl-tool installed." -ForegroundColor Green
Write-Host "  Binary: $dest" -ForegroundColor DarkGray
Write-Host "  CLI:    rdl-tool.exe --help" -ForegroundColor DarkGray
Write-Host "  MCP:    rdl-tool.exe --mcp" -ForegroundColor DarkGray
Write-Host ""
Write-Host "  Update:    powershell -ExecutionPolicy Bypass -File scripts/setup-rdl-tool.ps1" -ForegroundColor DarkGray
Write-Host "  Uninstall: Remove-Item $dest" -ForegroundColor DarkGray
Write-Host ""
