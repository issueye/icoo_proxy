# AI 本地代理网关 - 开发阶段计划文档

> **项目名称**: icoo_proxy (AI Local Proxy Gateway)
> **文档版本**: v1.0
> **创建日期**: 2026-04-14
> **关联文档**: 需求分析文档 v1.0

---

## 1. 项目概述

### 1.1 项目目标

将现有 `icoo_proxy` 桌面客户端改造为 **AI 本地代理网关**，实现多供应商管理、统一代理接口和 AI 协议互转三大核心能力。

### 1.2 当前项目资产盘点

| 资产 | 路径 | 可复用程度 | 说明 |
|------|------|-----------|------|
| Go 项目骨架 | `main.go`, `go.mod` | **高** | 保留 Wails 框架，新增网关 HTTP Server |
| HTTP 反向代理 | `internal/services/api_proxy.go` | **高** | 作为网关代理引擎的基础进行重构 |
| TOML 配置服务 | `internal/services/config.go` | **高** | 扩展配置结构，新增供应商/路由配置 |
| Wails 应用绑定 | `internal/services/app.go` | **高** | 继续作为前端与后端的桥接层 |
| Vue 3 前端框架 | `frontend/` | **高** | 保留技术栈，重构页面内容 |
| UI 组件库 | `frontend/src/components/ui/` | **高** | Button, Card, Input, Dialog 等组件直接复用 |
| 布局组件 | `frontend/src/components/layout/` | **高** | DataTable, ManagePage, PageHeader 等直接复用 |
| Settings 组件 | `frontend/src/components/settings/` | **中** | 当前仅保留已启用设置分区，未挂载旧面板应移除 |
| Pinia Store | `frontend/src/stores/` | **中** | provider.js 改造复用，新增 gateway/route store |
| HTTP 服务层 | `frontend/src/services/` | **中** | http.js 保留，API 层需重构 |
| 供应商管理页面 | `frontend/src/views/ProvidersView.vue` | **中** | 作为供应商管理页面的基础改造 |
| 主题系统 | `frontend/src/stores/theme.js` | **高** | 直接复用 |

### 1.3 技术选型确认

| 层级 | 技术 | 版本 | 说明 |
|------|------|------|------|
| 后端语言 | Go | 1.23 | 保持不变 |
| 桌面框架 | Wails | v2.11.0 | 保持不变 |
| HTTP 框架 | net/http + 自定义路由 | 标准库 | 网关核心使用标准库，轻量高效 |
| SSE 处理 | 标准库 + bufio | 标准库 | 流式响应处理 |
| 前端框架 | Vue 3 | 3.x | 保持不变 |
| 状态管理 | Pinia | 2.x | 保持不变 |
| 样式方案 | Tailwind CSS | 3.x | 保持不变 |
| UI 组件 | Radix Vue | 最新 | 保持不变 |
| 配置格式 | TOML | v2 | 保持不变 |
| 加密存储 | AES-GCM | 标准库 crypto/aes | API Key 加密 |

---

## 2. 技术架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                    icoo_proxy (Wails Desktop App)                    │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                     Vue 3 Frontend (管理界面)                  │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────────┐  │ │
│  │  │ 供应商   │ │ 路由规则 │ │ 请求日志 │ │ 网关设置        │  │ │
│  │  │ 管理     │ │ 管理     │ │ 监控     │ │ (端口/密钥/重试) │  │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────────┘  │ │
│  └───────────────────────────┬────────────────────────────────────┘ │
│                              │ Wails Bindings                       │
│  ┌───────────────────────────┴────────────────────────────────────┐ │
│  │                      Go Backend Services                       │ │
│  │                                                                │ │
│  │  ┌──────────────────────────────────────────────────────────┐  │ │
│  │  │              Gateway HTTP Server (:16790)                │  │ │
│  │  │                                                          │  │ │
│  │  │  ┌────────────┐  ┌────────────┐  ┌────────────────────┐ │  │ │
│  │  │  │  Router     │  │ Middleware │  │ Protocol Converter │ │  │ │
│  │  │  │  (路由匹配) │  │ (中间件链) │  │  (协议转换引擎)    │ │  │ │
│  │  │  └─────┬──────┘  └─────┬──────┘  └────────┬───────────┘ │  │ │
│  │  │        │               │                   │             │  │ │
│  │  │  ┌─────┴───────────────┴───────────────────┴──────────┐  │  │ │
│  │  │  │              Proxy Engine (代理引擎)                │  │  │ │
│  │  │  └──────────────────────┬─────────────────────────────┘  │  │ │
│  │  └─────────────────────────┼────────────────────────────────┘  │ │
│  │                            │                                    │ │
│  │  ┌─────────────────────────┴────────────────────────────────┐ │ │
│  │  │              Provider Manager (供应商管理器)              │ │ │
│  │  │                                                          │ │ │
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────┐  │ │ │
│  │  │  │ OpenAI   │ │Anthropic │ │  Gemini  │ │  Custom   │  │ │ │
│  │  │  │ Adapter  │ │ Adapter  │ │ Adapter  │ │  Adapter  │  │ │ │
│  │  │  └──────────┘ └──────────┘ └──────────┘ └───────────┘  │ │ │
│  │  └─────────────────────────────────────────────────────────┘ │ │
│  │                                                              │ │
│  │  ┌──────────────────┐ ┌──────────────┐ ┌────────────────┐  │ │
│  │  │  Config Service  │ │ Log Service  │ │ Route Service  │  │ │
│  │  └──────────────────┘ └──────────────┘ └────────────────┘  │ │
│  └──────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
        ┌─────┴─────┐ ┌─────┴─────┐ ┌─────┴─────┐
        │  OpenAI   │ │ Anthropic │ │  Gemini   │
        │  API      │ │  API      │ │  API      │
        └───────────┘ └───────────┘ └───────────┘
