# 不兼容式设计方案

## 1. 设计结论

`icoo_llm_bridge` 建议采用不兼容式重构，而不是在旧 `icoo_server` 上做渐进式兼容改造。

原因：

- 当前代理、管理 API、配置、持久化和运行时状态耦合较重。
- 旧项目存在安全默认值问题，例如部分管理接口未统一鉴权、本地判断实现不一致。
- 流量记录使用 LevelDB，业务配置使用 SQLite，存储模型分裂。
- `proxy.go` 代理流程过长，继续兼容旧结构会限制新架构拆分。
- 新技术栈明确要求 Gin + GORM + SQLite no-cgo，适合重新定义边界。

不兼容式设计的目标不是破坏用户习惯，而是把兼容成本集中到一次性迁移工具和桌面端适配中，换取更干净的运行时模型。

## 2. 明确放弃的兼容项

### 2.1 不兼容旧数据库文件

旧文件：

```text
.data/icoo_proxy.db
.data/traffic.leveldb
```

新文件：

```text
.data/icoo_llm_bridge.db
```

新系统不直接打开旧数据库。旧数据通过导入工具迁移：

```text
cmd/import-legacy
```

导入后，运行时只依赖新 SQLite。

### 2.2 不兼容旧管理 API 路径

旧管理 API 使用 `/api/*`，同时还存在 `/admin/models`、`/admin/routes`、`/admin/requests` 这类未统一保护的接口。

新管理 API 统一改为：

```text
/api/v1/*
```

旧 `/admin/*` 全部删除。旧 `/api/*` 不做运行时兼容，避免长期维护两套路由。

### 2.3 不兼容旧配置加载行为

旧行为：

- 优先读 `config.toml`。
- TOML 解析失败时可能静默回退 `.env`。

新行为：

- `--config` 指定配置文件。
- 默认读取 `config.toml`。
- 配置文件存在但解析失败时直接启动失败。
- `.env` 只用于开发环境变量覆盖，不作为主配置。

### 2.4 不兼容旧本地免鉴权语义

旧管理鉴权存在基于 `Host` 判断本地请求的问题。

新语义：

- 本地免鉴权仅在 `allow_local_without_auth = true` 时启用。
- 判断依据只能是连接来源 IP，即 loopback `127.0.0.0/8` 或 `::1`。
- 伪造 `Host: localhost` 无效。
- 生产模式建议禁用本地免鉴权。

### 2.5 不兼容旧流量存储

旧流量记录写 LevelDB。

新设计写 SQLite `traffic_records` 表。原因：

- 统一备份。
- 统一迁移。
- 支持 GORM 分页和聚合。
- 降低桌面端部署复杂度。

旧 LevelDB 流量可选择导入，也可以丢弃。

## 3. 新领域模型

### 3.1 Provider 替代 Supplier 命名

建议新系统内部统一使用 `Provider`，而不是 `Supplier`。

原因：

- LLM API 领域更常用 provider。
- 与 upstream vendor、model catalog 语义更清晰。

新表：

```text
providers
provider_models
```

建议把旧 `suppliers.models` 的 JSON 字符串拆为独立 `provider_models` 表。

### 3.2 Endpoint 明确为 IngressEndpoint

旧 `Endpoint` 语义偏泛。新系统建议使用：

```text
ingress_endpoints
```

字段：

```text
id
path
downstream_protocol
enabled
protected
built_in
description
created_at
updated_at
```

`protected` 用于定义代理入口是否需要 proxy key。默认 true。

### 3.3 RoutePolicy 升级为 RoutingRule

旧路由策略每个 downstream protocol 只有一个默认供应商，表达力有限。

新表：

```text
routing_rules
```

字段：

```text
id
name
priority
match_protocol
match_model_pattern
target_provider_id
target_model
enabled
created_at
updated_at
```

解析规则按 priority 从小到大执行。这样可以统一支持：

- 默认路由。
- 模型别名。
- `provider/model`。
- 模型通配。
- 未来的 fallback 链。

### 3.4 AuthKey 拆分用途

旧 AuthKey 同时参与管理和代理访问。

新表建议增加 scope：

```text
api_keys
```

字段：

```text
id
name
secret_hash
secret_preview
scopes
enabled
expires_at
created_at
updated_at
```

scope 示例：

- `admin`
- `proxy`

新系统不建议明文保存 secret。创建后只显示一次明文，数据库保存 hash。

## 4. 新 API 设计

### 4.1 管理 API

统一前缀：

```text
/api/v1
```

资源命名：

