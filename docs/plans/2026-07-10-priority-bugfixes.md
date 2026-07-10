# Priority Bug Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the confirmed local-browser access-control issue, provider secret exposure/update semantics, broken provider checks/model discovery, and the traffic writer counter race.

**Architecture:** Keep the existing Gin/service/repository layering. Enforce trusted-origin checks in global middleware, expose provider DTOs instead of persistence entities, preserve secrets server-side during partial updates, reuse the provider chat service for a minimal health probe, and make traffic queue accounting happen before a worker can call `Done`.

**Tech Stack:** Go 1.23, Gin, GORM/SQLite, Vue 3, Axios, Go testing.

---

### Task 1: Lock down browser-origin access

**Files:**
- Modify: `icoo_llm_bridge/internal/middleware/middleware.go`
- Modify: `icoo_llm_bridge/internal/middleware/middleware_test.go`
- Modify: `icoo_llm_bridge/internal/router/router.go`

**Steps:**
1. Add failing middleware tests for a hostile `Origin`, a Wails origin, a loopback development origin, and requests without an `Origin`.
2. Implement an origin allowlist that permits Wails schemes and loopback hosts, echoes the accepted origin, adds `Vary: Origin`, and rejects every other browser origin with 403.
3. Disable Gin's default trust of all proxies with `SetTrustedProxies(nil)`.
4. Run `go test ./internal/middleware ./internal/router`.

### Task 2: Stop returning provider secrets and preserve update metadata

**Files:**
- Modify: `icoo_llm_bridge/internal/service/admin_models.go`
- Modify: `icoo_llm_bridge/internal/service/admin_services.go`
- Modify: `icoo_llm_bridge/internal/app/container_test.go`
- Modify: `icoo_desktop/frontend/src/lib/apiClient.js`

**Steps:**
1. Add tests proving provider list/save responses do not contain `APIKeyCipher` or the raw secret.
2. Add a provider response DTO with a masked key preview.
3. On updates, load the existing provider, retain `CreatedAt`, and retain the stored key when the request key is blank.
4. Remove the frontend workaround that reads the raw provider secret before updating.
5. Run backend integration tests and the frontend build.

### Task 3: Restore provider health checks and model discovery

**Files:**
- Modify: `icoo_llm_bridge/internal/router/router.go`
- Modify: `icoo_llm_bridge/internal/controller/admin_resource_controller.go`
- Modify: `icoo_llm_bridge/internal/service/admin_models.go`
- Modify: `icoo_llm_bridge/internal/service/admin_services.go`
- Modify: `icoo_llm_bridge/internal/service/provider_chat_service.go`
- Modify: `icoo_llm_bridge/internal/service/services.go`
- Modify: `icoo_desktop/frontend/src/lib/apiClient.js`
- Add/modify tests under `icoo_llm_bridge/internal/service` and `icoo_llm_bridge/internal/app`.

**Steps:**
1. Add failing tests for `POST /api/v1/providers/:id/check` and model-fetch response handling.
2. Extend the provider chat service with a minimal non-stream health probe using the first enabled model.
3. Register the missing route and return the health shape expected by the frontend.
4. Return the already-unwrapped model array directly in Axios code.
5. Run focused backend tests and `npm run build`.

### Task 4: Fix traffic queue accounting

**Files:**
- Modify: `icoo_llm_bridge/internal/service/proxy_service.go`
- Modify: `icoo_llm_bridge/internal/service/proxy_service_test.go`

**Steps:**
1. Add a fast in-memory traffic writer stress test.
2. Increment the wait group before publishing to the channel and undo the increment if the queue is full.
3. Run the stress test repeatedly and run `go test -race ./...`.

### Task 5: Full verification

**Steps:**
1. Run `gofmt` on changed Go files.
2. Run `go test ./...` and `go vet ./...` in `icoo_llm_bridge`.
3. Run `go test -race ./...` in `icoo_llm_bridge`.
4. Run `npm run build` in `icoo_desktop/frontend`.
5. Run desktop tests and report any pre-existing environment-dependent failure separately.
6. Confirm `git diff --check` and verify the two original local proxy-cancellation changes remain intact.

### Task 6: Make server lifecycle ownership-safe

**Files:**
- Modify: `icoo_llm_bridge/internal/app/container.go`
- Modify: `icoo_llm_bridge/internal/app/container_test.go`
- Modify: `icoo_desktop/app.go`
- Modify: `icoo_desktop/app_test.go`

**Steps:**
1. Add a test proving `Container.Start` returns an address-in-use error synchronously.
2. Bind the listener before starting the serve goroutine.
3. Change desktop stop behavior to refuse terminating a healthy process it did not start.
4. Make saving desktop server settings restart an owned Bridge process.
5. Run bridge app tests and desktop tests.

### Task 7: Bound proxy request memory

**Files:**
- Modify: `icoo_llm_bridge/internal/config/config.go`
- Modify: `icoo_llm_bridge/internal/config/loader.go`
- Modify: `icoo_llm_bridge/configs/config.example.toml`
- Modify: `icoo_llm_bridge/internal/service/proxy_service.go`
- Modify: `icoo_llm_bridge/internal/service/proxy_service_test.go`

**Steps:**
1. Add a test that sends a body larger than the configured limit and expects HTTP 413 without contacting upstream.
2. Add a configurable `max_request_body_bytes` with a 64 MiB default.
3. Wrap request bodies with `http.MaxBytesReader` and classify `MaxBytesError` as 413.
4. Run focused proxy tests.

### Task 8: Add streaming response-header timeout

**Files:**
- Modify: `icoo_llm_bridge/internal/service/proxy_service.go`
- Modify: `icoo_llm_bridge/internal/service/http_proxy.go`
- Modify: `icoo_llm_bridge/internal/service/proxy_service_test.go`

**Steps:**
1. Add a test where upstream delays response headers beyond `stream_preflight_timeout` and expect HTTP 504.
2. Configure streaming transports with `ResponseHeaderTimeout` while keeping total client timeout disabled.
3. Apply the same response-header timeout when a provider proxy URL is configured.
4. Run streaming tests and race detection.

### Task 9: Apply desktop settings consistently

**Files:**
- Modify: `icoo_llm_bridge/internal/config/loader.go`
- Modify: `icoo_llm_bridge/internal/config/loader_test.go`
- Modify: `icoo_desktop/app.go`
- Modify: `icoo_desktop/app_test.go`

**Steps:**
1. Add config tests for the desktop's legacy flat chain-log TOML fields.
2. Accept both flat fields and the canonical `[log]` block.
3. Restart a desktop-owned Bridge after saving settings.
4. Run backend config tests, desktop tests, and the frontend build.

### Task 10: Classify downstream cancellations across the full proxy lifecycle

**Files:**
- Modify: `icoo_llm_bridge/internal/service/proxy_service.go`
- Modify: `icoo_llm_bridge/internal/service/proxy_service_test.go`
- Modify: `icoo_desktop/frontend/src/lib/apiClient.js`
- Modify: `icoo_desktop/frontend/src/stores/traffic.js`
- Modify: `icoo_desktop/frontend/src/views/TrafficView.vue`

**Steps:**
1. Add deterministic tests for cancellation during SSE preflight and after the first stream event.
2. Reuse cancellation/timeout classification when reading upstream bodies, preflighting streams, converting streams, and writing downstream responses.
3. Record downstream cancellation as 499 with `client canceled request`, never 502.
4. Report 499 separately from server errors in the traffic dashboard.
5. Run proxy tests repeatedly, race detection, and the frontend build.
