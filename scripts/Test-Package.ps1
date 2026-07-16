param(
  [Parameter(Mandatory = $true)]
  [string]$PackageDir
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$resolvedPackage = (Resolve-Path -LiteralPath $PackageDir).Path
$expected = @("bridge.exe", "icoo_desktop.exe")

foreach ($name in $expected) {
  $path = Join-Path $resolvedPackage $name
  if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
    throw "Missing package artifact: $path"
  }
  $item = Get-Item -LiteralPath $path
  if ($item.Length -lt 1MB) {
    throw "Package artifact is unexpectedly small: $path ($($item.Length) bytes)"
  }
  $stream = [System.IO.File]::OpenRead($path)
  try {
    $first = $stream.ReadByte()
    $second = $stream.ReadByte()
  } finally {
    $stream.Dispose()
  }
  if ($first -ne [byte][char]'M' -or $second -ne [byte][char]'Z') {
    throw "Package artifact is not a Windows PE executable: $path"
  }
}

$unexpectedExecutables = Get-ChildItem -LiteralPath $resolvedPackage -Filter *.exe -File |
  Where-Object { $_.Name -notin $expected }
if ($unexpectedExecutables) {
  throw "Unexpected executable artifacts: $($unexpectedExecutables.Name -join ', ')"
}

Write-Host "Package smoke test passed: $resolvedPackage" -ForegroundColor Green

