# icoo_llm_bridge 并行开发计划

计划日期：2026-05-17

## 1. 开发目标

本轮开发以 `docs/07-ccx-comparison-and-optimization-plan.md` 为依据，先推进可独立并行的基础优化，不一次性重构完整高可用架构。

本轮目标：

1. 修正代理运行时的 P0 风险：上游错误、响应头复制、普通请求和流式请求 timeout、流式响应预检测。
2. 补强协议转换矩阵的测试与可用方向，优先处理 OpenAI Chat 作为上游时的缺口。
3. 为后续 ProviderEndpoint、ProviderCredential、RoutePlan、FailoverExecutor 落地建立清晰设计和代码切入点。
4. 保持现有分层边界：Controller 不承载业务逻辑，Repository 不参与调度，`utils/ai_llm_proxy` 不依赖 Gin/GORM/Service。

## 2. 当前基线

当前代理链路：

```text
ProxyController
  -> ProxyService
  -> RouteResolver
  -> ai_llm_proxy.Converter
  -> single Provider base_url + single api_key
  -> upstream
  -> ai_llm_proxy.Converter
  -> TrafficRecord
```

当前已有测试入口：

- `internal/utils/ai_llm_proxy/converter_test.go`
- `internal/service/proxy_service_test.go`
- `internal/service/route_resolver_test.go`
- `internal/app/container_test.go`
- `internal/middleware/middleware_test.go`

## 3. 并行拆分

### Worker A：协议转换矩阵

负责范围：

- `internal/utils/ai_llm_proxy/`

任务：

- 补齐 OpenAI Chat upstream 到 Anthropic / OpenAI Responses 的非流式响应转换。
- 补齐 OpenAI Chat stream 到 Anthropic / OpenAI Responses 的最小流式转换。
- 增加矩阵测试，覆盖已支持、补齐和明确未支持的方向。
- 更新 `internal/utils/ai_llm_proxy/README.md` 中的支持状态。

边界：

- 不修改 `internal/service`。
- 不引入 Gin、GORM 或 Repository 依赖。
- 转换不要求第一版完全保留复杂 tool call，但必须保证 text、usage、finish reason 的基本语义。

### Worker B：代理运行时 P0 加固

负责范围：

- `internal/service/proxy_service.go`
- `internal/service/proxy_service_test.go`
- 必要时新增 `internal/service/*_test.go`

任务：

- 上游非 2xx 响应先按 downstream 协议返回错误，不再盲目进入成功响应转换。
- 响应体被读取并重写后，避免透传不可靠的 `content-encoding`、`content-length`、`transfer-encoding`。
- 普通请求 timeout 与流式请求 timeout 解耦，避免长 SSE 被 `WriteTimeout` 截断。
- 增加最小 stream preflight：写客户端 header 前读取有限首批 SSE 数据，识别明显错误或空流。
- 为以上行为增加服务级测试。

边界：

- 不修改 `internal/utils/ai_llm_proxy`。
- 不做 ProviderEndpoint / ProviderCredential 数据模型迁移。
- 不引入完整 failover，只解决当前单上游路径的正确性。

### Worker C：RoutePlan 与资源池设计落点

负责范围：

- `internal/model/domain/`
- `internal/service/route_resolver.go`
- `internal/service/route_resolver_test.go`
- 可新增只读设计骨架文件，但不做数据库迁移。

任务：

- 设计并落地轻量 `RouteCandidate` / `RoutePlan` domain 类型。
- 保留现有 `RouteResolver.Resolve` 兼容行为。
- 新增可测试的候选生成方法或辅助函数，为后续 Scheduler / FailoverExecutor 做准备。
- 增加 route plan 相关单元测试。

边界：

- 不修改 Provider 持久化模型。
- 不修改 ProxyService 请求执行逻辑。
- 不做跨 Provider failover，只准备类型和候选排序基础。

## 4. 合并顺序

建议合并顺序：

1. Worker C：先合入 domain 类型和 route plan 兼容扩展，风险较低。
2. Worker A：合入 converter 能力和测试，避免影响 service 层。
3. Worker B：最后合入 ProxyService 运行时变更，因为它触达请求主链路。

如果出现冲突，以以下原则处理：

- `utils/ai_llm_proxy` 由 Worker A 优先。
- `internal/service/proxy_service.go` 由 Worker B 优先。
- `route_resolver.go` 由 Worker C 优先。
- 不回滚其他 worker 的文件；需要适配时在自己的范围内调整。

## 5. 验证计划

每个 worker 完成后至少运行相关包测试：

```text
go test ./internal/utils/ai_llm_proxy
go test ./internal/service
go test ./internal/app
```

最终合并后运行：

```text
go test ./...
```

验收条件：

- 所有现有测试通过。
- 新增测试覆盖本轮变更。
- `docs/07` 和 `docs/08` 中的支持状态与实现一致。
- 没有引入跨层依赖倒置。

## 6. 本轮不做

- 不开发完整 ProviderEndpoint / ProviderCredential 数据库迁移。
- 不开发完整 Scheduler、FailoverExecutor、RuntimeHealthStore。
- 不开发 Gemini、Images。
- 不开发管理端 UI。
- 不修改认证模型和 API Key 加密策略。

这些内容进入下一轮，以本轮的 RoutePlan、converter、stream preflight 为基础继续推进。

## 7. 本轮执行结果

已完成：

- Worker A：补齐 OpenAI Chat upstream 到 Anthropic / OpenAI Responses 的非流式响应转换和基础 SSE 文本流转换，并更新转换矩阵测试。
- Worker B：加固 `ProxyService` 的上游非 2xx 错误处理、响应头复制、流式请求 timeout 和 stream preflight。
- Worker C：新增 `RoutePlan` / `RouteCandidate` 等轻量 domain 类型，并在 `RouteResolver` 中提供兼容的 `ResolvePlan` 落点。

保留风险：

- Anthropic request -> OpenAI Chat request、OpenAI Responses request -> OpenAI Chat request 仍未支持。
- OpenAI Chat SSE tool call 增量仍是降级路径，当前只保证文本、usage 和终止原因。
- stream preflight 仍是最小实现，首事件等待时间固定，后续应改为可配置。
