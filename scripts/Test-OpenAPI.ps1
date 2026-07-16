param(
  [string]$RouterPath = ".\bridge\internal\router\router.go",
  [string]$SpecPath = ".\docs\openapi.yaml"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$router = Get-Content -LiteralPath $RouterPath -Raw
$expected = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)

foreach ($match in [regex]::Matches($router, 'api\.(GET|POST|PUT|DELETE)\("([^"]+)"')) {
  $method = $match.Groups[1].Value
  $path = "/api/v1" + $match.Groups[2].Value
  $path = $path -replace ':([A-Za-z_]+)', '{$1}'
  [void]$expected.Add("$method $path")
}
foreach ($match in [regex]::Matches($router, 'engine\.(GET|POST|PUT|DELETE)\("([^"]+)"')) {
  $path = $match.Groups[2].Value
  if ($path -in @("/healthz", "/readyz")) {
    [void]$expected.Add("$($match.Groups[1].Value) $path")
  }
}

$documented = [System.Collections.Generic.HashSet[string]]::new([StringComparer]::OrdinalIgnoreCase)
$currentPath = ""
foreach ($line in Get-Content -LiteralPath $SpecPath) {
  if ($line -match '^  (/[^:]+):\s*$') {
    $currentPath = $Matches[1]
    continue
  }
  if ($currentPath -and $line -match '^    (get|post|put|delete):\s*$') {
    [void]$documented.Add("$($Matches[1].ToUpperInvariant()) $currentPath")
  }
}

$missing = @($expected | Where-Object { -not $documented.Contains($_) } | Sort-Object)
if ($missing.Count -gt 0) {
  throw "OpenAPI is missing registered routes: $($missing -join ', ')"
}

Write-Host "OpenAPI route coverage passed: $($expected.Count) registered operations" -ForegroundColor Green

