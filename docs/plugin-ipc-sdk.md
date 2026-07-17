# Plugin IPC SDK 指南

**模块：** `github.com/issueye/icoo_proxy/common/pluginipc`  
**契约：** [plugin-ipc-contract.md](./plugin-ipc-contract.md) v1（Frozen）  
**协议版本：** `ProtocolVersion = 1`

本文档介绍 **SDK 便捷层**（Client / Server 封装），帮助宿主与插件作者用更少样板代码正确使用 IPC。  
底层传输、JSON-RPC 帧、方法语义仍以契约文档为准；SDK **不改变** 线协议。

---

## 1. 分层一览

| 层 | 文件 | 受众 | 说明 |
|----|------|------|------|
| 协议原语 | `types.go` `framing.go` `conn.go` `transport_*.go` | 高级 / 调试 | 帧、多路 Conn、pipe/UDS |
| Client 原语 | `client.go` | 宿主 | `Handshake` / `Complete` / `OpenStream` … |
| Server 原语 | `server.go` | 插件 | `NewServer` / `RegisterComplete` / `RegisterProxyStream` |
| **SDK Client** | `sdk_client.go` `sdk_helpers.go` | **宿主推荐** | `Connect` / `NewProxyRequest` / `MapCallError` / `Stream.OK` |
| **SDK Server** | `sdk_server.go` `sdk_stream.go` `sdk_helpers.go` | **插件推荐** | `RunPlugin` / `ServeConn` / `RegisterProxyStreamEx` / 响应 helper |

```text
宿主进程                          插件进程
────────                          ────────
spawn + env                       flag/env
pluginipc.Connect  ── Dial ──►    pluginipc.RunPlugin
  Handshake                       Listen → AfterListen → Accept
                                  → PrepareHandshake → ServeConn
Complete / OpenStream  ◄──RPC──►  Handlers.Complete / Stream
```

**边界：**

- SDK **不**管理插件进程（`exec`、Job Object、registry、自动重启）——那是 `bridge/internal/pluginhost`。
- 插件 **禁止** import `bridge/internal/*`，只依赖 `common`。
- 参考实现：`plugins/mock`（最小 RunPlugin）、`plugins/grokbuild`（RunPlugin + Admin UI）。

---

## 2. 插件侧（Server SDK）

### 2.1 最小可运行插件

```go
package main

import (
    "context"
    "log"

    "github.com/issueye/icoo_proxy/common/pluginipc"
)

func main() {
    err := pluginipc.RunPlugin(
        pluginipc.PluginMeta{
            ID:           "hello",
            Version:      "0.1.0",
            UpstreamKind: "echo",
            Capabilities: []string{
                pluginipc.CapProxyComplete,
                pluginipc.CapProxyStream,
                pluginipc.CapModelsList,
                pluginipc.CapHealth,
            },
            SupportedIngress: []string{"openai-chat", "openai-responses", "anthropic"},
        },
        pluginipc.Handlers{
            Complete: func(ctx context.Context, req pluginipc.ProxyRequest) (*pluginipc.ProxyResponse, error) {
                // req.Body 已是 resolved（inline 或 raw-followup 合并后）
                return pluginipc.OKJSON([]byte(`{"ok":true}`), &pluginipc.Usage{OutputTokens: 1}), nil
            },
            Stream: func(ctx context.Context, req pluginipc.ProxyRequest) (
                *pluginipc.StreamOpenResult, *pluginipc.ProxyResponse,
                func(context.Context, *pluginipc.StreamWriter), error,
            ) {
                open := pluginipc.SSEOpen(pluginipc.NewStreamID("hello"))
                run := func(ctx context.Context, w *pluginipc.StreamWriter) {
                    select {
                    case <-ctx.Done():
                        return // honor stream.cancel
                    default:
                    }
                    _ = w.WriteChunk([]byte("data: {\"type\":\"message\"}\n\n"))
                    _ = w.End(&pluginipc.Usage{OutputTokens: 1})
                }
                // 闭包可捕获 prepare 阶段状态，无需 pendingRuns
                return open, nil, run, nil
            },
            ModelsList: func(ctx context.Context) (*pluginipc.ModelsListResult, error) {
                return &pluginipc.ModelsListResult{
                    Models: []pluginipc.ModelInfo{{ID: "hello-model", DisplayName: "Hello"}},
                }, nil
            },
        },
        pluginipc.PluginHooks{},
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

`RunPlugin` 内部顺序（与契约一致）：

1. 解析 `--endpoint` / `--data-dir` / `--plugin-id` + `ICOO_PLUGIN_TOKEN`
2. **`Listen` 立刻**（避免 host dial 超时）
3. `AfterListen` hook（可选：admin UI / 轻量 init）
4. `Accept` 单连接
5. **`PrepareHandshake`**（可选：注入动态 `AdminBaseURL` / `UIPages`）
6. `ServeConn` → `NewServer` + 默认 handshake/ping/shutdown/health + 注册 `Handlers`
7. 阻塞直到 signal / shutdown / 连接关闭

### 2.2 带 Admin UI 的完整示例（推荐）

生产插件（含 `plugins/grokbuild`）应使用 **A**。自定义 flag 在 `RunPlugin` **之前**注册即可。

```go
package main

