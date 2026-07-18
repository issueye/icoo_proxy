# icoo_proxy

Local-first LLM API bridge and desktop console.

`icoo_proxy` exposes OpenAI Chat Completions, OpenAI Responses, and Anthropic Messages compatible endpoints, then routes each request to configured upstream providers or process plugins. The backend lives in `bridge` (`github.com/issueye/icoo_proxy/bridge`), the desktop app in `desktop` (`github.com/issueye/icoo_proxy/desktop`), shared libraries in `common` (`github.com/issueye/icoo_proxy/common`), and optional process plugins under `plugins/*`. The monorepo uses a Go workspace (`go.work`, Go 1.23).

- Chinese documentation: [README.cn.md](README.cn.md)
- Documentation index: [docs/README.md](docs/README.md)
- Workspace guide: [docs/workspace.md](docs/workspace.md)
- Management OpenAPI: [docs/openapi.yaml](docs/openapi.yaml)
- Plugin IPC contract / SDK: [docs/plugin-ipc-contract.md](docs/plugin-ipc-contract.md) · [docs/plugin-ipc-sdk.md](docs/plugin-ipc-sdk.md)
- Version: root [`VERSION`](VERSION) (currently **2.0.1**) · License: [Apache-2.0](LICENSE)

## Features

- Local HTTP bridge for:
  - `POST /v1/chat/completions`
  - `POST /v1/responses`
  - `POST /v1/messages`
  - `GET /v1/models`
- Provider, model, endpoint, routing-rule, API-key, traffic, and process-plugin management APIs.
- Protocol conversion across OpenAI Chat, OpenAI Responses, and Anthropic Messages.
- Streaming SSE support, including same-protocol low-latency pass-through.
- Process plugins over Named Pipe (Windows) / UDS (Unix) JSON-RPC (`common/pluginipc`).
- Desktop management console built with Wails v2 and Vue 3 (does **not** link `bridge`/`common`).
- Dual SQLite storage: main DB + traffic DB (WAL).
- Provider health checks, runtime database diagnostics, and plugin extension pages (iframe via bridge reverse-proxy).

## Repository Layout

```text
.
├── go.work                 # Go workspace root
├── VERSION / CHANGELOG.md  # Single release version source
├── common/                 # Shared module (ai_llm_proxy, pluginipc, …)
├── bridge/                 # Gateway host (Gin + GORM/SQLite)
├── desktop/                # Wails + Vue console
├── plugins/                # Process plugins (mock, grokbuild, …)
├── docs/                   # Index, OpenAPI, IPC contracts, design/plans
├── icoo_proxy/             # Packaged executable output (not a source module)
├── scripts/                # OpenAPI + package smoke tests
└── build-all.ps1           # One-shot package script
```

## Requirements

- Windows PowerShell (primary packaging path)
- Go 1.23+
- Node.js and npm
- Wails CLI for desktop builds

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## Quick Start

```powershell
# Build everything into .\icoo_proxy\
.\build-all.ps1

# Or build pieces separately
cd bridge;  .\build.ps1
cd desktop; .\build.ps1 -BridgePath ..\bridge\build\bridge.exe
```

Default bridge listen address: `127.0.0.1:18181`.

```powershell
Invoke-RestMethod http://127.0.0.1:18181/healthz
Invoke-RestMethod http://127.0.0.1:18181/readyz
Invoke-RestMethod http://127.0.0.1:18181/api/v1/runtime/state
```

By default, local loopback requests can use admin APIs without an API key. Configure API keys before exposing the bridge outside localhost.

## Build The Bridge

```powershell
cd bridge
.\build.ps1
# .\build.ps1 -SkipTests
```

Output: `bridge\build\bridge.exe`

Run:

```powershell
cd bridge
.\build\bridge.exe
```

Config sample: [`bridge/configs/config.example.toml`](bridge/configs/config.example.toml).

## Build The Desktop App

```powershell
cd desktop
.\build.ps1 -BridgePath ..\bridge\build\bridge.exe
```

Output:

```text
desktop\build\bin\icoo_desktop.exe
desktop\build\bin\bridge.exe
```

Frontend-only development:

```powershell
cd desktop\frontend
npm install
npm run dev
```

Desktop does not import `bridge` or `common`. It spawns `bridge.exe` as a child process and talks HTTP to the management API. See [desktop/README.md](desktop/README.md).

## Desktop Console

The desktop console is the recommended way to operate a local `icoo_proxy` instance. It can launch or restart the bundled bridge, show connection state, manage providers/routes/keys/traffic/settings, discover process plugins, and embed plugin extension pages.

![Gateway overview](images/index.png)

### Provider Management

![Provider management](images/provider.png)

Create upstream providers with protocol (`openai-responses`, `openai-chat`, `anthropic`, or plugin-backed vendors), base URL, credentials, and enabled models. Health check sends a minimal real upstream request using the first enabled model (consumes a small quota).

### Routing Rules

![Routing rules](images/rules.png)

Map each downstream protocol to an upstream provider and protocol. Resolution order:

