# 架构设计

## 1. 总体架构

`icoo_llm_bridge` 采用 MVC + Service + Repository 的 Go Web 后端结构。这里的 MVC 不按传统服务端模板渲染理解，而按 API 服务拆分：

- Model：GORM 实体、领域模型、值对象、枚举。
- View：响应 DTO、分页结构、错误结构、序列化结构。
- Controller：Gin handler，负责 HTTP 入参、状态码、响应。

Service 和 Repository 作为工程化补充：

- Service：业务编排和领域逻辑。
- Repository：数据访问和事务边界。
- Middleware：鉴权、CORS、请求 ID、错误恢复、访问日志。

## 2. 目标目录结构

```text
icoo_llm_bridge/
  cmd/
    bridge/
      main.go
  configs/
    config.example.toml
  docs/
    01-requirements-analysis.md
    02-architecture-design.md
    03-api-and-module-plan.md
    04-migration-plan.md
  internal/
    app/
      app.go
      bootstrap.go
      container.go
      lifecycle.go
    config/
      config.go
      loader.go
    constants/
      protocol.go
      vendor.go
    model/
      entity/
        auth_key.go
        endpoint.go
        model_alias.go
        route_policy.go
        supplier.go
        traffic_record.go
        ui_pref.go
      domain/
        route.go
        supplier_snapshot.go
        token_usage.go
    view/
      response.go
      pagination.go
      state_view.go
      supplier_view.go
      endpoint_view.go
      traffic_view.go
    controller/
      admin/
        overview_controller.go
        supplier_controller.go
        endpoint_controller.go
        auth_key_controller.go
        route_policy_controller.go
        model_alias_controller.go
        settings_controller.go
        traffic_controller.go
      proxy/
        proxy_controller.go
      health/
        health_controller.go
    middleware/
      auth.go
      cors.go
      request_id.go
      recovery.go
    router/
      router.go
      admin_routes.go
      proxy_routes.go
      health_routes.go
    repository/
      db.go
      migrate.go
      auth_key_repository.go
      endpoint_repository.go
      supplier_repository.go
      route_policy_repository.go
      model_alias_repository.go
      traffic_repository.go
      ui_pref_repository.go
    contract/
      repository.go
      service.go
      converter.go
    service/
      admin/
        overview_service.go
      auth/
        auth_key_service.go
      endpoint/
        endpoint_service.go
      supplier/
        supplier_service.go
        supplier_health_service.go
      routing/
        catalog_service.go
        route_resolver.go
        supplier_model_cache.go
      proxy/
        proxy_service.go
        proxy_context.go
        upstream_client.go
        stream_relay.go
        request_archive.go
      traffic/
        traffic_service.go
      settings/
        settings_service.go
      translation/
        converter.go
        request_converter.go
        response_converter.go
        stream_converter.go
        usage.go
    utils/
      ai_llm_proxy/
        README.md
        types.go
        request_convert.go
        response_convert.go
        stream_convert.go
        json_helpers.go
      jsonx/
        redact.go
      httputilx/
        headers.go
      idgen/
        request_id.go
  migrations/
    001_init.sql
  go.mod
  README.md
```

## 3. 依赖方向

依赖只能从外层指向内层：

```text
cmd -> app/container -> router/controller -> service -> repository -> model/entity
                                             service -> model/domain
                                             service -> utils/ai_llm_proxy
controller -> view
middleware -> service/auth
```

禁止：

- Repository 引用 Gin。
- Model 引用 Controller。
- Translation 引用数据库。
- Controller 直接调用 GORM。
- Service 直接写 HTTP 响应。
- `utils/ai_llm_proxy` 反向引用 service、repository、controller、Gin 或 GORM。

## 4. 依赖注入与对象管理

项目采用手写依赖注入，`internal/app.Container` 是唯一组合根。所有长生命周期对象在启动阶段统一创建、校验和关闭。

Container 管理对象：

- Config
- Logger
- GORM DB
- Repositories
- Services
- Controllers
- Middlewares
- Gin Engine
- HTTP Server

设计原则：

