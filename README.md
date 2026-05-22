# icoo_proxy

Local-first LLM API bridge and desktop console.

`icoo_proxy` exposes OpenAI Chat Completions, OpenAI Responses, and Anthropic Messages compatible endpoints, then routes each request to configured upstream providers. The current replacement-ready backend is the Rust service in `icoo_llm_bridge_r`; the Go service in `icoo_llm_bridge` is kept as the previous implementation and parity reference.

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

## Repository Layout

```text
.
├── icoo_llm_bridge_r/   # Rust backend, preferred service for new builds
├── icoo_desktop/        # Wails desktop app and Vue frontend
├── icoo_llm_bridge/     # Go backend, previous implementation/reference
├── icoo_proxy/          # Packaged executable output directory
└── build-all.ps1        # Legacy all-in-one build script
```

## Requirements

- Windows PowerShell
- Rust toolchain with `cargo`
- Go toolchain
- Node.js and npm
- Wails CLI, for desktop builds

Install Wails if needed:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## Build The Rust Bridge

```powershell
cd icoo_llm_bridge_r
.\build.ps1
```

The output is:

```text
icoo_llm_bridge_r\build\bridge.exe
```

Skip tests when you only need a fast release build:

```powershell
.\build.ps1 -SkipTests
```

Use a custom Cargo cache only when needed:

```powershell
.\build.ps1 -CargoHome "E:\cargo-cache"
```

## Run The Bridge

```powershell
cd icoo_llm_bridge_r
.\build\bridge.exe --addr 127.0.0.1:18181 --data-dir .data
```

Health checks:

```powershell
Invoke-RestMethod http://127.0.0.1:18181/healthz
Invoke-RestMethod http://127.0.0.1:18181/readyz
Invoke-RestMethod http://127.0.0.1:18181/api/v1/runtime/state
```

By default, local loopback requests can use admin APIs without an API key. Configure API keys before exposing the bridge outside localhost.

## Build The Desktop App

Build the desktop app and bundle the Rust bridge:

```powershell
cd icoo_desktop
.\build.ps1 -BridgePath ..\icoo_llm_bridge_r\build\bridge.exe
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

Default routing rules can be edited in the desktop `规则设置` page. Model-specific aliases can be configured in `模型路由`.

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
cd icoo_llm_bridge_r
cargo test
```

Run frontend build:

```powershell
cd icoo_desktop\frontend
npm run build
```

Real upstream verification scripts:

```powershell
cd icoo_llm_bridge_r
.\scripts\verify-real-upstream.ps1 -BridgeUrl http://127.0.0.1:18181 -ApiKey "<key>" -ResponsesModel "<model>"
.\scripts\preflight-gray-replacement.ps1 -SourceDataDir "E:\path\to\data"
```

See also:

- [Go/Rust parity checklist](icoo_llm_bridge_r/docs/parity/go-rust-parity-checklist.md)
- [Rust gray release runbook](icoo_llm_bridge_r/docs/release/rust-gray-release-runbook.md)
- [Real upstream verification](icoo_llm_bridge_r/docs/release/real-upstream-verification.md)

## Current Status

The Rust bridge has been validated against real upstream providers for:

- OpenAI Responses to OpenAI Responses
- OpenAI Chat to OpenAI Responses
- OpenAI Chat to Anthropic
- OpenAI Responses to Anthropic
- Anthropic to OpenAI Responses
- Anthropic to Anthropic
- Multi-turn messages
- Tool calls
- Streaming responses
- Mixed concurrent requests

Known operational notes:

- Some providers may return `429 Too Many Requests` under concurrent load. Use provider-level concurrency limits, retry, or backoff in production.
- Full migration of older legacy database schemas is not part of the current Rust gray-release phase.
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
