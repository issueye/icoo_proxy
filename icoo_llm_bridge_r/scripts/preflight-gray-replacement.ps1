param(
  [string]$SourceDataDir = "",
  [string]$BridgeExe = "",
  [string]$BridgeUrl = "http://127.0.0.1:18182",
  [string]$ApiKey = "",
  [switch]$SkipCopy
)

$ErrorActionPreference = "Stop"

function Add-Result {
  param(
    [System.Collections.Generic.List[object]]$Results,
    [string]$Name,
    [string]$Status,
    [string]$Detail = ""
  )
  $Results.Add([pscustomobject]@{ Name = $Name; Status = $Status; Detail = $Detail }) | Out-Null
}

function Invoke-Check {
  param(
    [System.Collections.Generic.List[object]]$Results,
    [string]$Name,
    [scriptblock]$Block
  )
  try {
    $detail = & $Block
    Add-Result $Results $Name "PASS" $detail
  } catch {
    Add-Result $Results $Name "FAIL" $_.Exception.Message
  }
}

function New-Headers {
  $headers = @{}
  if (-not [string]::IsNullOrWhiteSpace($ApiKey)) {
    $headers["Authorization"] = "Bearer $ApiKey"
  }
  return $headers
}

function Resolve-DataDir {
  param([string]$Path)
  if (-not [string]::IsNullOrWhiteSpace($Path)) {
    return (Resolve-Path $Path).Path
  }
  $candidates = @(
    (Join-Path (Get-Location).Path ".data"),
    (Join-Path (Split-Path (Get-Location).Path -Parent) "icoo_proxy\.data"),
    (Join-Path (Split-Path (Get-Location).Path -Parent) "icoo_proxy")
  )
  foreach ($candidate in $candidates) {
    if (Test-Path (Join-Path $candidate "icoo_llm_bridge.db")) {
      return (Resolve-Path $candidate).Path
    }
  }
  throw "SourceDataDir was not provided and no icoo_llm_bridge.db was found in default locations"
}

if ([string]::IsNullOrWhiteSpace($BridgeExe)) {
  $BridgeExe = Join-Path (Get-Location).Path "build\bridge.exe"
}
$BridgeExe = (Resolve-Path $BridgeExe).Path

$results = [System.Collections.Generic.List[object]]::new()
$process = $null
$verifyData = $null

try {
  Invoke-Check $results "bridge executable" {
    if (-not (Test-Path $BridgeExe)) { throw "not found: $BridgeExe" }
    $item = Get-Item $BridgeExe
    "path=$($item.FullName); bytes=$($item.Length)"
  }

  $source = Resolve-DataDir $SourceDataDir
  Invoke-Check $results "source data directory" {
    if (-not (Test-Path (Join-Path $source "icoo_llm_bridge.db"))) { throw "icoo_llm_bridge.db not found in $source" }
    "path=$source"
  }

  if ($SkipCopy) {
    $verifyData = $source
  } else {
    $verifyData = Join-Path $env:TEMP ("icoo_llm_bridge_r_preflight_" + [Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $verifyData | Out-Null
    Copy-Item -Path (Join-Path $source "*") -Destination $verifyData -Recurse -Force
  }

  $uri = [Uri]$BridgeUrl
  $addr = $uri.Authority
  $stdout = Join-Path $verifyData "rust-preflight-stdout.log"
  $stderr = Join-Path $verifyData "rust-preflight-stderr.log"
  $process = Start-Process -FilePath $BridgeExe -ArgumentList @("--addr", $addr, "--data-dir", $verifyData) -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru

  Invoke-Check $results "startup" {
    for ($i = 0; $i -lt 40; $i++) {
      try {
        $health = Invoke-RestMethod -Uri ($BridgeUrl.TrimEnd('/') + "/healthz") -TimeoutSec 1
        if ($health.service -eq "icoo_llm_bridge") { return "pid=$($process.Id); data=$verifyData" }
      } catch {
        Start-Sleep -Milliseconds 250
      }
    }
    throw "service did not become ready"
  }

  Invoke-Check $results "readyz" {
    $r = Invoke-RestMethod -Uri ($BridgeUrl.TrimEnd('/') + "/readyz") -TimeoutSec 5
    if ($r.service -ne "icoo_llm_bridge") { throw "unexpected service: $($r.service)" }
    "ready=$($r.ready)"
  }

  Invoke-Check $results "runtime database diagnostics" {
    $r = Invoke-RestMethod -Uri ($BridgeUrl.TrimEnd('/') + "/api/v1/runtime/state") -Headers (New-Headers) -TimeoutSec 5
    $data = if ($null -ne $r.data) { $r.data } else { $r }
    $warnings = @($data.database.warnings)
    "main_ok=$($data.database.main_ok); traffic_ok=$($data.database.traffic_ok); warnings=$($warnings.Count); listen=$($data.listen_addr)"
  }

  foreach ($path in @("/api/v1/providers", "/api/v1/ingress-endpoints", "/api/v1/routing-rules", "/api/v1/api-keys", "/api/v1/traffic")) {
    Invoke-Check $results $path {
      $r = Invoke-WebRequest -Uri ($BridgeUrl.TrimEnd('/') + $path) -Headers (New-Headers) -TimeoutSec 10 -UseBasicParsing
      if ($r.StatusCode -lt 200 -or $r.StatusCode -ge 300) { throw "HTTP $($r.StatusCode)" }
      "HTTP $($r.StatusCode)"
    }
  }
} finally {
  if ($process -and -not $process.HasExited) {
    Stop-Process -Id $process.Id -Force
  }
}

$results | Format-Table -AutoSize

$failed = $results | Where-Object { $_.Status -eq "FAIL" }
if ($failed.Count -gt 0) {
  exit 1
}

