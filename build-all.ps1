param(
  [switch]$SkipTests,
  [switch]$SkipPlugins
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BridgeDir = Join-Path $RootDir "bridge"
$DesktopDir = Join-Path $RootDir "desktop"
$PackageDir = Join-Path $RootDir "icoo_proxy"
$GrokPluginDir = Join-Path $RootDir "plugins\grokbuild"
$MockPluginDir = Join-Path $RootDir "plugins\mock"

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

function Invoke-GoBuild {
  param(
    [Parameter(Mandatory = $true)][string]$ModuleDir,
    [Parameter(Mandatory = $true)][string]$Package,
    [Parameter(Mandatory = $true)][string]$Output
  )
  $outDir = Split-Path -Parent $Output
  if (-not (Test-Path $outDir)) {
    New-Item -ItemType Directory -Path $outDir -Force | Out-Null
  }
  Push-Location $ModuleDir
  try {
    & go build -o $Output $Package
    if ($LASTEXITCODE -ne 0) {
      throw "go build failed: $Package -> $Output"
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

Write-Step "[1/3] Building bridge (icoo/bridge)"
Invoke-ProjectBuild -ProjectDir $BridgeDir -BuildArgs $CommonArgs

$BridgeOutput = Join-Path $BridgeDir "build\bridge.exe"
if (-not (Test-Path $BridgeOutput)) {
  throw "bridge.exe not found after build: $BridgeOutput"
}

Write-Step "[2/3] Building desktop (icoo/desktop)"
$DesktopArgs = @($CommonArgs + @("-BridgePath", $BridgeOutput))
Invoke-ProjectBuild -ProjectDir $DesktopDir -BuildArgs $DesktopArgs

$DesktopOutput = Join-Path $DesktopDir "build\bin\icoo_desktop.exe"
if (-not (Test-Path $DesktopOutput)) {
  throw "icoo_desktop.exe not found after build: $DesktopOutput"
}

$PluginOutputs = @()
if (-not $SkipPlugins) {
  Write-Step "[3/3] Building process plugins"
  if (Test-Path $GrokPluginDir) {
    $GrokOut = Join-Path $GrokPluginDir "build\plugin-grokbuild.exe"
    Invoke-GoBuild -ModuleDir $GrokPluginDir -Package "./cmd/plugin-grokbuild" -Output $GrokOut
    $PluginOutputs += $GrokOut
    # Also place next to bridge for relative executable resolution.
    Copy-Item -LiteralPath $GrokOut -Destination (Join-Path $BridgeDir "build\plugin-grokbuild.exe") -Force
    Copy-Item -LiteralPath $GrokOut -Destination (Join-Path $DesktopDir "build\bin\plugin-grokbuild.exe") -Force
  }
  if (Test-Path $MockPluginDir) {
    $MockOut = Join-Path $MockPluginDir "build\mockplugin.exe"
    Invoke-GoBuild -ModuleDir $MockPluginDir -Package "./cmd/mockplugin" -Output $MockOut
    $PluginOutputs += $MockOut
  }
} else {
  Write-Step "[3/3] Skipping process plugins (-SkipPlugins)"
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
  foreach ($plugin in $PluginOutputs) {
    $name = Split-Path -Leaf $plugin
    Copy-Item -LiteralPath $plugin -Destination (Join-Path $PackageDir $name) -Force
  }
  $ExampleCfg = Join-Path $PackageDir "config.example.grokbuild.toml"
  $SrcExample = Join-Path $RootDir "icoo_proxy\config.example.grokbuild.toml"
  if (-not (Test-Path $SrcExample)) {
    $SrcExample = Join-Path $RootDir "bridge\configs\config.example.toml"
  }
  if (Test-Path (Join-Path $RootDir "icoo_proxy\config.example.grokbuild.toml")) {
    Copy-Item -LiteralPath (Join-Path $RootDir "icoo_proxy\config.example.grokbuild.toml") -Destination $ExampleCfg -Force
  }
} catch {
  throw "Failed to package executables. Close running icoo_desktop/bridge.exe from $PackageDir, then run build-all.ps1 again. Original error: $($_.Exception.Message)"
}

Write-Host ""
Write-Host "All builds completed" -ForegroundColor Green
Write-Host "  Bridge:  $BridgeOutput" -ForegroundColor Green
Write-Host "  Desktop: $DesktopOutput" -ForegroundColor Green
foreach ($plugin in $PluginOutputs) {
  Write-Host "  Plugin:  $plugin" -ForegroundColor Green
}
Write-Host "  Package: $PackageDir" -ForegroundColor Green
