# icoo_llm_bridge 架构文档

本文档集用于指导将现有 `icoo_server` 重构为 `icoo_llm_bridge`。

目标技术栈：

- Go
- Gin
- GORM
- SQLite without CGO，建议使用 `github.com/glebarez/sqlite`
- MVC 分层结构

## 文档目录

- [需求分析](docs/01-requirements-analysis.md)
- [架构设计](docs/02-architecture-design.md)
- [接口与模块规划](docs/03-api-and-module-plan.md)
- [迁移计划](docs/04-migration-plan.md)
- [不兼容式设计方案](docs/05-breaking-design.md)
- [依赖注入与对象管理](docs/06-dependency-injection.md)

## 重构目标

`icoo_llm_bridge` 定位为本地优先的 LLM API Bridge。它对下游暴露 Anthropic Messages、OpenAI Chat Completions、OpenAI Responses 兼容入口，对上游按供应商、路由策略和模型别名转发请求，并在必要时完成协议转换。

本次设计重点不是简单搬迁代码，而是把当前集中在 `proxy.go`、`admin.go`、`main.go` 中的职责拆开：

- Controller 只负责 HTTP 参数、鉴权上下文、响应封装。
- Service 负责业务编排，包括代理链路、路由解析、供应商管理、流量统计。
- Model/Repository 负责 GORM 实体、查询、事务和持久化。
- View/DTO 负责对外响应结构，避免数据库实体直接泄露到 API。
- `app.Container` 作为统一对象管理入口，采用手写构造函数注入，不使用全局单例。

## 关键约束

- SQLite 必须使用 no-cgo driver，便于桌面端和跨平台分发。
- 协议转换逻辑必须可单测，不依赖 Gin 或数据库。
- 代理流式转发必须保持低延迟，不应为了记录日志而阻塞主响应链路。
- 管理 API 默认不得因伪造 `Host` 头绕过鉴权。
- `internal/utils/ai_llm_proxy` 作为底层协议转换工具包，只允许被 service 层调用，不能依赖业务容器。

## 当前设计取向

采用不兼容式重构优先。新项目不强制保留旧接口、旧数据库文件、旧配置格式和旧路由命名；兼容性通过一次性迁移工具解决，运行时模型以 `icoo_llm_bridge` 的新领域设计为准。

## 当前骨架运行

```powershell
go test ./...
go run ./cmd/bridge
```

构建 Windows 可执行文件：

```powershell
.\build.ps1
```

默认监听 `127.0.0.1:18181`，配置样例见 [config.example.toml](configs/config.example.toml)。
