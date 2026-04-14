# AI 本地代理网关 - 需求分析文档

> **项目名称**: icoo_proxy (AI Local Proxy Gateway)
> **文档版本**: v1.0
> **创建日期**: 2026-04-14
> **项目状态**: 需求分析阶段

---

## 1. 项目概述与背景

### 1.1 项目背景

当前 AI 应用生态中存在多个主流服务供应商（OpenAI、Anthropic、Google、Azure、AWS Bedrock 等），各供应商采用不同的 API 协议与数据格式。这导致以下问题：

- **接入成本高**：每个 AI 应用需针对不同供应商分别实现对接逻辑
- **协议不兼容**：OpenAI、Anthropic、Google Gemini 等协议之间存在格式差异，无法互通
- **管理分散**：API Key、用量、配置散落在各处，缺乏统一管理
- **切换困难**：更换供应商需要修改应用代码，缺乏灵活的切换机制

### 1.2 项目目标

将现有的 `icoo_proxy`（Wails 桌面应用 + AI 聊天客户端）改造为一个 **AI 本地代理网关**，实现：

1. **统一供应商管理**：集中管理多个 AI 供应商的连接配置、API Key、模型列表
2. **统一代理口径**：对外提供一个与 OpenAI API 兼容的标准接口，下游应用无需修改即可使用
3. **协议互转**：在 OpenAI、Anthropic、Google Gemini 等不同 AI 协议之间进行透明转换

### 1.3 项目定位

| 维度 | 描述 |
|------|------|
| 运行模式 | 本地桌面应用 (Wails) + 独立代理服务 |
| 目标用户 | AI 应用开发者、研究人员、个人用户 |
| 核心价值 | 一个入口、多供应商、协议互通 |
| 技术栈 | Go 1.23 + Wails v2 + Vue 3 + Pinia + Tailwind CSS |

---

## 2. 核心需求分析

### 2.1 供应商管理（Provider Management）

#### 2.1.1 功能描述

提供对 AI 服务供应商的完整生命周期管理，包括供应商的注册、配置、健康检查和注销。

#### 2.1.2 详细需求

| 需求编号 | 需求名称 | 优先级 | 描述 |
|----------|---------|--------|------|
| PM-001 | 供应商注册 | P0 | 支持添加新的 AI 供应商，配置名称、API Base URL、API Key、协议类型 |
| PM-002 | 供应商编辑 | P0 | 支持修改已注册供应商的所有配置项 |
| PM-003 | 供应商删除 | P0 | 支持删除不再使用的供应商配置 |
| PM-004 | 供应商启停 | P1 | 支持启用/禁用供应商，禁用后不再参与路由 |
| PM-005 | 连通性测试 | P0 | 提供一键测试供应商连接是否正常的能力 |
| PM-006 | 模型列表同步 | P0 | 自动或手动从供应商拉取可用模型列表 |
| PM-007 | 预置供应商模板 | P1 | 内置主流供应商的默认配置模板（OpenAI、Anthropic、Google、Azure、Ollama 等） |
| PM-008 | 供应商分组 | P2 | 支持将供应商按标签/分组进行管理 |

#### 2.1.3 支持的供应商类型

| 供应商 | 协议类型 | API Base URL 示例 | 备注 |
|--------|---------|-------------------|------|
| OpenAI | openai | `https://api.openai.com/v1` | Chat Completions API |
| Anthropic | anthropic | `https://api.anthropic.com/v1` | Messages API |
| Google Gemini | gemini | `https://generativelanguage.googleapis.com/v1beta` | generateContent API |
| Azure OpenAI | azure-openai | `https://{resource}.openai.azure.com/openai` | Azure 部署格式 |
| AWS Bedrock | bedrock | `https://bedrock-runtime.{region}.amazonaws.com` | AWS SigV4 鉴权 |
| Ollama (本地) | openai | `http://localhost:11434/v1` | 兼容 OpenAI 协议 |
| DeepSeek | openai | `https://api.deepseek.com/v1` | 兼容 OpenAI 协议 |
| 通义千问 | openai | `https://dashscope.aliyuncs.com/compatible-mode/v1` | 兼容 OpenAI 协议 |
| 智谱 AI | openai | `https://open.bigmodel.cn/api/paas/v4` | 兼容 OpenAI 协议 |
| 自定义供应商 | openai/anthropic/gemini | 用户自定义 | 通用 OpenAI 兼容接口 |