```

### 2.2 Go 后端模块划分

```
internal/
├── gateway/                    # 网关核心模块
│   ├── server.go              # HTTP 服务器，监听端口，启动网关
│   ├── router.go              # 路由匹配引擎，model → provider 映射
│   ├── handler.go             # 请求处理器，/v1/chat/completions 等
│   ├── middleware.go          # 中间件链（日志、限流、认证、CORS）
│   └── proxy_engine.go        # 代理引擎，请求转发与响应回传
│
├── provider/                   # 供应商管理模块
│   ├── manager.go             # 供应商生命周期管理（CRUD、启停、健康检查）
│   ├── registry.go            # 供应商注册表，运行时状态维护
│   └── health_checker.go      # 供应商健康检查
│
├── protocol/                   # 协议转换模块
│   ├── converter.go           # 转换引擎入口（统一接口）
│   ├── types.go               # 内部统一消息类型定义
│   ├── openai/                # OpenAI 协议适配
│   │   ├── adapter.go         # OpenAI 适配器（实现 ProtocolAdapter 接口）
│   │   ├── request.go         # OpenAI 请求结构体与解析
│   │   ├── response.go        # OpenAI 响应结构体与生成
│   │   └── stream.go          # OpenAI SSE 流式处理
│   ├── anthropic/             # Anthropic 协议适配
│   │   ├── adapter.go
│   │   ├── request.go
│   │   ├── response.go
│   │   └── stream.go
│   ├── gemini/                # Google Gemini 协议适配
│   │   ├── adapter.go
│   │   ├── request.go
│   │   ├── response.go
│   │   └── stream.go
│   └── mapping.go             # 字段映射规则与转换辅助函数
│
├── config/                     # 配置管理模块
│   ├── config.go              # 配置结构定义与文件读写
│   ├── encrypt.go             # API Key 加密/解密
│   └── watcher.go             # 配置文件变更监听（热加载）
│
├── route/                      # 路由规则模块
│   ├── rule.go                # 路由规则数据结构
│   ├── matcher.go             # 模式匹配引擎（精确/前缀/正则）
│   └── manager.go             # 路由规则 CRUD
│
├── log/                        # 日志与监控模块
│   ├── logger.go              # 请求日志记录器
│   ├── usage.go               # Token 用量统计
│   └── store.go               # 日志存储（内存环形缓冲 + 文件）
│
└── services/                   # 现有服务（保留并改造）
    ├── app.go                 # Wails 应用绑定（扩展网关控制方法）
    ├── config.go              # 保留，与 config 模块整合
    └── api_proxy.go           # 改造为网关代理引擎的一部分