import (
    "context"
    "flag"
    "io"
    "log"
    "net"
    "net/http"

    "github.com/issueye/icoo_proxy/common/pluginipc"
)

func main() {
    // 自定义 flag 必须在 RunPlugin / ParsePluginFlags 前注册
    httpProxy := flag.String("http-proxy", "", "outbound proxy")

    var (
        adminURL    string
        adminCloser io.Closer
        // 业务依赖用外层 var，AfterListen 赋值；Handlers 闭包运行时再解引用
    )

    meta := pluginipc.PluginMeta{
        ID:           "hello-ui",
        Version:      "0.1.0",
        UpstreamKind: "demo",
        Capabilities: []string{
            pluginipc.CapProxyComplete, pluginipc.CapProxyStream,
            pluginipc.CapModelsList, pluginipc.CapHealth, pluginipc.CapUI,
        },
        SupportedIngress: []string{"openai-chat", "openai-responses", "anthropic"},
    }

    handlers := pluginipc.Handlers{
        Complete: func(ctx context.Context, req pluginipc.ProxyRequest) (*pluginipc.ProxyResponse, error) {
            _ = httpProxy // 业务可用
            return pluginipc.OKJSON([]byte(`{"ok":true}`), nil), nil
        },
        Stream: func(ctx context.Context, req pluginipc.ProxyRequest) (
            *pluginipc.StreamOpenResult, *pluginipc.ProxyResponse,
            func(context.Context, *pluginipc.StreamWriter), error,
        ) {
            open := pluginipc.SSEOpen(pluginipc.NewStreamID("ui"))
            run := func(ctx context.Context, w *pluginipc.StreamWriter) {
                // 必须响应 cancel：host 客户端断开时会 stream.Cancel
                select {
                case <-ctx.Done():
                    return
                default:
                }
                _ = w.WriteChunk([]byte("data: {}\n\n"))
                _ = w.End(nil)
            }
            return open, nil, run, nil
        },
        ModelsList: func(ctx context.Context) (*pluginipc.ModelsListResult, error) {
            return &pluginipc.ModelsListResult{
                Models: []pluginipc.ModelInfo{{ID: "demo"}},
            }, nil
        },
    }

    err := pluginipc.RunPlugin(meta, handlers, pluginipc.PluginHooks{
        AfterListen: func(ctx context.Context, env pluginipc.PluginEnv) error {
            // 务必保持快速；host dial 默认约 30s
            ln, err := net.Listen("tcp", "127.0.0.1:0")
            if err != nil {
                return err
            }
            mux := http.NewServeMux()
            mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                _, _ = io.WriteString(w, "<html>credentials</html>")
            })
            srv := &http.Server{Handler: mux}
            go func() { _ = srv.Serve(ln) }()
            adminURL = "http://" + ln.Addr().String()
            adminCloser = closerFunc(func() error { return srv.Close() })
            return nil
        },
        PrepareHandshake: func(env pluginipc.PluginEnv, m pluginipc.PluginMeta) (pluginipc.PluginMeta, error) {
            m.AdminBaseURL = adminURL
    m.AdminToken = adminToken
            m.UIPages = []pluginipc.UIPage{{
                ID: "credentials", Title: "凭据", Path: "/", Icon: "key", Group: "插件",
            }}
            return m, nil
        },
        OnShutdown: func() {
            if adminCloser != nil {
                _ = adminCloser.Close()
            }
        },
        Health: func(ctx context.Context) (*pluginipc.HealthResult, error) {
            return &pluginipc.HealthResult{OK: true, Status: "healthy"}, nil
        },
    })
    if err != nil {
        log.Fatal(err)
    }
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }
```

**B. 手动 `Listen` + `ServeConn`（仅当需要完全自管 accept 循环时）**

```go
ln, _ := pluginipc.Listen(ctx, pluginipc.ListenConfig{Endpoint: endpoint})
// init store / admin → 得到 adminURL
conn, _ := ln.Accept()
meta.AdminBaseURL = adminURL
srv := pluginipc.ServeConn(conn, pluginipc.ServerOptions{
    HostToken:  token,
    Handshake:  pluginipc.HandshakeFrom(meta),
    OnShutdown: onShutdown,
}, handlers, healthFn)
<-ctx.Done()
_ = srv.Close()
```

> 注意：`plugins/grokbuild` **已迁移到模式 A**（`RunPlugin` + `PrepareHandshake`）。新插件请优先复制 A / mock，不必再手写 accept 循环。

### 2.3 统一 Stream API：`RegisterProxyStreamEx`

旧 API（两段 + 自建 map）：

```go
var pending sync.Map
srv.RegisterProxyStream(prepare, run) // prepare 与 run 靠 streamID 桥接
```

新 API（推荐）：

```go
srv.RegisterProxyStreamEx(func(ctx context.Context, req pluginipc.ProxyRequest) (
    open *pluginipc.StreamOpenResult,
    errResp *pluginipc.ProxyResponse,
    run func(context.Context, *pluginipc.StreamWriter),
    err error,
) {
    if noCreds {
        return nil, pluginipc.Unauthorized("no credentials"), nil, nil
    }
    if badIngress {
        return nil, nil, nil, pluginipc.RPCUnsupportedIngress(fmt.Errorf("..."))
    }
    open := pluginipc.SSEOpen(pluginipc.NewStreamID("gb"))
    upstreamBody := doUpstream(ctx, req)
    run := func(ctx context.Context, w *pluginipc.StreamWriter) {
        defer upstreamBody.Close()
        // Copy 直到 EOF 或 ctx cancel
        buf := make([]byte, 32<<10)
        for {
            select {
            case <-ctx.Done():
                return
            default:
            }
            n, err := upstreamBody.Read(buf)
            if n > 0 {
                _ = w.WriteChunk(buf[:n])
            }
            if err != nil {
                break
            }
        }
        _ = w.End(nil)
    }
    return open, nil, run, nil
})
```

**规则（契约 Issue 18）：**

- 成功 open 必须带非空 `stream_id`，status 2xx
- 非 2xx 用 `errResp *ProxyResponse`；**不要**再启动 SSE
- `run` 仅在 open 帧写完后由 SDK 调用（open-before-chunk）
- host `stream.Cancel` → 插件侧取消 **run 的 `ctx`**；`run` 必须监听 `ctx.Done()` 以中止上游读取

### 2.4 响应与错误 helper

| Helper | 用途 |
|--------|------|
| `OKJSON(body, usage)` | 200 + `application/json` |
| `JSONStatus(status, body, usage)` | 任意 HTTP status + JSON body |
| `JSONError(status, msg)` | JSON 错误体（含 type/code） |
| `Unauthorized(msg)` | 401 |
| `BadGateway(msg)` | 502 |
| `SSEOpen(streamID)` | stream open 200 + `text/event-stream` |
| `NewStreamID(prefix)` | 随机 stream id |
| `RPCUnsupportedIngress(err)` | RPC `-32002` |
| `UpstreamRPCError(status, msg)` | RPC `-32003` + `data.status` |
| `HandshakeFrom(meta)` | 填 `IPCProtocolVersion` |
| `(*StreamWriter).AsWriter()` | `io.Writer` 适配 |

**错误模型：**

| 场景 | 返回 |
|------|------|
| 参数 / ingress 不支持 | `return nil, pluginipc.RPCUnsupportedIngress(err)` |
| 无凭据 / 上游 4xx/5xx | `Unauthorized` / `JSONStatus(status, body, nil), nil` |
| 需要 host 按上游 status 映射 | `return nil, pluginipc.UpstreamRPCError(429, "rate limited")` |
| 流式业务失败（open 前） | `return nil, errResp, nil, nil` |
| 流式中途失败 | `w.Error(code, msg)` |
| 流式取消 | `run` 内检查 `ctx.Done()` |

### 2.5 测试：`ServeConn` + `net.Pipe`

```go
c1, c2 := net.Pipe()
srv := pluginipc.ServeConn(c2, pluginipc.ServerOptions{
    HostToken: "tok",
    Handshake: pluginipc.HandshakeFrom(pluginipc.PluginMeta{ID: "t", Version: "0"}),
}, pluginipc.Handlers{
    Complete: func(ctx context.Context, req pluginipc.ProxyRequest) (*pluginipc.ProxyResponse, error) {
        return pluginipc.OKJSON([]byte(`{}`), nil), nil
    },
}, nil)
cli := pluginipc.NewClient(c1, pluginipc.ClientOptions{})
// Handshake + Complete ...
```

### 2.6 CLI / 环境约定

| 来源 | 名称 | 说明 |
|------|------|------|
| flag / env | `--endpoint` / `ICOO_PLUGIN_ENDPOINT` | pipe 或 sock 路径 |
| flag | `--data-dir` | 插件数据目录 |
| flag | `--plugin-id` | 覆盖 handshake plugin_id |
| env **必填** | `ICOO_PLUGIN_TOKEN` | 与 host 握手 |

额外 flag 请在调用 `ParsePluginFlags` / `RunPlugin` **之前** `flag.String(...)` 注册。

---

## 3. 宿主侧（Client SDK）

### 3.1 连接与握手

进程仍由 host 自己 spawn；SDK 只负责 Dial + Handshake。`bridge/internal/pluginhost` 的最小形态：

```go
endpoint, _ := pluginipc.NewEndpoint(id, dataDir)
token, _ := pluginipc.NewHostToken()
cmd := exec.Command(exe,
    "--endpoint", endpoint,
    "--data-dir", pluginDataDir,
    "--plugin-id", id,
)
cmd.Env = append(os.Environ(),
    "ICOO_PLUGIN_TOKEN="+token,
    "ICOO_PLUGIN_ENDPOINT="+endpoint,
)
if err := cmd.Start(); err != nil {
    return err
}
cli, hs, err := pluginipc.Connect(ctx, pluginipc.ConnectConfig{
    Endpoint:             endpoint,
    Token:                token,
    HostVersion:          pluginipc.DefaultHostVersion, // "icoo_llm_bridge"
    DialTimeout:          30 * time.Second,
    HandshakeTimeout:     15 * time.Second,
    MaxFrameBytes:        64 << 20,
    MaxConcurrentStreams: 32,
})
if err != nil {
    _ = cmd.Process.Kill()
    _, _ = cmd.Wait()
    return err
}
// hs.PluginID, hs.Capabilities, hs.AdminBaseURL, hs.UIPages ...
// 之后：心跳 Ping、进程 Wait 监视、Stop 时 Shutdown+Close
```

### 3.2 组装代理请求

```go
req := pluginipc.NewProxyRequest(pluginipc.ProxyRequestInput{
    Ingress: "anthropic", // 或 openai-chat / openai-responses
    Path:    r.URL.Path,
    Method:  r.Method,
    Headers: pluginipc.HeadersFromHTTP(r.Header), // 自动 Filter + anthropic-version
    Body:    body,
    Model:   route.Model,
    Stream:  wantsStream,
})
```

`NewProxyRequest` 内部：

1. `FilterHeaders`（剥离 Authorization / Cookie / x-api-key …）
2. `EnsureAnthropicVersion`（anthropic ingress 缺省 `2023-06-01`）

### 3.3 非流式

```go
resp, err := cli.Complete(ctx, req)
if err != nil {
    status, msg := pluginipc.MapCallError(err)
    // write HTTP status/msg
    return
}
if !resp.Success() {
    // 写 resp.StatusOrOK() + resp.Body
    return
}
// 成功：resp.Body / resp.Usage
```

### 3.4 流式（务必检查 OK）

```go
stream, err := cli.OpenStream(ctx, req)
if err != nil {
    status, msg := pluginipc.MapCallError(err)
    return
}
defer stream.Close()