### 2.2 统一代理接口（Unified Proxy Interface）

#### 2.2.1 功能描述

对外暴露一个与 OpenAI API 兼容的标准 HTTP 接口，所有下游应用（如 Cursor、Continue、ChatBox 等）只需将 API Base URL 指向本网关即可使用。

#### 2.2.2 详细需求

| 需求编号 | 需求名称 | 优先级 | 描述 |
|----------|---------|--------|------|
| UP-001 | OpenAI 兼容接口 | P0 | 实现 `/v1/chat/completions` 接口，完全兼容 OpenAI Chat Completions API |
| UP-002 | 流式响应支持 | P0 | 支持 SSE (Server-Sent Events) 流式输出，兼容 OpenAI 的 stream 格式 |
| UP-003 | 模型列表接口 | P0 | 实现 `/v1/models` 接口，返回所有已配置供应商的模型聚合列表 |
| UP-004 | 非 stream 响应 | P0 | 支持标准 JSON 响应模式 |
| UP-005 | Embeddings 接口 | P1 | 实现 `/v1/embeddings` 接口 |
| UP-006 | 模型路由规则 | P1 | 支持按模型名称自动路由到对应供应商 |
| UP-007 | 请求转发 | P0 | 将统一接口请求转发到目标供应商，处理鉴权转换 |
| UP-008 | 响应格式转换 | P0 | 将目标供应商的响应转换为 OpenAI 格式返回 |
| UP-009 | 错误统一处理 | P0 | 将不同供应商的错误格式统一为 OpenAI 错误格式 |
| UP-010 | 请求/响应日志 | P1 | 记录所有经过网关的请求和响应，用于调试和审计 |

#### 2.2.3 统一网关 API 规范

网关对外暴露的 API 与 OpenAI API 保持一致：

```
Base URL: http://localhost:{port}/v1

接口列表:
  POST /v1/chat/completions    - Chat 补全（核心接口）
  GET  /v1/models              - 模型列表
  POST /v1/embeddings          - 文本向量化
  GET  /v1/health              - 网关健康检查
```

#### 2.2.4 请求转发流程

```
客户端请求 → 网关统一接口 → 协议识别 → 路由匹配 → 供应商选择 → 协议转换 → 供应商API → 协议反转换 → 客户端响应
```

### 2.3 协议互转（Protocol Conversion）

#### 2.3.1 功能描述

实现不同 AI 协议之间的消息格式转换，使请求可以从一种协议格式透明地转换为另一种协议格式。

#### 2.3.2 详细需求

| 需求编号 | 需求名称 | 优先级 | 描述 |
|----------|---------|--------|------|
| PC-001 | OpenAI → Anthropic 转换 | P0 | 将 OpenAI Chat Completions 请求转换为 Anthropic Messages API 格式 |
| PC-002 | OpenAI → Gemini 转换 | P0 | 将 OpenAI Chat Completions 请求转换为 Google Gemini API 格式 |
| PC-003 | Anthropic → OpenAI 反转换 | P0 | 将 Anthropic Messages 响应转换为 OpenAI Chat Completions 格式 |
| PC-004 | Gemini → OpenAI 反转换 | P0 | 将 Gemini generateContent 响应转换为 OpenAI Chat Completions 格式 |
| PC-005 | 流式响应转换 | P0 | 支持 SSE 流式数据的实时格式转换 |
| PC-006 | Tool/Function Calling 转换 | P1 | 支持工具调用消息在不同协议间的转换 |
| PC-007 | 多模态消息转换 | P1 | 支持图片、文件等多模态内容的格式转换 |
| PC-008 | Token 计数转换 | P2 | 统一不同供应商的 Token 计数格式 |
| PC-009 | 错误格式转换 | P0 | 将各供应商的错误响应统一为 OpenAI 错误格式 |

---

## 3. 功能性需求详细描述

### 3.1 供应商管理模块

#### 3.1.1 供应商配置数据模型

