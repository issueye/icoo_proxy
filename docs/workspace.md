# Go Workspace Layout

`icoo_proxy` is a multi-module monorepo managed by Go workspaces (`go.work`).

## Modules

| Directory | Module path | Role |
|-----------|-------------|------|
| `common/` | `github.com/issueye/icoo_proxy/common` | Shared pure libraries |
| `bridge/` | `github.com/issueye/icoo_proxy/bridge` | LLM bridge HTTP service (Gin/GORM host) |
| `desktop/` | `github.com/issueye/icoo_proxy/desktop` | Wails + Vue desktop console |
| `plugins/mock/` | `github.com/issueye/icoo_proxy/plugins/mock` | Mock process plugin (SDK sample) |
| `plugins/grokbuild (RunPlugin + PrepareHandshake)/` | `github.com/issueye/icoo_proxy/plugins/grokbuild` | Grok Build / SuperGrok process plugin |

```text
icoo_proxy/
├── go.work
├── common/
│   ├── constants/           # protocol / vendor
│   ├── domain/              # route, usage, provider snapshot
│   ├── idgen/
│   ├── view/                # API envelope DTOs
│   ├── ai_llm_proxy/        # protocol converter matrix
│   └── pluginipc/           # process-plugin IPC + Client/Server SDK
├── bridge/
│   ├── cmd/bridge/
│   └── internal/            # app, config, controller, service, entity, repo, pluginhost…
├── desktop/
└── plugins/
    ├── mock/                # SDK sample plugin (RunPlugin)
    └── grokbuild/
```

## Layering

```text
                    ┌──────────── desktop ────────────┐
                    │  spawn bridge; manage UX        │
                    └───────────────┬─────────────────┘
                                    │ HTTP / child process
┌──────────── plugins/* ────────────┤
│  import common/pluginipc only     │
└───────────────┬───────────────────┘
                │ IPC
┌───────────────▼───────────────────┐
│              bridge               │
│  Gin HTTP · GORM · plugin host    │
│  imports common/* freely          │
└───────────────┬───────────────────┘
                │
┌───────────────▼───────────────────┐
│              common               │
│  constants · domain · idgen ·     │
│  view · ai_llm_proxy · pluginipc  │
└───────────────────────────────────┘
```

### Stays in bridge (not common)

- `internal/app` composition root
- `internal/config` bridge TOML + plugins host config
- `internal/controller` / `middleware` / `router` (Gin)
- `internal/model/entity` GORM entities
- `internal/repository` SQLite
- `internal/service` business orchestration
- `internal/pluginhost` process lifecycle (Job Object / PGID)

## Commands

```powershell
# from repo root
go test ./common/...
go test ./bridge/...
go test ./desktop
go test ./plugins/mock/...
go test ./plugins/grokbuild/...

cd bridge;  .\build.ps1
cd desktop; .\build.ps1 -BridgePath ..\bridge\build\bridge.exe

# One-shot: bridge + desktop + plugins (plugin-grokbuild, mockplugin)
.\build-all.ps1
# .\build-all.ps1 -SkipPlugins
```

## Adding shared code

1. Prefer `common/<pkg>` for anything pure or reused by plugins.
2. Run `go test ./common/...` and `go test ./bridge/...`.
3. Update `common/README.md` package table.

## Migration history

| Old path | New path |
|----------|----------|
| `icoo_llm_bridge/` | `bridge/` |
| `icoo_desktop/` | `desktop/` |
| `bridge/pkg/pluginipc` | `common/pluginipc` |
| `bridge/internal/constants` | `common/constants` |
| `bridge/internal/model/domain` | `common/domain` |
| `bridge/internal/utils/idgen` | `common/idgen` |
| `bridge/internal/view` | `common/view` |
| `bridge/internal/utils/ai_llm_proxy` | `common/ai_llm_proxy` |
