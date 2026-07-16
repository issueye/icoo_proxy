# Protocol Converter Bugfix Plan

> **Scope:** `icoo_llm_bridge/internal/utils/ai_llm_proxy`
> **Goal:** Repair confirmed protocol-conversion bugs and add regression tests that cover request asymmetry, usage mapping errors, stream semantics, and request-body model rewriting.
> **Approach:** Fix the minimum set of defects first, then add targeted tests. Do not refactor the whole converter unless required by a follow-on task.

---

## Task 1: Fix Chat → Responses request forcing stream=true

**Bug:** `chatcompletions_to_responses.go` always sets `Stream: true`, ignoring downstream intent. This breaks non-streaming Chat callers that expect a single non-streamed Responses object.

**Files:**
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/chatcompletions_to_responses.go`
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/converter_test.go`

**Steps:**
1. Add a failing test proving a non-stream Chat request converts to a non-stream Responses request.
2. Change `ChatCompletionsToResponses` to read the original Chat `Stream` value instead of hardcoding `true`.
3. Keep existing tool/system/user mapping behavior unchanged.
4. Run `go test ./internal/utils/ai_llm_proxy`.

---

## Task 2: Remove request/response path asymmetry for Chat → Anthropic and Responses → Chat

**Bug:** `ConvertRequest` does not support `Chat → Anthropic` and `Responses → Chat`, but `ConvertResponse` does. This creates a misleading surface and can cause unexpected 4xx paths.

**Files:**
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/converter.go`
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/converter_test.go`

**Steps:**
1. Decide scope for this plan:
   - preferred short-term: make unsupported request paths fail with an explicit stable error.
   - optional follow-on: add real request conversion if routing requires it.
2. Add a failing test proving the two unsupported request paths return a known error instead of ambiguous behavior.
3. Implement explicit unsupported-request handling in `ConvertRequest`.
4. Run `go test ./internal/utils/ai_llm_proxy`.

**Decision record for this plan:** This task only adds explicit failure semantics. Real `Chat → Anthropic` and `Responses → Chat` request conversion is tracked separately.

---

## Task 3: Scope `rewriteJSONModel` to the top-level model field only

**Bug:** `rewriteJSONModel` rewrites every `model` key in the request body, even nested ones. This is unsafe for any future nested object or internal metadata.

**Files:**
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/converter.go`
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/converter_test.go`

**Steps:**
1. Add a test body containing nested `model` fields and assert they remain unchanged after rewrite.
2. Change `rewriteJSONModel` to rewrite only the top-level `model` key.
3. Run `go test ./internal/utils/ai_llm_proxy`.

---

## Task 4: Fix Responses → Anthropic usage mapping

**Bug:** `responses_to_anthropic.go` computes `input_tokens = max(0, input_tokens - cached_tokens)`. This is semantically wrong and can silently zero out real prompt usage when cached token counts are high. Cache-specific fields are also not mapped.

**Files:**
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/responses_to_anthropic.go`
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/responses_to_anthropic_response.go`
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/converter_test.go`

**Steps:**
1. Add a failing test with `input_tokens=100, cached_tokens=150` showing wrong negative subtraction.
2. Map usage as:
   - `input_tokens` keeps the original `input_tokens`
   - `cache_creation_input_tokens` mapped from `cache_creation_input_tokens`
   - `cache_read_input_tokens` mapped from `cache_read_input_tokens`
   - `output_tokens` mapped from `output_tokens`
3. If an upstream field is missing, omit the Anthropic cache field rather than inventing values.
4. Run `go test ./internal/utils/ai_llm_proxy`.

---

## Task 5: Add robust error usage extraction

**Bug:** `extractUsage` only does a shallow parse of `payload["usage"]`. Many real-world error or edge-case payloads will silently produce empty usage.

**Files:**
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/converter.go`
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/converter_test.go`

**Steps:**
1. Add tests for missing `usage`, nested `usage`, and differently shaped usage objects.
2. Make `extractUsage` tolerant of common nested shapes and missing fields.
3. Return `nil` on unknown shapes instead of silently returning empty usage if the shape is unrecognizable; let callers decide fallback behavior.
4. Run `go test ./internal/utils/ai_llm_proxy`.

---

## Task 6: Clarify and harden stream conversion cancellation behavior

**Bug risk:** Stream converters stop reading after terminal events, but there is no explicit propagation of `ctx.Done()` from downstream disconnect to upstream cancelation. This can leave upstream work running after the client disconnects.

**Files:**
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/anthropic_to_responses_stream.go`
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/chatcompletions_stream.go`
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/converter_test.go`

**Steps:**
1. Add a test that cancels downstream context during streaming and asserts upstream body reads stop promptly.
2. Ensure every streaming converter checks context before blocking reads and exits cleanly on cancellation.
3. Keep terminal-event fast-exit behavior, but do not rely on it alone for cleanup.
4. Run `go test ./internal/utils/ai_llm_proxy`.

**Note for this plan:** Full 499 classification is owned by `proxy_service.go` and is already covered in the existing priority bugfix plan. This task only removes protocol-layer resource leakage.

---

## Task 7: Document stream tool-call limitations explicitly

**Status:** Not a silent crash, but a known semantic limitation that is easy to misuse.

**Files:**
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/chatcompletions_stream.go`
- Modify: `icoo_llm_bridge/internal/utils/ai_llm_proxy/README.md`

**Steps:**
1. Add a clear comment in `ChatChunkToResponsesEvents` stating that tool-call delta streaming is intentionally limited to text/usage/termination in this implementation.
2. Add a follow-up task note in README under stream limitations so future contributors do not accidentally claim full tool-call streaming support.
3. No behavioral code change in this task.

---

## Task 8: Full verification for protocol converter changes

**Steps:**
1. Run `gofmt` on changed files under `icoo_llm_bridge/internal/utils/ai_llm_proxy`.
2. Run `go test ./internal/utils/ai_llm_proxy`.
3. Run `go test -race ./internal/utils/ai_llm_proxy`.
4. Run `go vet ./internal/utils/ai_llm_proxy`.
5. Record any pre-existing environment-dependent failures separately.

---

## Execution order recommendation

1. Task 1
2. Task 3
3. Task 4
4. Task 5
5. Task 2
6. Task 6
7. Task 7
8. Task 8

Reasoning: fix caller-visible request behavior first, then safer mapping, then explicit unsupported-path handling, then streaming hygiene.
