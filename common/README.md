# icoo/common

Shared Go libraries for the `icoo_proxy` monorepo. Modules that depend on this package: `bridge`, `plugins/*`. Prefer pure logic and protocol contracts here; keep Gin/GORM/process-host wiring in `bridge`.

## Packages

| Package | Import path suffix | Responsibility |
|---------|-------------------|----------------|
| `constants` | `.../common/constants` | Protocol & vendor enums |
| `domain` | `.../common/domain` | Persistence-free domain types (route, usage, snapshots) |
| `idgen` | `.../common/idgen` | ID helpers |
| `view` | `.../common/view` | Admin API response / pagination DTOs |
| `ai_llm_proxy` | `.../common/ai_llm_proxy` | Multi-protocol request/response/stream conversion |
| `pluginipc` | `.../common/pluginipc` | Process-plugin IPC (pipe/UDS + JSON-RPC) **and Client/Server SDK** (`Connect`, `RunPlugin`, helpers) |

## Rules

1. **No Gin, no GORM, no SQLite** in `common` (except future intentional shared adapters).
2. Plugins may import `constants`, `pluginipc`, and (if needed) `ai_llm_proxy` / `domain`.
3. Plugins **must not** import `bridge/internal/...`.
4. Prefer adding new pure helpers here over duplicating them in bridge or plugins.
5. Plugin IPC usage guide: [`docs/plugin-ipc-sdk.md`](../docs/plugin-ipc-sdk.md); wire contract: [`docs/plugin-ipc-contract.md`](../docs/plugin-ipc-contract.md).

## Local consume

```go
// bridge/go.mod or plugins/*/go.mod
require github.com/issueye/icoo_proxy/common v0.0.0

replace github.com/issueye/icoo_proxy/common => ../common // or ../../common
```
