param(
  [switch]$SkipTests,
  [switch]$SkipFrontend,
  [string]$BridgePath = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ProjectRoot
$VersionFile = Join-Path $RootDir "VERSION"
$OutputDir = Join-Path $ProjectRoot "build\bin"
$DesktopOutput = Join-Path $OutputDir "icoo_desktop.exe"
$BundledBridgeOutput = Join-Path $OutputDir "bridge.exe"
$FrontendDir = Join-Path $ProjectRoot "frontend"
$WailsJSDir = Join-Path $FrontendDir "wailsjs"

if (-not (Test-Path -LiteralPath $VersionFile -PathType Leaf)) {
  throw "Version file not found: $VersionFile"
}
$Version = (Get-Content -LiteralPath $VersionFile -Raw).Trim()
if ($Version -notmatch '^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$') {
  throw "Invalid semantic version in ${VersionFile}: '$Version'"
}

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

if (-not (Get-Command "wails" -ErrorAction SilentlyContinue)) {
  throw "Required command not found: wails. Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
}
if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) { throw "Required command not found: go" }
if (-not (Get-Command "npm" -ErrorAction SilentlyContinue)) { throw "Required command not found: npm" }

function Reset-WailsBindings {
  $ResolvedFrontend = (Resolve-Path -LiteralPath $FrontendDir).Path
  if (Test-Path -LiteralPath $WailsJSDir) {
    $ResolvedWailsJS = (Resolve-Path -LiteralPath $WailsJSDir).Path
    if (-not $ResolvedWailsJS.StartsWith($ResolvedFrontend + [IO.Path]::DirectorySeparatorChar)) {
      throw "Refusing to remove unexpected wailsjs path: $ResolvedWailsJS"
    }
    [System.IO.Directory]::Delete(('\\?\' + $ResolvedWailsJS), $true)
  }
  New-Item -ItemType Directory -Path $WailsJSDir -Force | Out-Null
  if ($IsWindows -or $env:OS -match "Windows") {
    & icacls $WailsJSDir /inheritance:e | Out-Null
    & icacls $WailsJSDir /grant '*S-1-1-0:(OI)(CI)F' | Out-Null
  }
}

if (-not $SkipTests) {
  Write-Step "Running Go tests and vet"
  Push-Location $ProjectRoot
  try {
    Invoke-Checked { go test . }
    Invoke-Checked { go vet . }
  } finally { Pop-Location }
}

Write-Step "Generating Wails bindings"
Push-Location $ProjectRoot
try {
  Reset-WailsBindings
  Invoke-Checked { wails generate module }
  if (-not (Test-Path (Join-Path $WailsJSDir "runtime\runtime.js"))) {
    throw "Wails runtime binding was not generated."
  }
  if (-not (Test-Path (Join-Path $WailsJSDir "go\main\App.js"))) {
    throw "Wails Go binding was not generated."
  }
} finally { Pop-Location }

Write-Step "Building icoo_desktop with wails build"
Push-Location $ProjectRoot
try {
  if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
  }

  # wails build embeds frontend/dist (via //go:embed) and produces a production binary.
  # frontend:install / frontend:build come from wails.json unless -SkipFrontend.
  $ldflags = "-s -w -X main.Version=$Version"
  # On Windows, force .exe so tooling and packaging always find the same path.
  $outputName = "icoo_desktop"
  if ($env:OS -match "Windows" -or $IsWindows) {
    $outputName = "icoo_desktop.exe"
  }
  $wailsArgs = @(
    "build",
    "-clean",
    "-ldflags", $ldflags,
    "-o", $outputName
  )
  if ($SkipFrontend) {
    $wailsArgs += "-s"
  }

  Invoke-Checked { & wails @wailsArgs }

  # Normalize legacy output without extension (some Wails versions omit .exe).
  $legacyNoExt = Join-Path $OutputDir "icoo_desktop"
  if ((Test-Path -LiteralPath $legacyNoExt -PathType Leaf) -and -not (Test-Path -LiteralPath $DesktopOutput -PathType Leaf)) {
    Move-Item -LiteralPath $legacyNoExt -Destination $DesktopOutput -Force
  }
  if ((Test-Path -LiteralPath $legacyNoExt -PathType Leaf) -and (Test-Path -LiteralPath $DesktopOutput -PathType Leaf)) {
    # Prefer the .exe path; drop the extensionless duplicate.
    Remove-Item -LiteralPath $legacyNoExt -Force -ErrorAction SilentlyContinue
  }

  if (-not (Test-Path -LiteralPath $DesktopOutput -PathType Leaf)) {
    throw "wails build finished but icoo_desktop.exe was not found under build\bin"
  }
} finally { Pop-Location }

if ([string]::IsNullOrWhiteSpace($BridgePath)) {
  $Candidates = @(
    (Join-Path $RootDir "bridge\build\bridge.exe"),
    (Join-Path $RootDir "bridge\bridge.exe"),
    (Join-Path $RootDir "icoo_llm_bridge\build\bridge.exe")
  )
  foreach ($Candidate in $Candidates) {
    if (Test-Path $Candidate) {
      $BridgePath = $Candidate
      break
    }
  }
}

if (-not [string]::IsNullOrWhiteSpace($BridgePath) -and (Test-Path $BridgePath)) {
  Write-Step "Bundling bridge"
  if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
  }
  Copy-Item -LiteralPath $BridgePath -Destination $BundledBridgeOutput -Force
} else {
  Write-Warning "bridge.exe not found. Desktop can still connect to a remote bridge URL, but local wake will fail until bridge.exe is placed next to icoo_desktop.exe."
}

# Bundle process plugins when present next to bridge or in package tree.
$PluginCandidates = @(
  (Join-Path $RootDir "plugins\grokbuild\build\plugin-grokbuild.exe"),
  (Join-Path $RootDir "bridge\build\plugin-grokbuild.exe"),
  (Join-Path $RootDir "icoo_proxy\plugin-grokbuild.exe")
)
foreach ($plugin in $PluginCandidates) {
  if (Test-Path -LiteralPath $plugin -PathType Leaf) {
    Write-Step "Bundling plugin-grokbuild"
    Copy-Item -LiteralPath $plugin -Destination (Join-Path $OutputDir "plugin-grokbuild.exe") -Force
    break
  }
}

Write-Step "Build completed (wails build)"
Write-Host "Version: $Version" -ForegroundColor Green
Write-Host "Desktop: $DesktopOutput" -ForegroundColor Green
if (Test-Path $BundledBridgeOutput) {
  Write-Host "Bridge:  $BundledBridgeOutput" -ForegroundColor Green
}
Write-Host "  Production binary built via: wails build" -ForegroundColor DarkGray
