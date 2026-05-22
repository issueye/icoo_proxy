# Rust Replacement Readiness Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Raise `icoo_llm_bridge_r` from a functional Rust rewrite to a Go-version replacement candidate that can be used to gray-replace the Go service with clear rollback.

**Architecture:** Keep the Rust service separated under `icoo_llm_bridge_r` and preserve the existing Axum/Tokio/Reqwest/Rusqlite layering. This phase does not attempt full legacy database migration because that has been accepted as non-blocking. It also does not use mock upstream services; verification must run against copied local configuration and real upstream providers where credentials are available.

**Tech Stack:** Rust 2021, Tokio, Axum, Reqwest, Futures, Rusqlite, Serde JSON, PowerShell build script, existing Go implementation as behavior reference.

---

## Current Baseline

The Rust version currently scores about 78/100 for replacement readiness.

Already completed:
- Independent Rust project exists at `icoo_llm_bridge_r`.
- Go version remains untouched at `icoo_llm_bridge`.
- Core config, startup, health check, admin API, route resolution, proxy forwarding, API key auth, traffic recording, and main protocol conversion paths are implemented.
- Same-protocol SSE pass-through is low latency.
- Tests pass: `cargo test`, `.\build.ps1 -SkipTests`, and local smoke checks.

Known non-blocking gap:
- Earlier old database schemas are only partially compatible. Current plan keeps this as-is because the current requirement allows it.

Primary remaining risks:
- Cross-protocol streaming conversion still buffers in some paths.
- Protocol conversion has good main-path coverage but not exhaustive Go parity.
- Real upstream compatibility has not been systematically verified.
- Release/runbook guardrails are not yet explicit enough for a safe replacement.

---

### Task 1: Create a Go-vs-Rust Parity Checklist

**Files:**
- Create: `icoo_llm_bridge_r/docs/parity/go-rust-parity-checklist.md`
- Read: `icoo_llm_bridge/internal/service/proxy_service.go`
- Read: `icoo_llm_bridge/internal/utils/ai_llm_proxy/*.go`
- Read: `icoo_llm_bridge_r/src/proxy.rs`
- Read: `icoo_llm_bridge_r/src/protocol.rs`

**Step 1: Inventory public behavior**

List every public behavior that must match:
- Fixed endpoints: `/v1/messages`, `/v1/chat/completions`, `/v1/responses`
- Dynamic endpoints
- Admin API endpoints
- Auth behavior
- Routing behavior
- Traffic recording
- Protocol conversion request paths
- Protocol conversion response paths
- Streaming conversion paths

**Step 2: Mark current coverage**

For each item, mark one of:
- `done`
- `covered by test`
- `partially covered`
- `not implemented`
- `accepted gap`

**Step 3: Add a short risk label**

Use:
- `P0`: replacement blocker
- `P1`: gray-release risk
- `P2`: follow-up

**Step 4: Verify the checklist is actionable**

Run:

```powershell
Get-Content .\docs\parity\go-rust-parity-checklist.md
```

Expected: the file exists and clearly lists remaining work.

---

### Task 2: Add Cross-Protocol Streaming Tests

**Files:**
- Modify: `icoo_llm_bridge_r/src/protocol.rs`
- Modify: `icoo_llm_bridge_r/tests/proxy_e2e.rs`

**Step 1: Add Responses-to-Chat streaming tool-call test**

Add a protocol unit test where Responses stream events contain:
- `response.created`
- `response.output_item.added` with `function_call`
- `response.function_call_arguments.delta`
- `response.completed`

Expected Chat stream chunks include:
- assistant role
- `tool_calls` delta
- function arguments delta
- final `finish_reason: "tool_calls"`
- `[DONE]`

**Step 2: Add Responses-to-Chat reasoning stream test**

Add a protocol unit test where Responses stream events contain:
- `response.reasoning_summary_text.delta`
- `response.output_text.delta`
- `response.completed`