```

### 2.3 前端模块划分

```
frontend/src/
├── views/
│   ├── ProvidersView.vue       # 供应商管理页面（改造）
│   ├── RoutesView.vue          # 路由规则管理页面（新增）
│   ├── LogsView.vue            # 请求日志与监控页面（新增）
│   ├── GatewayView.vue         # 网关状态总览页面（新增）
│   └── SettingsView.vue        # 设置页面（改造）
│
├── components/
│   ├── gateway/                # 网关相关组件（新增）
│   │   ├── GatewayStatus.vue        # 网关运行状态显示
│   │   ├── GatewayControl.vue       # 启动/停止/重启控制
│   │   └── StatsCards.vue           # 统计数据卡片
│   ├── provider/               # 供应商相关组件（新增）
│   │   ├── ProviderForm.vue         # 供应商配置表单
│   │   ├── ProviderCard.vue         # 供应商信息卡片
│   │   ├── ProviderTestResult.vue   # 连通性测试结果
│   │   └── ProviderTemplateSelect.vue # 预置模板选择
│   ├── route/                  # 路由相关组件（新增）
│   │   ├── RouteRuleForm.vue        # 路由规则表单
│   │   └── RouteRuleTable.vue       # 路由规则列表
│   ├── logs/                   # 日志相关组件（新增）
│   │   ├── LogTable.vue             # 请求日志表格
│   │   ├── LogDetail.vue            # 日志详情弹窗
│   │   └── UsageChart.vue           # 用量统计图表
│   ├── layout/                 # 布局组件（保留）
│   └── ui/                     # UI 基础组件（保留）
│
├── stores/
│   ├── provider.js             # 供应商状态（改造）
│   ├── route.js                # 路由规则状态（新增）
│   ├── gateway.js              # 网关状态（新增）
│   ├── log.js                  # 日志状态（新增）
│   └── theme.js                # 主题状态（保留）
│
├── services/
│   ├── http.js                 # HTTP 客户端（保留）
│   ├── wails.js                # Wails 桥接（保留）
│   ├── unified-api.js          # 统一 API 适配（保留）
│   ├── gateway-api.js          # 网关 API（新增）
│   ├── provider-api.js         # 供应商 API（改造）
│   ├── route-api.js            # 路由规则 API（新增）
│   └── log-api.js              # 日志 API（新增）
```

### 2.4 关键数据流

#### 2.4.1 请求代理数据流

```
外部客户端
    │
    │ POST /v1/chat/completions (OpenAI 格式)
    ▼
Gateway HTTP Server (:16790)
    │
    ├─→ Middleware: CORS → Logger → RateLimiter
    │
    ├─→ Router: 解析 model="claude-sonnet-4-20250514"
    │       │
    │       └─→ 匹配路由规则 → provider = "anthropic-main"
    │
    ├─→ Protocol Converter:
    │       │
    │       ├─→ OpenAI Request → InternalMessage (内部统一格式)
    │       └─→ InternalMessage → Anthropic Request
    │
    ├─→ Proxy Engine:
    │       │
    │       └─→ 添加 Anthropic API Key → 发送 HTTP 请求 → 接收响应
    │
    ├─→ Protocol Converter (响应方向):
    │       │
    │       ├─→ Anthropic Response → InternalMessage
    │       └─→ InternalMessage → OpenAI Response
    │
    └─→ 返回 OpenAI 格式响应给客户端
```

#### 2.4.2 流式响应数据流

```
供应商 SSE Stream
    │
    ▼
io.Pipe (流式管道，零缓冲)
    │
    ├─→ 逐行读取 SSE data
    ├─→ 解析供应商格式 (如 Anthropic content_block_delta)
    ├─→ 转换为 OpenAI 格式 (chat.completion.chunk)
    ├─→ 写入 Response Writer (SSE 格式输出)
    │
    ▼
客户端接收 SSE Stream
```

### 2.5 关键接口定义

```go
// ProtocolAdapter 协议适配器接口
type ProtocolAdapter interface {
    // ParseRequest 将供应商特定请求解析为内部统一格式
    ParseRequest(body []byte) (*InternalRequest, error)

    // BuildRequest 将内部统一格式构建为供应商特定请求
    BuildRequest(req *InternalRequest) ([]byte, error)

    // ParseResponse 将供应商特定响应解析为内部统一格式
    ParseResponse(body []byte) (*InternalResponse, error)

    // BuildResponse 将内部统一格式构建为供应商特定响应
    BuildResponse(resp *InternalResponse) ([]byte, error)

    // ParseStreamEvent 解析单个 SSE 流式事件
    ParseStreamEvent(data []byte) (*InternalStreamChunk, error)

    // BuildStreamEvent 构建单个 SSE 流式事件
    BuildStreamEvent(chunk *InternalStreamChunk) (string, error)

    // StreamDone 返回流结束标记
    StreamDone() string
}

