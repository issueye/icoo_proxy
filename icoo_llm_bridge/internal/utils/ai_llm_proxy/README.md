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

已接入的非流式 JSON 转换方向：

- Anthropic request -> OpenAI Responses request
- OpenAI Chat Completions request -> OpenAI Responses request
- OpenAI Chat Completions request -> Anthropic request
- OpenAI Responses request -> Anthropic request
- Anthropic response -> OpenAI Responses response
- Anthropic response -> OpenAI Chat Completions response
- OpenAI Responses response -> Anthropic response
- OpenAI Responses response -> OpenAI Chat Completions response

暂未接入的方向会返回明确错误，避免隐式错误转换。

已接入的 SSE 流式转换方向：

- OpenAI Responses stream -> Anthropic stream
- OpenAI Responses stream -> OpenAI Chat Completions stream
- Anthropic stream -> OpenAI Responses stream
- Anthropic stream -> OpenAI Chat Completions stream

同协议 stream 直接透传。