Expected Chat stream chunks preserve reasoning as `reasoning_content` deltas where possible.

**Step 3: Add Chat-to-Responses tool-call stream test**

Add a protocol unit test where Chat stream chunks contain:
- `delta.tool_calls`
- function name
- function argument fragments
- `finish_reason: "tool_calls"`

Expected Responses stream includes function-call item and argument delta events.

**Step 4: Run focused protocol tests**

Run:

```powershell
$env:CARGO_HOME=(Join-Path (Get-Location) '.cargo-home'); cargo test protocol::tests
```

Expected: newly added tests fail if the converter is still missing these stream fields.

---

### Task 3: Implement Missing Cross-Protocol Stream Field Conversion

**Files:**
- Modify: `icoo_llm_bridge_r/src/protocol.rs`

**Step 1: Extend Responses-to-Chat stream conversion**

Update `responses_stream_to_chat` to handle:
- `response.output_item.added` for `function_call`
- `response.function_call_arguments.delta`
- `response.reasoning_summary_text.delta`

Use Chat Completions-compatible deltas:
- `delta.tool_calls`
- `delta.reasoning_content`

**Step 2: Track finish reason**

Track whether any tool call was seen.

Expected:
- tool call stream ends with `finish_reason: "tool_calls"`
- text-only stream ends with `finish_reason: "stop"`

**Step 3: Extend Chat-to-Responses stream conversion**

Update `chat_stream_to_responses` to handle:
- `delta.tool_calls`
- function-call item creation
- argument deltas
- final completed event

Keep the implementation conservative and stateful. Do not rewrite the whole converter.

**Step 4: Run focused tests**

Run:

```powershell
$env:CARGO_HOME=(Join-Path (Get-Location) '.cargo-home'); cargo test protocol::tests
```

Expected: all protocol tests pass.

---

### Task 4: Add Real Upstream Gray-Readiness Verification

**Files:**
- Create: `icoo_llm_bridge_r/docs/release/real-upstream-verification.md`
- Create: `icoo_llm_bridge_r/scripts/verify-real-upstream.ps1`
- Modify: `icoo_llm_bridge_r/docs/parity/go-rust-parity-checklist.md`

**Step 1: Define the real-upstream verification matrix**

Document the exact checks that must run before gray replacement:
- OpenAI Chat downstream to OpenAI Chat upstream
- OpenAI Chat downstream to OpenAI Responses upstream
- OpenAI Responses downstream to OpenAI Responses upstream
- OpenAI Responses downstream to Anthropic upstream
- Anthropic Messages downstream to Anthropic upstream
- Anthropic Messages downstream to OpenAI Responses upstream
- Same-protocol streaming
- Cross-protocol streaming
- Upstream error propagation

**Step 2: Create a copied-data verification procedure**

The procedure must never point the Rust verification run at the production data directory directly.

Document:
- how to stop the Go service or leave it running on its existing port
- how to copy the Go service data directory to a temporary Rust verification directory
- how to start Rust on a different port
- how to confirm providers, models, routing rules, endpoints, and API keys are readable
- how to run a minimal request through each enabled route

**Step 3: Create the PowerShell verification script**

Create `scripts/verify-real-upstream.ps1` with parameters:

```powershell
param(
  [string]$BridgeUrl = "http://127.0.0.1:18182",
  [string]$ApiKey = "",
  [string]$ChatModel = "",
  [string]$ResponsesModel = "",
  [string]$AnthropicModel = ""
)
```

The script should:
- call `/healthz`
- call `/api/v1/runtime/state`
- call `/api/v1/providers`
- call `/api/v1/ingress-endpoints`
- call `/api/v1/routing-rules`
- send one non-stream Chat request if `$ChatModel` is set
- send one stream Chat request if `$ChatModel` is set
- send one non-stream Responses request if `$ResponsesModel` is set
- send one stream Responses request if `$ResponsesModel` is set
- send one Anthropic Messages request if `$AnthropicModel` is set
- print a compact pass/fail table