```go
type Provider struct {
    ID          string            `json:"id" toml:"id"`
    Name        string            `json:"name" toml:"name"`
    Type        string            `json:"type" toml:"type"`          // openai, anthropic, gemini, azure-openai, bedrock
    APIBase     string            `json:"api_base" toml:"api_base"`
    APIKey      string            `json:"api_key" toml:"api_key"`
    Enabled     bool              `json:"enabled" toml:"enabled"`
    Models      []ModelInfo       `json:"models" toml:"models"`
    ExtraConfig map[string]string `json:"extra_config" toml:"extra_config"` // 供应商特有配置
    Priority    int               `json:"priority" toml:"priority"`  // 路由优先级
    CreatedAt   time.Time         `json:"created_at" toml:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at" toml:"updated_at"`
}

type ModelInfo struct {
    ID          string `json:"id" toml:"id"`
    Name        string `json:"name" toml:"name"`         // 显示名称
    ProviderID  string `json:"provider_id" toml:"provider_id"`
    MaxTokens   int    `json:"max_tokens" toml:"max_tokens"`
    InputPrice  string `json:"input_price" toml:"input_price"`   // 输入单价
    OutputPrice string `json:"output_price" toml:"output_price"` // 输出单价
}
```

#### 3.1.2 处理逻辑

1. **添加供应商**：用户填写供应商类型、名称、API Base、API Key → 系统验证配置有效性 → 尝试连通性测试 → 保存到本地配置文件
2. **同步模型列表**：根据供应商类型调用对应的 Models API → 解析返回数据 → 更新本地模型缓存
3. **连通性测试**：使用配置信息发送一个轻量级请求（如 list models）→ 返回成功/失败状态

### 3.2 统一代理模块

#### 3.2.1 请求处理流程

```
1. 接收请求 → 解析 model 字段
2. 根据 model 查找对应的供应商配置
3. 确定目标供应商的协议类型
4. 如果目标协议与网关协议不一致，进行请求格式转换
5. 添加目标供应商的鉴权信息（API Key 等）
6. 转发请求到目标供应商
7. 接收响应（区分 stream / non-stream）
8. 如果目标协议与网关协议不一致，进行响应格式转换
9. 返回统一格式的响应给客户端
```

#### 3.2.2 模型路由规则

| 路由策略 | 描述 | 示例 |
|---------|------|------|
| 精确匹配 | 模型名完全匹配供应商中的模型ID | `gpt-4o` → OpenAI 供应商 |
| 前缀匹配 | 模型名前缀匹配供应商类型 | `claude-*` → Anthropic 供应商 |
| 自定义映射 | 用户定义模型名到供应商的映射规则 | `my-model` → 映射到 Ollama 的 `llama3` |
| 默认路由 | 未匹配时使用的默认供应商 | 所有未知模型 → 默认供应商 |
| 优先级路由 | 同一模型在多个供应商可用时，按优先级选择 | `gpt-4o` 在 OpenAI(P=1) 和 Azure(P=2) → 选 OpenAI |

#### 3.2.3 请求/响应处理

**请求处理 (OpenAI 格式输入):**

```json
// 客户端发送（OpenAI 格式）
POST /v1/chat/completions
{
  "model": "claude-sonnet-4-20250514",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": true
}

// 网关内部处理：
// 1. 识别 model="claude-sonnet-4-20250514" → 路由到 Anthropic 供应商
// 2. 转换请求格式：OpenAI → Anthropic
// 3. 发送到 Anthropic API
// 4. 转换响应格式：Anthropic → OpenAI
// 5. 返回 OpenAI 格式响应给客户端
```

### 3.3 协议转换模块

#### 3.3.1 OpenAI Chat Completions 格式（内部基准格式）

```json
// 请求
{
  "model": "string",
  "messages": [{"role": "system|user|assistant|tool", "content": "string"}],
  "temperature": 0.7,
  "max_tokens": 1024,
  "stream": false,
  "tools": [{"type": "function", "function": {"name": "", "parameters": {}}}]
}

// 响应 (非流式)
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "model": "gpt-4o",
  "choices": [{
    "index": 0,
    "message": {"role": "assistant", "content": "Hello!"},
    "finish_reason": "stop"
  }],
  "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
}

