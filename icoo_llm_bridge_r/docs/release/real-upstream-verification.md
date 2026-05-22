# Real Upstream Verification

This procedure verifies that `icoo_llm_bridge_r` can gray-replace the Go service against real upstream providers. It intentionally does not use a mock upstream.

## Rules

- Do not point the Rust verification run at the production data directory directly.
- Copy the Go service data directory to a temporary verification directory first.
- Run Rust on a different port from the Go service.
- Only test providers and models that are already configured and allowed for verification.
- Record every provider/model pair tested and keep request IDs for traffic lookups.

## Setup

1. Build Rust:

```powershell
cd E:\code\issueye\icoo_proxy\icoo_llm_bridge_r
.\build.ps1 -SkipTests
```

2. Copy the Go data directory:

```powershell
$sourceData = "E:\path\to\go\.data"
$verifyData = Join-Path $env:TEMP ("icoo_llm_bridge_r_verify_" + [Guid]::NewGuid().ToString("N"))
Copy-Item -Path $sourceData -Destination $verifyData -Recurse
```

3. Start Rust on a separate port:

```powershell
.\build\bridge.exe --addr 127.0.0.1:18182 --data-dir $verifyData
```

4. Run the real-upstream verification script from another PowerShell window:

```powershell
.\scripts\verify-real-upstream.ps1 `
  -BridgeUrl http://127.0.0.1:18182 `
  -ApiKey "<proxy-key>" `
  -ChatModel "<chat-route-model>" `
  -ResponsesModel "<responses-route-model>" `
  -AnthropicModel "<anthropic-route-model>"
```

Omit model parameters for protocols that are not configured in the copied data directory. The script skips omitted models.

## Preflight

Before sending real upstream requests, run:

```powershell
.\scripts\preflight-gray-replacement.ps1 `
  -SourceDataDir "E:\path\to\go-data" `
  -BridgeUrl http://127.0.0.1:18184 `
  -ApiKey "<key-if-needed>"
```

This confirms that the copied data directory can start under Rust and that admin list endpoints are readable. It is useful for catching accepted old-schema warnings before traffic is sent to real providers.

## Verification Matrix

Run at least one configured route from each available category:

| Check | Required Before Gray Replacement | Notes |
| --- | --- | --- |
| `/healthz` | yes | Must return service `icoo_llm_bridge`. |
| `/readyz` | yes | Must return ready status. |
| `/api/v1/runtime/state` | yes | Must show correct listen address and DB diagnostics. |
| `/api/v1/providers` | yes | Confirms copied provider config is readable. |
| `/api/v1/ingress-endpoints` | yes | Confirms endpoints are readable. |
| `/api/v1/routing-rules` | yes | Confirms route rules are readable for current schema. |
| Chat non-stream request | if configured | Verifies route, model rewrite, upstream auth, response conversion. |
| Chat stream request | if configured | Verifies SSE behavior. |
| Responses non-stream request | if configured | Verifies route and response shape. |
| Responses stream request | if configured | Verifies Responses SSE behavior. |
| Anthropic Messages request | if configured | Verifies Anthropic-compatible path. |
| Upstream error propagation | if safely reproducible | Use a known invalid model or disabled route only in a non-production copy. |

## Record

For each verification run, record:

- Go service build/version being compared:
- Rust executable path:
- Rust executable timestamp:
- Copied data directory:
- Bridge URL:
- Providers tested:
- Models tested:
- Successful request IDs:
- Failed request IDs:
- Upstream errors and whether Rust preserved status/body:
- Decision: pass/fail for gray replacement:

## Pass Criteria

Gray replacement can proceed when:

- Health, readiness, runtime state, providers, endpoints, and routing rules checks pass.
- Every configured protocol path intended for gray traffic passes at least one real request.
- Streaming passes for any protocol path that will receive streaming traffic.
- Traffic records are written for successful and failed proxy attempts.
- Known accepted database limitations are understood and not triggered by the copied production schema.
- Rollback path in `rust-gray-release-runbook.md` is ready.
