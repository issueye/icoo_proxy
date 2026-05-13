param(
  [switch]$SkipTests,
  [string]$BridgePath = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ProjectRoot
$OutputDir = Join-Path $ProjectRoot "build\bin"
$DesktopOutput = Join-Path $OutputDir "icoo_desktop.exe"
$BundledBridgeOutput = Join-Path $OutputDir "bridge.exe"

function Write-Step {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Host ""
  Write-Host "==> $Message" -ForegroundColor Cyan
}

function Invoke-Checked {
  param([Parameter(Mandatory = $true)][scriptblock]$Script)
  & $Script
  if ($LASTEXITCODE -ne 0) { throw "Command failed with exit code $LASTEXITCODE." }
}

if (-not (Get-Command "wails" -ErrorAction SilentlyContinue)) { throw "Required command not found: wails" }

if (-not $SkipTests) {
  Write-Step "Running Go vet"
  Push-Location $ProjectRoot
  try { Invoke-Checked { go vet ./... } } finally { Pop-Location }
}

Write-Step "Building icoo_desktop (frontend + Wails)"
Push-Location $ProjectRoot
try { Invoke-Checked { wails build } } finally { Pop-Location }

if ([string]::IsNullOrWhiteSpace($BridgePath)) {
  $Candidates = @(
    (Join-Path $RootDir "icoo_llm_bridge\build\bridge.exe"),
    (Join-Path $RootDir "icoo_llm_bridge\build\icoo_llm_bridge.exe"),
    (Join-Path $RootDir "icoo_llm_bridge\bridge.exe")
  )
  foreach ($Candidate in $Candidates) {
    if (Test-Path $Candidate) {
      $BridgePath = $Candidate
      break
    }
  }
}

if (-not [string]::IsNullOrWhiteSpace($BridgePath) -and (Test-Path $BridgePath)) {
  Write-Step "Bundling icoo_llm_bridge"
  if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
  }
  Copy-Item -LiteralPath $BridgePath -Destination $BundledBridgeOutput -Force
} else {
  Write-Warning "bridge.exe not found. Desktop can still connect to a remote bridge URL, but local wake will fail until bridge.exe is placed next to icoo_desktop.exe."
}

Write-Step "Build completed"
Write-Host "Desktop: $DesktopOutput" -ForegroundColor Green
Write-Host "Bridge:  $BundledBridgeOutput" -ForegroundColor Green
Write-Host "  (Run with bridge.exe in same directory or configure remote URL)" -ForegroundColor DarkGray