- 业务包不读取全局变量。
- Service 通过构造函数接收接口，而不是直接 new repository。
- Controller 通过构造函数接收 service interface。
- Router 只接收 controller 和 middleware，不创建业务对象。
- 测试时可以替换 repository、converter、upstream client、traffic recorder。

示例结构：

```go
type Container struct {
    Config Config
    DB     *gorm.DB

    Repositories Repositories
    Services     Services
    Controllers  Controllers

    Router *gin.Engine
    Server *http.Server
}
```

启动阶段：

```text
NewContainer
  -> load config
  -> open db
  -> migrate
  -> build repositories
  -> build utilities
  -> build services
  -> build controllers
  -> build router
```

关闭阶段由 `Container.Close()` 统一释放数据库连接、日志文件、后台任务和 HTTP server。

## 5. Gin 路由分组

```text
GET    /
GET    /healthz
GET    /readyz

POST   /v1/messages
POST   /anthropic/v1/messages
POST   /v1/chat/completions
POST   /openai/v1/chat/completions
POST   /v1/responses
POST   /openai/v1/responses

GET    /api/overview
GET    /api/state
POST   /api/proxy/reload

GET    /api/suppliers
GET    /api/suppliers/all
POST   /api/suppliers
DELETE /api/suppliers/:id
GET    /api/suppliers/health
POST   /api/suppliers/:id/health/check

GET    /api/endpoints
GET    /api/endpoints/all
POST   /api/endpoints
DELETE /api/endpoints/:id

GET    /api/auth-keys
GET    /api/auth-keys/all
POST   /api/auth-keys
DELETE /api/auth-keys/:id
POST   /api/auth-keys/:id/reveal-secret

GET    /api/route-policies
POST   /api/route-policies

GET    /api/model-aliases
POST   /api/model-aliases
DELETE /api/model-aliases/:id

GET    /api/settings
PUT    /api/settings
GET    /api/ui-prefs
PUT    /api/ui-prefs

GET    /api/traffic
DELETE /api/traffic
```

所有 `/api/*` 路由统一挂载 `AdminAuthMiddleware`。旧的 `/admin/models`、`/admin/routes`、`/admin/requests` 不建议继续公开；如必须兼容，应重定向到 `/api/*` 并走同一鉴权链。

## 6. MVC 职责拆分

### 6.1 Controller

Controller 示例职责：

- 从 Gin `Context` 读取 path/query/body/header。
- 调用 service。
- 将业务错误映射为 HTTP 状态码。
- 返回 `view.AdminResponse` 或协议原始响应。

Controller 不负责：

- 供应商合法性判断。
- 路由解析。
- 协议转换。
- GORM 查询。

### 6.2 Model

Model 分两类：

- `entity`：数据库表结构，包含 GORM tag。
- `domain`：业务过程对象，例如 `Route`、`SupplierSnapshot`、`ProxyContext`、`TokenUsage`。

这种拆分可以避免数据库结构污染业务流程，也方便未来调整表结构。

### 6.3 View

View 放 API 响应 DTO：

- `AdminResponse`
- `AdminError`
- `PageResult`
- `StateView`
- `SupplierView`
- `EndpointView`
- `TrafficView`

列表接口默认返回脱敏字段。Secret 明文只能由专用 view 返回。

## 7. 数据库设计

建议全部迁移到 SQLite + GORM。

核心表：

- `suppliers`
- `endpoints`
- `auth_keys`
- `route_policies`
- `model_aliases`
- `ui_prefs`
- `traffic_records`

新增 `traffic_records` 建议字段：

```text
id TEXT PRIMARY KEY
request_id TEXT UNIQUE INDEX
endpoint TEXT
downstream_protocol TEXT
upstream_protocol TEXT
model TEXT
status_code INTEGER
duration_ms INTEGER
input_tokens INTEGER
output_tokens INTEGER
total_tokens INTEGER
error TEXT
created_at DATETIME INDEX
```

可选调试归档表或文件：

```text
request_archives
  id
  request_id
  direction
  phase
  headers_json
  body_text
  truncated
  created_at
```

如果继续用文件保存请求体，路径必须由配置控制，并默认关闭。

## 8. 代理链路设计

代理服务建议拆成 pipeline：