```text
GET    /api/v1/overview
GET    /api/v1/runtime/state
POST   /api/v1/runtime/reload

GET    /api/v1/providers
POST   /api/v1/providers
GET    /api/v1/providers/:id
PUT    /api/v1/providers/:id
DELETE /api/v1/providers/:id
POST   /api/v1/providers/:id/check

GET    /api/v1/ingress-endpoints
POST   /api/v1/ingress-endpoints
PUT    /api/v1/ingress-endpoints/:id
DELETE /api/v1/ingress-endpoints/:id

GET    /api/v1/routing-rules
POST   /api/v1/routing-rules
PUT    /api/v1/routing-rules/:id
DELETE /api/v1/routing-rules/:id

GET    /api/v1/api-keys
POST   /api/v1/api-keys
DELETE /api/v1/api-keys/:id

GET    /api/v1/traffic
DELETE /api/v1/traffic
```

### 4.2 代理 API

代理入口仍可保留官方兼容路径，因为这是下游 SDK 兼容性，不属于内部设计包袱：

```text
POST /v1/messages
POST /v1/chat/completions
POST /v1/responses
```

建议删除命名空间重复路径，除非桌面端明确依赖：

```text
/anthropic/v1/messages
/openai/v1/chat/completions
/openai/v1/responses
```

如果保留，应作为 `ingress_endpoints` seed 数据，而不是硬编码。

## 5. 新数据库表建议

```text
providers
provider_models
ingress_endpoints
routing_rules
api_keys
ui_preferences
traffic_records
request_archives
runtime_events
```

### 5.1 providers

```text
id TEXT PRIMARY KEY
name TEXT UNIQUE
protocol TEXT
vendor TEXT
base_url TEXT
api_key_cipher TEXT
only_stream BOOLEAN
user_agent TEXT
enabled BOOLEAN
description TEXT
created_at DATETIME
updated_at DATETIME
```

### 5.2 provider_models

```text
id TEXT PRIMARY KEY
provider_id TEXT INDEX
name TEXT
max_tokens INTEGER
enabled BOOLEAN
created_at DATETIME
updated_at DATETIME
UNIQUE(provider_id, name)
```

### 5.3 routing_rules

```text
id TEXT PRIMARY KEY
name TEXT
priority INTEGER INDEX
match_protocol TEXT
match_model_pattern TEXT
target_provider_id TEXT
target_model TEXT
enabled BOOLEAN
created_at DATETIME
updated_at DATETIME
```

### 5.4 api_keys

```text
id TEXT PRIMARY KEY
name TEXT
secret_hash TEXT UNIQUE
secret_preview TEXT
scopes TEXT
enabled BOOLEAN
expires_at DATETIME NULL
created_at DATETIME
updated_at DATETIME
```

## 6. 路由解析新规则

新 RouteResolver 使用统一规则引擎：

1. 如果 model 是 `provider/model`，优先尝试 provider 直连。
2. 遍历启用的 `routing_rules`，按 priority 匹配 `protocol + model pattern`。
3. 如果没有传 model，匹配 `match_model_pattern = ""` 或 `*` 的默认规则。
4. 规则命中后解析 provider 和 target model。
5. 未命中返回协议兼容错误。

这样可以删除旧的 default route、alias、route policy 三套概念，统一成 routing rule。

## 7. 不兼容带来的收益

- 管理 API 更清晰，版本化后可演进。
- 数据库结构更规范，模型列表从 JSON 字符串变为关系表。
- API Key 可改为 hash 存储，不再需要 reveal 明文。
- 路由策略、模型别名和默认路由统一为 routing rules。
- LevelDB 被移除，部署和备份更简单。
- 旧安全问题不需要在兼容层里继续保留。

## 8. 需要付出的代价

- 桌面端需要适配 `/api/v1/*`。
- 旧数据库需要导入工具。
- 旧配置文件需要迁移到新格式。
- 旧 AuthKey 如果只保存明文，可导入为新 hash；导入后不能再 reveal。
- 如果外部用户依赖 `/anthropic/*` 或 `/openai/*` 命名空间路径，需要通过 seed endpoint 显式启用。

## 9. 推荐最终方案

采用以下决策：

- 新项目名：`icoo_llm_bridge`
- 依赖注入：使用手写构造函数注入，`app.Container` 统一管理对象生命周期
- 管理 API：只提供 `/api/v1/*`
- 数据库：只使用 `.data/icoo_llm_bridge.db`
- 存储：全部 SQLite + GORM
- 供应商命名：`Provider`
- 模型存储：独立 `provider_models`
- 路由系统：统一 `routing_rules`
- 协议转换工具：旧 `internal/pkg/ai_llm_proxy` 迁移为 `internal/utils/ai_llm_proxy`
- API Key：hash 存储，scope 区分 admin/proxy
- 兼容方式：一次性 import，不做运行时双栈兼容
