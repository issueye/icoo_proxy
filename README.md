# icoo_proxy

Local-first LLM API bridge and desktop console.

`icoo_proxy` exposes OpenAI Chat Completions, OpenAI Responses, and Anthropic Messages compatible endpoints, then routes each request to configured upstream providers. The backend service lives in `icoo_llm_bridge`, and the desktop app lives in `icoo_desktop`.

Chinese documentation: [README.cn.md](README.cn.md)

## Features

- Local HTTP bridge for:
  - `POST /v1/chat/completions`
  - `POST /v1/responses`
  - `POST /v1/messages`
- Provider, model, endpoint, routing-rule, API-key, and traffic management APIs.
- Protocol conversion across OpenAI Chat, OpenAI Responses, and Anthropic Messages.
- Streaming SSE support, including same-protocol low-latency pass-through.
- Desktop management console built with Wails and Vue.
- SQLite storage with separate main and traffic databases.
- Provider health checks from the desktop console.
- Runtime database diagnostics for gray replacement safety.
- Desktop pages for gateway overview, provider health, routing rules, custom endpoints, local API keys, traffic inspection, and runtime settings.

## Repository Layout

```text
.
├── icoo_desktop/        # Wails desktop app and Vue frontend
├── icoo_llm_bridge/     # Go backend service
├── icoo_proxy/          # Packaged executable output directory
└── build-all.ps1        # All-in-one build script
```

## Requirements

- Windows PowerShell
- Go toolchain
- Node.js and npm
- Wails CLI, for desktop builds

Install Wails if needed:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## Build The Bridge

```powershell
cd icoo_llm_bridge
.\build.ps1
```

The output is:

```text
icoo_llm_bridge\build\bridge.exe
```

Skip tests when you only need a fast build:

```powershell
.\build.ps1 -SkipTests
```

## Run The Bridge

```powershell
cd icoo_llm_bridge
.\build\bridge.exe
```

Health checks:

```powershell
Invoke-RestMethod http://127.0.0.1:18181/healthz
Invoke-RestMethod http://127.0.0.1:18181/readyz
Invoke-RestMethod http://127.0.0.1:18181/api/v1/runtime/state
```

By default, local loopback requests can use admin APIs without an API key. Configure API keys before exposing the bridge outside localhost.

## Build The Desktop App

Build the desktop app and bundle the bridge:

```powershell
cd icoo_desktop
.\build.ps1 -BridgePath ..\icoo_llm_bridge\build\bridge.exe
```

The output is:

```text
icoo_desktop\build\bin\icoo_desktop.exe
icoo_desktop\build\bin\bridge.exe
```

For frontend-only development:

```powershell
cd icoo_desktop\frontend
npm install
npm run dev
```

## Desktop Console

The desktop console is the recommended way to operate a local `icoo_proxy` instance. It can launch or restart the bundled bridge, show the current connection state, and manage the gateway without editing SQLite data by hand.

![Gateway overview](images/index.png)

The overview page shows the listener address, access mode, provider count, enabled routing policies, upstream readiness, and the supported ingress paths. The screenshots in this section use `127.0.0.1:18181` in local trusted mode.

### Provider Management

![Provider management](images/provider.png)

Use `Provider` to search and filter upstream providers, create new provider entries, edit credentials and base URLs, check provider health, inspect enabled models, and delete entries. Provider rows show the protocol, base URL, enabled state, and operational tags such as `only_stream`.

### Routing Rules

![Routing rules](images/rules.png)

Use `规则设置` to map each downstream protocol to an upstream provider and protocol. The console currently lists Anthropic Messages, OpenAI Chat, and OpenAI Responses as downstream protocols, and shows whether each mapping is enabled, pending selection, or unconfigured.

### Ingress Endpoints

![Ingress endpoints](images/endpoint.png)

Use `端点` to view or add ingress paths. The default enabled endpoints include:

- `/v1/chat/completions` for OpenAI Chat compatible clients.
- `/v1/messages` for Anthropic Messages compatible clients.
- `/v1/responses` and `/responses` for OpenAI Responses compatible clients.

### API Keys

![API keys](images/keys.png)

Use `授权 Key` to create local client keys when you need authenticated access. Clients can authenticate with either `Bearer` tokens or the `x-api-key` header. When no key is configured and the bridge is bound to localhost, local trusted mode can still allow local admin usage.

### Traffic Monitor

![Traffic monitor](images/traffic.png)

Use `流量监控` to inspect recent proxy requests, success and error counts, average latency, token totals, endpoints, downstream/upstream protocols, hit routing rules, status codes, and per-request timing. The page supports protocol filtering, manual refresh, auto refresh, and clearing recorded requests.

### Runtime Settings

![Runtime settings](images/settings.png)

Use `项目设置` to adjust console appearance, button density, and bridge runtime settings such as host, port, read/write timeouts, shutdown timeout, default max tokens, and chain-log parameters. Save settings and restart the bridge from the top-right action to apply runtime changes.

## Configure Providers

In the desktop app, open `Provider` and create a provider with:

- Name
- Protocol: `openai-responses`, `openai-chat`, or `anthropic`
- Base URL
- API key
- Enabled models

The provider health button sends a minimal real upstream request using the first enabled model. This validates connectivity and credentials, but it also consumes a small model request.

## Routing

The bridge resolves routes in this order:

1. Direct provider/model routing, for example `provider-name/model-name`.
2. Enabled routing rules ordered by priority.

Default protocol routing can be edited in the desktop `规则设置` page. Model-specific aliases can be configured in `模型路由`.

## Main APIs

Proxy APIs:

```text
POST /v1/messages
POST /v1/chat/completions
POST /v1/responses
```

Admin APIs:

```text
GET  /api/v1/runtime/state
GET  /api/v1/providers
POST /api/v1/providers
POST /api/v1/providers/:id/check
GET  /api/v1/providers/:id/models
GET  /api/v1/ingress-endpoints
GET  /api/v1/routing-rules
GET  /api/v1/api-keys
GET  /api/v1/traffic
```

## Verification

Run backend tests:

```powershell
cd icoo_llm_bridge
go test ./...
```

Run frontend build:

```powershell
cd icoo_desktop\frontend
npm run build
```

## Current Status

The bridge supports:

- OpenAI Responses to OpenAI Responses
- OpenAI Chat to OpenAI Responses
- OpenAI Chat to Anthropic
- OpenAI Responses to Anthropic
- Anthropic to OpenAI Responses
- Anthropic to Anthropic
- Multi-turn messages
- Tool calls
- Streaming responses
- Traffic recording and desktop-side inspection

Known operational notes:

- Some providers may return `429 Too Many Requests` under concurrent load. Use provider-level concurrency limits, retry, or backoff in production.
- Some cross-protocol streaming paths still buffer during conversion, while same-protocol SSE pass-through is low latency.

## Packaging

The preferred manual package layout is:

```text
icoo_proxy\
├── icoo_desktop.exe
└── bridge.exe
```

Start `icoo_desktop.exe`; the desktop app can launch the bundled `bridge.exe` when it is placed in the same directory.

## License

No license file is currently included. Add one before public distribution.
