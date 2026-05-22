param(
  [switch]$SkipTests,
  [string]$CargoHome = ""
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

if (-not (Get-Command "cargo" -ErrorAction SilentlyContinue)) {
  throw "Required command not found: cargo"
}

if (-not (Test-Path $OutputDir)) {
  New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

Push-Location $ProjectRoot
$PreviousCargoHome = $env:CARGO_HOME
try {
  if (-not [string]::IsNullOrWhiteSpace($CargoHome)) {
    $env:CARGO_HOME = $CargoHome
  }
  if (-not $SkipTests) {
    Invoke-Checked { cargo test }
  }
  Invoke-Checked { cargo build --release }
  $BuiltBinary = Join-Path $ProjectRoot "target\release\icoo_llm_bridge.exe"
  if (-not (Test-Path $BuiltBinary)) {
    throw "Rust build output not found: $BuiltBinary"
  }
  Copy-Item -LiteralPath $BuiltBinary -Destination $OutputFile -Force
} finally {
  if ($null -eq $PreviousCargoHome) {
    Remove-Item Env:\CARGO_HOME -ErrorAction SilentlyContinue
  } else {
    $env:CARGO_HOME = $PreviousCargoHome
  }
  Pop-Location
}

Write-Host "Build completed: $OutputFile"