Run:

```powershell
.\scripts\verify-real-upstream.ps1 -BridgeUrl http://127.0.0.1:18182 -ApiKey <key> -ChatModel <model>
```

Expected: script reports pass for each route that has a supplied model. Routes without supplied models are skipped, not failed.

**Step 4: Record required manual observations**

In `real-upstream-verification.md`, require the operator to record:
- Go service version/build being compared
- Rust executable path and timestamp
- copied data directory path
- upstream providers tested
- models tested
- request IDs for successful traffic records
- any upstream errors and whether Rust preserved status/body correctly

**Step 5: Update parity checklist**

Mark real-upstream verification as a gray-release gate.

Expected:
- no mock server is introduced
- no test depends on external credentials by default
- real upstream checks are repeatable through the script and documented procedure

---

### Task 5: Add Gray Replacement Safety Checks

**Files:**
- Modify: `icoo_llm_bridge_r/src/db.rs`
- Modify: `icoo_llm_bridge_r/src/http_app.rs`
- Modify: `icoo_llm_bridge_r/tests/proxy_e2e.rs`
- Create: `icoo_llm_bridge_r/docs/release/rust-gray-release-runbook.md`

**Step 1: Add startup schema diagnostics**

At startup, check important tables and columns:
- `providers`
- `provider_models`
- `ingress_endpoints`
- `routing_rules`
- `api_keys`
- traffic database `traffic_records`

Do not auto-migrate old database gaps in this task.

**Step 2: Expose diagnostics in runtime state**

Add a compact field to `/api/v1/runtime/state`, for example:

```json
{
  "database": {
    "main_ok": true,
    "traffic_ok": true,
    "warnings": []
  }
}
```

**Step 3: Test current schema returns no warnings**

Add an e2e test using a fresh Rust-created database.

Expected:
- `main_ok = true`
- `traffic_ok = true`
- warnings empty

**Step 4: Test accepted old schema warning**

Create a temp old-style database missing `routing_rules.upstream_protocol`.

Expected:
- service still starts
- runtime state reports a warning
- this remains an accepted non-blocking compatibility gap

**Step 5: Write gray replacement runbook**

Document:
- how to run Rust and Go side by side
- how to point Rust at a copied data directory
- smoke test commands
- real-upstream verification commands
- traffic switch criteria
- rollback steps
- known accepted database compatibility limitation

---

### Task 6: Final Replacement Verification

**Files:**
- Modify only files needed for failed tests.

**Step 1: Run Rust full tests**

Run:

```powershell
$env:CARGO_HOME=(Join-Path (Get-Location) '.cargo-home'); cargo test
```

Expected: all tests pass.

**Step 2: Run Rust release build**

Run:

```powershell
.\build.ps1 -SkipTests
```

Expected:
- `icoo_llm_bridge_r/build/bridge.exe` exists

**Step 3: Smoke test executable**

Run the built executable with a temp data directory and non-default port.

Check:
- `/healthz`
- `/readyz`
- `/api/v1/runtime/state`
- one admin list endpoint
- real-upstream verification script for configured providers

Expected: all return successful responses.

**Step 4: Update the score**

Update the replacement score in a short note:

```text
Before: 78/100
After: expected 88/100 or higher if all tasks pass
Remaining accepted gaps: old database full migration, any explicitly deferred provider-specific edge cases
```

---

## Exit Criteria

This phase is complete when:
- Cross-protocol streaming tool/reasoning behavior has tests and implementation.
- Real-upstream verification procedure and script exist; execute them with configured providers before traffic switch.
- Runtime state exposes database diagnostics.
- Gray-release runbook exists.
- `cargo test` passes.
- `.\build.ps1 -SkipTests` passes.
- Built executable passes smoke checks.

After completion, `icoo_llm_bridge_r` should be suitable for controlled gray replacement of the Go service, with rollback handled by switching traffic back to the Go executable.
