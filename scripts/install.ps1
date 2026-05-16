# hookset installer for Windows
# Usage: irm https://raw.githubusercontent.com/bulga138/hookset/master/scripts/install.ps1 | iex
#
# Environment overrides:
#   $env:HOOKSET_VERSION      — install a specific version (default: latest)
#   $env:HOOKSET_INSTALL_DIR  — install directory (default: $env:LOCALAPPDATA\hookset\bin)
#   $env:HOOKSET_BINARY_PATH  — skip download, copy from this local path (air-gapped)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$Repo       = "bulga138/hookset"
$Binary     = "hookset.exe"
$InstallDir = if ($env:HOOKSET_INSTALL_DIR) { $env:HOOKSET_INSTALL_DIR }
              else { Join-Path $env:LOCALAPPDATA "hookset\bin" }

# ── helpers ──────────────────────────────────────────────────

function Write-Ok  ($msg) { Write-Host "  $([char]0x2713) $msg" -ForegroundColor Green }
function Write-Err ($msg) { Write-Host "  $([char]0x2717) $msg" -ForegroundColor Red }
function Write-Info($msg) { Write-Host "[hookset] $msg" -ForegroundColor Cyan }
function Die       ($msg) { Write-Err $msg; exit 1 }

# ── git version check ─────────────────────────────────────────

function Check-Git {
  try {
    $ver = (git --version) -replace 'git version ', ''
    $parts = $ver -split '\.'
    $maj = [int]$parts[0]; $min = [int]$parts[1]
    if ($maj -gt 2 -or ($maj -eq 2 -and $min -ge 54)) {
      Write-Ok "Git $ver — config-based hooks supported"
    } else {
      Write-Err "Git $ver is too old — hookset requires git >= 2.54"
      Write-Err "Upgrade: https://git-scm.com/downloads"
    }
  } catch {
    Write-Err "Git not found — hookset requires git >= 2.54"
  }
}

# ── detect arch ───────────────────────────────────────────────

function Get-Arch {
  $a = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
  switch ($a) {
    'X64'   { return 'amd64' }
    'Arm64' { return 'arm64' }
    default { Die "Unsupported architecture: $a" }
  }
}

# ── latest version ────────────────────────────────────────────

function Get-LatestVersion {
  $url = "https://api.github.com/repos/$Repo/releases/latest"
  $resp = Invoke-RestMethod -Uri $url -Headers @{ 'User-Agent' = 'hookset-installer' }
  return $resp.tag_name -replace '^v', ''
}

# ── main ──────────────────────────────────────────────────────

Write-Host ""
Write-Info "hookset installer for Windows"
Write-Host ""

Check-Git

$Arch = Get-Arch

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$Dest = Join-Path $InstallDir $Binary
$Tmp  = [System.IO.Path]::GetTempFileName()

if ($env:HOOKSET_BINARY_PATH) {
  Write-Info "Using local binary: $env:HOOKSET_BINARY_PATH"
  Copy-Item $env:HOOKSET_BINARY_PATH $Tmp -Force
} else {
  $Version = if ($env:HOOKSET_VERSION) { $env:HOOKSET_VERSION } else { Get-LatestVersion }
  $Asset   = "hookset_${Version}_windows_${Arch}.zip"
  $Url     = "https://github.com/$Repo/releases/download/v${Version}/$Asset"
  Write-Info "Downloading hookset $Version (windows/$Arch)..."

  $TmpZip = "$Tmp.zip"
  Invoke-WebRequest -Uri $Url -OutFile $TmpZip -UseBasicParsing
  Expand-Archive -Path $TmpZip -DestinationPath (Split-Path $Tmp) -Force
  Move-Item (Join-Path (Split-Path $Tmp) "hookset.exe") $Tmp -Force
  Remove-Item $TmpZip -Force
}

Copy-Item $Tmp $Dest -Force
Remove-Item $Tmp -ErrorAction SilentlyContinue
Write-Ok "Installed -> $Dest"

# ── PATH setup ────────────────────────────────────────────────

$UserPath = [System.Environment]::GetEnvironmentVariable('PATH', 'User')
if ($UserPath -notlike "*$InstallDir*") {
  [System.Environment]::SetEnvironmentVariable(
    'PATH', "$InstallDir;$UserPath", 'User'
  )
  Write-Ok "Added $InstallDir to user PATH"
  Write-Host "  Restart your terminal for PATH changes to take effect." -ForegroundColor DarkGray
}

Write-Host ""
Write-Host "Done! Run: hookset version" -ForegroundColor Green
Write-Host ""
