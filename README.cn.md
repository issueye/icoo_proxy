# icoo_proxy

本地优先的 LLM API 转换网关和桌面管理端。

`icoo_proxy` 对下游暴露 OpenAI Chat Completions、OpenAI Responses、Anthropic Messages 兼容接口，并根据供应商、模型和路由策略把请求转发到不同上游。后端服务位于 `icoo_llm_bridge`，桌面端位于 `icoo_desktop`。

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
- 桌面端提供网关概览、供应商健康检查、路由规则、自定义端点、本地授权 Key、流量查看和运行参数配置。

## 仓库结构

```text
.
├── icoo_desktop/        # Wails 桌面端和 Vue 前端
├── icoo_llm_bridge/     # Go 后端服务
├── icoo_proxy/          # 打包输出目录
└── build-all.ps1        # 一键构建脚本
```

## 环境要求

- Windows PowerShell
- Go 工具链
- Node.js 和 npm
- Wails CLI，用于构建桌面端

如果没有安装 Wails：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 构建网关

```powershell
cd icoo_llm_bridge
.\build.ps1
```

输出文件：

```text
icoo_llm_bridge\build\bridge.exe
```

只想快速构建发布程序，可以跳过测试：

```powershell
.\build.ps1 -SkipTests
```

## 运行网关

```powershell
cd icoo_llm_bridge
.\build\bridge.exe
```

健康检查：

```powershell
Invoke-RestMethod http://127.0.0.1:18181/healthz
Invoke-RestMethod http://127.0.0.1:18181/readyz
Invoke-RestMethod http://127.0.0.1:18181/api/v1/runtime/state
```

默认情况下，本机回环地址可以免 API Key 访问管理接口。不要在未配置 API Key 的情况下把服务暴露到非本机网络。

## 构建桌面端

先构建网关，然后构建桌面端并把 `bridge.exe` 打包进去：

```powershell
cd icoo_desktop
.\build.ps1 -BridgePath ..\icoo_llm_bridge\build\bridge.exe
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

## 桌面管理端

桌面管理端是本地运行 `icoo_proxy` 的推荐入口。它可以启动或重启打包后的网关，显示当前连接状态，并通过界面管理供应商、路由、端点、授权 Key、流量和运行参数。

![网关概览](images/index.png)

网关概览页会显示监听地址、访问模式、供应商数量、启用策略数量、上游就绪状态和支持的入口路径。下面截图使用的是 `127.0.0.1:18181` 的本地信任模式。

### Provider 管理

![Provider 管理](images/provider.png)

`Provider` 页面用于搜索和筛选上游供应商，新建供应商，编辑凭据和基础地址，执行健康检查，查看已启用模型，以及删除供应商。列表会展示协议、地址、启用状态和 `only_stream` 等运行标签。

### 路由规则

![路由规则](images/rules.png)

`规则设置` 页面用于把不同下游协议映射到指定上游供应商和上游协议。当前内置展示 Anthropic Messages、OpenAI Chat、OpenAI Responses 三类下游协议，并标明每条规则是已启用、待选择还是未配置。

### 入口端点

![入口端点](images/endpoint.png)

`端点` 页面用于查看或新增入口路径。默认启用的端点包括：

- `/v1/chat/completions`：兼容 OpenAI Chat 客户端。
- `/v1/messages`：兼容 Anthropic Messages 客户端。
- `/v1/responses` 和 `/responses`：兼容 OpenAI Responses 客户端。

### 授权 Key

![授权 Key](images/keys.png)

`授权 Key` 页面用于创建本地客户端访问凭据。客户端可以通过 `Bearer` token 或 `x-api-key` 请求头鉴权。未配置 Key 且服务仅绑定本机时，本地信任模式仍可让本机管理操作生效。

### 流量监控

![流量监控](images/traffic.png)

`流量监控` 页面用于查看最近代理请求、成功和错误数量、平均耗时、Token 汇总、端点、下游和上游协议、命中的路由规则、状态码和单次请求耗时。页面支持按协议筛选、手动刷新、自动刷新和清空请求记录。

### 项目设置

![项目设置](images/settings.png)

`项目设置` 页面用于调整控制台外观、按钮密度，以及网关运行参数，例如监听地址、端口、读写超时、关闭超时、默认最大 Token 和链路日志参数。修改后可以通过右上角保存并重载让配置生效。

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

运行后端测试：

```powershell
cd icoo_llm_bridge
go test ./...
```

运行前端构建：

```powershell
cd icoo_desktop\frontend
npm run build
```

## 当前状态

网关支持以下场景：

- OpenAI Responses 到 OpenAI Responses
- OpenAI Chat 到 OpenAI Responses
- OpenAI Chat 到 Anthropic
- OpenAI Responses 到 Anthropic
- Anthropic 到 OpenAI Responses
- Anthropic 到 Anthropic
- 多轮对话
- 工具调用
- 流式响应
- 流量记录和桌面端查看

当前需要注意：

- 部分供应商在并发较高时会返回 `429 Too Many Requests`。生产环境建议增加供应商级并发限制、重试或退避策略。
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
