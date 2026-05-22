param(
  [string]$BridgeUrl = "http://127.0.0.1:18182",
  [string]$ApiKey = "",
  [string]$ChatModel = "",
  [string]$ResponsesModel = "",
  [string]$AnthropicModel = ""
)

$ErrorActionPreference = "Stop"

function New-Headers {
  $headers = @{}
  if (-not [string]::IsNullOrWhiteSpace($ApiKey)) {
    $headers["Authorization"] = "Bearer $ApiKey"
  }
  return $headers
}

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

function Invoke-JsonPost {
  param(
    [string]$Path,
    [object]$Body
  )
  $json = $Body | ConvertTo-Json -Depth 20 -Compress
  Invoke-WebRequest `
    -Uri ($BridgeUrl.TrimEnd('/') + $Path) `
    -Method POST `
    -Headers (New-Headers) `
    -ContentType "application/json" `
    -Body $json `
    -TimeoutSec 120 `
    -UseBasicParsing
}

function Assert-Status2xx {
  param($Response)
  if ($Response.StatusCode -lt 200 -or $Response.StatusCode -ge 300) {
    throw "HTTP $($Response.StatusCode)"
  }
}

$results = [System.Collections.Generic.List[object]]::new()

Invoke-Check $results "healthz" {
  $r = Invoke-RestMethod -Uri ($BridgeUrl.TrimEnd('/') + "/healthz") -TimeoutSec 10
  if ($r.service -ne "icoo_llm_bridge") { throw "unexpected service: $($r.service)" }
  "service=$($r.service)"
}

Invoke-Check $results "readyz" {
  $r = Invoke-RestMethod -Uri ($BridgeUrl.TrimEnd('/') + "/readyz") -TimeoutSec 10
  if ($r.service -ne "icoo_llm_bridge") { throw "unexpected service: $($r.service)" }
  "ready=$($r.ready)"
}

Invoke-Check $results "runtime state" {
  $r = Invoke-RestMethod -Uri ($BridgeUrl.TrimEnd('/') + "/api/v1/runtime/state") -Headers (New-Headers) -TimeoutSec 10
  $data = if ($null -ne $r.data) { $r.data } else { $r }
  if ($data.service -ne "icoo_llm_bridge") { throw "unexpected service: $($data.service)" }
  "listen=$($data.listen_addr)"
}

foreach ($path in @("/api/v1/providers", "/api/v1/ingress-endpoints", "/api/v1/routing-rules")) {
  Invoke-Check $results $path {
    $r = Invoke-WebRequest -Uri ($BridgeUrl.TrimEnd('/') + $path) -Headers (New-Headers) -TimeoutSec 10 -UseBasicParsing
    Assert-Status2xx $r
    "HTTP $($r.StatusCode)"
  }
}

if ([string]::IsNullOrWhiteSpace($ChatModel)) {
  Add-Result $results "chat non-stream" "SKIP" "ChatModel not provided"
  Add-Result $results "chat stream" "SKIP" "ChatModel not provided"
} else {
  Invoke-Check $results "chat non-stream" {
    $r = Invoke-JsonPost "/v1/chat/completions" @{
      model = $ChatModel
      messages = @(@{ role = "user"; content = "Reply with exactly: ok" })
      stream = $false
      max_tokens = 16
    }
    Assert-Status2xx $r
    "HTTP $($r.StatusCode); request_id=$($r.Headers['x-icoo-request-id'])"
  }
  Invoke-Check $results "chat stream" {
    $r = Invoke-JsonPost "/v1/chat/completions" @{
      model = $ChatModel
      messages = @(@{ role = "user"; content = "Reply with exactly: ok" })
      stream = $true
      max_tokens = 16
    }
    Assert-Status2xx $r
    if ($r.Content -notmatch "data:") { throw "stream response has no SSE data" }
    "HTTP $($r.StatusCode); request_id=$($r.Headers['x-icoo-request-id'])"
  }
}

if ([string]::IsNullOrWhiteSpace($ResponsesModel)) {
  Add-Result $results "responses non-stream" "SKIP" "ResponsesModel not provided"
  Add-Result $results "responses stream" "SKIP" "ResponsesModel not provided"
} else {
  Invoke-Check $results "responses non-stream" {
    $r = Invoke-JsonPost "/v1/responses" @{
      model = $ResponsesModel
      input = "Reply with exactly: ok"
      stream = $false
      max_output_tokens = 16
    }
    Assert-Status2xx $r
    "HTTP $($r.StatusCode); request_id=$($r.Headers['x-icoo-request-id'])"
  }
  Invoke-Check $results "responses stream" {
    $r = Invoke-JsonPost "/v1/responses" @{
      model = $ResponsesModel
      input = "Reply with exactly: ok"
      stream = $true
      max_output_tokens = 16
    }
    Assert-Status2xx $r
    if ($r.Content -notmatch "data:|event:") { throw "stream response has no SSE data" }
    "HTTP $($r.StatusCode); request_id=$($r.Headers['x-icoo-request-id'])"
  }
}

if ([string]::IsNullOrWhiteSpace($AnthropicModel)) {
  Add-Result $results "anthropic messages" "SKIP" "AnthropicModel not provided"
} else {
  Invoke-Check $results "anthropic messages" {
    $r = Invoke-JsonPost "/v1/messages" @{
      model = $AnthropicModel
      messages = @(@{ role = "user"; content = "Reply with exactly: ok" })
      max_tokens = 16
      stream = $false
    }
    Assert-Status2xx $r
    "HTTP $($r.StatusCode); request_id=$($r.Headers['x-icoo-request-id'])"
  }
}

$results | Format-Table -AutoSize

$failed = $results | Where-Object { $_.Status -eq "FAIL" }
if ($failed.Count -gt 0) {
  exit 1
}