if !stream.OK() {
    // 禁止 WriteHeader(200) / 禁止 SSE
    _, errBody, status := stream.ErrorBody(ctx)
    // 写 status + errBody
    return
}

// 客户端断开时取消上游
go func() {
    <-ctx.Done()
    cctx, ccancel := context.WithTimeout(context.Background(), 2*time.Second)
    if err := stream.Cancel(cctx); err != nil { /* log */ }
    ccancel()
}()

// 仅在 OK 后写 SSE 头与 body
for {
    ev, err := stream.Recv(ctx)
    if err != nil {
        break
    }
    switch ev.Kind {
    case "chunk":
        if _, err := w.Write(ev.Chunk.Data); err != nil {
            cctx, ccancel := context.WithTimeout(context.Background(), 2*time.Second)
    if err := stream.Cancel(cctx); err != nil { /* log */ }
    ccancel()
            return
        }
    case "end":
        // usage := ev.End.Usage
        return
    case "error":
        // log ev.Error.Message
        return
    }
}
```

### 3.5 生命周期 RPC

| 方法 | API | 典型用途 |
|------|-----|----------|
| handshake | `Connect` / `Handshake` | 启动 |
| ping | `Ping` | 心跳 |
| health | `Health` | 管理面 |
| get_info | `GetInfo` | 调试 |
| models.list | `ListModels` | 拉模型 |
| shutdown | `Shutdown` | 优雅停止 |

---

## 4. Streaming 取消：`stream.cancel`

### 4.1 时序

```text
Host OpenStream ──► Plugin prepare
                      │ 2xx open  (register cancelable run ctx)
                      ▼