```text
ProxyController
  -> ProxyService.Handle(ctx, downstreamProtocol)
    -> ValidateDownstreamRequest
    -> ReadRequestBody
    -> ResolveRoute
    -> ConvertRequest
    -> BuildUpstreamRequest
    -> SendUpstreamRequest
    -> HandleUpstreamResponse
    -> RecordTraffic
```

`ProxyService` 不直接依赖 Gin，只接收内部定义的 request adapter 或标准 `http.Request/http.ResponseWriter`。如果保持流式响应，允许 proxy 层持有 `http.ResponseWriter`，但要把转换、路由、记录拆成独立组件。

## 9. 协议转换设计

`service/translation` 负责业务层转换编排，底层协议结构和纯转换函数放到 `internal/utils/ai_llm_proxy`。

`internal/utils/ai_llm_proxy` 约束：

- 输入：协议类型、目标模型、请求或响应 JSON、SSE event。
- 输出：转换后的 JSON、SSE event 或 token usage。
- 不引用 Gin。
- 不引用 GORM。
- 不读取配置文件。
- 不调用 repository。
- 不保存任何运行时状态。

`service/translation` 约束：

- 负责选择转换方向。
- 负责把业务 Route、默认 max tokens 等参数转换为工具包输入。
- 负责把工具包错误转换为业务错误。

建议接口：

```go
type Converter interface {
    ConvertRequest(input RequestConvertInput) ([]byte, error)
    ConvertResponse(input ResponseConvertInput) ([]byte, error)
}
```

流式转换单独接口：

```go
type StreamConverter interface {
    ConvertStream(input StreamConvertInput) (TokenUsage, error)
}
```

## 10. 配置设计

建议配置文件：

```toml
host = "127.0.0.1"
port = 18181
read_timeout_seconds = 15
write_timeout_seconds = 300
shutdown_timeout_seconds = 10
allow_unauthenticated_local = true
default_max_tokens = 32768

[log]
chain_log_path = ".data/bridge-chain.log"
chain_log_bodies = false
chain_log_max_body_bytes = 8192

[archive]
enabled = false
down_request_dir = ".data/down_request"
up_request_dir = ".data/up_request"
```

配置加载规则：

1. 显式 `--config` 优先。
2. 默认读取工作目录 `config.toml`。
3. 开发环境允许 `.env` 覆盖部分变量。
4. 配置文件存在但解析失败时必须启动失败。

## 11. 安全设计

管理鉴权：

- `/api/*` 默认要求 `x-api-key` 或 `Authorization: Bearer`。
- 本地免鉴权只允许 `RemoteAddr` 是 loopback。
- 不允许通过 `Host` 判断本地请求。

代理鉴权：

- 使用配置 API Key + 数据库启用 AuthKey 合并后的 key ring。
- 空 key ring 时，默认只允许本地请求。
- 建议生产配置要求至少一个 API Key。

敏感信息：

- `Authorization`、`x-api-key`、`api_key`、`secret` 统一脱敏。
- Supplier 列表返回 `api_key_masked`，不返回 `api_key`。

## 12. 错误处理

定义统一错误类型：

```go
type AppError struct {
    Code string
    Message string
    HTTPStatus int
}
```

代理协议错误需要按下游协议输出：

- Anthropic：`{"type":"error","error":{"type":"invalid_request_error","message":"..."}}`
- OpenAI：`{"error":{"type":"invalid_request_error","message":"..."}}`

管理 API 使用统一结构：

```json
{
  "data": {},
  "error": null
}
```

## 13. 测试策略

单元测试：

- translation 转换矩阵。
- route resolver 优先级。
- supplier model cache。
- auth middleware 本地地址判断。
- repository CRUD。
- container 构造顺序和 Close 释放。
- utils/ai_llm_proxy 纯函数转换。

集成测试：

- Gin router 管理 API。
- SQLite migration。
- 代理请求到 mock upstream。
- SSE 透传和转换。

回归测试：

- 保留现有 `internal/services/translation` 测试样例。
- 将旧 `internal/pkg/ai_llm_proxy` 整理迁移为 `internal/utils/ai_llm_proxy`，并移除外部测试依赖不完整的问题。
