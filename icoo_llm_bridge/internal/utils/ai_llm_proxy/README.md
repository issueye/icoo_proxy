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

当前降级规则：

- 第一版优先保证 text、usage、finish reason 的基本语义。
- OpenAI Chat 非流式 tool calls 会转换为 Responses `function_call`，再可转换为 Anthropic `tool_use`。
- OpenAI Chat SSE tool call 增量暂不完整转换；流式路径只保证文本、usage 和终止原因。