Host ◄── open result ── Plugin
Host 开始 Recv          Plugin run 写 chunk
  ...
客户端断开 / 写下游失败
Host stream.Cancel ──► Plugin MethodStreamCancel
                      cancel(run ctx)
Plugin run 见 ctx.Done() → 停上游读，return
（可选 stream.end / 连接关闭；重复 Cancel 幂等）
```

### 4.2 义务

| 角色 | 必须 |
|------|------|
| Host | 客户端 `Request.Context` 取消时调用 `stream.Cancel`；写下游失败时也建议 Cancel；`defer stream.Close()` |
| Plugin SDK | `RegisterProxyStream` / `RegisterProxyStreamEx` **默认**注册 cancel handler，取消 run `ctx` |
| Plugin 业务 | `run` 内监听 `ctx.Done()`，关闭上游 body，停止 `WriteChunk` |

### 4.3 `Cancel` vs `Close`

| API | 含义 |
|-----|------|
| `stream.Cancel` | 向插件发 `stream.cancel` RPC，请求中止 **该 stream** 的 run |
| `stream.Close` | 释放 host 侧 stream 资源 / 注销接收；**不保证**插件立刻停上游 |

open 前失败（`errResp` 非 2xx）不会注册 run，无需 Cancel。已 `end` / 连接断开后的 Cancel 应为幂等 no-op。

---


### 4.1 可观测性与背压

| 项 | 行为 |
|----|------|
| Host 取消 | `plugin_proxy` 在 client disconnect / 下游写失败时调用 `stream.Cancel`，记录成功/失败日志，RPC 超时 2s |
| 插件 hook | `ServerOptions.OnStreamCancel(streamID, found)` 可选回调 |
| 事件缓冲 | host 侧 stream channel 容量 64；**满时非阻塞丢弃**并 `Conn.StreamDrops()` 计数，避免堵死 demux |
| 语义边界 | `stream.cancel` **只**取消 open 成功后的 run ctx；prepare 阶段仍用 Background |

## 5. 可靠性（SDK + Host 边界）

### 5.1 插件启动

| 项 | 建议 |
|----|------|
| Listen 优先 | `RunPlugin` 保证 Listen → AfterListen → Accept |
| AfterListen 时限 | 远小于 host `DialTimeout`（默认 30s）；OAuth 等需自带短超时 |
| Token | 仅 env；常量时间比较（契约） |
| 单连接 | 当前契约单 host 连接；断线后由 **host respawn**，SDK 不自动 reconnect |

### 5.2 运行中

| 项 | 默认 / 说明 |
|----|-------------|
| Ping | host 周期 `Ping`（bridge 默认 5s） |
| MaxConcurrentStreams | 默认 32 |
| max frame / inline | 默认跟 host 请求体上限 / 256KiB |
| orphan stream 缓冲 | host conn 在 register 前最多缓存 64 个 stream 事件 |

### 5.3 停止

| 路径 | 行为 |
|------|------|
| `plugin.shutdown` | 触发 `OnShutdown` + 关连接 |
| SIGINT/SIGTERM | `RunPlugin` 默认 `signal.NotifyContext` |
| host `Stop` | Shutdown RPC → Close client → Wait/Kill 进程 |

### 5.4 Host 进程可靠性（`pluginhost`，非 SDK）

bridge 宿主额外提供：

- **进程监视** `cmd.Wait` → status=`error`，可选自动重启
- **心跳失败** 连续 N 次（默认 3）→ 关 Client + 自动 `Restart`
- **退避** 1s / 2s / 5s / 30s，避免重启风暴
- **Job Object**（Windows `KILL_ON_JOB_CLOSE`）；`Manager.Close()` 在 `StopAll` 后释放
- 配置：`plugins.auto_restart`（默认 true）、`plugins.auto_restart_fail_threshold`（默认 3）

```toml
[plugins]
auto_restart = true
auto_restart_fail_threshold = 3
heartbeat_interval_seconds = 5
```

---

## 6. 能力常量

```go
pluginipc.CapProxyComplete // "proxy.complete"
pluginipc.CapProxyStream   // "proxy.stream"
pluginipc.CapModelsList    // "models.list"
pluginipc.CapHealth        // "health"
pluginipc.CapUI            // "ui"
```

插件可在 `Capabilities` 中追加自有字符串（如 `oauth.device`）；host 目前主要用于展示，不强制 gate。

---

## 7. 目录与依赖

```text
plugins/<id>/
  go.mod                 # require common; replace => ../../common
  info.toml              # 安装发现元数据
  cmd/plugin-<id>/main.go
