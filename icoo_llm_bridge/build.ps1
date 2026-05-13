param(
  [switch]$SkipTests
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$OutputDir = Join-Path $ProjectRoot "build"
$OutputFile = Join-Path $OutputDir "bridge.exe"

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
  Invoke-Checked { go build -o $OutputFile ./cmd/bridge }
} finally {
  Pop-Location
}

Write-Host "Build completed: $OutputFile"