// 响应 (流式 SSE)
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}
data: [DONE]
```

#### 3.3.2 Anthropic Messages API 格式

```json
// 请求
{
  "model": "claude-sonnet-4-20250514",
  "messages": [{"role": "user", "content": "Hello"}],
  "system": "You are helpful.",
  "max_tokens": 1024,
  "stream": false
}

// 响应 (非流式)
{
  "id": "msg_xxx",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "Hello!"}],
  "model": "claude-sonnet-4-20250514",
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 10, "output_tokens": 5}
}

// 响应 (流式 SSE)
event: content_block_delta
data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}
event: message_stop
data: {"type":"message_stop"}
```

#### 3.3.3 Google Gemini API 格式

```json
// 请求
POST /v1beta/models/{model}:generateContent
{
  "contents": [{"role": "user", "parts": [{"text": "Hello"}]}],
  "generationConfig": {"temperature": 0.7, "maxOutputTokens": 1024}
}

// 响应
{
  "candidates": [{
    "content": {"parts": [{"text": "Hello!"}], "role": "model"},
    "finishReason": "STOP"
  }],
  "usageMetadata": {"promptTokenCount": 10, "candidatesTokenCount": 5, "totalTokenCount": 15}
}
```

#### 3.3.4 核心字段映射表

| 概念 | OpenAI | Anthropic | Gemini |
|------|--------|-----------|--------|
| 系统提示 | `messages[role=system]` | `system` 字段 | `systemInstruction` |
| 用户消息 | `messages[role=user]` | `messages[role=user]` | `contents[role=user]` |
| 助手消息 | `messages[role=assistant]` | `messages[role=assistant]` | `contents[role=model]` |
| 文本内容 | `content: string` | `content: [{type:"text", text}]` | `parts: [{text}]` |
| 图片内容 | `content: [{type:"image_url"}]` | `content: [{type:"image", source}]` | `parts: [{inlineData}]` |
| 温度参数 | `temperature` | `temperature` | `generationConfig.temperature` |
| 最大 Token | `max_tokens` | `max_tokens` | `generationConfig.maxOutputTokens` |
| 停止原因 | `finish_reason: "stop"` | `stop_reason: "end_turn"` | `finishReason: "STOP"` |
| 输入 Token | `usage.prompt_tokens` | `usage.input_tokens` | `usageMetadata.promptTokenCount` |
| 输出 Token | `usage.completion_tokens` | `usage.output_tokens` | `usageMetadata.candidatesTokenCount` |
| 工具调用 | `tools + tool_calls` | `tools + tool_use` | `tools + functionCall` |

---

## 4. 非功能性需求

### 4.1 性能要求

| 指标 | 要求 | 说明 |
|------|------|------|
| 首字节延迟 | < 50ms (网关自身开销) | 不包括供应商响应时间 |
| 并发请求 | 支持 100+ 并发连接 | 本地使用场景 |
| 流式首包延迟 | < 100ms | 从供应商返回第一个 token 到客户端接收 |
| 内存占用 | < 200MB | 正常运行状态 |
| 协议转换延迟 | < 10ms per request | 格式转换处理时间 |

### 4.2 安全要求

| 需求 | 描述 |
|------|------|
| API Key 加密存储 | 供应商的 API Key 在本地配置文件中加密存储，不明文保存 |
| 本地访问限制 | 网关默认仅监听 localhost，不暴露到公网 |
| 请求日志脱敏 | 日志中不记录完整的 API Key 和敏感用户内容（可配置） |
| CORS 控制 | 可配置允许的跨域来源 |

### 4.3 可靠性要求

| 需求 | 描述 |
|------|------|
| 配置持久化 | 所有配置保存到本地 TOML/JSON 文件，重启不丢失 |
| 优雅降级 | 单个供应商不可用不影响其他供应商的正常使用 |
| 自动重试 | 请求失败时支持可配置的自动重试（次数、间隔） |
| 健康检查 | 定期检查已配置供应商的可用性 |

### 4.4 可扩展性要求

| 需求 | 描述 |
|------|------|
| 协议插件化 | 新协议类型可通过实现标准接口来添加，无需修改核心代码 |
| 供应商模板 | 支持用户自定义供应商模板，兼容各种 OpenAI 兼容接口 |
| 中间件机制 | 支持请求/响应中间件，用于日志、限流、缓存等扩展 |
| 配置热加载 | 修改配置后无需重启网关即可生效 |

---

## 5. 数据模型设计

### 5.1 核心数据实体

```
┌──────────────┐     ┌──────────────┐     ┌──────────────────┐
│   Provider   │────<│    Model     │     │   Route Rule     │
│──────────────│     │──────────────│     │──────────────────│
│ id           │     │ id           │     │ id               │
│ name         │     │ provider_id  │     │ pattern          │
│ type         │     │ model_id     │     │ provider_id      │
│ api_base     │     │ display_name │     │ target_model     │
│ api_key      │     │ max_tokens   │     │ priority         │
│ enabled      │     │ enabled      │     │ enabled          │
│ priority     │     └──────────────┘     └──────────────────┘
│ extra_config │
│ created_at   │     ┌──────────────┐     ┌──────────────────┐
│ updated_at   │     │  Log Entry   │     │  Gateway Config  │
└──────────────┘     │──────────────│     │──────────────────│
                     │ id           │     │ listen_port      │
                     │ timestamp    │     │ default_provider │
                     │ provider_id  │     │ log_level        │
                     │ model        │     │ log_retention    │
                     │ direction    │     │ retry_count      │
                     │ status_code  │     │ retry_interval   │
                     │ latency_ms   │     │ cors_origins     │
                     │ tokens_in    │     │ rate_limit       │
                     │ tokens_out   │     └──────────────────┘
                     │ error        │
                     └──────────────┘
