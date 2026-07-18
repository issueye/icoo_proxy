# Go Workspace Layout

`icoo_proxy` 是由 Go workspace（`go.work`）管理的多模块 monorepo。

- Go：`1.23`
- 当前版本源：根目录 [`VERSION`](../VERSION)（`2.0.1`）
- 文档索引：[`docs/README.md`](./README.md)

## Modules

| Directory | Module path | Role |
|-----------|-------------|------|
| `common/` | `github.com/issueye/icoo_proxy/common` | 共享纯库（协议、领域、IPC SDK） |
| `bridge/` | `github.com/issueye/icoo_proxy/bridge` | LLM 网关宿主（Gin + GORM/SQLite + plugin host） |
| `desktop/` | `github.com/issueye/icoo_proxy/desktop` | Wails + Vue 桌面管理端（不链接 bridge/common） |
| `plugins/mock/` | `github.com/issueye/icoo_proxy/plugins/mock` | 进程插件 SDK 样例 |
| `plugins/grokbuild/` | `github.com/issueye/icoo_proxy/plugins/grokbuild` | Grok Build / SuperGrok 进程插件 |

```text
icoo_proxy/
├── go.work
├── VERSION / CHANGELOG.md / LICENSE
├── build-all.ps1
├── docs/                    # 文档索引、OpenAPI、IPC 契约、设计/计划
├── common/
│   ├── constants/           # protocol / vendor
│   ├── domain/              # route, usage, provider snapshot
│   ├── idgen/
│   ├── view/                # API envelope DTOs
│   ├── ai_llm_proxy/        # 协议转换矩阵
│   └── pluginipc/           # 进程插件 IPC + Client/Server SDK
├── bridge/
│   ├── cmd/bridge/
│   ├── configs/
│   └── internal/            # app, config, controller, service, entity, repo, pluginhost…
├── desktop/
│   ├── frontend/            # Vue 3 + Vite
│   └── *.go                 # Wails 宿主、托盘、bridge 子进程
├── plugins/
│   ├── mock/                # SDK sample (RunPlugin)
│   └── grokbuild/           # RunPlugin + PrepareHandshake + admin UI
└── icoo_proxy/              # 打包输出目录（非源码模块）
```

## Layering

```text
                    ┌──────────── desktop ────────────┐
                    │  spawn bridge; manage UX        │
                    │  NO import bridge/common        │
                    └───────────────┬─────────────────┘
                                    │ HTTP / child process
┌──────────── plugins/* ────────────┤
│  import common/* only             │
│  MUST NOT import bridge/internal  │
└───────────────┬───────────────────┘
                │ IPC (Named Pipe / UDS + JSON-RPC)
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

### 留在 bridge（不得下沉到 common）

- `internal/app` 组合根 / DI 容器
- `internal/config` Bridge TOML 与 plugins host 配置
- `internal/controller` / `middleware` / `router`（Gin）
- `internal/model/entity` GORM 实体
- `internal/repository` SQLite 访问
- `internal/service` 业务编排（代理、路由、流量、管理）
- `internal/pluginhost` 进程生命周期（Job Object / PGID）

### Desktop 边界

- 通过 HTTP 调用 Bridge 管理 API（默认 `http://127.0.0.1:18181`）。
- 负责启动/停止同目录或配置路径下的 `bridge.exe`。
- 插件扩展页经 Bridge 反代嵌入（`/api/v1/plugins/:id/ui/*`），不直连插件随机端口。

### Plugin 边界

- 每个插件独立 Go module，只依赖 `common`。
- 分发包布局：`plugins/<id>/info.toml` + 可执行文件。
- 运行时状态：`data_dir/plugins/<id>/`（凭据、registry 等）。
- 契约：[`plugin-ipc-contract.md`](./plugin-ipc-contract.md)、SDK：[`plugin-ipc-sdk.md`](./plugin-ipc-sdk.md)。

## 数据与配置（Bridge）

| 项 | 默认 |
|----|------|
| 监听 | `127.0.0.1:18181` |
| 主库 | `.data/icoo_llm_bridge.db` |
| 流量库 | `.data/icoo_llm_bridge_traffic.db`（WAL） |
| 链路日志 | `.data/bridge-chain.log` |
| 配置样例 | `bridge/configs/config.example.toml` |

## Commands

```powershell
# from repo root (go.work)
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
# .\build-all.ps1 -SkipTests
```

## Adding shared code

1. 纯逻辑或插件也需要的能力优先放 `common/<pkg>`。
2. 跑 `go test ./common/...` 与 `go test ./bridge/...`（插件相关再跑 `plugins/*`）。
3. 更新 [`common/README.md`](../common/README.md) 包表与本文件（若边界变化）。

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
