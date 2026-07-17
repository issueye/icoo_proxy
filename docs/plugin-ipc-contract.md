# Plugin IPC Contract (v1)

> Frozen for implementation. Full design: [docs/design/2026-07-16-process-plugin-architecture.md](./design/2026-07-16-process-plugin-architecture.md)

## Module graph

| Package | Module | Import path |
|---------|--------|-------------|
| IPC SDK | `github.com/issueye/icoo_proxy/common` | `github.com/issueye/icoo_proxy/common/pluginipc` |
| Protocol / domain | `github.com/issueye/icoo_proxy/common` | `.../common/constants`, `.../common/domain`, `.../common/ai_llm_proxy` |
| Host | `github.com/issueye/icoo_proxy/bridge` | `github.com/issueye/icoo_proxy/bridge/internal/pluginhost` |
| Grok plugin | `github.com/issueye/icoo_proxy/plugins/grokbuild` | own module; **must not** import `bridge/internal/...` |

## Package layout (discovery)

Bridge / desktop look for a top-level **`plugins/`** directory (next to `bridge.exe` and under cwd). Each plugin is one subdirectory with `info.toml`:

```text
plugins/
  grokbuild/
    info.toml              # name, version, description, executable, …
    plugin-grokbuild.exe
  mock/
    info.toml
    mockplugin.exe
```

Example `info.toml`:

```toml
id = "grokbuild"
name = "GrokBuild / SuperGrok Proxy"
version = "0.3.2"
description = "…"
executable = "plugin-grokbuild.exe"
capabilities = ["proxy.complete", "proxy.stream", "models.list", "health", "ui"]
supported_ingress = ["anthropic", "openai-responses", "openai-chat"]
```

Runtime state (registry.json, credentials) remains under **`data_dir/plugins/`**, not the package tree. Legacy flat `plugin-*.exe` next to bridge is still scanned as a fallback.

Plugin `go.mod` (workspace-aware):

```go
module github.com/issueye/icoo_proxy/plugins/grokbuild

go 1.23

require github.com/issueye/icoo_proxy/common v0.0.0

replace github.com/issueye/icoo_proxy/common => ../../common
```

Root `go.work` lists `./common`, `./bridge`, `./desktop`, `./plugins/*`.

## Transport

| OS | Endpoint format | Listener | Dialer |
|----|-----------------|----------|--------|
| Windows | `\\.\pipe\icoo-plugin-<plugin_id>-<8hex>` | Plugin (go-winio Named Pipe) | Host |
| Unix | `<data_dir>/plugins/<id>/run-<8hex>.sock` | Plugin (pathname UDS) | Host |

- Random 8-hex suffix is **required** every spawn.
- Windows SDDL default: `D:P(A;;GA;;;OW)` (owner only).
- Unix: parent dir `0700`, socket `0600`.

## Process spawn contract

```text
plugin-grokbuild \
  --endpoint <pipe-or-sock> \
  --data-dir <path> \
  --plugin-id grokbuild
```

Environment:

| Variable | Required | Description |
|----------|----------|-------------|
| `ICOO_PLUGIN_TOKEN` | yes | 32-byte hex host token; constant-time compare on handshake |
| `ICOO_PLUGIN_ENDPOINT` | no | Same as `--endpoint` if CLI omitted |
| `ICOO_PLUGIN_LOG` | no | Optional log path |

## Framing

```text
+----------------+---------------------------+
| u32 BE length  | payload (length bytes)    |
+----------------+---------------------------+
```

- Control messages: UTF-8 JSON-RPC 2.0.
- Large bodies: `body_encoding=raw-followup` then an immediate **raw** frame (not JSON).
- `WriteMessage(ctrl, raw)` must hold `writeMu` across **both** frames.
- Demux `expect_raw_body` attaches the next frame before dispatch.

Defaults:

- `max_frame_bytes` follows `max_request_body_bytes` (default 64 MiB) when 0.
- Inline body allowed only when `len(body) ≤ 256 KiB`.
- Stream chunk payload max 64 KiB per notification.

## Lifecycle RPC

| Method | Direction | Notes |
|--------|-----------|-------|
| `plugin.handshake` | H→P | token, versions, capabilities |
| `plugin.ping` | H→P | heartbeat |
| `plugin.get_info` | H→P | redacted health summary |
| `plugin.shutdown` | H→P | graceful exit |
| `plugin.health` | H→P | Admin Check |
| `models.list` | H→P | model catalog |
| `proxy.complete` | H→P | non-stream |
| `proxy.stream.open` | H→P | returns `stream_id` **before** any `stream.*` |
| `stream.chunk` | P→H | seq + data |
| `stream.end` | P→H | terminal + usage |
| `stream.error` | P→H | terminal error |
| `stream.cancel` | H→P | client cancel |

## Streaming rules

1. Plugin **must not** emit `stream.*` before the open **result** frame is fully written.
2. Host **must not** write SSE until open result is 2xx with `stream_id`.
3. Non-2xx open ⇒ map to complete-style HTTP error; **never** `WriteHeader(200)` / SSE.
4. Max concurrent streams default **32** per plugin.

## Header allowlist (host → plugin)

Allowed (case-insensitive):

- `content-type`, `accept`, `user-agent`
- `anthropic-version`, `anthropic-beta`
- `x-claude-code-session-id`, `x-session-id`, `x-grok-conv-id`
- `openai-organization`, `openai-project`

Denied: `authorization`, `x-api-key`, `cookie`, `proxy-authorization`, hop-by-hop headers.

Default inject when Anthropic ingress and missing: `anthropic-version: 2023-06-01`.

## Error codes → HTTP

| code | meaning | HTTP |
|------|---------|------|
| -32700 | parse error | 502 |
| -32600 | invalid request | 502 |
| -32601 | method not found | 502 |
| -32602 | invalid params | 400 |
| -32603 | internal error | 502 |
| -32001 | unauthorized token | 502 |
| -32002 | unsupported ingress | 400 |
| -32003 | upstream error | data.status or 502 |
| -32004 | stream not found | 502 |
| -32005 | shutting down | 503 |
| -32006 | too many streams | 503 |
| -32007 | frame/body too large | 413 |

## Routing (host)

- Provider: `Vendor=plugin` + `plugin_id`.
- Proxy branch **before** `ConvertRequest`; raw downstream body; skip bridge converter.
- Traffic: `UpstreamProtocol=plugin:<plugin_id>`.

## GrokBuild plugin

- Optional; default `enabled=false`.
- Hybrid process plugin (primary) + permanent HTTP sidecar path (`Vendor=custom` + loopback).
- MVP-A: text complete + stream messages; no tools/thinking parity.
- Credentials stay in plugin `data_dir`.