```

### 5.2 配置文件结构 (icoo_proxy.toml)

```toml
[gateway]
listen_port = 16790
default_provider = ""
log_level = "info"
log_retention_days = 7
retry_count = 2
retry_interval_ms = 500

[security]
encrypt_api_keys = true
allowed_origins = ["*"]

[[providers]]
id = "openai-main"
name = "OpenAI"
type = "openai"
api_base = "https://api.openai.com/v1"
api_key = "encrypted:xxxx"
enabled = true
priority = 1

[[providers]]
id = "anthropic-main"
name = "Anthropic"
type = "anthropic"
api_base = "https://api.anthropic.com/v1"
api_key = "encrypted:xxxx"
enabled = true
priority = 2

[[providers]]
id = "ollama-local"
name = "Ollama Local"
type = "openai"
api_base = "http://localhost:11434/v1"
api_key = ""
enabled = true
priority = 3

[[route_rules]]
pattern = "gpt-*"
provider_id = "openai-main"
priority = 10

[[route_rules]]
pattern = "claude-*"
provider_id = "anthropic-main"
priority = 10
```

---

## 6. 用例分析

### 6.1 用例：添加新供应商

| 项目 | 描述 |
|------|------|
| 用例编号 | UC-001 |
| 用例名称 | 添加新 AI 供应商 |
| 参与者 | 用户 |
| 前置条件 | 网关应用已启动 |
| 主流程 | 1. 用户打开供应商管理页面<br>2. 点击"添加供应商"<br>3. 选择供应商类型（或自定义）<br>4. 填写名称、API Base URL、API Key<br>5. 点击"测试连接"<br>6. 连接成功后点击"保存" |
| 替代流程 | 3a. 选择预置模板，自动填充 API Base<br>5a. 连接失败，提示错误信息 |
| 后置条件 | 新供应商已保存到配置，可被路由使用 |

### 6.2 用例：通过网关调用 AI 模型

| 项目 | 描述 |
|------|------|
| 用例编号 | UC-002 |
| 用例名称 | 通过统一接口调用 AI 模型 |
| 参与者 | 外部应用（如 Cursor、ChatBox） |
| 前置条件 | 至少一个供应商已配置并启用 |
| 主流程 | 1. 外部应用将 API Base 指向 `http://localhost:16790/v1`<br>2. 发送 OpenAI 格式的 Chat Completions 请求<br>3. 网关根据 model 字段匹配供应商和路由规则<br>4. 执行协议转换（如需要）<br>5. 转发请求到目标供应商<br>6. 接收并转换响应<br>7. 返回 OpenAI 格式响应给外部应用 |
| 替代流程 | 3a. 未匹配到供应商，返回 404 错误<br>5a. 供应商返回错误，转换为统一错误格式返回 |
| 后置条件 | 请求日志已记录，用量统计已更新 |