1. Direct provider/model routing, for example `provider-name/model-name`.
2. Enabled routing rules ordered by priority.

### Ingress Endpoints

![Ingress endpoints](images/endpoint.png)

Default enabled paths include:

- `/v1/chat/completions` — OpenAI Chat compatible clients
- `/v1/messages` — Anthropic Messages compatible clients
- `/v1/responses` and `/responses` — OpenAI Responses compatible clients

### API Keys

![API keys](images/keys.png)

Clients authenticate with `Authorization: Bearer …` or `x-api-key`. Local trusted mode may allow unauthenticated admin access when bound to loopback and no keys are required by policy.

### Traffic Monitor

![Traffic monitor](images/traffic.png)

Inspect recent proxy requests, success/error counts, latency, tokens, protocols, hit rules, status codes (including client cancel `499`), and timings.

### Runtime Settings

![Runtime settings](images/settings.png)

Adjust console appearance and bridge runtime parameters (host, port, timeouts, default max tokens, chain-log options). Save and restart the owned bridge process to apply runtime changes.

## Process Plugins

Plugins are **separate processes**, not shared libraries:

- Discovery package: `plugins/<id>/info.toml` + executable (next to `bridge.exe` or cwd).
- Transport: Windows Named Pipe / Unix UDS; framing: length-prefix JSON-RPC 2.0 (+ optional raw body frames).
- Host owns lifecycle (spawn, heartbeat, auto-restart, Job Object / PGID kill-on-close).
- Plugins import `common` only; **must not** import `bridge/internal/...`.

| Plugin | Role |
|--------|------|
| [`plugins/mock`](plugins/mock/README.md) | Minimal SDK sample for integration tests |
| [`plugins/grokbuild`](plugins/grokbuild/README.md) | Optional Grok Build / SuperGrok adapter (default off; see disclaimer) |

Bridge config sketch:

```toml
[plugins.entries.grokbuild]
enabled = true
executable = "plugins/grokbuild/plugin-grokbuild.exe"
data_dir = ".data/plugins/grokbuild"
```

Full contract: [docs/plugin-ipc-contract.md](docs/plugin-ipc-contract.md).

## Protocol Conversion Matrix

Authoritative implementation: [`common/ai_llm_proxy`](common/ai_llm_proxy/README.md). Unsupported request directions return stable `not implemented` errors.

### Request (downstream → upstream)

| Downstream \\ Upstream | Anthropic | OpenAI Chat | OpenAI Responses |
| --- | --- | --- | --- |
| Anthropic | pass-through | **not implemented** | supported |
| OpenAI Chat | supported | pass-through | supported |
| OpenAI Responses | supported | **not implemented** | pass-through |

### Non-stream response & SSE (upstream → downstream)

All 3×3 directions are supported. Same-protocol SSE is low-latency pass-through; some cross-protocol tool-call streams buffer for semantic completeness.

## Main APIs

Proxy:

```text
POST /v1/messages
POST /v1/chat/completions
POST /v1/responses
GET  /v1/models
```

Admin (see OpenAPI for full surface, including plugins):

```text
GET  /api/v1/runtime/state
GET  /api/v1/providers
POST /api/v1/providers
POST /api/v1/providers/:id/check
GET  /api/v1/ingress-endpoints
GET  /api/v1/routing-rules
GET  /api/v1/api-keys
GET  /api/v1/traffic
GET  /api/v1/plugins
GET  /api/v1/plugins/ui-pages
```

## Storage

| File | Purpose |
|------|---------|
| `.data/icoo_llm_bridge.db` | Providers, routes, keys, UI prefs, … |
| `.data/icoo_llm_bridge_traffic.db` | Traffic records (separate DB, WAL) |
| `.data/bridge-chain.log` | Optional chain log |
| `.data/plugins/<id>/` | Plugin runtime state / credentials |

## Verification

```powershell
# from repo root (go.work)
go test ./common/...
go test ./bridge/...
go test ./desktop
go test ./plugins/mock/...
go test ./plugins/grokbuild/...

cd desktop\frontend
npm ci
npm run lint
npm run format:check
npm test
npm run build

# contracts + package
.\scripts\Test-OpenAPI.ps1
.\build-all.ps1
.\scripts\Test-Package.ps1 -PackageDir .\icoo_proxy
```

## Operational Notes

- Some providers return `429` under concurrent load; use concurrency limits, retry, or backoff.
- Cross-protocol streaming may buffer (especially tool-call reordering for Anthropic blocks).
- Loopback-first deployment is the supported model; do not expose unauthenticated admin APIs.
- Process plugins receive allowlisted headers only; host tokens stay out of plugin admin list APIs.

## Packaging

Preferred layout after `.\build-all.ps1`:

```text
icoo_proxy\
├── icoo_desktop.exe
├── bridge.exe
└── plugins\
    ├── mock\
    │   ├── info.toml
    │   └── mockplugin.exe
    └── grokbuild\
        ├── info.toml
        └── plugin-grokbuild.exe
```

Start `icoo_desktop.exe`; it can launch the bundled `bridge.exe` from the same directory.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
