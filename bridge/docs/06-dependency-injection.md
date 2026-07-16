# 依赖注入与对象管理

## 1. 设计结论

`icoo_llm_bridge` 使用手写依赖注入，不引入 Wire、Dig、Fx 等 DI 框架。原因是项目对象图清晰、生命周期可控，手写构造函数更容易调试，也更适合桌面端分发。

统一对象管理由 `internal/app.Container` 负责。Container 是组合根，只允许在启动和关闭阶段使用。业务逻辑不得把 Container 当 service locator 使用。

## 2. Container 职责

Container 负责：

- 加载配置。
- 初始化 logger。
- 打开 SQLite/GORM。
- 执行 migration 和 seed。
- 构建 repository。
- 构建 utils。
- 构建 service。
- 构建 controller。
- 构建 middleware。
- 构建 Gin router。
- 统一关闭资源。

Container 不负责：

- 写业务规则。
- 处理 HTTP 请求。
- 解析路由策略。
- 做协议转换。

## 3. 推荐结构

```go
type Container struct {
    Config config.Config
    Logger *slog.Logger
    DB     *gorm.DB

    Utils        Utils
    Repositories Repositories
    Services     Services
    Controllers  Controllers
    Middlewares  Middlewares

    Router *gin.Engine
    Server *http.Server
}
```

对象分组：

```go
type Utils struct {
    LLMProxy ai_llm_proxy.Converter
}

type Repositories struct {
    Provider      repository.ProviderRepository
    ProviderModel repository.ProviderModelRepository
    Endpoint      repository.EndpointRepository
    RoutingRule   repository.RoutingRuleRepository
    APIKey        repository.APIKeyRepository
    Traffic       repository.TrafficRepository
    UIPreference  repository.UIPreferenceRepository
}

type Services struct {
    Auth        service.AuthService
    Provider    service.ProviderService
    Endpoint    service.EndpointService
    Routing     service.RouteResolver
    Translation service.TranslationService
    Proxy       service.ProxyService
    Traffic     service.TrafficService
    Runtime     service.RuntimeService
}
```

## 4. 构造顺序

```text
NewContainer
  -> LoadConfig
  -> NewLogger
  -> OpenDB
  -> AutoMigrate
  -> SeedDefaults
  -> NewRepositories
  -> NewUtils
  -> NewServices
  -> NewMiddlewares
  -> NewControllers
  -> NewRouter
  -> NewHTTPServer
```

关键点：

- Repository 只能依赖 DB。
- Utils 只能依赖静态配置或无依赖。
- Service 依赖 repository interface、utils interface、logger、配置快照。
- Controller 依赖 service interface。
- Router 依赖 controller 和 middleware。

## 5. 禁止模式

禁止在业务代码中使用：

```go
app.GetContainer().Services.Proxy.Handle(...)
```

禁止 service 里直接创建 repository：

```go
func NewProxyService(db *gorm.DB) *ProxyService {
    repo := repository.NewProviderRepository(db)
    ...
}
```

推荐：

```go
func NewProxyService(
    auth AuthService,
    routes RouteResolver,
    converter TranslationService,
    upstream UpstreamClient,
    traffic TrafficService,
) *ProxyService
```

## 6. utils/ai_llm_proxy 设计

路径：

```text
internal/utils/ai_llm_proxy
```

定位：

- 从旧 `internal/pkg/ai_llm_proxy` 迁移而来。
- 作为底层协议转换工具包。
- 不承担业务路由、鉴权、上游请求、日志和持久化。

建议接口：

```go
type Converter interface {
    ConvertRequest(input RequestInput) ([]byte, error)
    ConvertResponse(input ResponseInput) ([]byte, error)
    ConvertStream(input StreamInput) (StreamResult, error)
    ExtractUsage(protocol Protocol, body []byte) TokenUsage
}
```

目录：

```text
internal/utils/ai_llm_proxy/
  types.go
  request_convert.go
  response_convert.go
  stream_convert.go
  usage.go
  json_helpers.go
```

依赖边界：

- 可以依赖标准库。
- 可以依赖 `internal/constants` 和纯 model value object。
- 不依赖 Gin。
- 不依赖 GORM。
- 不依赖 repository。
- 不依赖 service。
- 不依赖 app.Container。

## 7. 测试策略

Container 测试：

- 使用临时 SQLite 文件或内存 SQLite。
- 验证 `NewContainer` 能完成所有构造。
- 验证 `Close` 可重复调用且不 panic。

Service 测试：

- 用 fake repository。
- 用 fake upstream client。
- 用 fake converter。

utils/ai_llm_proxy 测试：

- 使用表驱动测试覆盖 3 协议请求转换。
- 使用表驱动测试覆盖 3 协议响应转换。
- 单独测试 SSE event 转换。
- 不启动 Gin。
- 不打开数据库。
