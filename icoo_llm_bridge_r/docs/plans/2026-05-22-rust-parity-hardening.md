# Rust Parity Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Bring `icoo_llm_bridge_r` closer to Go-version replacement readiness by closing the highest-risk gaps: low-latency streaming proxy behavior, protocol converter parity tests, and admin API compatibility tests.

**Architecture:** Keep the current Rust layering: Axum handlers call proxy/admin services, repositories own SQLite, and `protocol.rs` remains pure conversion logic. Streaming work should preserve the existing non-stream path while adding a streaming response type so pass-through streams can be returned as an Axum body without buffering the whole upstream response.

**Tech Stack:** Rust 2021, Tokio, Axum, Reqwest streaming, Futures, Rusqlite, Serde JSON, PowerShell build script.

---

### Task 1: Low-Latency Pass-Through Streaming

**Files:**
- Modify: `src/proxy.rs`
- Test: `tests/proxy_e2e.rs`

**Step 1: Write a failing streaming latency test**

Add an e2e test where upstream sends one SSE event, flushes, waits, then sends the final event. The client reads `bytes_stream()` from the bridge and must receive the first event before the upstream finishes.

**Step 2: Run the focused test**

Run: `cargo test streaming_passthrough_returns_first_event_before_upstream_finishes --test proxy_e2e`

Expected: FAIL because current proxy buffers `resp.bytes().await`.

**Step 3: Introduce a proxy output enum**

Change `forward` from returning only `Vec<u8>` to returning either buffered bytes or an Axum `Body` stream.

**Step 4: Stream pass-through upstream responses**

When upstream content type is `text/event-stream` and downstream protocol equals upstream protocol, return `Body::from_stream(resp.bytes_stream())` immediately with `text/event-stream` headers.

**Step 5: Keep converted streams buffered for now**

For cross-protocol streams, keep the current buffered conversion path until the converter is refactored to async frame-by-frame conversion. This is an explicit intermediate step.

**Step 6: Run the focused test**

Run: `cargo test streaming_passthrough_returns_first_event_before_upstream_finishes --test proxy_e2e`

Expected: PASS.

### Task 2: Streaming Conversion Parity Tests

**Files:**
- Modify: `src/protocol.rs`

**Step 1: Port Go converter tests for stream terminal behavior**

Add Rust tests equivalent to:
- Responses stream to Chat stops after `response.completed` without EOF.
- Chat stream to Responses stops after `[DONE]` without EOF.
- Chat stream to Anthropic includes `message_stop`.

**Step 2: Run protocol tests**

Run: `cargo test protocol::tests`

Expected: PASS or expose missing conversion behavior.

**Step 3: Fix any missing stream finalization behavior**

Patch `src/protocol.rs` only where the tests show a parity gap.

### Task 3: Admin API Compatibility Tests

**Files:**
- Modify: `tests/proxy_e2e.rs`

**Step 1: Add API key reveal/update test**

Create key, reveal secret, update metadata without secret, reveal same secret again.

**Step 2: Add built-in endpoint delete protection test**

Call `DELETE /api/v1/ingress-endpoints/endpoint-v1-responses` and assert `400` with `built-in endpoint cannot be deleted`.

**Step 3: Add pagination alias test**

Verify both `page_size` and `pageSize` produce `data.page_size`.

**Step 4: Run e2e tests**

Run: `cargo test --test proxy_e2e`

Expected: PASS.

### Task 4: Converter Request/Response Parity Expansion

**Files:**
- Modify: `src/protocol.rs`

**Step 1: Add tests for multimodal text/image preservation**

Cover Chat content array and Anthropic image blocks where the current simplified converter may lose fields.

**Step 2: Add tests for reasoning/tool fields**

Cover `reasoning_effort`, Responses `reasoning`, and basic `tools/tool_choice` pass-through.

**Step 3: Patch converter gaps conservatively**

Preserve existing JSON fields where possible rather than inventing new mappings.

### Task 5: Final Verification

**Files:**
- Modify only files needed for failing tests.

**Step 1: Run Rust tests**

Run: `cargo test`

Expected: PASS.

**Step 2: Run Rust build**

Run: `.\build.ps1 -SkipTests`

Expected: `build\bridge.exe` exists.

**Step 3: Smoke test executable**

Run the built bridge with a temp data dir and check `/healthz` and `/api/v1/runtime/state`.

Expected: both respond with service `icoo_llm_bridge`.
