# internal/utils/ai_llm_proxy

该目录规划为 `icoo_llm_bridge` 的底层 LLM 协议转换工具包。

定位：

- 保存 Anthropic、OpenAI Chat、OpenAI Responses 的协议结构。
- 提供请求、响应、SSE 和 usage 的纯函数转换。
- 从旧项目 `internal/pkg/ai_llm_proxy` 整理迁移。

边界：

- 不依赖 Gin。
- 不依赖 GORM。
- 不依赖 repository。
- 不依赖 service。
- 不依赖 `app.Container`。

调用关系：

```text
service/translation -> utils/ai_llm_proxy
service/proxy       -> service/translation
```

当前已迁入旧项目 `internal/pkg/ai_llm_proxy` 的非测试源码，并通过 `NewProtocolConverter` 暴露给代理服务使用。

## 转换矩阵

请求转换：

| Downstream -> Upstream | Anthropic | OpenAI Chat | OpenAI Responses |
| --- | --- | --- | --- |
| Anthropic | 透传 | 未支持 | 已支持 |
| OpenAI Chat | 已支持 | 透传 | 已支持 |
| OpenAI Responses | 已支持 | 未支持 | 透传 |

非流式响应转换：

| Upstream -> Downstream | Anthropic | OpenAI Chat | OpenAI Responses |
| --- | --- | --- | --- |
| Anthropic | 透传 | 已支持 | 已支持 |
| OpenAI Chat | 已支持 | 透传 | 已支持 |
| OpenAI Responses | 已支持 | 已支持 | 透传 |

SSE 流式转换：

| Upstream -> Downstream | Anthropic | OpenAI Chat | OpenAI Responses |
| --- | --- | --- | --- |
| Anthropic | 透传 | 已支持 | 已支持 |
| OpenAI Chat | 已支持 | 透传 | 已支持 |
| OpenAI Responses | 已支持 | 已支持 | 透传 |

暂未接入的方向会返回明确错误，避免隐式错误转换。

矩阵验证：

- `protocol_matrix_test.go` 使用表驱动测试覆盖请求、非流式响应和 SSE 的全部 3 x 3 方向。
- 请求矩阵中，Anthropic -> OpenAI Chat 与 OpenAI Responses -> OpenAI Chat 会返回固定的 `not implemented` 错误；其余方向使用最小有效请求验证成功。
- 非流式响应的全部方向使用最小有效响应验证成功。
- SSE 矩阵使用各上游协议的最小终止流验证全部方向可执行；具体文本、usage、终止原因和工具调用语义由专项测试覆盖。

当前降级规则：

- 第一版优先保证 text、usage、finish reason 的基本语义。
- OpenAI Chat 非流式 tool calls 会转换为 Responses `function_call`，再可转换为 Anthropic `tool_use`。
- OpenAI Chat SSE tool call 增量已通过专项测试覆盖到 Responses `function_call` 和 Anthropic `tool_use`，包括交错的多个工具调用及分段参数。为把交错的并行调用串行映射到 Anthropic content blocks，当前实现会缓冲工具参数到 `finish_reason` 或 EOF，再按原始碎片顺序回放；该路径保证语义完整，但不是低延迟透传。
