# icoo_proxy

本地优先的 LLM API 转换网关和桌面管理端。

`icoo_proxy` 对下游暴露 OpenAI Chat Completions、OpenAI Responses、Anthropic Messages 兼容接口，并根据供应商、模型和路由策略把请求转发到不同上游。当前推荐使用的后端是 `icoo_llm_bridge_r` 里的 Rust 版本；`icoo_llm_bridge` 里的 Go 版本保留为旧实现和行为对照。

English documentation: [README.md](README.md)

## 功能特性

- 本地 HTTP 网关，支持：
  - `POST /v1/chat/completions`
  - `POST /v1/responses`
  - `POST /v1/messages`
- 管理供应商、模型、端点、路由规则、API Key 和流量记录。
- 支持 OpenAI Chat、OpenAI Responses、Anthropic Messages 之间的协议转换。
- 支持 SSE 流式响应，同协议流式请求可低延迟透传。
- 基于 Wails + Vue 的桌面管理端。
- 使用 SQLite 存储，主数据库和流量数据库分离。
- 桌面端支持供应商连接测试。
- 运行状态接口会返回数据库诊断信息，便于灰度替换。

## 仓库结构

```text
.
├── icoo_llm_bridge_r/   # Rust 后端，当前推荐用于新构建
├── icoo_desktop/        # Wails 桌面端和 Vue 前端
├── icoo_llm_bridge/     # Go 后端，旧实现/对照版本
├── icoo_proxy/          # 打包输出目录
└── build-all.ps1        # 旧的一键构建脚本
```

## 环境要求

- Windows PowerShell
- Rust 工具链和 `cargo`
- Go 工具链
- Node.js 和 npm
- Wails CLI，用于构建桌面端

如果没有安装 Wails：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 构建 Rust 网关

```powershell
cd icoo_llm_bridge_r
.\build.ps1
```

输出文件：

```text
icoo_llm_bridge_r\build\bridge.exe
```

只想快速构建发布程序，可以跳过测试：

```powershell
.\build.ps1 -SkipTests
```

只有在确实需要时才指定 Cargo 缓存目录：

```powershell
.\build.ps1 -CargoHome "E:\cargo-cache"
```

## 运行网关

```powershell
cd icoo_llm_bridge_r
.\build\bridge.exe --addr 127.0.0.1:18181 --data-dir .data
```

健康检查：

```powershell
Invoke-RestMethod http://127.0.0.1:18181/healthz
Invoke-RestMethod http://127.0.0.1:18181/readyz
Invoke-RestMethod http://127.0.0.1:18181/api/v1/runtime/state
```

默认情况下，本机回环地址可以免 API Key 访问管理接口。不要在未配置 API Key 的情况下把服务暴露到非本机网络。

## 构建桌面端

先构建 Rust 网关，然后构建桌面端并把 `bridge.exe` 打包进去：

```powershell
cd icoo_desktop
.\build.ps1 -BridgePath ..\icoo_llm_bridge_r\build\bridge.exe
```

输出文件：

```text
icoo_desktop\build\bin\icoo_desktop.exe
icoo_desktop\build\bin\bridge.exe
```

仅开发前端时：

```powershell
cd icoo_desktop\frontend
npm install
npm run dev
```

## 配置供应商

在桌面端进入 `Provider` 页面，新建供应商并填写：

- 名称
- 协议：`openai-responses`、`openai-chat` 或 `anthropic`
- 基础地址
- API Key
- 可用模型

供应商列表里的健康检查按钮会使用第一个已启用模型发送一次最小真实上游请求。它可以验证地址和凭据是否可用，但也会消耗一次很小的模型调用额度。

## 路由规则

网关按以下顺序解析路由：

1. 直连路由，例如 `provider-name/model-name`。
2. 按优先级排序的启用路由规则。

默认协议路由可以在桌面端 `规则设置` 页面编辑。模型别名和指定模型路由可以在 `模型路由` 页面配置。

## 主要接口

代理接口：

```text
POST /v1/messages
POST /v1/chat/completions
POST /v1/responses
```

管理接口：

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

## 验证

运行 Rust 后端测试：

```powershell
cd icoo_llm_bridge_r
cargo test
```

运行前端构建：

```powershell
cd icoo_desktop\frontend
npm run build
```

真实上游验证脚本：

```powershell
cd icoo_llm_bridge_r
.\scripts\verify-real-upstream.ps1 -BridgeUrl http://127.0.0.1:18181 -ApiKey "<key>" -ResponsesModel "<model>"
.\scripts\preflight-gray-replacement.ps1 -SourceDataDir "E:\path\to\data"
```

更多文档：

- [Go/Rust parity checklist](icoo_llm_bridge_r/docs/parity/go-rust-parity-checklist.md)
- [Rust gray release runbook](icoo_llm_bridge_r/docs/release/rust-gray-release-runbook.md)
- [Real upstream verification](icoo_llm_bridge_r/docs/release/real-upstream-verification.md)

## 当前状态

Rust 网关已经用真实上游验证过以下场景：

- OpenAI Responses 到 OpenAI Responses
- OpenAI Chat 到 OpenAI Responses
- OpenAI Chat 到 Anthropic
- OpenAI Responses 到 Anthropic
- Anthropic 到 OpenAI Responses
- Anthropic 到 Anthropic
- 多轮对话
- 工具调用
- 流式响应
- 混合并发请求

当前需要注意：

- 部分供应商在并发较高时会返回 `429 Too Many Requests`。生产环境建议增加供应商级并发限制、重试或退避策略。
- Rust 灰度阶段没有包含旧数据库完整迁移。
- 部分跨协议流式转换仍会缓冲；同协议 SSE 透传已经是低延迟路径。

## 打包方式

推荐手动交付目录结构：

```text
icoo_proxy\
├── icoo_desktop.exe
└── bridge.exe
```

启动 `icoo_desktop.exe` 即可。只要 `bridge.exe` 放在同一目录，桌面端就可以唤起本地网关。

## 许可证

当前仓库还没有包含许可证文件。公开分发前请补充许可证。
