# 迁移计划

## 1. 迁移原则

- 先迁移测试覆盖最强的纯业务模块，再迁移 HTTP 层。
- 不追求运行时完全兼容旧行为，优先保证新模型清晰、安全默认值正确。
- 每个阶段都保证 `go test ./...` 可运行。
- 不把旧项目里的 `server_test.exe`、`.data`、日志、数据库复制到新项目。
- 旧数据通过一次性导入工具迁移，不直接复用旧数据库文件。

## 2. 阶段拆分

### 阶段 1：新项目骨架

产出：

- `go.mod`
- `cmd/bridge/main.go`
- `internal/app`
- `internal/config`
- `internal/router`
- `internal/middleware`
- `internal/repository/db.go`

依赖：

- `github.com/gin-gonic/gin`
- `gorm.io/gorm`
- `github.com/glebarez/sqlite`
- `github.com/BurntSushi/toml`

验收：

- 服务能启动。
- `/healthz` 返回 ok。
- SQLite no-cgo 能正常打开并迁移空表。

### 阶段 2：Model 和 Repository

迁移：

- Supplier
- Endpoint
- AuthKey
- RoutePolicy
- ModelAlias
- UiPref
- TrafficRecord

验收：

- AutoMigrate 成功。
- Repository CRUD 测试通过。
- 默认 endpoints 和 route policies seed 成功。

### 阶段 3：Service 层

迁移：

- SupplierService
- EndpointService
- AuthKeyService
- RoutePolicyService
- ModelAliasService
- RouteResolver
- SupplierModelCache
- TrafficService

验收：

- 路由解析优先级测试通过。
- 供应商模型缓存并发读测试通过。
- AuthKey 合并和脱敏测试通过。

### 阶段 4：协议转换

迁移：

- `internal/services/translation` 到 `internal/service/translation`
- Anthropic model
- OpenAI Chat model
- OpenAI Responses model

处理：

- 将当前 `internal/pkg/ai_llm_proxy` 整理迁移到 `internal/utils/ai_llm_proxy`。
- 迁移后移除不完整的外部测试依赖，保持工具包纯函数和可单测。

验收：

- 请求转换矩阵测试通过。
- 响应转换矩阵测试通过。
- SSE 转换测试通过。

### 阶段 5：ProxyService

拆分当前大文件：

- `proxy_context.go`
- `proxy_service.go`
- `route_step.go`
- `convert_step.go`
- `upstream_client.go`
- `response_handler.go`
- `stream_relay.go`
- `archive_service.go`

验收：

- mock upstream 集成测试通过。
- 同协议透传通过。
- 跨协议 JSON 转换通过。
- SSE 透传和转换通过。
- 错误响应符合下游协议格式。

### 阶段 6：Gin Controller 和 Router

迁移：

- HealthController
- ProxyController
- Admin controllers

验收：

- 所有管理 API 路由可访问。
- `/api/*` 统一鉴权。
- 默认代理 endpoint 正确注册。
- 内置 endpoint 删除失败。

### 阶段 7：配置和热重载

迁移：

- `config.toml`
- `.env` 开发覆盖
- runtime config snapshot
- route resolver cache rebuild

验收：

- 配置解析失败时启动失败。
- 修改供应商、策略、别名后 reload 生效。
- reload 不中断已进入的代理请求。

## 3. 数据迁移

旧 SQLite 文件：

```text
.data/icoo_proxy.db
```

新 SQLite 文件建议：

```text
.data/icoo_llm_bridge.db
```

表兼容策略：

- 旧表字段语义只作为导入来源，不要求新表字段一一对应。
- `traffic.leveldb` 不建议原地读取，可提供一次性导入工具。
- 新增 `traffic_records` 表后，后续流量直接写 SQLite。
- 新系统启动时只读取新库，不在运行时兼容旧库结构。

迁移工具建议：

```text
cmd/migrate/
  main.go
```

功能：

- 读取旧 SQLite。
- 复制 suppliers、endpoints、auth_keys、route_policies、model_aliases、ui_prefs。
- 可选读取旧 LevelDB 流量并写入 traffic_records。

## 4. 安全修复清单

必须修复：

- 管理 API 本地免鉴权不能基于 `Host`。
- 旧 `/admin/*` 未鉴权接口必须删除或纳入鉴权。
- CORS 默认不能在生产环境无条件 `*`。
- 配置文件解析失败不能静默降级。

建议修复：

- Secret 明文 reveal 增加审计日志。
- 请求体归档默认关闭。
- 链路日志默认不记录 body。
- API Key 比较可改为常量时间比较。

## 5. 测试清单

基础：

- `go test ./...`
- `go test -race ./...`

HTTP：

- 管理 API 未带 key 返回 401。
- 本地请求在允许本地免鉴权时通过。
- 伪造 `Host: localhost` 的远程请求不应通过。

代理：

- 缺少 model 且无默认路由返回协议错误。
- `supplier/model` 命中指定供应商。
- alias 命中指定供应商和模型。
- upstream 4xx 错误透传。
- upstream timeout 返回 502。

流式：

- 下游 stream=true 时保持 SSE。
- 下游 stream=false 且上游 forced stream 时可聚合。
- 客户端断开时 upstream body 能关闭。

## 6. 里程碑

M1：项目骨架、SQLite/GORM、健康检查完成。

M2：管理 API 和数据模型完成。

M3：路由解析和协议转换完成。

M4：代理主链路完成。

M5：流量记录、热重载、安全修复完成。

M6：迁移工具、回归测试、打包脚本完成。

## 7. 建议执行顺序

1. 创建 `icoo_llm_bridge` Go module。
2. 先复制并重命名协议枚举、模型结构和 translation 测试。
3. 搭建 Gin + GORM + SQLite no-cgo 骨架。
4. 实现 repository 和 service。
5. 迁移 proxy pipeline。
6. 接入 controller。
7. 加回归测试。
8. 再考虑删除旧项目或切换桌面端调用目标。

## 8. 不兼容式迁移结论

推荐迁移方式：

1. 新建 `icoo_llm_bridge` 独立 Go module。
2. 新建 `.data/icoo_llm_bridge.db`，不复用 `.data/icoo_proxy.db`。
3. 提供 `cmd/import-legacy` 一次性导入旧 suppliers、auth keys、route policies、aliases、endpoints。
4. 管理 API 改为 `/api/v1/*`。
5. 代理 endpoint 改为显式 endpoint 表驱动，只保留必要官方兼容路径。
6. 桌面端或调用方适配新 API。
