# Go/Rust Parity Checklist

This checklist tracks whether `icoo_llm_bridge_r` is ready to gray-replace the Go service.

Status values:
- `done`: implemented and manually reviewed
- `covered by test`: implemented and covered by automated tests
- `partially covered`: implemented but missing important coverage
- `not implemented`: missing
- `accepted gap`: explicitly out of scope for this gray-release phase

Risk values:
- `P0`: replacement blocker
- `P1`: gray-release risk
- `P2`: follow-up

## Runtime And Configuration

| Area | Status | Risk | Notes |
| --- | --- | --- | --- |
| Separate Rust project under `icoo_llm_bridge_r` | done | P2 | Go version remains in `icoo_llm_bridge`. |
| Config file loading | covered by test | P1 | Supports host, port, timeouts, auth bypass, data/db paths, log/archive settings. |
| CLI overrides | done | P1 | Supports `--config`, `--data-dir`, `--addr`. |
| Health endpoints | covered by test | P1 | `/healthz`, `/readyz`, runtime state smoke checked. |
| Release build output | covered by test | P1 | `build.ps1 -SkipTests` writes `build/bridge.exe`. |

## Database

| Area | Status | Risk | Notes |
| --- | --- | --- | --- |
| Main DB tables | covered by test | P1 | Providers, models, endpoints, rules, API keys, UI preferences. |
| Separate traffic DB | covered by test | P1 | Matches current Go layout. |
| Current schema compatibility | partially covered | P1 | Current schema works; diagnostics planned in this phase. |
| Older legacy DB migration | accepted gap | P2 | Full migration intentionally deferred; read compatibility exists for missing `routing_rules.upstream_protocol` and `api_keys.secret_cipher`. |

## Admin API

| Area | Status | Risk | Notes |
| --- | --- | --- | --- |
| Providers CRUD | covered by test | P1 | Main list/create/update/delete paths implemented. |
| Provider models CRUD | partially covered | P1 | Implemented; broader parity coverage remains useful. |
| Ingress endpoints CRUD | covered by test | P1 | Built-in endpoint delete protection covered. |
| Routing rules CRUD | partially covered | P1 | Implemented; active-request protection covered in service logic. |
| API keys | covered by test | P1 | Create/list/delete/reveal/update preservation covered. |
| Traffic list/clear | covered by test | P1 | Uses separate traffic DB. |
| Pagination aliases | covered by test | P2 | `page_size` and `pageSize` covered. |

## Auth And Routing

| Area | Status | Risk | Notes |
| --- | --- | --- | --- |
| Local auth bypass | covered by test | P1 | Same intended behavior as Go config. |
| API key hash verification | covered by test | P1 | Supports proxy/admin scopes. |
| Direct provider/model routing | covered by test | P1 | `provider/model` style route works. |
| Routing-rule resolution | partially covered | P1 | Priority and model matching covered at unit level; gray validation should confirm real data. |
| Route metadata in traffic records | covered by test | P1 | Rule id/name/source recorded. |

## Proxy Behavior

| Area | Status | Risk | Notes |
| --- | --- | --- | --- |
| Fixed proxy endpoints | covered by test | P1 | `/v1/messages`, `/v1/chat/completions`, `/v1/responses`. |
| Dynamic proxy endpoints | covered by test | P1 | Enabled endpoint paths route through fallback. |
| Model rewrite | covered by test | P1 | Request model rewritten to routed upstream model. |
| Upstream auth/header construction | partially covered | P1 | Implemented; real-upstream verification required. |
| Traffic recording on success/failure | covered by test | P1 | Records status, usage, route data, body preview. |
| Upstream error propagation | partially covered | P1 | Implemented; real-upstream verification required. |

## Protocol Conversion

| Area | Status | Risk | Notes |
| --- | --- | --- | --- |
| Chat request to Responses | covered by test | P1 | Includes multimodal, tools, reasoning. |
| Chat request to Anthropic | partially covered | P1 | Basic conversion implemented. |
| Anthropic request to Responses | partially covered | P1 | Basic conversion implemented. |
| Responses request to Anthropic | covered by test | P1 | Includes multimodal, tools, reasoning. |
| Responses response to Chat | covered by test | P1 | Includes text, reasoning, function calls. |
| Chat response to Responses | covered by test | P1 | Includes tool calls. |
| Chat response to Anthropic | covered by test | P1 | Includes tool calls. |
| Responses response to Anthropic | covered by test | P1 | Includes function calls. |
| Anthropic response to Chat/Responses | partially covered | P1 | Basic text conversion implemented. |

## Streaming

| Area | Status | Risk | Notes |
| --- | --- | --- | --- |
| Same-protocol SSE pass-through | covered by test | P1 | Low-latency pass-through implemented. |
| Responses stream to Chat text | covered by test | P1 | Terminal handling covered. |
| Responses stream to Chat tool calls | covered by test | P1 | Converts `response.output_item.added` and argument deltas to Chat `tool_calls`. |
| Responses stream to Chat reasoning deltas | covered by test | P1 | Converts reasoning summary deltas to Chat `reasoning_content`. |
| Chat stream to Responses text | covered by test | P1 | Terminal handling covered. |
| Chat stream to Responses tool calls | covered by test | P1 | Converts Chat `delta.tool_calls` to Responses function-call stream events. |
| Chat stream to Anthropic text | covered by test | P1 | Emits `message_stop`. |
| Cross-protocol streaming low latency | partially covered | P1 | Some paths still buffer through Chat conversion. |

## Gray Replacement Gates

| Gate | Status | Risk | Notes |
| --- | --- | --- | --- |
| Full Rust tests pass | covered by test | P1 | Run `cargo test`. |
| Rust release build passes | covered by test | P1 | Run `.\build.ps1 -SkipTests`. |
| Executable smoke test passes | covered by test | P1 | Health/runtime checked. |
| Real-upstream verification | done | P1 | No mock; documented procedure and script exist. Must be run with real configured providers before traffic switch. |
| Runtime DB diagnostics | covered by test | P1 | Runtime state reports current schema OK and accepted old-schema warnings; preflight script checks copied data directories. |
| Gray replacement runbook | done | P1 | Side-by-side run, verification, traffic switch, and rollback documented. |
