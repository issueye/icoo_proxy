# 接口与模块规划

## 1. 模块映射

| 当前模块 | 新模块 | 说明 |
| --- | --- | --- |
| `cmd/server/main.go` | `cmd/bridge/main.go` + `internal/app` | main 只负责启动入口，生命周期进入 app |
| `internal/api` | `internal/router` + `internal/middleware` + `internal/controller/health` | Gin 路由和中间件替代 net/http mux |
| `internal/adminapi` | `internal/controller/admin` | 每类资源独立 controller |
| `internal/services/proxy.go` | `internal/service/proxy/*` | 拆分代理上下文、pipeline、upstream client、stream relay |
| `internal/services/translation` | `internal/service/translation` + `internal/utils/ai_llm_proxy` | service 编排转换方向，utils 保持纯函数转换逻辑 |
| `internal/pkg/ai_llm_proxy` | `internal/utils/ai_llm_proxy` | 作为底层协议转换工具包迁移，不能依赖 Gin/GORM/service |
| `internal/services/catalog.go` | `internal/service/routing` | 路由解析独立模块 |
| `internal/storage` | `internal/repository` | GORM 初始化、迁移、repository |
| `internal/traffic` | `internal/service/traffic` + `traffic_repository` | 迁移 LevelDB 到 SQLite |
| `internal/models` | `internal/model/entity` + `internal/model/domain` + `internal/view` | 实体、领域对象、响应 DTO 拆分 |

## 2. Controller 规划

### 2.1 HealthController

接口：

- `GET /`
- `GET /healthz`
- `GET /readyz`

依赖：

- `OverviewService`
- `RuntimeStateService`

### 2.2 ProxyController

接口：

- 动态注册所有启用 endpoints。

职责：

- 根据 endpoint path 找到下游协议。
- 调用 `ProxyService.Handle`。
- 不参与路由策略或转换判断。

### 2.3 SupplierController

接口：

- `GET /api/suppliers`
- `GET /api/suppliers/all`
- `POST /api/suppliers`
- `DELETE /api/suppliers/:id`
- `GET /api/suppliers/health`
- `POST /api/suppliers/:id/health/check`

Service：

- `SupplierService`
- `SupplierHealthService`

### 2.4 EndpointController

接口：

- `GET /api/endpoints`
- `GET /api/endpoints/all`
- `POST /api/endpoints`
- `DELETE /api/endpoints/:id`

规则：

- 内置 endpoint 不允许删除。
- path 必须以 `/` 开头。
- protocol 必须是支持协议之一。

### 2.5 AuthKeyController

接口：

- `GET /api/auth-keys`
- `GET /api/auth-keys/all`
- `POST /api/auth-keys`
- `DELETE /api/auth-keys/:id`
- `POST /api/auth-keys/:id/reveal-secret`

规则：

- 列表只返回 masked secret。
- reveal 需要已通过管理鉴权。

### 2.6 RoutePolicyController

接口：

- `GET /api/route-policies`
- `POST /api/route-policies`

规则：

- 每个 downstream protocol 只允许一个策略。
- 启用策略时，供应商必须启用且有默认模型。

### 2.7 ModelAliasController

接口：

- `GET /api/model-aliases`
- `POST /api/model-aliases`
- `DELETE /api/model-aliases/:id`

规则：

- alias name 唯一。
- supplier 必须存在。
- model 不允许为空。

### 2.8 SettingsController

接口：

- `GET /api/settings`
- `PUT /api/settings`
- `GET /api/ui-prefs`
- `PUT /api/ui-prefs`

说明：

- 项目配置仍可保存在 `config.toml`。
- UI 偏好保存在 SQLite。

### 2.9 TrafficController

接口：

- `GET /api/traffic`
- `DELETE /api/traffic`

查询参数：

- `filter`
- `page`
- `pageSize`

返回：

- items
- total
- protocol_options
- token_stats
- total_requests
- success_count
- error_count
- average_latency

## 3. Service 规划

### 3.1 ProxyService

核心接口：

```go
type ProxyService interface {
    Handle(ctx context.Context, req ProxyRequest, downstream Protocol) ProxyResult
}
```

关键依赖：

- `AuthKeyService`
- `RouteResolver`
- `translation.Converter`
- `utils/ai_llm_proxy`
- `UpstreamClient`
- `TrafficService`
- `ArchiveService`
- `ChainLogger`

### 3.2 RouteResolver

核心接口：

```go
type RouteResolver interface {
    Resolve(downstream Protocol, requestedModel string) (Route, error)
    RebuildCache(ctx context.Context) error
}
```

解析顺序必须与当前行为兼容：

1. 空模型走默认策略。
2. `supplier/model` 直连供应商模型。
3. 当前 downstream 的策略供应商模型。
4. 模型别名。
5. 默认策略 fallback。