### 6.3 用例：配置模型路由

| 项目 | 描述 |
|------|------|
| 用例编号 | UC-003 |
| 用例名称 | 配置模型路由规则 |
| 参与者 | 用户 |
| 前置条件 | 至少两个供应商已配置 |
| 主流程 | 1. 用户打开路由规则管理页面<br>2. 添加路由规则：填写模型匹配模式、目标供应商、优先级<br>3. 保存规则<br>4. 规则立即生效 |
| 替代流程 | 2a. 规则冲突检测，提示用户 |
| 后置条件 | 新路由规则已生效 |

### 6.4 用例：查看请求日志

| 项目 | 描述 |
|------|------|
| 用例编号 | UC-004 |
| 用例名称 | 查看请求日志与用量统计 |
| 参与者 | 用户 |
| 前置条件 | 网关已处理过请求 |
| 主流程 | 1. 用户打开日志/监控页面<br>2. 查看请求列表（时间、模型、供应商、状态、延迟、Token 用量）<br>3. 可按供应商、模型、时间范围筛选<br>4. 查看单条请求的详细信息 |
| 替代流程 | 无 |
| 后置条件 | 无 |

---

## 7. 约束与假设

### 7.1 约束

| 约束 | 描述 |
|------|------|
| 平台约束 | 主要支持 Windows 平台（当前 Wails 构建限制） |
| 运行环境 | 需要本地运行网关服务，不提供云部署 |
| 网络要求 | 需要能访问目标 AI 供应商的 API 端点 |
| 存储限制 | 配置和日志使用本地文件系统存储 |
| 并发限制 | 面向单用户本地使用场景，非高并发生产环境 |

### 7.2 假设

| 假设 | 描述 |
|------|------|
| 用户具备基本的 AI API 使用经验 | 了解 API Key、模型名称等概念 |
| 目标供应商 API 相对稳定 | 不会频繁变更 API 格式 |
| 本地网络环境正常 | 能够正常访问外部 API |
| 用户设备性能足够 | 能够运行 Wails 桌面应用 |

---

## 8. 风险分析

| 风险编号 | 风险描述 | 可能性 | 影响度 | 应对策略 |
|---------|---------|--------|--------|---------|
| R-001 | AI 供应商 API 格式变更导致转换失败 | 中 | 高 | 建立版本化协议适配器，快速跟进更新 |
| R-002 | 流式响应转换中的性能瓶颈 | 低 | 中 | 使用 io.Pipe 流式处理，避免全量缓冲 |
| R-003 | API Key 泄露风险 | 低 | 高 | 本地加密存储，默认仅监听 localhost |
| R-004 | 大量并发请求导致内存溢出 | 低 | 中 | 实现请求队列和限流机制 |
| R-005 | 协议字段不完全对齐导致信息丢失 | 中 | 中 | 建立完整的字段映射表，不支持的字段记录告警日志 |
| R-006 | 供应商 API 限流导致请求失败 | 中 | 中 | 实现多供应商故障转移和重试机制 |
| R-007 | Wails 框架限制导致某些功能难以实现 | 低 | 低 | 核心网关功能与 Wails UI 解耦，网关可独立运行 |

---

## 附录

### 附录 A：术语表

| 术语 | 说明 |
|------|------|
| Provider | AI 服务供应商（如 OpenAI、Anthropic） |
| Gateway | 网关，本项目的核心代理服务 |
| Protocol | AI 服务的 API 协议格式 |
| SSE | Server-Sent Events，服务端推送事件流 |
| Route Rule | 路由规则，决定请求转发到哪个供应商 |
| Upstream | 上游服务，即 AI 供应商的 API |
| Downstream | 下游客户端，即使用网关的应用 |

### 附录 B：参考文档

- [OpenAI Chat Completions API](https://platform.openai.com/docs/api-reference/chat)
- [Anthropic Messages API](https://docs.anthropic.com/en/api/messages)
- [Google Gemini API](https://ai.google.dev/api/generate-content)
- [Azure OpenAI API](https://learn.microsoft.com/en-us/azure/ai-services/openai/reference)
