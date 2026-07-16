param(
  [Parameter(Mandatory = $true)]
  [string]$PackageDir
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$resolvedPackage = (Resolve-Path -LiteralPath $PackageDir).Path
$required = @("bridge.exe", "icoo_desktop.exe")
# Optional process plugins packaged next to core binaries.
$optionalPluginPatterns = @("plugin-*.exe", "mockplugin.exe")

function Assert-WindowsPE {
  param([Parameter(Mandatory = $true)][string]$Path)
  $item = Get-Item -LiteralPath $Path
  if ($item.Length -lt 1MB) {
    throw "Package artifact is unexpectedly small: $Path ($($item.Length) bytes)"
  }
  $stream = [System.IO.File]::OpenRead($Path)
  try {
    $first = $stream.ReadByte()
    $second = $stream.ReadByte()
  } finally {
    $stream.Dispose()
  }
  if ($first -ne [byte][char]'M' -or $second -ne [byte][char]'Z') {
    throw "Package artifact is not a Windows PE executable: $Path"
  }
}

$allowed = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
foreach ($name in $required) {
  $path = Join-Path $resolvedPackage $name
  if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
    throw "Missing package artifact: $path"
  }
  Assert-WindowsPE -Path $path
  [void]$allowed.Add($name)
}

foreach ($pattern in $optionalPluginPatterns) {
  Get-ChildItem -LiteralPath $resolvedPackage -Filter $pattern -File -ErrorAction SilentlyContinue | ForEach-Object {
    Assert-WindowsPE -Path $_.FullName
    [void]$allowed.Add($_.Name)
  }
}

$unexpectedExecutables = Get-ChildItem -LiteralPath $resolvedPackage -Filter *.exe -File |
  Where-Object { -not $allowed.Contains($_.Name) }
if ($unexpectedExecutables) {
  throw "Unexpected executable artifacts: $($unexpectedExecutables.Name -join ', ')"
}

Write-Host "Package smoke test passed: $resolvedPackage" -ForegroundColor Green
Write-Host "  Required: $($required -join ', ')" -ForegroundColor Green
$plugins = @($allowed | Where-Object { $_ -notin $required })
if ($plugins.Count -gt 0) {
  Write-Host "  Plugins:  $($plugins -join ', ')" -ForegroundColor Green
}
