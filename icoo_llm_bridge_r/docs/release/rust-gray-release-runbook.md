# Rust Gray Replacement Runbook

This runbook describes how to gray-replace the Go `icoo_llm_bridge` service with `icoo_llm_bridge_r`.

## Preconditions

- Rust build passes with `.\build.ps1 -SkipTests`.
- Rust tests pass with `cargo test`.
- `build\bridge.exe` exists.
- Real upstream verification has been run for every route that will receive gray traffic.
- Rollback path to the Go service is ready.
- A copied data directory has been tested. Do not use the production data directory directly for verification.

## Build

```powershell
cd E:\code\issueye\icoo_proxy\icoo_llm_bridge_r
$env:CARGO_HOME=(Join-Path (Get-Location) '.cargo-home')
cargo test
.\build.ps1 -SkipTests
```

Expected:
- all tests pass
- `E:\code\issueye\icoo_proxy\icoo_llm_bridge_r\build\bridge.exe` exists

## Side-By-Side Startup

Keep the Go service on its current port. Start Rust on a separate port with a copied data directory:

```powershell
$verifyData = "E:\path\to\copied\.data"
.\build\bridge.exe --addr 127.0.0.1:18182 --data-dir $verifyData
```

Check:

```powershell
Invoke-RestMethod http://127.0.0.1:18182/healthz
Invoke-RestMethod http://127.0.0.1:18182/readyz
Invoke-RestMethod http://127.0.0.1:18182/api/v1/runtime/state
```

The runtime state must show:
- service `icoo_llm_bridge`
- `running = true`
- correct listen address
- database `main_ok = true` and `traffic_ok = true` for the schema intended for gray traffic

Warnings are acceptable only if they match a known non-blocking limitation, such as older legacy schema gaps that are not used for the gray run.

## Preflight With Copied Data

Run the preflight script before real upstream verification. It copies a data directory, starts Rust on a separate port, checks health/readiness/runtime state, and confirms admin list endpoints are readable.

```powershell
.\scripts\preflight-gray-replacement.ps1 `
  -SourceDataDir "E:\path\to\go-data" `
  -BridgeUrl http://127.0.0.1:18184 `
  -ApiKey "<admin-or-proxy-key-if-local-auth-disabled>"
```

The script may report database warnings for accepted old schema gaps. The gray run can proceed only if the relevant admin endpoints still pass and the warning is understood.

## Real Upstream Verification

Run the script with the models that will be used for gray traffic:

```powershell
.\scripts\verify-real-upstream.ps1 `
  -BridgeUrl http://127.0.0.1:18182 `
  -ApiKey "<proxy-key>" `
  -ChatModel "<chat-route-model>" `
  -ResponsesModel "<responses-route-model>" `
  -AnthropicModel "<anthropic-route-model>"
```

Pass criteria:
- health, readiness, runtime state, providers, endpoints, and routing rules pass
- every supplied model path passes
- skipped paths are intentional because no model was supplied
- traffic records are visible after proxy requests

## Gray Traffic Switch

Start with a small share of traffic or a single controlled caller.

Before switching:
- confirm the Rust process is stable
- confirm real-upstream verification passed
- confirm logs and traffic records are being written
- confirm API key scopes work for the caller

During gray traffic:
- watch request latency
- watch upstream error rates
- compare response shapes for representative calls
- inspect `/api/v1/traffic` for route/source/model correctness
- keep the Go process available for rollback

## Rollback

Rollback means routing traffic back to the Go service.

Use the existing service supervisor, reverse proxy, or caller configuration to restore the Go service endpoint.

After rollback:
- confirm Go `/healthz` or equivalent endpoint is healthy
- confirm requests are no longer reaching Rust
- keep Rust logs and copied data directory for analysis
- do not delete traffic evidence until failures are understood

## Accepted Limitations

- Full migration of older legacy database schemas is not part of this phase.
- Older databases missing `routing_rules.upstream_protocol` and `api_keys.secret_cipher` are readable. New writes preserve compatibility with those missing columns, but they do not add the columns automatically.
- Real upstream verification depends on available configured providers and credentials.
- Cross-protocol streaming has targeted tool-call and reasoning coverage, but provider-specific SSE variants should still be watched during gray traffic.
