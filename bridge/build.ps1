param(
  [switch]$SkipTests
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ProjectRoot
$VersionFile = Join-Path $RootDir "VERSION"
$OutputDir = Join-Path $ProjectRoot "build"
$OutputFile = Join-Path $OutputDir "bridge.exe"

if (-not (Test-Path -LiteralPath $VersionFile -PathType Leaf)) {
  throw "Version file not found: $VersionFile"
}
$Version = (Get-Content -LiteralPath $VersionFile -Raw).Trim()
if ($Version -notmatch '^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$') {
  throw "Invalid semantic version in ${VersionFile}: '$Version'"
}

function Invoke-Checked {
  param([Parameter(Mandatory = $true)][scriptblock]$Script)
  & $Script
  if ($LASTEXITCODE -ne 0) {
    throw "Command failed with exit code $LASTEXITCODE."
  }
}

if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
  throw "Required command not found: go"
}

if (-not (Test-Path $OutputDir)) {
  New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

Push-Location $ProjectRoot
try {
  if (-not $SkipTests) {
    Invoke-Checked { go test ./... }
  }
  Invoke-Checked { go build -ldflags "-s -w -X 'github.com/issueye/icoo_proxy/bridge/internal/service.Version=$Version'" -o $OutputFile ./cmd/bridge }
} finally {
  Pop-Location
}

Write-Host "Build completed: $OutputFile (version $Version)"
