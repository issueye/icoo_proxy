# 需求分析

## 1. 项目定位

现有 `icoo_server` 是一个 LLM 协议代理服务，主要服务桌面端或本地网关场景。它把不同下游 API 协议统一接入，并按配置选择上游供应商、目标模型和协议转换方式。

重构后的 `icoo_llm_bridge` 应保持现有能力，同时提高模块边界、可测试性、安全性和后续扩展能力。

## 2. 当前功能点盘点

### 2.1 代理入口

当前默认支持 6 个代理路径：

- `/v1/messages`
- `/anthropic/v1/messages`
- `/v1/chat/completions`
- `/openai/v1/chat/completions`
- `/v1/responses`
- `/openai/v1/responses`

每个 Endpoint 绑定一个下游协议：

- `anthropic`
- `openai-chat`
- `openai-responses`

代理入口要求 `POST` 请求。请求进入后完成 API Key 校验、请求体读取、模型提取、路由解析、请求转换、上游转发、响应转换、流量记录。

### 2.2 协议转换

当前请求转换矩阵完整覆盖 3 x 3：

| 下游 | 上游 | 行为 |
| --- | --- | --- |
| Anthropic | Anthropic | 改写 model |
| Anthropic | OpenAI Chat | 转换请求体 |
| Anthropic | OpenAI Responses | 转换请求体 |
| OpenAI Chat | Anthropic | 转换请求体，补齐 max_tokens |
| OpenAI Chat | OpenAI Chat | 改写 model |
| OpenAI Chat | OpenAI Responses | 转换请求体 |
| OpenAI Responses | Anthropic | 转换请求体，补齐 max_tokens |
| OpenAI Responses | OpenAI Chat | 转换请求体 |
| OpenAI Responses | OpenAI Responses | 改写 model |

响应转换矩阵同样覆盖 3 x 3。流式响应已覆盖若干关键链路：

- OpenAI Responses SSE 到 Anthropic SSE
- OpenAI Responses SSE 到 OpenAI Chat SSE
- Anthropic SSE 到 OpenAI Chat SSE
- OpenAI Responses SSE 聚合为 JSON 后再转换给非流式下游

### 2.3 路由解析

当前路由解析顺序：

1. 请求未传 `model` 时，使用当前下游协议的默认路由策略。
2. 请求 `model` 为 `供应商/模型` 格式时，优先命中供应商模型缓存。
3. 请求 `model` 与当前下游协议的启用路由策略匹配时，按策略供应商解析该供应商下的模型。
4. 命中模型别名时，按别名目标供应商和模型转发。
5. 仍未命中时，使用默认路由策略，并把请求模型作为上游模型。

这套规则需要保留，但建议在新项目中独立为 `RouterService` 或 `RouteResolver`，不要继续耦合在代理服务内部。

### 2.4 管理 API

当前管理能力：

- 服务概览和运行状态。
- 代理热重载。
- 供应商分页、列表、保存、删除、健康检查。
- 端点分页、列表、保存、删除。
- API Key 分页、列表、保存、删除、显示明文 Secret。
- 路由策略列表、保存。
- 模型别名列表、保存、删除。
- 项目设置读取和保存。
- UI 偏好读取和保存。
- 流量分页查询和清空。

### 2.5 数据持久化

当前 SQLite/GORM 表：

- `suppliers`
- `endpoints`
- `auth_keys`
- `route_policies`
- `model_aliases`
- `ui_prefs`

当前流量记录使用 LevelDB。重构要求技术栈为 Go + SQLite(no cgo) + GORM + Gin，因此建议把流量记录也迁移到 SQLite 表，避免双存储带来的备份、迁移和一致性问题。

### 2.6 配置能力

当前支持：

- `config.toml`
- `.env` 兼容模式
- 监听地址、端口、超时、API Key、本地免鉴权、链路日志、默认 max tokens

新项目建议保留 `config.toml` 为主配置，`.env` 只用于开发覆盖；配置解析失败必须直接报错，不应静默回退。

## 3. 非功能需求

### 3.1 安全

- 管理 API 必须统一走鉴权中间件。
- 本地免鉴权必须基于 `RemoteAddr` 回环地址判断，不能基于 `Host` 头。
- Secret 默认不出现在列表接口，只允许通过显式 reveal 接口返回。
- 请求/响应体落盘默认关闭；开启时必须有最大字节限制和敏感字段脱敏。

### 3.2 可维护性

- 代理流程需要拆成小型 pipeline step。
- 协议转换保持纯函数，单元测试无需启动 HTTP 服务。
- 数据访问通过 repository 封装，service 不直接拼 GORM 查询。
- Controller 不写业务规则。

### 3.3 可观测性

- 每个代理请求生成 `request_id`。
- 响应头返回 `X-ICOO-Request-ID`。
- 记录下游协议、上游协议、模型、状态码、耗时、token 用量和错误摘要。
- 链路日志保留阶段事件：downstream request、route resolved、request converted、upstream request、response converted、request completed。

### 3.4 兼容性

- 保持现有默认代理路径。
- 保持现有管理 API 语义，但可调整为 Gin 路由组。
- 保持现有数据库字段语义，允许通过迁移脚本升级。

## 4. 优先级

P0：

- Gin 路由和中间件基础框架。
- SQLite no-cgo + GORM 初始化和 migration。
- 供应商、端点、鉴权 Key、路由策略、模型别名。
- 代理主链路和 3 协议转换。
- 管理 API 统一鉴权。

P1：

- 流量记录迁移到 SQLite。
- 健康检查。
- 请求/响应体调试归档。
- 热重载。

P2：

- 更细粒度的供应商 failover。
- 请求限流。
- OpenAPI 文档。
- Web UI 专用聚合接口。
