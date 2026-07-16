param(
  [switch]$SkipTests
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BridgeDir = Join-Path $RootDir "bridge"
$DesktopDir = Join-Path $RootDir "desktop"
$PackageDir = Join-Path $RootDir "icoo_proxy"

function Write-Step {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Host ""
  Write-Host "============================================================" -ForegroundColor Cyan
  Write-Host "  $Message" -ForegroundColor Cyan
  Write-Host "============================================================" -ForegroundColor Cyan
}

function Invoke-ProjectBuild {
  param(
    [Parameter(Mandatory = $true)][string]$ProjectDir,
    [string[]]$BuildArgs = @()
  )
  Push-Location $ProjectDir
  try {
    & pwsh -NoProfile -ExecutionPolicy Bypass -File ".\build.ps1" @BuildArgs
    if ($LASTEXITCODE -ne 0) {
      throw "Build failed in $ProjectDir"
    }
  } finally {
    Pop-Location
  }
}

if (-not (Test-Path $BridgeDir)) {
  throw "bridge directory not found: $BridgeDir"
}
if (-not (Test-Path $DesktopDir)) {
  throw "desktop directory not found: $DesktopDir"
}

$CommonArgs = @()
if ($SkipTests) {
  $CommonArgs += "-SkipTests"
}

Write-Step "[1/2] Building bridge (icoo/bridge)"
Invoke-ProjectBuild -ProjectDir $BridgeDir -BuildArgs $CommonArgs

$BridgeOutput = Join-Path $BridgeDir "build\bridge.exe"
if (-not (Test-Path $BridgeOutput)) {
  throw "bridge.exe not found after build: $BridgeOutput"
}

Write-Step "[2/2] Building desktop (icoo/desktop)"
$DesktopArgs = @($CommonArgs + @("-BridgePath", $BridgeOutput))
Invoke-ProjectBuild -ProjectDir $DesktopDir -BuildArgs $DesktopArgs

$DesktopOutput = Join-Path $DesktopDir "build\bin\icoo_desktop.exe"
if (-not (Test-Path $DesktopOutput)) {
  throw "icoo_desktop.exe not found after build: $DesktopOutput"
}

Write-Step "Packaging executables"
if (-not (Test-Path $PackageDir)) {
  New-Item -ItemType Directory -Path $PackageDir | Out-Null
}

$StaleFiles = @(
  (Join-Path $PackageDir "icoo_server.exe"),
  (Join-Path $PackageDir "icoo_llm_bridge.exe")
)
foreach ($StaleFile in $StaleFiles) {
  if (Test-Path $StaleFile) {
    Remove-Item -LiteralPath $StaleFile -Force
  }
}

$PackageBridge = Join-Path $PackageDir "bridge.exe"
$PackageDesktop = Join-Path $PackageDir "icoo_desktop.exe"
try {
  Copy-Item -LiteralPath $BridgeOutput -Destination $PackageBridge -Force
  Copy-Item -LiteralPath $DesktopOutput -Destination $PackageDesktop -Force
} catch {
  throw "Failed to package executables. Close running icoo_desktop/bridge.exe from $PackageDir, then run build-all.ps1 again. Original error: $($_.Exception.Message)"
}

Write-Host ""
Write-Host "All builds completed" -ForegroundColor Green
Write-Host "  Bridge:  $BridgeOutput" -ForegroundColor Green
Write-Host "  Desktop: $DesktopOutput" -ForegroundColor Green
Write-Host "  Package: $PackageDir" -ForegroundColor Green