### 3.3 SupplierService

职责：

- CRUD。
- 模型列表标准化。
- API Key 脱敏。
- 默认模型校验。

### 3.4 TrafficService

职责：

- 记录代理请求摘要。
- 分页查询。
- token 汇总。
- 成功率、错误数、平均延迟。

实现：

- 使用 SQLite `traffic_records`。
- `created_at` 建索引，按时间倒序查询。

### 3.5 TranslationService

职责：

- 请求 JSON 转换。
- 响应 JSON 转换。
- SSE 事件转换。
- usage 提取。
- 作为 service 层适配器调用 `internal/utils/ai_llm_proxy`。

约束：

- 无数据库依赖。
- 无 Gin 依赖。
- 高覆盖率单元测试。

### 3.6 Utils/ai_llm_proxy

职责：

- 保存 Anthropic、OpenAI Chat、OpenAI Responses 的协议结构体。
- 提供请求转换纯函数。
- 提供响应转换纯函数。
- 提供 SSE event 转换纯函数。
- 提供 JSON helper 和 usage 提取 helper。

边界：

- 不读取配置。
- 不创建 HTTP client。
- 不访问数据库。
- 不记录业务日志。
- 不依赖 Container。

目录建议：

```text
internal/utils/ai_llm_proxy/
  types.go
  request_convert.go
  response_convert.go
  stream_convert.go
  usage.go
  json_helpers.go
```

## 4. Repository 规划

每个 repository 提供最小查询能力：

```go
type SupplierRepository interface {
    List(ctx context.Context) ([]entity.Supplier, error)
    Page(ctx context.Context, query SupplierQuery) (Page[entity.Supplier], error)
    FindByID(ctx context.Context, id string) (entity.Supplier, error)
    Save(ctx context.Context, item *entity.Supplier) error
    Delete(ctx context.Context, id string) error
}
```

事务规则：

- 单表简单 CRUD 不显式事务。
- 热重载或批量迁移可使用 `db.Transaction`。
- Repository 返回底层错误，Service 转成业务错误。

## 5. 中间件规划

### 5.1 RequestIDMiddleware

- 如果请求头已有 `X-Request-ID` 可复用。
- 否则生成 `req-` 前缀短 ID。
- 响应写入 `X-ICOO-Request-ID`。

### 5.2 AdminAuthMiddleware

- 作用于 `/api` group。
- 支持 `x-api-key` 和 `Authorization: Bearer`。
- 本地免鉴权只基于 `ClientIP()` 或 `RemoteAddr` 解析结果。

### 5.3 CORSMiddleware

- 桌面端可配置允许来源。
- 默认开发环境允许 `localhost`。
- 不建议生产环境默认 `*`。

### 5.4 RecoveryMiddleware

- 捕获 panic。
- 记录 request_id。
- 管理 API 返回统一错误。
- 代理 API 返回符合下游协议的错误。

## 6. 启动流程

```text
main
  -> app.NewContainer
       -> load config
       -> init logger
       -> open gorm sqlite
       -> auto migrate
       -> seed default data
       -> build repositories
       -> build utils
       -> build services
       -> build controllers
       -> build middlewares
       -> build gin router
  -> start http server
  -> graceful shutdown
  -> container.Close
```

## 7. 依赖注入对象规划

统一对象管理放在 `internal/app/container.go`。

Repository group：

```go
type Repositories struct {
    Provider      ProviderRepository
    ProviderModel ProviderModelRepository
    Endpoint      EndpointRepository
    RoutingRule   RoutingRuleRepository
    APIKey        APIKeyRepository
    Traffic       TrafficRepository
    UIPreference  UIPreferenceRepository
}
```

Service group：

```go
type Services struct {
    Auth        AuthService
    Provider    ProviderService
    Endpoint    EndpointService
    Routing     RouteResolver
    Translation TranslationService
    Proxy       ProxyService
    Traffic     TrafficService
    Runtime     RuntimeService
}
```

Controller group：

```go
type Controllers struct {
    Health   *HealthController
    Proxy    *ProxyController
    Provider *ProviderController
    Routing  *RoutingRuleController
    APIKey   *APIKeyController
    Traffic  *TrafficController
    Runtime  *RuntimeController
}
```

构造规则：

- `NewRepositories(db)` 只接收数据库。
- `NewServices(repositories, utils, configSnapshot, logger)` 只接收接口集合和配置快照。
- `NewControllers(services)` 只接收 service interface。
- `NewRouter(controllers, middlewares)` 只负责挂路由。

## 8. 热重载设计

热重载不应重启整个 HTTP server。建议：

1. 重新读取配置。
2. 重新加载 auth keys、suppliers、route policies、aliases、endpoints。
3. 重建 route resolver cache。
4. 原子替换 runtime config snapshot。

代理请求读取配置时使用只读快照，避免半更新状态。
