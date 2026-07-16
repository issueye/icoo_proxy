# icoo_proxy Development Plan

> Date: 2026-07-16
> Baseline: `main` at `578d125`
> Goal: strengthen protocol correctness, cancellation behavior, frontend quality gates, and release governance without changing the local-first product model.

## Current baseline

- Bridge and desktop Go test suites pass.
- `go vet ./...` passes for `icoo_llm_bridge`.
- Browser-origin filtering, provider-secret masking, bounded request bodies, stream response-header timeout, traffic queue accounting, and 499 cancellation classification are implemented.
- The protocol converter test coverage is about 44%; frontend source has no automated test command.
- The older protocol plan contains stale assumptions: Chat to Anthropic request conversion already exists, and model rewriting already targets the top-level `model` property.

## Engineering rules

1. Preserve behavior unless a task defines the intended protocol change and a regression test proves it.
2. Keep converter code independent from Gin, GORM, repositories, and the application container.
3. Add focused tests for every protocol or lifecycle change before considering it complete.
4. Do not infer token accounting semantics. Capture representative upstream payloads and define the expected cross-protocol contract first.
5. Keep generated frontend output out of hand edits; regenerate it only through the frontend build.

## Phase 1: correctness and testability

### P1. Preserve downstream stream intent

**Owner boundary:** `internal/utils/ai_llm_proxy/chatcompletions_to_responses.go` and a dedicated regression test file.

- Map the Chat Completions `stream` value into the Responses request instead of forcing `true`.
- Cover omitted, false, and true stream values.
- Verify non-stream requests continue through the non-stream response path.

**Acceptance:** focused converter tests and the full Bridge test suite pass.

### P2. Propagate cancellation through stream conversion

**Owner boundary:** converter adapter types, stream scanning/conversion functions, and dedicated cancellation tests.

- Add `context.Context` to `StreamInput`.
- Stop same-protocol copy and cross-protocol SSE scanning promptly when the context is canceled.
- Ensure the proxy service passes the downstream request context into the converter.
- Preserve terminal-event fast exit and existing 499 traffic classification.

**Acceptance:** deterministic cancellation tests prove the reader stops promptly; service and converter tests pass under the race detector.

### P3. Establish a frontend test baseline

**Owner boundary:** frontend package configuration and new test-only files.

- Add a unit-test runner compatible with Vue 3 and Vite 5.
- Add deterministic tests for pure client-side normalization or store behavior without requiring Wails.
- Add a `test` script suitable for CI.

**Acceptance:** `npm test` and `npm run build` pass.

## Phase 2: protocol contract hardening

### P4. Define usage and cache-token semantics

- Collect representative OpenAI Responses, Chat Completions, and Anthropic usage payloads.
- Decide whether Anthropic `input_tokens` represents uncached input when `cache_read_input_tokens` is present.
- Add fixtures for cached input, missing usage, nested error usage, and incomplete responses.
- Only then change `ExtractUsage` or Responses-to-Anthropic mapping.

**Acceptance:** a documented mapping table and fixture-driven tests cover all supported protocol pairs.

### P5. Complete streaming tool-call conversion

- Implement Chat SSE tool-call delta accumulation and conversion to Responses and Anthropic events.
- Cover parallel tool calls, fragmented JSON arguments, finish reasons, and usage-only terminal chunks.
- Remove the documented lossy fallback only after all cases pass.

**Acceptance:** tool-call streams round-trip through every advertised upstream/downstream pair without losing call IDs, names, or arguments.

### P6. Verify the complete protocol matrix

- Maintain explicit request, non-stream response, and stream-response matrices.
- Test supported cells and stable errors for unsupported cells.
- Reconcile root README claims with the tested matrix.

**Acceptance:** every documented matrix cell has a test or is explicitly marked unsupported.

## Phase 3: frontend and API maintainability

### P7. Split the frontend API adapter

- Separate transport setup, DTO normalization, Wails configuration, and resource-specific clients.
- Keep stores responsible for UI state rather than HTTP response-shape repair.
- Introduce shared error normalization and typed contracts, preferably through TypeScript or generated definitions.

**Acceptance:** no single API module owns unrelated providers, traffic, settings, routing, and model-catalog behavior; tests cover public client functions.

### P8. Add UI quality gates

- Add linting, formatting checks, component tests, and critical workflow tests.
- Cover provider creation, route selection, settings restart behavior, and traffic cancellation display.

**Acceptance:** all checks run non-interactively and fail on regressions.

## Phase 4: release governance

### P9. Unify versioning and artifacts

- Use one version source for Bridge and desktop builds.
- Verify the displayed version matches both binaries.
- Stop committing hashed frontend assets unless repository policy explicitly requires them; otherwise add a build-time freshness check.

### P10. Add delivery metadata and automation

- Add CI for Go tests, vet, race-focused packages, frontend tests, and frontend build.
- Add LICENSE, CHANGELOG, and an OpenAPI description for management endpoints.
- Add a Windows packaging smoke test for the two-executable distribution.

**Acceptance:** a clean checkout can produce verified release artifacts using documented commands.

## Parallel execution map

| Worker | Initial task | Files allowed to overlap |
| --- | --- | --- |
| protocol-stream | P1 | None outside its converter file and dedicated test file |
| stream-cancel | P2 | Adapter/converter/service stream call sites and dedicated tests |
| frontend-tests | P3 | `frontend/package*.json` and new test files |
| integrator | Review and verification | May resolve conflicts after workers finish |

## Execution status

Completed on 2026-07-16:

- [x] P1: Chat to Responses preserves omitted, false, and true stream intent, including a proxy-service regression test for the non-stream JSON path.
- [x] P2: stream conversion accepts downstream context, cancels same- and cross-protocol conversion, and closes closable upstream readers.
- [x] P3: Vitest baseline added with four Pinia model-catalog store tests.
- [x] P4 deterministic scope: usage extraction is protocol-specific, supports top-level and `response.usage` envelopes, and no longer double-counts mixed protocol fields. Cached-token billing semantics are documented separately and intentionally unchanged pending accessible vendor evidence.
- [x] P5: Chat SSE tool-call fragments round-trip to Responses function calls and Anthropic tool-use blocks, including interleaved parallel calls. The Anthropic path buffers tool fragments until completion to preserve block ordering.
- [x] P6: table-driven tests cover all 27 request, non-stream response, and SSE protocol-matrix cells; unsupported request cells return exact stable errors.
- [x] P7: the frontend API adapter is split into transport, normalization, providers/models, routing/resources, runtime/traffic, and Wails settings modules while preserving the original 41-function public barrel.
- [x] P8: ESLint, progressive Prettier checks, Vitest store tests, and Vue component tests run non-interactively in CI.
- [x] P9: root `VERSION` is the single release source; both binaries receive `2.0.1` through linker injection, and the packaged Bridge reports it through `/api/v1/runtime/state`.
- [x] P10 engineering scope: CI, CHANGELOG, OpenAPI validation/route coverage, complete Windows packaging, and PE artifact smoke tests are implemented.
- [x] Full Bridge tests, focused race tests, Bridge vet, desktop tests, frontend tests, and frontend production build pass.

Owner decisions completed:

- [x] Apache License 2.0 selected and added to the repository.

External evidence still required:

- Re-verify cached-token billing parity when official vendor documentation or real API fixtures are accessible; the current network returned regional HTTP 403 responses.

## Verification commands

```powershell
cd icoo_llm_bridge
go test ./...
go test -race ./internal/service ./internal/utils/ai_llm_proxy
go vet ./...

cd ..\icoo_desktop
go test .

cd frontend
npm test
npm run build
```