```

`go.mod` 示例：

```go
module github.com/issueye/icoo_proxy/plugins/hello

go 1.23

require github.com/issueye/icoo_proxy/common v0.0.0

replace github.com/issueye/icoo_proxy/common => ../../common
```

根目录 `go.work` 应包含 `./common` 与 `./plugins/<id>`。

---

## 8. 从旧代码迁移

| 旧写法 | 新写法 |
|--------|--------|
| 手写 Listen/Accept/NewServer/flag | `RunPlugin` 或 `ServeConn` |
| AfterListen 后无法改 handshake | `PluginHooks.PrepareHandshake` |
| `RegisterProxyStream` + `sync.Map` | `RegisterProxyStreamEx` 返回 run 闭包 |
| 手写 JSON 401/502 body | `Unauthorized` / `BadGateway` / `JSONError` / `JSONStatus` |
| 手写 SSE headers | `SSEOpen` + `NewStreamID` |
| `StreamWriter` 自定义 `io.Writer` | `w.AsWriter()` |
| Dial+NewClient+Handshake | `Connect` |
| 手写 FilterHeaders 组装 | `NewProxyRequest` + `HeadersFromHTTP` |
| 字符串匹配错误 | `MapCallError`（`-32003` 读 `data.status`） |
| 手写 non-2xx stream drain | `!stream.OK()` + `ErrorBody` |
| 忽略 `stream.cancel` | 默认取消 run `ctx`；`run` 必须响应 |

### 8.1 grokbuild 迁移对照（已完成）

| 原 hybrid | 现 RunPlugin |
|-----------|--------------|
| `ParsePluginFlags` + 手写 Listen | `RunPlugin` 内置 |
| store/oauth/admin 在 Accept 前 | `AfterListen` |
| `acceptOne` | SDK Accept |
| 手写 `meta.AdminBaseURL` | `PrepareHandshake` |
| `ServeConn` + select | `RunPlugin` wait |
| `OnShutdown` 关 conn+admin | 只关 admin（SDK 关 conn） |
| `--http-proxy` | **仍**在 `RunPlugin` 前 `flag.String` |

业务层 `proxyhandler` 无需改动（已用 StreamEx helpers）。完整源码：`plugins/grokbuild/cmd/plugin-grokbuild/main.go`。

底层 API **全部保留**，可渐进迁移。

---

## 9. 测试

```bash
cd common
go test ./pluginipc/ -count=1

