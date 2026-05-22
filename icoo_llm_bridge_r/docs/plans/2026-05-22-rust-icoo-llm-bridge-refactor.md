# Rust icoo_llm_bridge Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rebuild `icoo_llm_bridge` in Rust while keeping the executable name, command-line flags, HTTP API, SQLite persistence, proxy routing, and protocol conversion behavior equivalent to the current Go service.

**Architecture:** Keep the existing layered shape: HTTP handlers call services, services call repositories, repositories own SQLite access, and protocol conversion remains pure and testable. The Rust crate will use `axum` for HTTP, `tokio` for async runtime, `sqlx` with SQLite for persistence, `serde`/`serde_json` for protocol bodies, and `reqwest` for upstream calls.

**Tech Stack:** Rust 2021, Tokio, Axum, SQLx SQLite, Reqwest, Serde, TOML, SHA-2, UUID/Rand, PowerShell build script.

---

### Task 1: Rust Crate And Build Contract

**Files:**
- Create: `Cargo.toml`
- Create: `src/main.rs`
- Create: `src/lib.rs`
- Modify: `build.ps1`

**Steps:**
1. Add a Rust binary crate named `icoo_llm_bridge` that builds to `target/release/icoo_llm_bridge.exe`.
2. Keep CLI flags compatible with Go: `-config`, `-data-dir`, `-addr`.
3. Update `build.ps1` so it checks `cargo`, runs `cargo test` unless `-SkipTests`, builds release, and copies the binary to `build/bridge.exe`.
4. Verify `.\build.ps1 -SkipTests` produces `build/bridge.exe`.

### Task 2: Configuration And Runtime State

**Files:**
- Create: `src/config.rs`
- Create: `src/http.rs`
- Create: `src/error.rs`

**Steps:**
1. Port Go defaults exactly: host `127.0.0.1`, port `18181`, data dir `.data`, db paths, timeout defaults, local auth default, body logging defaults, archive defaults.
2. Parse `config.toml` fields including legacy `allow_unauthenticated_local`.
3. Apply `-data-dir` and `-addr` overrides with the same path and address semantics.
4. Implement health/runtime routes: `/`, `/healthz`, `/readyz`, `/api/v1/runtime/state`.
5. Test JSON shapes for health and runtime state.

### Task 3: SQLite Schema, Repositories, And Seeds

**Files:**
- Create: `src/model.rs`
- Create: `src/db.rs`
- Create: `src/repository.rs`

**Steps:**
1. Create main DB tables: `providers`, `provider_models`, `ingress_endpoints`, `routing_rules`, `api_keys`, `ui_preferences`.
2. Create separate traffic DB table: `traffic_records`.
3. Preserve Go table and column names so existing `.data/*.db` files remain readable.
4. Seed the three built-in endpoints every startup with IDs `endpoint-v1-messages`, `endpoint-v1-chat-completions`, and `endpoint-v1-responses`.
5. Test initialization creates separate main and traffic databases.

### Task 4: Admin API Equivalence

**Files:**
- Create: `src/admin.rs`
- Create: `src/auth.rs`
- Modify: `src/http.rs`

**Steps:**
1. Implement CORS, request ID, and admin auth behavior.
2. Preserve local bypass based on actual peer IP, not `Host`.
3. Implement provider, provider model, ingress endpoint, routing rule, API key, and traffic endpoints under `/api/v1`.
4. Preserve response wrapper: `{ "data": ... }` and `{ "error": { "code": "BAD_REQUEST", "message": "..." } }`.
5. Preserve pagination query names `page`, `page_size`, `pageSize`, default `20`, max `200`.
6. Test CRUD for providers, API key reveal/update, built-in endpoint delete rejection, and traffic list/clear.

### Task 5: Route Resolver And Request Tracking

**Files:**
- Create: `src/routing.rs`
- Create: `src/tracker.rs`

**Steps:**
1. Port direct route behavior for `provider/model`.
2. Port enabled rule matching by protocol, glob pattern, and ascending priority.
3. Preserve error messages used by the Go tests.
4. Track active routing rules and reject modifying/deleting active rules.
5. Test route resolver cases from `internal/service/route_resolver_test.go`.

### Task 6: Protocol Conversion

**Files:**
- Create: `src/protocol/mod.rs`
- Create: `src/protocol/types.rs`
- Create: `src/protocol/converter.rs`
- Create: `src/protocol/sse.rs`

**Steps:**
1. Port request conversion matrix from `internal/utils/ai_llm_proxy/README.md`.
2. Preserve pass-through behavior when downstream and upstream protocols match, then rewrite top-level `model`.
3. Preserve unsupported request direction errors for Anthropic -> Chat and Responses -> Chat.
4. Port non-stream response conversion for Anthropic, Chat Completions, and Responses.
5. Port SSE conversion enough to preserve text deltas, usage, finish reason, terminal-event early return, and `[DONE]`.
6. Test the converter scenarios covered by `converter_test.go`.

### Task 7: Proxy Service

**Files:**
- Create: `src/proxy.rs`
- Modify: `src/http.rs`

**Steps:**
1. Implement fixed proxy routes `/v1/messages`, `/v1/chat/completions`, `/v1/responses`.
2. Implement dynamic endpoint matching from enabled ingress endpoints.
3. Preserve proxy auth, API key extraction, upstream URL joining, upstream auth headers, safe response header copying, stream headers, and upstream non-2xx error handling.
4. Record traffic with independent context and configured request body preview behavior.
5. Implement stream preflight rejection for empty stream and error events.
6. Test forwarding, model rewrite, error handling, stream conversion, and traffic recording.

### Task 8: Final Verification

**Files:**
- Modify as needed only for issues found during verification.

**Steps:**
1. Run `cargo test`.
2. Run `.\build.ps1`.
3. Start `build/bridge.exe -data-dir <temp> -addr 127.0.0.1:<free-port>`.
4. Verify `/healthz` and `/api/v1/runtime/state`.
5. Confirm `build-all.ps1 -SkipTests` still picks up `icoo_llm_bridge\build\bridge.exe` for the desktop build.