// ProviderAdapter 供应商适配器接口
type ProviderAdapter interface {
    // GetProtocolType 返回供应商使用的协议类型
    GetProtocolType() string

    // BuildHTTPRequest 构建发往供应商的 HTTP 请求
    BuildHTTPRequest(ctx context.Context, provider *Provider, body []byte) (*http.Request, error)

    // HealthCheck 供应商健康检查
    HealthCheck(ctx context.Context, provider *Provider) error

    // ListModels 获取供应商可用模型列表
    ListModels(ctx context.Context, provider *Provider) ([]ModelInfo, error)
}
```

---

## 3. 开发阶段划分

### 阶段一：基础架构重构

> **目标**：建立网关核心框架，实现基础代理能力

#### 任务清单

| 编号 | 任务 | 涉及文件 | 说明 |
|------|------|---------|------|
| T1-01 | 创建网关 HTTP Server | `internal/gateway/server.go` | 监听指定端口（默认 16790），启动独立 HTTP 服务器 |
| T1-02 | 实现请求路由分发 | `internal/gateway/router.go` | 将 `/v1/chat/completions`、`/v1/models` 等路由到对应 handler |
| T1-03 | 实现中间件框架 | `internal/gateway/middleware.go` | 可链式调用的中间件模式（Logger、Recovery、CORS） |
| T1-04 | 重构代理引擎 | `internal/gateway/proxy_engine.go` | 基于 api_proxy.go 重构，支持动态目标、连接池、超时管理 |
| T1-05 | 定义内部统一消息类型 | `internal/protocol/types.go` | InternalRequest、InternalResponse、InternalStreamChunk |
| T1-06 | 实现请求 handler | `internal/gateway/handler.go` | chatCompletionsHandler、modelsHandler、healthHandler |
| T1-07 | 扩展配置结构 | `internal/config/config.go` | 新增 [gateway]、[[providers]]、[[route_rules]] 配置段 |
| T1-08 | 实现配置热加载 | `internal/config/watcher.go` | 监听 TOML 文件变更，触发回调重新加载 |
| T1-09 | 集成网关到 Wails 生命周期 | `internal/services/app.go` | Startup 时启动网关，Shutdown 时优雅关闭 |
| T1-10 | 更新 main.go | `main.go` | 调整 assetHandler，将网关端口配置传入 |

#### 技术要点

1. **网关独立端口**：网关 HTTP Server 监听独立端口（16790），与 Wails WebView 的资源服务分离
2. **中间件链设计**：采用 `func(http.Handler) http.Handler` 模式，支持灵活组合
3. **优雅关闭**：使用 `http.Server.Shutdown()` 确保在途请求完成

#### 验收标准

- [ ] 网关 HTTP Server 可独立启动并监听端口
- [ ] `/v1/health` 返回 `{"status": "ok"}`
- [ ] `/v1/chat/completions` 可将请求透传到配置的单个供应商
- [ ] 支持 SSE 流式透传
- [ ] 配置文件修改后网关自动感知（无需重启）
- [ ] Wails 应用启动时网关自动启动

---

### 阶段二：供应商管理系统

> **目标**：实现完整的供应商 CRUD、配置管理和健康检查

#### 任务清单

| 编号 | 任务 | 涉及文件 | 说明 |
|------|------|---------|------|
| T2-01 | 实现供应商管理器 | `internal/provider/manager.go` | 供应商 CRUD、缓存、线程安全读写 |
| T2-02 | 实现供应商注册表 | `internal/provider/registry.go` | 运行时供应商状态维护（活跃/不可用/已禁用） |
| T2-03 | 实现健康检查器 | `internal/provider/health_checker.go` | 定时/按需检查供应商可用性 |
| T2-04 | 实现 API Key 加密 | `internal/config/encrypt.go` | AES-GCM 加密存储，启动时用机器特征密钥解密 |
| T2-05 | 实现预置供应商模板 | `internal/provider/templates.go` | 内置主流供应商的默认配置（API Base、请求头格式） |
| T2-06 | 扩展 Wails 绑定方法 | `internal/services/app.go` | 新增 GetProviders、AddProvider、UpdateProvider 等方法 |
| T2-07 | 实现供应商 API 服务层 | `frontend/src/services/provider-api.js` | 前端 API 调用封装 |
| T2-08 | 改造供应商 Pinia Store | `frontend/src/stores/provider.js` | 供应商列表、当前编辑、加载状态 |
| T2-09 | 改造供应商管理页面 | `frontend/src/views/ProvidersView.vue` | 使用 ManagePage 布局组件重构 |
| T2-10 | 实现供应商表单组件 | `frontend/src/components/provider/ProviderForm.vue` | 供应商配置编辑表单 |
| T2-11 | 实现供应商卡片组件 | `frontend/src/components/provider/ProviderCard.vue` | 供应商信息展示与操作 |
| T2-12 | 实现连通性测试组件 | `frontend/src/components/provider/ProviderTestResult.vue` | 测试连接结果显示 |
| T2-13 | 实现模板选择组件 | `frontend/src/components/provider/ProviderTemplateSelect.vue` | 预置供应商模板快速选择 |

#### 技术要点

1. **加密方案**：使用机器特征（如主机名 + 用户名哈希）生成 AES 密钥，对 API Key 进行 AES-GCM 加密
2. **健康检查策略**：启动时全量检查 + 定时轮询（可配置间隔） + 手动触发
3. **供应商状态机**：`unknown → healthy → unhealthy → disabled`

#### 验收标准

- [ ] 可通过 UI 添加、编辑、删除、启停供应商
- [ ] API Key 在配置文件中加密存储
- [ ] 连通性测试功能正常工作
- [ ] 预置模板可快速创建常见供应商
- [ ] 自定义供应商可配置任意 API Base

---

### 阶段三：协议转换引擎（核心）

> **目标**：实现 OpenAI ↔ Anthropic、OpenAI ↔ Gemini 的完整协议互转

#### 任务清单

| 编号 | 任务 | 涉及文件 | 说明 |
|------|------|---------|------|
| T3-01 | 定义 ProtocolAdapter 接口 | `internal/protocol/converter.go` | 统一转换接口 |
| T3-02 | 实现内部统一消息结构 | `internal/protocol/types.go` | 覆盖 text、image、tool_call 等所有内容类型 |
| T3-03 | 实现 OpenAI 适配器 | `internal/protocol/openai/adapter.go` | OpenAI 格式 ↔ InternalMessage |
| T3-04 | OpenAI 请求/响应结构体 | `internal/protocol/openai/request.go`, `response.go` | 完整的 OpenAI API 类型定义 |
| T3-05 | OpenAI SSE 流处理 | `internal/protocol/openai/stream.go` | 解析/生成 OpenAI SSE 格式 |
| T3-06 | 实现 Anthropic 适配器 | `internal/protocol/anthropic/adapter.go` | Anthropic 格式 ↔ InternalMessage |
| T3-07 | Anthropic 请求/响应结构体 | `internal/protocol/anthropic/request.go`, `response.go` | 完整的 Anthropic API 类型定义 |
| T3-08 | Anthropic SSE 流处理 | `internal/protocol/anthropic/stream.go` | 解析/生成 Anthropic SSE 格式 |
| T3-09 | 实现 Gemini 适配器 | `internal/protocol/gemini/adapter.go` | Gemini 格式 ↔ InternalMessage |
| T3-10 | Gemini 请求/响应结构体 | `internal/protocol/gemini/request.go`, `response.go` | 完整的 Gemini API 类型定义 |
| T3-11 | Gemini SSE 流处理 | `internal/protocol/gemini/stream.go` | 解析/生成 Gemini SSE 格式 |
| T3-12 | 实现字段映射规则 | `internal/protocol/mapping.go` | system prompt、tool calling、多模态等特殊字段映射 |
| T3-13 | 实现转换引擎集成 | `internal/protocol/converter.go` | 根据 (源协议, 目标协议) 选择转换路径 |
| T3-14 | 编写协议转换单元测试 | `internal/protocol/*_test.go` | 覆盖所有转换路径的测试用例 |

#### 技术要点

1. **转换路径优化**：OpenAI 作为基准格式，所有协议先转为 InternalMessage 再转为目标协议。即 `A → Internal → B`，而非 `A → B` 的直接转换
2. **流式处理使用 io.Pipe**：避免全量缓冲，实现逐 token 转发
3. **Tool Calling 映射**：三种协议的 tool calling 格式差异较大，需要特别注意：
   - OpenAI: `tools[].function` + `tool_calls[].function`
   - Anthropic: `tools[].input_schema` + `content[].type="tool_use"`
   - Gemini: `tools[].functionDeclarations` + `functionCall`
4. **System Prompt 处理**：Anthropic 单独的 `system` 字段 vs OpenAI 的 `messages[role=system]`

#### 验收标准

- [ ] OpenAI 格式请求 → Anthropic 供应商 → OpenAI 格式响应（非流式）
- [ ] OpenAI 格式请求 → Gemini 供应商 → OpenAI 格式响应（非流式）
- [ ] OpenAI 格式请求 → Anthropic 供应商 → OpenAI 格式响应（流式 SSE）
- [ ] OpenAI 格式请求 → Gemini 供应商 → OpenAI 格式响应（流式 SSE）
- [ ] Tool Calling 消息在三种协议间正确转换
- [ ] 多模态（图片）消息正确转换
- [ ] 错误响应统一转换为 OpenAI 错误格式
- [ ] 单元测试覆盖率 > 80%

---

### 阶段四：统一 API 网关与路由策略

> **目标**：实现智能路由、故障转移和完整的网关功能

#### 任务清单

| 编号 | 任务 | 涉及文件 | 说明 |
|------|------|---------|------|
| T4-01 | 实现路由规则匹配引擎 | `internal/route/matcher.go` | 支持精确匹配、前缀匹配、正则匹配 |
| T4-02 | 实现路由规则管理 | `internal/route/manager.go` | 路由规则 CRUD，优先级排序 |
| T4-03 | 实现模型自动发现路由 | `internal/gateway/router.go` | 根据供应商类型自动匹配模型名前缀 |
| T4-04 | 实现故障转移机制 | `internal/gateway/proxy_engine.go` | 请求失败时自动切换到备选供应商 |
| T4-05 | 实现请求重试 | `internal/gateway/proxy_engine.go` | 可配置重试次数和间隔 |
| T4-06 | 实现 `/v1/models` 聚合接口 | `internal/gateway/handler.go` | 聚合所有供应商的模型列表 |
| T4-07 | 实现错误统一处理 | `internal/gateway/handler.go` | 所有供应商错误统一为 OpenAI error 格式 |
| T4-08 | 实现路由规则 API | `internal/services/app.go` | Wails 绑定路由规则操作 |
| T4-09 | 实现路由 API 服务层 | `frontend/src/services/route-api.js` | 前端 API 封装 |
| T4-10 | 实现路由 Pinia Store | `frontend/src/stores/route.js` | 路由规则状态管理 |
| T4-11 | 实现路由规则管理页面 | `frontend/src/views/RoutesView.vue` | 路由规则的 CRUD 界面 |
| T4-12 | 实现路由规则表单 | `frontend/src/components/route/RouteRuleForm.vue` | 规则编辑表单 |
| T4-13 | 实现路由规则列表 | `frontend/src/components/route/RouteRuleTable.vue` | 规则列表与排序 |

#### 技术要点

1. **路由匹配优先级**：自定义规则（优先级值） > 前缀匹配 > 默认路由
2. **故障转移策略**：同一模型在多供应商可用时，按优先级尝试，失败后自动 fallback
3. **模型列表聚合**：定期从各供应商拉取模型列表，合并去重，标记来源供应商

#### 验收标准

- [ ] 模型名 `gpt-4o` 自动路由到 OpenAI 供应商
- [ ] 模型名 `claude-*` 自动路由到 Anthropic 供应商
- [ ] 自定义路由规则可覆盖默认匹配
- [ ] 主供应商失败时自动切换到备选供应商
- [ ] `/v1/models` 返回所有供应商的模型聚合列表
- [ ] 第三方应用（如 Cursor）可通过网关正常调用

---

### 阶段五：管理界面改造

> **目标**：完成前端管理界面的全面改造，提供网关总览和日志监控

#### 任务清单

| 编号 | 任务 | 涉及文件 | 说明 |
|------|------|---------|------|
| T5-01 | 实现网关状态 Store | `frontend/src/stores/gateway.js` | 网关运行状态、统计信息 |
| T5-02 | 实现网关总览页面 | `frontend/src/views/GatewayView.vue` | 网关状态、供应商概览、统计卡片 |
| T5-03 | 实现网关控制组件 | `frontend/src/components/gateway/GatewayControl.vue` | 启动/停止/重启网关 |
| T5-04 | 实现网关状态组件 | `frontend/src/components/gateway/GatewayStatus.vue` | 运行状态、监听端口、连接数 |
| T5-05 | 实现统计卡片组件 | `frontend/src/components/gateway/StatsCards.vue` | 请求数、成功率、平均延迟、Token 用量 |
| T5-06 | 实现请求日志 Store | `frontend/src/stores/log.js` | 日志列表、筛选条件、分页 |
| T5-07 | 实现请求日志页面 | `frontend/src/views/LogsView.vue` | 请求日志查询与展示 |
| T5-08 | 实现日志表格组件 | `frontend/src/components/logs/LogTable.vue` | 日志列表表格 |
| T5-09 | 实现日志详情组件 | `frontend/src/components/logs/LogDetail.vue` | 单条请求的详细信息 |
| T5-10 | 实现用量统计组件 | `frontend/src/components/logs/UsageChart.vue` | Token 用量图表 |
| T5-11 | 改造设置页面 | `frontend/src/views/SettingsView.vue` | 新增网关设置项（端口、重试、日志级别等） |
| T5-12 | 改造侧边导航 | `frontend/src/components/ChatSidebar.vue` | 更新导航项（网关、供应商、路由、日志、设置） |
| T5-13 | 改造前端路由 | `frontend/src/router/index.js` | 新增网关、路由规则、日志页面路由 |

#### 验收标准

- [ ] 网关总览页面展示运行状态和关键指标
- [ ] 可在 UI 中启动/停止网关
- [ ] 供应商管理页面完整可用（CRUD、测试、模型列表）
- [ ] 路由规则管理页面完整可用
- [ ] 请求日志页面可按条件查询和查看详情
- [ ] 设置页面可配置网关参数

---

### 阶段六：高级功能

> **目标**：添加限流、负载均衡、缓存等增强功能，提升网关生产可用性

#### 任务清单

| 编号 | 任务 | 涉及文件 | 说明 |
|------|------|---------|------|
| T6-01 | 实现请求限流 | `internal/gateway/middleware.go` | 令牌桶/滑动窗口限流，保护供应商配额 |
| T6-02 | 实现负载均衡 | `internal/gateway/router.go` | 同模型多供应商的轮询/加权/最少连接策略 |
| T6-03 | 实现响应缓存 | `internal/gateway/cache.go` | 对相同请求缓存响应（可选开启） |
| T6-04 | 实现请求/响应 Hook | `internal/gateway/hooks.go` | 可扩展的请求预处理和响应后处理 |
| T6-05 | 实现详细审计日志 | `internal/log/logger.go` | 可配置的详细日志记录（包含完整请求/响应体） |
| T6-06 | 实现日志持久化与轮转 | `internal/log/store.go` | 日志写入文件，按大小/时间轮转 |
| T6-07 | 实现 Token 用量统计与配额 | `internal/log/usage.go` | 按供应商、模型、日期统计 Token 用量 |
| T6-08 | 实现 Embeddings 接口代理 | `internal/gateway/handler.go` | `/v1/embeddings` 接口 |
| T6-09 | 实现 Azure OpenAI 适配器 | `internal/protocol/azure/adapter.go` | Azure 特有的部署名/版本号格式 |
| T6-10 | 实现 AWS Bedrock 适配器 | `internal/protocol/bedrock/adapter.go` | AWS SigV4 签名认证 |
| T6-11 | 实现导出/导入配置 | `internal/config/config.go` | 支持导出完整配置和导入恢复 |
| T6-12 | 实现系统托盘集成 | `internal/services/tray.go` | 最小化到系统托盘，后台运行 |

#### 验收标准

- [ ] 限流功能可有效控制请求速率
- [ ] 多供应商负载均衡正常工作
- [ ] 日志可持久化到文件并自动轮转
- [ ] Token 用量统计准确
- [ ] Azure OpenAI 和 Bedrock 适配器可正常工作
- [ ] 配置可导出和导入

---

## 4. 关键技术方案

### 4.1 协议转换策略

**核心设计原则：以 InternalMessage 为中间格式**

```
源协议 → InternalMessage → 目标协议
```

所有协议都通过 InternalMessage 进行间接转换，而非两两直接转换。这样新增一种协议只需实现 `ProtocolAdapter` 接口，N 种协议只需 N 个适配器，而非 N*(N-1) 个。

**InternalMessage 结构定义：**

```go
type InternalMessage struct {
    Model       string
    Messages    []Message
    System      string              // system prompt
    Temperature *float64
    MaxTokens   *int
    Stream      bool
    Tools       []Tool
}

type Message struct {
    Role    string          // system, user, assistant, tool
    Content []ContentBlock  // 支持多内容块（文本、图片、工具调用等）
}

type ContentBlock struct {
    Type     string      // text, image, tool_use, tool_result
    Text     string
    ImageURL string
    ToolUse  *ToolUse
    ToolResult *ToolResult
}
```

### 4.2 流式响应处理方案

使用 `io.Pipe` 实现零缓冲流式转发：

```go
func (e *ProxyEngine) streamProxy(ctx context.Context, resp http.ResponseWriter, upstreamResp *http.Response, adapter ProtocolAdapter) {
    flusher, ok := resp.(http.Flusher)
    if !ok {
        // fallback to non-stream
        return
    }

    // 设置 SSE 响应头
    resp.Header().Set("Content-Type", "text/event-stream")
    resp.Header().Set("Cache-Control", "no-cache")
    resp.Header().Set("Connection", "keep-alive")

    scanner := bufio.NewScanner(upstreamResp.Body)
    for scanner.Scan() {
        line := scanner.Text()
        if !strings.HasPrefix(line, "data: ") {
            continue
        }
        data := strings.TrimPrefix(line, "data: ")
        if data == "[DONE]" || data == "" {
            // 转换各协议的结束标记为 OpenAI 的 [DONE]
            fmt.Fprintf(resp, "data: [DONE]\n\n")
            flusher.Flush()
            break
        }

        // 解析供应商 SSE 事件 → 转换 → 输出 OpenAI SSE
        chunk, err := adapter.ParseStreamEvent([]byte(data))
        if err != nil {
            continue
        }
        openaiEvent, err := openaiAdapter.BuildStreamEvent(chunk)
        if err != nil {
            continue
        }
        fmt.Fprintf(resp, "data: %s\n\n", openaiEvent)
        flusher.Flush()
    }
}
```

### 4.3 错误处理与重试机制

```go
type RetryConfig struct {
    MaxRetries    int           // 最大重试次数
    RetryInterval time.Duration // 重试间隔
    RetryOnStatus []int         // 触发重试的 HTTP 状态码
}

func (e *ProxyEngine) doRequestWithRetry(ctx context.Context, req *http.Request, cfg RetryConfig) (*http.Response, error) {
    var lastErr error
    for i := 0; i <= cfg.MaxRetries; i++ {
        resp, err := e.httpClient.Do(req)
        if err != nil {
            lastErr = err
            time.Sleep(cfg.RetryInterval)
            continue
        }
        if slices.Contains(cfg.RetryOnStatus, resp.StatusCode) {
            lastErr = fmt.Errorf("status %d", resp.StatusCode)
            resp.Body.Close()
            time.Sleep(cfg.RetryInterval)
            continue
        }
        return resp, nil
    }
    return nil, fmt.Errorf("after %d retries: %w", cfg.MaxRetries, lastErr)
}
```

### 4.4 配置热加载

使用 `fsnotify` 监听配置文件变更：

```go
func (w *ConfigWatcher) Watch() {
    watcher, _ := fsnotify.NewWatcher()
    watcher.Add(w.configPath)

    for {
        select {
        case event, ok := <-watcher.Events:
            if !ok { return }
            if event.Has(fsnotify.Write) {
                newCfg, err := config.Load(w.configPath)
                if err == nil {
                    w.onChange(newCfg) // 回调通知
                }
            }
        case <-w.ctx.Done():
            watcher.Close()
            return
        }
    }
}
```

---

## 5. 风险与应对策略

| 风险 | 可能性 | 影响 | 应对策略 |
|------|--------|------|---------|
| 协议格式变更 | 中 | 高 | 每个协议适配器独立版本化，快速发布更新 |
| 流式转换性能瓶颈 | 低 | 中 | 使用 io.Pipe 流式处理，逐行转换，避免缓冲 |
| 供应商 API 限流 | 高 | 中 | 实现故障转移到备选供应商，配置重试策略 |
| Tool Calling 字段不对齐 | 中 | 中 | 建立完整映射表，不支持的参数记录告警日志并跳过 |
| Wails 框架兼容性问题 | 低 | 低 | 网关核心不依赖 Wails，可独立作为 Go 服务运行 |
| 前端状态管理复杂度 | 中 | 低 | 使用 Pinia store 拆分，每个功能模块独立 store |

---

## 6. 质量保证计划

### 6.1 测试策略

| 测试类型 | 覆盖范围 | 工具 |
|---------|---------|------|
| 单元测试 | 协议转换、路由匹配、配置解析 | Go testing + testify |
| 集成测试 | 端到端请求转发（使用 mock 服务器） | httptest.Server |
| 前端测试 | 组件渲染、用户交互 | Vitest + Vue Test Utils |
| 手动测试 | 实际供应商对接、UI 流程 | 测试 checklist |

### 6.2 代码质量

| 指标 | 目标 |
|------|------|
| 协议转换模块单元测试覆盖率 | > 80% |
| Go 代码 go vet 零告警 | 100% |
| 关键函数注释覆盖率 | 100% |
| API 兼容性测试 | 每个协议至少 10 个转换测试用例 |

---

## 7. 部署与发布计划

### 7.1 构建产物

| 产物 | 说明 |
|------|------|
| `icoo_proxy.exe` | Windows 桌面应用（Wails 构建），包含网关 + 管理界面 |
| `icoo_proxy.toml` | 默认配置文件模板 |

### 7.2 发布策略

1. **版本号**：遵循语义化版本 `v{major}.{minor}.{patch}`
2. **配置迁移**：新版配置文件向下兼容，新增字段提供默认值
3. **更新日志**：每个版本附带 CHANGELOG

### 7.3 兼容性声明

| 外部兼容 | 说明 |
|---------|------|
| OpenAI API | 兼容 Chat Completions API v1（2024-01+） |
| Anthropic API | 兼容 Messages API v1 |
| Google Gemini API | 兼容 v1beta |
| 客户端工具 | 兼容所有使用 OpenAI SDK 或 OpenAI 兼容格式的工具 |