# host
go test ./bridge/internal/pluginhost/ ./bridge/internal/service/ -count=1 -run Plugin
```

覆盖：握手、inline/raw body、stream open-before-chunk、SDK helpers、`Connect`/`RunPlugin`、`RegisterProxyStreamEx`、`PrepareHandshake`、`stream.cancel`、`AsWriter`、`MapCallError`（含 `-32003 data.status`）。

---

## 10. 安全提示（插件作者）

1. **永远不要**把客户端 `Authorization` / API Key 当上游凭据——host 已剥离；凭据由插件自己的 store/OAuth 管理。  
2. Admin UI 仅监听 **loopback**，并通过握手下发 AdminToken；bridge UI 反代注入 X-ICOO-Plugin-Admin-Token，浏览器侧不持有该密钥。
3. AdminToken 不得出现在 PluginView / ui-pages 等公开管理 API 响应中。  
3. 可执行插件等同本机代码执行——安装来源需可信。  
4. 不要在无密钥时把 bridge 暴露到公网。

---

## 11. API 速查

### Server

```text
ParsePluginFlags() (PluginEnv, error)
RunPlugin(meta, handlers, hooks) error
  hooks.AfterListen / PrepareHandshake / OnShutdown / Health
ServeConn(conn, opts, handlers, health) *Server
HandshakeFrom(meta) HandshakeResult
(*Server) RegisterProxyStreamEx(StreamHandler)  // 内置 stream.cancel → ctx
(*Server) RegisterHealth(fn)
(*StreamWriter) AsWriter() io.Writer
OKJSON / JSONStatus / JSONError / Unauthorized / BadGateway / SSEOpen / NewStreamID
RPCUnsupportedIngress(err) error
UpstreamRPCError(status, msg) error
```

### Client

```text
Connect(ctx, ConnectConfig) (*Client, *HandshakeResult, error)
NewProxyRequest(ProxyRequestInput) ProxyRequest
HeadersFromHTTP(http.Header) map[string]string
MapCallError(err) (status int, message string)
(*Stream) OK() bool
(*Stream) ErrorBody(ctx) (headers, body, status)
(*Stream) Cancel(ctx) error
(*ProxyResponse) StatusOrOK() int
(*ProxyResponse) Success() bool
```

---

## 12. 相关文档

- [Plugin IPC Contract v1](./plugin-ipc-contract.md) — 线协议冻结基线  
- [Process plugin architecture](./design/2026-07-16-process-plugin-architecture.md)  
- [Plugin extension pages](./design/2026-07-16-plugin-extension-pages.md)  
- 示例插件：[`plugins/mock`](../plugins/mock)、[`plugins/grokbuild`](../plugins/grokbuild)  
- OpenAPI：`docs/openapi.yaml`（管理面 REST，非 IPC）


## Host policy: admin_enabled

Per-plugin `admin_enabled` gates UI reverse-proxy and ui-pages (403 `PLUGIN_UI_DISABLED` when off). Install defaults true when package advertises `ui`. Windows pipe SDDL default is owner-only `D:P(A;;GA;;;OW)`.
