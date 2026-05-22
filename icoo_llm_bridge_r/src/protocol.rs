use std::{
    collections::HashMap,
    io::{BufRead, BufReader, Read, Write},
};

use anyhow::bail;
use serde_json::{json, Map, Value};

use crate::model::{
    TokenUsage, PROTOCOL_ANTHROPIC, PROTOCOL_OPENAI_CHAT, PROTOCOL_OPENAI_RESPONSES,
};

pub fn convert_request(
    downstream: &str,
    upstream: &str,
    model: &str,
    body: &[u8],
) -> anyhow::Result<Vec<u8>> {
    if downstream.is_empty() || upstream.is_empty() {
        bail!("protocols are required");
    }
    let out = match (downstream, upstream) {
        (a, b) if a == b => body.to_vec(),
        (PROTOCOL_OPENAI_CHAT, PROTOCOL_OPENAI_RESPONSES) => chat_request_to_responses(body)?,
        (PROTOCOL_ANTHROPIC, PROTOCOL_OPENAI_RESPONSES) => anthropic_request_to_responses(body)?,
        (PROTOCOL_OPENAI_CHAT, PROTOCOL_ANTHROPIC) => chat_request_to_anthropic(body)?,
        (PROTOCOL_OPENAI_RESPONSES, PROTOCOL_ANTHROPIC) => responses_request_to_anthropic(body)?,
        _ => bail!(
            "request conversion from {} to {} is not implemented",
            downstream,
            upstream
        ),
    };
    rewrite_model(out, model)
}

pub fn convert_response(
    downstream: &str,
    upstream: &str,
    model: &str,
    body: &[u8],
) -> anyhow::Result<Vec<u8>> {
    if downstream.is_empty() || upstream.is_empty() {
        bail!("protocols are required");
    }
    match (upstream, downstream) {
        (a, b) if a == b => Ok(body.to_vec()),
        (PROTOCOL_OPENAI_RESPONSES, PROTOCOL_OPENAI_CHAT) => {
            responses_response_to_chat(body, model)
        }
        (PROTOCOL_OPENAI_RESPONSES, PROTOCOL_ANTHROPIC) => {
            responses_response_to_anthropic(body, model)
        }
        (PROTOCOL_OPENAI_CHAT, PROTOCOL_OPENAI_RESPONSES) => chat_response_to_responses(body),
        (PROTOCOL_OPENAI_CHAT, PROTOCOL_ANTHROPIC) => chat_response_to_anthropic(body, model),
        (PROTOCOL_ANTHROPIC, PROTOCOL_OPENAI_CHAT) => anthropic_response_to_chat(body, model),
        (PROTOCOL_ANTHROPIC, PROTOCOL_OPENAI_RESPONSES) => anthropic_response_to_responses(body),
        _ => bail!(
            "response conversion from {} to {} is not implemented",
            upstream,
            downstream
        ),
    }
}

pub fn convert_stream(
    downstream: &str,
    upstream: &str,
    model: &str,
    reader: impl Read,
    mut writer: impl Write,
) -> anyhow::Result<TokenUsage> {
    if downstream == upstream {
        let mut r = BufReader::new(reader);
        std::io::copy(&mut r, &mut writer)?;
        return Ok(TokenUsage::default());
    }
    match (upstream, downstream) {
        (PROTOCOL_OPENAI_RESPONSES, PROTOCOL_OPENAI_CHAT) => {
            responses_stream_to_chat(reader, writer, model)
        }
        (PROTOCOL_ANTHROPIC, PROTOCOL_OPENAI_CHAT) => {
            anthropic_stream_to_chat(reader, writer, model)
        }
        (PROTOCOL_OPENAI_CHAT, PROTOCOL_OPENAI_RESPONSES) => {
            chat_stream_to_responses(reader, writer, model)
        }
        (PROTOCOL_OPENAI_CHAT, PROTOCOL_ANTHROPIC) => {
            chat_stream_to_anthropic(reader, writer, model)
        }
        (PROTOCOL_OPENAI_RESPONSES, PROTOCOL_ANTHROPIC) => {
            responses_stream_to_anthropic(reader, writer, model)
        }
        (PROTOCOL_ANTHROPIC, PROTOCOL_OPENAI_RESPONSES) => {
            anthropic_stream_to_responses(reader, writer, model)
        }
        _ => bail!(
            "stream conversion from {} to {} is not implemented",
            upstream,
            downstream
        ),
    }
}

pub fn extract_usage(body: &[u8]) -> TokenUsage {
    let Ok(payload) = serde_json::from_slice::<Value>(body) else {
        return TokenUsage::default();
    };
    let usage = payload.get("usage").and_then(Value::as_object);
    let Some(usage) = usage else {
        return TokenUsage::default();
    };
    TokenUsage {
        input_tokens: int(usage.get("input_tokens")) + int(usage.get("prompt_tokens")),
        output_tokens: int(usage.get("output_tokens")) + int(usage.get("completion_tokens")),
        total_tokens: int(usage.get("total_tokens")),
    }
    .normalize()
}

pub fn request_wants_stream(body: &[u8]) -> bool {
    serde_json::from_slice::<Value>(body)
        .ok()
        .and_then(|v| v.get("stream").and_then(Value::as_bool).map(bool::from))
        .unwrap_or(false)
}

pub fn chat_include_usage(body: &[u8]) -> bool {
    serde_json::from_slice::<Value>(body)
        .ok()
        .and_then(|v| {
            v.get("stream_options")
                .and_then(|v| v.get("include_usage"))
                .and_then(Value::as_bool)
        })
        .unwrap_or(false)
}

pub fn write_chat_completion_as_stream(
    body: &[u8],
    include_usage: bool,
    mut writer: impl Write,
) -> anyhow::Result<()> {
    let payload: Value = serde_json::from_slice(body)?;
    let choices = payload
        .get("choices")
        .and_then(Value::as_array)
        .ok_or_else(|| anyhow::anyhow!("chat completion response has no choices"))?;
    let choice = choices
        .first()
        .ok_or_else(|| anyhow::anyhow!("chat completion response has no choices"))?;
    let id = payload.get("id").cloned().unwrap_or(json!(""));
    let created = payload.get("created").cloned().unwrap_or(json!(0));
    let model = payload.get("model").cloned().unwrap_or(json!(""));
    write_sse_json(
        &mut writer,
        &json!({"id": id, "object":"chat.completion.chunk", "created": created, "model": model, "choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}),
    )?;
    if let Some(content) = choice
        .pointer("/message/content")
        .and_then(|v| v.as_str())
        .map(str::to_string)
    {
        write_sse_json(
            &mut writer,
            &json!({"id": payload["id"], "object":"chat.completion.chunk", "created": payload["created"], "model": payload["model"], "choices":[{"index":0,"delta":{"content":content},"finish_reason":null}]}),
        )?;
    }
    let finish = choice
        .get("finish_reason")
        .and_then(Value::as_str)
        .unwrap_or("stop");
    write_sse_json(
        &mut writer,
        &json!({"id": payload["id"], "object":"chat.completion.chunk", "created": payload["created"], "model": payload["model"], "choices":[{"index":0,"delta":{"content":""},"finish_reason":finish}]}),
    )?;
    if include_usage {
        if let Some(usage) = payload.get("usage") {
            write_sse_json(
                &mut writer,
                &json!({"id": payload["id"], "object":"chat.completion.chunk", "created": payload["created"], "model": payload["model"], "choices":[], "usage": usage}),
            )?;
        }
    }
    writer.write_all(b"data: [DONE]\n\n")?;
    Ok(())
}

fn chat_request_to_responses(body: &[u8]) -> anyhow::Result<Vec<u8>> {
    let req: Value = serde_json::from_slice(body)?;
    let messages = req
        .get("messages")
        .cloned()
        .unwrap_or_else(|| Value::Array(vec![]));
    let mut out = Map::new();
    copy_if(&req, &mut out, "model");
    if let Some(instructions) = req.get("instructions").cloned().or_else(|| {
        req.get("messages")
            .and_then(Value::as_array)
            .map(|items| {
                items
                    .iter()
                    .filter(|m| m.get("role").and_then(Value::as_str) == Some("system"))
                    .filter_map(|m| text_from_content(m.get("content")?))
                    .collect::<Vec<_>>()
                    .join("\n")
            })
            .filter(|s| !s.is_empty())
            .map(Value::String)
    }) {
        out.insert("instructions".to_string(), instructions);
    }
    out.insert(
        "input".to_string(),
        chat_messages_to_responses_input(&messages),
    );
    if let Some(v) = req
        .get("max_completion_tokens")
        .or_else(|| req.get("max_tokens"))
    {
        out.insert("max_output_tokens".to_string(), v.clone());
    }
    for key in ["temperature", "top_p", "stream", "service_tier"] {
        copy_if(&req, &mut out, key);
    }
    let tools = chat_tools_to_responses_tools(req.get("tools"), req.get("functions"));
    if !tools.is_empty() {
        out.insert("tools".to_string(), Value::Array(tools));
    }
    if let Some(v) = req.get("tool_choice") {
        out.insert("tool_choice".to_string(), v.clone());
    } else if let Some(v) = req.get("function_call") {
        out.insert(
            "tool_choice".to_string(),
            chat_function_call_to_tool_choice(v),
        );
    }
    if let Some(v) = req.get("reasoning_effort") {
        out.insert(
            "reasoning".to_string(),
            json!({"effort": v, "summary": "auto"}),
        );
    }
    Ok(serde_json::to_vec(&Value::Object(out))?)
}

fn anthropic_request_to_responses(body: &[u8]) -> anyhow::Result<Vec<u8>> {
    let req: Value = serde_json::from_slice(body)?;
    let mut out = Map::new();
    copy_if(&req, &mut out, "model");
    if let Some(system) = req.get("system").and_then(text_from_content) {
        out.insert("instructions".to_string(), Value::String(system));
    }
    out.insert(
        "input".to_string(),
        req.get("messages").cloned().unwrap_or_else(|| json!([])),
    );
    if let Some(v) = req.get("max_tokens") {
        out.insert(
            "max_output_tokens".to_string(),
            json!(int(Some(v)).max(128)),
        );
    }
    for key in ["temperature", "top_p", "stream", "tools", "tool_choice"] {
        copy_if(&req, &mut out, key);
    }
    Ok(serde_json::to_vec(&Value::Object(out))?)
}

fn chat_request_to_anthropic(body: &[u8]) -> anyhow::Result<Vec<u8>> {
    let req: Value = serde_json::from_slice(body)?;
    let mut out = Map::new();
    copy_if(&req, &mut out, "model");
    let max_tokens = req
        .get("max_tokens")
        .or_else(|| req.get("max_completion_tokens"))
        .map(|v| int(Some(v)).max(128))
        .unwrap_or(128);
    out.insert("max_tokens".to_string(), json!(max_tokens));
    out.insert(
        "messages".to_string(),
        req.get("messages").cloned().unwrap_or_else(|| json!([])),
    );
    for key in [
        "temperature",
        "top_p",
        "stream",
        "tools",
        "tool_choice",
        "stop",
    ] {
        copy_if(&req, &mut out, key);
    }
    Ok(serde_json::to_vec(&Value::Object(out))?)
}

fn responses_request_to_anthropic(body: &[u8]) -> anyhow::Result<Vec<u8>> {
    let req: Value = serde_json::from_slice(body)?;
    let mut out = Map::new();
    copy_if(&req, &mut out, "model");
    if let Some(v) = req.get("instructions") {
        out.insert("system".to_string(), v.clone());
    }
    let max_tokens = req
        .get("max_output_tokens")
        .map(|v| int(Some(v)).max(128))
        .unwrap_or(128);
    out.insert("max_tokens".to_string(), json!(max_tokens));
    let (system_from_input, messages) = responses_input_to_anthropic_messages(req.get("input"));
    if !out.contains_key("system") {
        if let Some(system) = system_from_input {
            out.insert("system".to_string(), system);
        }
    }
    out.insert("messages".to_string(), messages);
    for key in ["temperature", "top_p", "stream"] {
        copy_if(&req, &mut out, key);
    }
    let tools = responses_tools_to_anthropic_tools(req.get("tools"));
    if !tools.is_empty() {
        out.insert("tools".to_string(), Value::Array(tools));
    }
    if let Some(v) = req.get("tool_choice") {
        out.insert(
            "tool_choice".to_string(),
            responses_tool_choice_to_anthropic(v),
        );
    }
    if let Some(effort) = req
        .pointer("/reasoning/effort")
        .and_then(Value::as_str)
        .map(responses_effort_to_anthropic)
    {
        out.insert("output_config".to_string(), json!({"effort": effort}));
        if effort != "low" {
            out.insert(
                "thinking".to_string(),
                json!({"type": "enabled", "budget_tokens": thinking_budget(effort)}),
            );
        }
    }
    Ok(serde_json::to_vec(&Value::Object(out))?)
}

fn responses_response_to_chat(body: &[u8], model: &str) -> anyhow::Result<Vec<u8>> {
    let resp: Value = serde_json::from_slice(body)?;
    let text = response_output_text(&resp);
    let reasoning = response_reasoning_text(&resp);
    let tool_calls = response_function_calls_to_chat(&resp);
    let mut message = Map::new();
    message.insert("role".to_string(), json!("assistant"));
    message.insert("content".to_string(), json!(text));
    if !reasoning.is_empty() {
        message.insert("reasoning_content".to_string(), json!(reasoning));
    }
    if !tool_calls.is_empty() {
        message.insert("tool_calls".to_string(), Value::Array(tool_calls));
    }
    let finish_reason = if message.get("tool_calls").is_some() {
        "tool_calls"
    } else {
        finish_from_responses(&resp)
    };
    let usage = responses_usage_to_chat(resp.get("usage"));
    Ok(serde_json::to_vec(&json!({
        "id": resp.get("id").cloned().unwrap_or(json!("")),
        "object": "chat.completion",
        "created": chrono::Utc::now().timestamp(),
        "model": resp.get("model").and_then(Value::as_str).unwrap_or(model),
        "choices": [{"index":0,"message":Value::Object(message),"finish_reason": finish_reason}],
        "usage": usage
    }))?)
}

fn anthropic_response_to_chat(body: &[u8], model: &str) -> anyhow::Result<Vec<u8>> {
    let resp: Value = serde_json::from_slice(body)?;
    let text = anthropic_content_text(resp.get("content"));
    let usage = anthropic_usage_to_chat(resp.get("usage"));
    Ok(serde_json::to_vec(&json!({
        "id": resp.get("id").cloned().unwrap_or(json!("")),
        "object": "chat.completion",
        "created": chrono::Utc::now().timestamp(),
        "model": model,
        "choices": [{"index":0,"message":{"role":"assistant","content":text},"finish_reason": chat_finish_from_anthropic(resp.get("stop_reason").and_then(Value::as_str))}],
        "usage": usage
    }))?)
}

fn chat_response_to_responses(body: &[u8]) -> anyhow::Result<Vec<u8>> {
    let resp: Value = serde_json::from_slice(body)?;
    let choice = resp
        .get("choices")
        .and_then(Value::as_array)
        .and_then(|v| v.first())
        .cloned()
        .unwrap_or_else(|| json!({}));
    let content = choice
        .pointer("/message/content")
        .and_then(Value::as_str)
        .unwrap_or("");
    let finish = choice
        .get("finish_reason")
        .and_then(Value::as_str)
        .unwrap_or("stop");
    let mut output = vec![
        json!({"type":"message","id":"msg_0","role":"assistant","content":[{"type":"output_text","text":content}],"status":"completed"}),
    ];
    if let Some(tool_calls) = choice
        .pointer("/message/tool_calls")
        .and_then(Value::as_array)
    {
        for call in tool_calls {
            if call
                .get("type")
                .and_then(Value::as_str)
                .unwrap_or("function")
                != "function"
            {
                continue;
            }
            output.push(json!({
                "type": "function_call",
                "call_id": call.get("id").and_then(Value::as_str).unwrap_or(""),
                "name": call.pointer("/function/name").and_then(Value::as_str).unwrap_or(""),
                "arguments": call.pointer("/function/arguments").and_then(Value::as_str).unwrap_or("")
            }));
        }
    }
    let mut out = json!({
        "id": resp.get("id").cloned().unwrap_or(json!("")),
        "object": "response",
        "model": resp.get("model").cloned().unwrap_or(json!("")),
        "status": if finish == "length" {"incomplete"} else {"completed"},
        "output": output,
        "usage": chat_usage_to_responses(resp.get("usage"))
    });
    if finish == "length" {
        out["incomplete_details"] = json!({"reason":"max_output_tokens"});
    }
    Ok(serde_json::to_vec(&out)?)
}

fn chat_response_to_anthropic(body: &[u8], model: &str) -> anyhow::Result<Vec<u8>> {
    let resp: Value = serde_json::from_slice(body)?;
    let choice = resp
        .get("choices")
        .and_then(Value::as_array)
        .and_then(|v| v.first())
        .cloned()
        .unwrap_or_else(|| json!({}));
    let content = choice
        .pointer("/message/content")
        .and_then(Value::as_str)
        .unwrap_or("");
    let mut blocks = Vec::new();
    if !content.is_empty() {
        blocks.push(json!({"type":"text","text":content}));
    }
    if let Some(tool_calls) = choice
        .pointer("/message/tool_calls")
        .and_then(Value::as_array)
    {
        for call in tool_calls {
            if call
                .get("type")
                .and_then(Value::as_str)
                .unwrap_or("function")
                != "function"
            {
                continue;
            }
            let raw_args = call
                .pointer("/function/arguments")
                .and_then(Value::as_str)
                .unwrap_or("{}");
            let input = serde_json::from_str::<Value>(raw_args).unwrap_or_else(|_| json!({}));
            blocks.push(json!({
                "type": "tool_use",
                "id": call.get("id").and_then(Value::as_str).unwrap_or(""),
                "name": call.pointer("/function/name").and_then(Value::as_str).unwrap_or(""),
                "input": input
            }));
        }
    }
    if blocks.is_empty() {
        blocks.push(json!({"type":"text","text":""}));
    }
    Ok(serde_json::to_vec(&json!({
        "id": resp.get("id").cloned().unwrap_or(json!("")),
        "type":"message",
        "role":"assistant",
        "model": model,
        "content": blocks,
        "stop_reason": anthropic_stop_from_chat(choice.get("finish_reason").and_then(Value::as_str)),
        "usage": chat_usage_to_anthropic(resp.get("usage"))
    }))?)
}

fn responses_response_to_anthropic(body: &[u8], model: &str) -> anyhow::Result<Vec<u8>> {
    let resp: Value = serde_json::from_slice(body)?;
    let blocks = responses_output_to_anthropic_blocks(&resp);
    Ok(serde_json::to_vec(&json!({
        "id": resp.get("id").cloned().unwrap_or(json!("")),
        "type":"message",
        "role":"assistant",
        "model": model,
        "content": blocks,
        "stop_reason": "end_turn",
        "usage": responses_usage_to_anthropic(resp.get("usage"))
    }))?)
}

fn anthropic_response_to_responses(body: &[u8]) -> anyhow::Result<Vec<u8>> {
    let resp: Value = serde_json::from_slice(body)?;
    Ok(serde_json::to_vec(&json!({
        "id": resp.get("id").cloned().unwrap_or(json!("")),
        "object":"response",
        "model": resp.get("model").cloned().unwrap_or(json!("")),
        "status":"completed",
        "output":[{"type":"message","id":"msg_0","role":"assistant","content":[{"type":"output_text","text":anthropic_content_text(resp.get("content"))}],"status":"completed"}],
        "usage": anthropic_usage_to_responses(resp.get("usage"))
    }))?)
}

fn responses_stream_to_chat(
    reader: impl Read,
    mut writer: impl Write,
    model: &str,
) -> anyhow::Result<TokenUsage> {
    let mut usage = TokenUsage::default();
    let mut saw_tool_call = false;
    let mut output_to_tool_index = HashMap::<i64, i64>::new();
    let mut next_tool_index = 0_i64;
    for frame in read_sse(reader)? {
        let Ok(event) = serde_json::from_str::<Value>(&frame.data) else {
            continue;
        };
        let typ = event
            .get("type")
            .and_then(Value::as_str)
            .unwrap_or(&frame.event);
        match typ {
            "response.created" => {
                write_sse_json(
                    &mut writer,
                    &chat_chunk(model, json!({"role":"assistant"}), None, None),
                )?;
            }
            "response.output_text.delta" => {
                if let Some(delta) = event.get("delta").and_then(Value::as_str) {
                    write_sse_json(
                        &mut writer,
                        &chat_chunk(model, json!({"content":delta}), None, None),
                    )?;
                }
            }
            "response.output_item.added" => {
                if event.pointer("/item/type").and_then(Value::as_str) == Some("function_call") {
                    saw_tool_call = true;
                    let output_index = int(event.get("output_index"));
                    let tool_index = next_tool_index;
                    next_tool_index += 1;
                    output_to_tool_index.insert(output_index, tool_index);
                    write_sse_json(
                        &mut writer,
                        &chat_chunk(
                            model,
                            json!({"tool_calls":[{
                                "index": tool_index,
                                "id": event.pointer("/item/call_id").and_then(Value::as_str).unwrap_or(""),
                                "type": "function",
                                "function": {
                                    "name": event.pointer("/item/name").and_then(Value::as_str).unwrap_or(""),
                                    "arguments": ""
                                }
                            }]}),
                            None,
                            None,
                        ),
                    )?;
                }
            }
            "response.function_call_arguments.delta" => {
                if let Some(delta) = event.get("delta").and_then(Value::as_str) {
                    let output_index = int(event.get("output_index"));
                    if let Some(tool_index) = output_to_tool_index.get(&output_index) {
                        write_sse_json(
                            &mut writer,
                            &chat_chunk(
                                model,
                                json!({"tool_calls":[{
                                    "index": tool_index,
                                    "function": {"arguments": delta}
                                }]}),
                                None,
                                None,
                            ),
                        )?;
                    }
                }
            }
            "response.reasoning_summary_text.delta" => {
                if let Some(delta) = event.get("delta").and_then(Value::as_str) {
                    write_sse_json(
                        &mut writer,
                        &chat_chunk(model, json!({"reasoning_content":delta}), None, None),
                    )?;
                }
            }
            "response.completed" | "response.done" | "response.incomplete" | "response.failed" => {
                if let Some(u) = event.pointer("/response/usage") {
                    usage = usage_from_responses(u);
                }
                let finish = if saw_tool_call {
                    "tool_calls"
                } else if event.pointer("/response/status").and_then(Value::as_str)
                    == Some("incomplete")
                    && event
                        .pointer("/response/incomplete_details/reason")
                        .and_then(Value::as_str)
                        == Some("max_output_tokens")
                {
                    "length"
                } else {
                    "stop"
                };
                write_sse_json(
                    &mut writer,
                    &chat_chunk(model, json!({}), Some(finish), None),
                )?;
                break;
            }
            _ => {}
        }
    }
    writer.write_all(b"data: [DONE]\n\n")?;
    Ok(usage.normalize())
}

fn anthropic_stream_to_chat(
    reader: impl Read,
    mut writer: impl Write,
    model: &str,
) -> anyhow::Result<TokenUsage> {
    let mut usage = TokenUsage::default();
    write_sse_json(
        &mut writer,
        &chat_chunk(model, json!({"role":"assistant"}), None, None),
    )?;
    for frame in read_sse(reader)? {
        let Ok(event) = serde_json::from_str::<Value>(&frame.data) else {
            continue;
        };
        let typ = event
            .get("type")
            .and_then(Value::as_str)
            .unwrap_or(&frame.event);
        match typ {
            "message_start" => {
                if let Some(u) = event.pointer("/message/usage") {
                    usage.input_tokens = int(u.get("input_tokens"));
                }
            }
            "content_block_delta" => {
                if let Some(text) = event.pointer("/delta/text").and_then(Value::as_str) {
                    write_sse_json(
                        &mut writer,
                        &chat_chunk(model, json!({"content":text}), None, None),
                    )?;
                }
            }
            "message_delta" => {
                if let Some(v) = event.pointer("/usage/output_tokens") {
                    usage.output_tokens = int(Some(v));
                }
            }
            "message_stop" => {
                write_sse_json(
                    &mut writer,
                    &chat_chunk(model, json!({}), Some("stop"), None),
                )?;
                break;
            }
            _ => {}
        }
    }
    writer.write_all(b"data: [DONE]\n\n")?;
    Ok(usage.normalize())
}

fn chat_stream_to_responses(
    reader: impl Read,
    mut writer: impl Write,
    model: &str,
) -> anyhow::Result<TokenUsage> {
    let mut usage = TokenUsage::default();
    let mut tool_call_indices = HashMap::<i64, bool>::new();
    writer.write_all(format!("event: response.created\ndata: {}\n\n", json!({"type":"response.created","response":{"id":"resp_stream","object":"response","model":model,"status":"in_progress","output":[]}})).as_bytes())?;
    for frame in read_sse(reader)? {
        if frame.data.trim() == "[DONE]" {
            break;
        }
        let Ok(chunk) = serde_json::from_str::<Value>(&frame.data) else {
            continue;
        };
        if let Some(u) = chunk.get("usage") {
            usage = usage_from_chat(u);
        }
        if let Some(choice) = chunk
            .get("choices")
            .and_then(Value::as_array)
            .and_then(|v| v.first())
        {
            if let Some(content) = choice.pointer("/delta/content").and_then(Value::as_str) {
                writer.write_all(format!("event: response.output_text.delta\ndata: {}\n\n", json!({"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":content})).as_bytes())?;
            }
            if let Some(tool_calls) = choice
                .pointer("/delta/tool_calls")
                .and_then(Value::as_array)
            {
                for call in tool_calls {
                    let index = int(call.get("index"));
                    if !tool_call_indices.contains_key(&index) {
                        tool_call_indices.insert(index, true);
                        writer.write_all(
                            format!(
                                "event: response.output_item.added\ndata: {}\n\n",
                                json!({
                                    "type": "response.output_item.added",
                                    "output_index": index,
                                    "item": {
                                        "type": "function_call",
                                        "call_id": call.get("id").and_then(Value::as_str).unwrap_or(""),
                                        "name": call.pointer("/function/name").and_then(Value::as_str).unwrap_or(""),
                                        "arguments": ""
                                    }
                                })
                            )
                            .as_bytes(),
                        )?;
                    }
                    if let Some(arguments) =
                        call.pointer("/function/arguments").and_then(Value::as_str)
                    {
                        if !arguments.is_empty() {
                            writer.write_all(
                                format!(
                                    "event: response.function_call_arguments.delta\ndata: {}\n\n",
                                    json!({
                                        "type": "response.function_call_arguments.delta",
                                        "output_index": index,
                                        "delta": arguments
                                    })
                                )
                                .as_bytes(),
                            )?;
                        }
                    }
                }
            }
            if choice
                .get("finish_reason")
                .and_then(Value::as_str)
                .is_some()
            {
                break;
            }
        }
    }
    let normalized = usage.clone().normalize();
    writer.write_all(format!("event: response.completed\ndata: {}\n\n", json!({"type":"response.completed","response":{"id":"resp_stream","object":"response","model":model,"status":"completed","output":[],"usage":{"input_tokens":usage.input_tokens,"output_tokens":usage.output_tokens,"total_tokens":normalized.total_tokens}}})).as_bytes())?;
    Ok(normalized)
}

fn chat_stream_to_anthropic(
    reader: impl Read,
    mut writer: impl Write,
    model: &str,
) -> anyhow::Result<TokenUsage> {
    let usage = chat_stream_to_anthropic_inner(reader, &mut writer, model)?;
    Ok(usage)
}

fn chat_stream_to_anthropic_inner(
    reader: impl Read,
    writer: &mut impl Write,
    model: &str,
) -> anyhow::Result<TokenUsage> {
    let mut usage = TokenUsage::default();
    writer.write_all(format!("event: message_start\ndata: {}\n\n", json!({"type":"message_start","message":{"id":"msg_stream","type":"message","role":"assistant","model":model,"content":[],"usage":{"input_tokens":0,"output_tokens":0}}})).as_bytes())?;
    writer.write_all(b"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")?;
    for frame in read_sse(reader)? {
        if frame.data.trim() == "[DONE]" {
            break;
        }
        let Ok(chunk) = serde_json::from_str::<Value>(&frame.data) else {
            continue;
        };
        if let Some(u) = chunk.get("usage") {
            usage = usage_from_chat(u);
        }
        if let Some(choice) = chunk
            .get("choices")
            .and_then(Value::as_array)
            .and_then(|v| v.first())
        {
            if let Some(content) = choice.pointer("/delta/content").and_then(Value::as_str) {
                writer.write_all(format!("event: content_block_delta\ndata: {}\n\n", json!({"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":content}})).as_bytes())?;
            }
            if choice
                .get("finish_reason")
                .and_then(Value::as_str)
                .is_some()
            {
                break;
            }
        }
    }
    writer.write_all(
        b"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n",
    )?;
    writer.write_all(format!("event: message_delta\ndata: {}\n\n", json!({"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"input_tokens":usage.input_tokens,"output_tokens":usage.output_tokens}})).as_bytes())?;
    writer.write_all(b"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")?;
    Ok(usage.normalize())
}

fn responses_stream_to_anthropic(
    reader: impl Read,
    writer: impl Write,
    model: &str,
) -> anyhow::Result<TokenUsage> {
    let mut chat_buf = Vec::new();
    let usage = responses_stream_to_chat(reader, &mut chat_buf, model)?;
    let _ = chat_stream_to_anthropic(chat_buf.as_slice(), writer, model)?;
    Ok(usage)
}

fn anthropic_stream_to_responses(
    reader: impl Read,
    writer: impl Write,
    model: &str,
) -> anyhow::Result<TokenUsage> {
    let mut chat_buf = Vec::new();
    let usage = anthropic_stream_to_chat(reader, &mut chat_buf, model)?;
    let _ = chat_stream_to_responses(chat_buf.as_slice(), writer, model)?;
    Ok(usage)
}

#[derive(Default)]
struct SSEFrame {
    event: String,
    data: String,
}

fn read_sse(reader: impl Read) -> anyhow::Result<Vec<SSEFrame>> {
    let mut frames = Vec::new();
    let mut event = String::new();
    let mut data = Vec::<String>::new();
    let mut br = BufReader::new(reader);
    let mut line = String::new();
    loop {
        line.clear();
        let n = br.read_line(&mut line)?;
        if n == 0 {
            if !data.is_empty() {
                frames.push(SSEFrame {
                    event: event.clone(),
                    data: data.join("\n").trim().to_string(),
                });
            }
            break;
        }
        let trimmed = line.trim_end_matches(['\n', '\r']);
        if trimmed.is_empty() {
            if !data.is_empty() {
                let joined = data.join("\n").trim().to_string();
                frames.push(SSEFrame {
                    event: event.clone(),
                    data: joined.clone(),
                });
                if joined == "[DONE]" {
                    break;
                }
            }
            event.clear();
            data.clear();
        } else if let Some(rest) = trimmed.strip_prefix("event:") {
            event = rest.trim().to_string();
        } else if let Some(rest) = trimmed.strip_prefix("data:") {
            data.push(rest.strip_prefix(' ').unwrap_or(rest).to_string());
        }
    }
    Ok(frames)
}

fn rewrite_model(body: Vec<u8>, model: &str) -> anyhow::Result<Vec<u8>> {
    if model.is_empty() {
        return Ok(body);
    }
    let mut payload: Value = serde_json::from_slice(&body)?;
    payload["model"] = Value::String(model.to_string());
    Ok(serde_json::to_vec(&payload)?)
}

fn copy_if(src: &Value, dst: &mut Map<String, Value>, key: &str) {
    if let Some(value) = src.get(key) {
        dst.insert(key.to_string(), value.clone());
    }
}

fn chat_messages_to_responses_input(messages: &Value) -> Value {
    let Some(items) = messages.as_array() else {
        return messages.clone();
    };
    Value::Array(
        items
            .iter()
            .map(|item| {
                let Some(obj) = item.as_object() else {
                    return item.clone();
                };
                let mut out = obj.clone();
                if let Some(content) = obj.get("content") {
                    out.insert(
                        "content".to_string(),
                        chat_content_to_responses_content(content),
                    );
                }
                Value::Object(out)
            })
            .collect(),
    )
}

fn chat_content_to_responses_content(content: &Value) -> Value {
    let Some(parts) = content.as_array() else {
        return content.clone();
    };
    Value::Array(
        parts
            .iter()
            .filter_map(|part| {
                let typ = part.get("type").and_then(Value::as_str).unwrap_or("");
                match typ {
                    "text" => part
                        .get("text")
                        .and_then(Value::as_str)
                        .filter(|text| !text.is_empty())
                        .map(|text| json!({"type": "input_text", "text": text})),
                    "image_url" => part
                        .pointer("/image_url/url")
                        .and_then(Value::as_str)
                        .filter(|url| !is_empty_base64_data_uri(url))
                        .map(|url| json!({"type": "input_image", "image_url": url})),
                    _ => Some(part.clone()),
                }
            })
            .collect(),
    )
}

fn chat_tools_to_responses_tools(tools: Option<&Value>, functions: Option<&Value>) -> Vec<Value> {
    let mut out = Vec::new();
    if let Some(items) = tools.and_then(Value::as_array) {
        for item in items {
            if item.get("type").and_then(Value::as_str) != Some("function") {
                continue;
            }
            if let Some(function) = item.get("function").and_then(Value::as_object) {
                out.push(chat_function_to_responses_tool(function));
            }
        }
    }
    if let Some(items) = functions.and_then(Value::as_array) {
        for item in items {
            if let Some(function) = item.as_object() {
                out.push(chat_function_to_responses_tool(function));
            }
        }
    }
    out
}

fn chat_function_to_responses_tool(function: &Map<String, Value>) -> Value {
    let mut out = Map::new();
    out.insert("type".to_string(), json!("function"));
    for key in ["name", "description", "parameters", "strict"] {
        if let Some(value) = function.get(key) {
            out.insert(key.to_string(), value.clone());
        }
    }
    Value::Object(out)
}

fn chat_function_call_to_tool_choice(value: &Value) -> Value {
    if value.is_string() {
        return value.clone();
    }
    if let Some(name) = value.get("name").and_then(Value::as_str) {
        return json!({"type": "function", "name": name});
    }
    value.clone()
}

fn normalize_responses_input(input: Option<&Value>) -> Value {
    match input {
        Some(Value::String(text)) => json!([{"role":"user","content":text}]),
        Some(Value::Array(items)) => Value::Array(items.clone()),
        Some(v) => v.clone(),
        None => json!([]),
    }
}

fn responses_input_to_anthropic_messages(input: Option<&Value>) -> (Option<Value>, Value) {
    match normalize_responses_input(input) {
        Value::Array(items) => {
            let mut system = None;
            let mut messages = Vec::new();
            for item in items {
                let Some(obj) = item.as_object() else {
                    continue;
                };
                let role = obj.get("role").and_then(Value::as_str).unwrap_or("");
                let typ = obj.get("type").and_then(Value::as_str).unwrap_or("");
                match (role, typ) {
                    ("system", _) => {
                        if let Some(text) = obj.get("content").and_then(text_from_content) {
                            if !text.is_empty() {
                                system = Some(Value::String(text));
                            }
                        }
                    }
                    (_, "function_call") => {
                        let input = obj
                            .get("arguments")
                            .and_then(Value::as_str)
                            .and_then(|args| serde_json::from_str::<Value>(args).ok())
                            .unwrap_or_else(|| json!({}));
                        messages.push(json!({
                            "role": "assistant",
                            "content": [{
                                "type": "tool_use",
                                "id": responses_call_id_to_anthropic(obj.get("call_id").and_then(Value::as_str).unwrap_or("")),
                                "name": obj.get("name").and_then(Value::as_str).unwrap_or(""),
                                "input": input
                            }]
                        }));
                    }
                    (_, "function_call_output") => {
                        messages.push(json!({
                            "role": "user",
                            "content": [{
                                "type": "tool_result",
                                "tool_use_id": responses_call_id_to_anthropic(obj.get("call_id").and_then(Value::as_str).unwrap_or("")),
                                "content": obj.get("output").cloned().unwrap_or_else(|| json!(""))
                            }]
                        }));
                    }
                    ("assistant", _) => {
                        messages.push(json!({
                            "role": "assistant",
                            "content": responses_content_to_anthropic_content(obj.get("content"), true)
                        }));
                    }
                    ("user", _) | ("", _) => {
                        messages.push(json!({
                            "role": "user",
                            "content": responses_content_to_anthropic_content(obj.get("content"), false)
                        }));
                    }
                    _ => {
                        messages.push(json!({
                            "role": role,
                            "content": responses_content_to_anthropic_content(obj.get("content"), false)
                        }));
                    }
                }
            }
            (system, Value::Array(messages))
        }
        other => (None, other),
    }
}

fn responses_content_to_anthropic_content(content: Option<&Value>, assistant: bool) -> Value {
    let Some(content) = content else {
        return if assistant {
            json!([{"type": "text", "text": ""}])
        } else {
            json!("")
        };
    };
    if let Some(text) = content.as_str() {
        return if assistant {
            json!([{"type": "text", "text": text}])
        } else {
            json!(text)
        };
    }
    let Some(parts) = content.as_array() else {
        return content.clone();
    };
    let mut blocks = Vec::new();
    for part in parts {
        let typ = part.get("type").and_then(Value::as_str).unwrap_or("");
        match typ {
            "input_text" | "output_text" | "text" => {
                if let Some(text) = part.get("text").and_then(Value::as_str) {
                    if !text.is_empty() {
                        blocks.push(json!({"type": "text", "text": text}));
                    }
                }
            }
            "input_image" => {
                if let Some(source) = part
                    .get("image_url")
                    .and_then(Value::as_str)
                    .and_then(data_uri_to_anthropic_image_source)
                {
                    blocks.push(json!({"type": "image", "source": source}));
                }
            }
            _ => {}
        }
    }
    if blocks.is_empty() {
        if assistant {
            json!([{"type": "text", "text": ""}])
        } else {
            json!("")
        }
    } else {
        Value::Array(blocks)
    }
}

fn responses_tools_to_anthropic_tools(tools: Option<&Value>) -> Vec<Value> {
    let mut out = Vec::new();
    if let Some(items) = tools.and_then(Value::as_array) {
        for item in items {
            let typ = item.get("type").and_then(Value::as_str).unwrap_or("");
            match typ {
                "function" => out.push(json!({
                    "name": item.get("name").and_then(Value::as_str).unwrap_or(""),
                    "description": item.get("description").cloned().unwrap_or_else(|| json!("")),
                    "input_schema": item.get("parameters").cloned().unwrap_or_else(|| json!({"type": "object", "properties": {}}))
                })),
                "web_search" | "google_search" | "web_search_20250305" => out.push(json!({
                    "type": "web_search_20250305",
                    "name": "web_search"
                })),
                _ => out.push(item.clone()),
            }
        }
    }
    out
}

fn responses_tool_choice_to_anthropic(value: &Value) -> Value {
    if let Some(choice) = value.as_str() {
        return match choice {
            "auto" => json!({"type": "auto"}),
            "required" => json!({"type": "any"}),
            "none" => json!({"type": "none"}),
            _ => value.clone(),
        };
    }
    if value.get("type").and_then(Value::as_str) == Some("function") {
        let name = value
            .get("name")
            .or_else(|| value.pointer("/function/name"))
            .and_then(Value::as_str)
            .unwrap_or("");
        if !name.is_empty() {
            return json!({"type": "tool", "name": name});
        }
    }
    value.clone()
}

fn responses_effort_to_anthropic(effort: &str) -> &str {
    if effort == "xhigh" {
        "max"
    } else {
        effort
    }
}

fn thinking_budget(effort: &str) -> i64 {
    match effort {
        "low" => 1024,
        "medium" => 4096,
        "high" => 10240,
        "max" => 32768,
        _ => 10240,
    }
}

fn data_uri_to_anthropic_image_source(raw: &str) -> Option<Value> {
    let rest = raw.strip_prefix("data:")?;
    let (media_type, data) = rest.split_once(";base64,")?;
    Some(json!({"type": "base64", "media_type": media_type, "data": data}))
}

fn is_empty_base64_data_uri(raw: &str) -> bool {
    let Some(rest) = raw.strip_prefix("data:") else {
        return false;
    };
    let Some((_, data)) = rest.split_once(";base64,") else {
        return false;
    };
    data.trim().is_empty()
}

fn responses_call_id_to_anthropic(id: &str) -> String {
    if let Some(rest) = id.strip_prefix("fc_") {
        if rest.starts_with("toolu_") || rest.starts_with("call_") {
            return rest.to_string();
        }
    }
    if !id.starts_with("toolu_") && !id.starts_with("call_") {
        return format!("toolu_{id}");
    }
    id.to_string()
}

fn text_from_content(value: &Value) -> Option<String> {
    match value {
        Value::String(s) => Some(s.clone()),
        Value::Array(items) => Some(
            items
                .iter()
                .filter_map(|part| part.get("text").and_then(Value::as_str))
                .collect::<Vec<_>>()
                .join(""),
        ),
        _ => None,
    }
}

fn response_output_text(resp: &Value) -> String {
    resp.get("output")
        .and_then(Value::as_array)
        .map(|items| {
            items
                .iter()
                .filter_map(|item| item.get("content").and_then(Value::as_array))
                .flat_map(|content| content.iter())
                .filter_map(|part| part.get("text").and_then(Value::as_str))
                .collect::<Vec<_>>()
                .join("")
        })
        .unwrap_or_default()
}

fn response_reasoning_text(resp: &Value) -> String {
    resp.get("output")
        .and_then(Value::as_array)
        .map(|items| {
            items
                .iter()
                .filter(|item| item.get("type").and_then(Value::as_str) == Some("reasoning"))
                .filter_map(|item| item.get("summary").and_then(Value::as_array))
                .flat_map(|summary| summary.iter())
                .filter(|part| part.get("type").and_then(Value::as_str) == Some("summary_text"))
                .filter_map(|part| part.get("text").and_then(Value::as_str))
                .collect::<Vec<_>>()
                .join("")
        })
        .unwrap_or_default()
}

fn response_function_calls_to_chat(resp: &Value) -> Vec<Value> {
    let mut out = Vec::new();
    if let Some(items) = resp.get("output").and_then(Value::as_array) {
        for item in items {
            if item.get("type").and_then(Value::as_str) != Some("function_call") {
                continue;
            }
            out.push(json!({
                "id": item.get("call_id").and_then(Value::as_str).unwrap_or(""),
                "type": "function",
                "function": {
                    "name": item.get("name").and_then(Value::as_str).unwrap_or(""),
                    "arguments": item.get("arguments").and_then(Value::as_str).unwrap_or("")
                }
            }));
        }
    }
    out
}

fn responses_output_to_anthropic_blocks(resp: &Value) -> Vec<Value> {
    let mut blocks = Vec::new();
    if let Some(items) = resp.get("output").and_then(Value::as_array) {
        for item in items {
            match item.get("type").and_then(Value::as_str).unwrap_or("") {
                "message" => {
                    if let Some(content) = item.get("content").and_then(Value::as_array) {
                        for part in content {
                            if let Some(text) = part.get("text").and_then(Value::as_str) {
                                blocks.push(json!({"type":"text","text":text}));
                            }
                        }
                    }
                }
                "function_call" => {
                    let raw_args = item
                        .get("arguments")
                        .and_then(Value::as_str)
                        .unwrap_or("{}");
                    let input =
                        serde_json::from_str::<Value>(raw_args).unwrap_or_else(|_| json!({}));
                    blocks.push(json!({
                        "type": "tool_use",
                        "id": item.get("call_id").and_then(Value::as_str).unwrap_or(""),
                        "name": item.get("name").and_then(Value::as_str).unwrap_or(""),
                        "input": input
                    }));
                }
                _ => {}
            }
        }
    }
    if blocks.is_empty() {
        blocks.push(json!({"type":"text","text":""}));
    }
    blocks
}

fn anthropic_content_text(content: Option<&Value>) -> String {
    content
        .and_then(Value::as_array)
        .map(|items| {
            items
                .iter()
                .filter_map(|part| part.get("text").and_then(Value::as_str))
                .collect::<Vec<_>>()
                .join("")
        })
        .unwrap_or_default()
}

fn finish_from_responses(resp: &Value) -> &'static str {
    match resp.get("status").and_then(Value::as_str) {
        Some("incomplete") => "length",
        Some("failed") => "content_filter",
        _ => "stop",
    }
}

fn chat_finish_from_anthropic(stop: Option<&str>) -> &'static str {
    match stop {
        Some("max_tokens") => "length",
        Some("tool_use") => "tool_calls",
        _ => "stop",
    }
}

fn anthropic_stop_from_chat(finish: Option<&str>) -> &'static str {
    match finish {
        Some("length") => "max_tokens",
        Some("tool_calls") => "tool_use",
        _ => "end_turn",
    }
}

fn responses_usage_to_chat(usage: Option<&Value>) -> Value {
    let u = usage_from_responses_value(usage);
    json!({"prompt_tokens":u.input_tokens,"completion_tokens":u.output_tokens,"total_tokens":u.total_tokens})
}

fn responses_usage_to_anthropic(usage: Option<&Value>) -> Value {
    let u = usage_from_responses_value(usage);
    json!({"input_tokens":u.input_tokens,"output_tokens":u.output_tokens,"cache_creation_input_tokens":0,"cache_read_input_tokens":0})
}

fn chat_usage_to_responses(usage: Option<&Value>) -> Value {
    let u = usage_from_chat_value(usage);
    json!({"input_tokens":u.input_tokens,"output_tokens":u.output_tokens,"total_tokens":u.total_tokens})
}

fn chat_usage_to_anthropic(usage: Option<&Value>) -> Value {
    let u = usage_from_chat_value(usage);
    json!({"input_tokens":u.input_tokens,"output_tokens":u.output_tokens,"cache_creation_input_tokens":0,"cache_read_input_tokens":0})
}

fn anthropic_usage_to_chat(usage: Option<&Value>) -> Value {
    let u = usage_from_anthropic_value(usage);
    json!({"prompt_tokens":u.input_tokens,"completion_tokens":u.output_tokens,"total_tokens":u.total_tokens})
}

fn anthropic_usage_to_responses(usage: Option<&Value>) -> Value {
    let u = usage_from_anthropic_value(usage);
    json!({"input_tokens":u.input_tokens,"output_tokens":u.output_tokens,"total_tokens":u.total_tokens})
}

fn usage_from_responses(v: &Value) -> TokenUsage {
    usage_from_responses_value(Some(v))
}

fn usage_from_chat(v: &Value) -> TokenUsage {
    usage_from_chat_value(Some(v))
}

fn usage_from_responses_value(usage: Option<&Value>) -> TokenUsage {
    TokenUsage {
        input_tokens: usage
            .and_then(|u| u.get("input_tokens"))
            .map_or(0, |v| int(Some(v))),
        output_tokens: usage
            .and_then(|u| u.get("output_tokens"))
            .map_or(0, |v| int(Some(v))),
        total_tokens: usage
            .and_then(|u| u.get("total_tokens"))
            .map_or(0, |v| int(Some(v))),
    }
    .normalize()
}

fn usage_from_chat_value(usage: Option<&Value>) -> TokenUsage {
    TokenUsage {
        input_tokens: usage
            .and_then(|u| u.get("prompt_tokens"))
            .map_or(0, |v| int(Some(v))),
        output_tokens: usage
            .and_then(|u| u.get("completion_tokens"))
            .map_or(0, |v| int(Some(v))),
        total_tokens: usage
            .and_then(|u| u.get("total_tokens"))
            .map_or(0, |v| int(Some(v))),
    }
    .normalize()
}

fn usage_from_anthropic_value(usage: Option<&Value>) -> TokenUsage {
    TokenUsage {
        input_tokens: usage
            .and_then(|u| u.get("input_tokens"))
            .map_or(0, |v| int(Some(v))),
        output_tokens: usage
            .and_then(|u| u.get("output_tokens"))
            .map_or(0, |v| int(Some(v))),
        total_tokens: 0,
    }
    .normalize()
}

fn chat_chunk(model: &str, delta: Value, finish: Option<&str>, usage: Option<Value>) -> Value {
    let mut value = json!({
        "id":"chatcmpl_bridge",
        "object":"chat.completion.chunk",
        "created": chrono::Utc::now().timestamp(),
        "model": model,
        "choices":[{"index":0,"delta":delta,"finish_reason":finish}]
    });
    if let Some(usage) = usage {
        value["usage"] = usage;
    }
    value
}

fn write_sse_json(mut writer: impl Write, value: &Value) -> anyhow::Result<()> {
    writer.write_all(format!("data: {}\n\n", serde_json::to_string(value)?).as_bytes())?;
    Ok(())
}

fn int(value: Option<&Value>) -> i64 {
    match value {
        Some(Value::Number(n)) => n.as_i64().unwrap_or(0),
        _ => 0,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn chat_request_to_responses_rewrites_model() {
        let out = convert_request(
            PROTOCOL_OPENAI_CHAT,
            PROTOCOL_OPENAI_RESPONSES,
            "target-model",
            br#"{"model":"chat-model","messages":[{"role":"user","content":"hello"}]}"#,
        )
        .unwrap();
        let payload: Value = serde_json::from_slice(&out).unwrap();
        assert_eq!(payload["model"], "target-model");
        assert!(payload.get("input").is_some());
    }

    #[test]
    fn responses_response_to_chat_empty_content() {
        let out = convert_response(
            PROTOCOL_OPENAI_CHAT,
            PROTOCOL_OPENAI_RESPONSES,
            "gpt-5.5",
            br#"{"id":"resp_1","object":"response","model":"gpt-5.5","status":"completed","output":[]}"#,
        )
        .unwrap();
        let payload: Value = serde_json::from_slice(&out).unwrap();
        assert_eq!(payload["object"], "chat.completion");
        assert_eq!(payload["choices"][0]["message"]["content"], "");
    }

    #[test]
    fn unsupported_request_direction() {
        let err = convert_request(
            PROTOCOL_ANTHROPIC,
            PROTOCOL_OPENAI_CHAT,
            "",
            br#"{"model":"m"}"#,
        )
        .unwrap_err()
        .to_string();
        assert!(err.contains("not implemented"));
    }

    #[test]
    fn responses_stream_to_chat_done() {
        let input = concat!(
            "event: response.created\n",
            "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt\",\"status\":\"in_progress\"}}\n\n",
            "event: response.output_text.delta\n",
            "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n",
            "event: response.completed\n",
            "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt\",\"status\":\"completed\"}}\n\n",
        );
        let mut out = Vec::new();
        convert_stream(
            PROTOCOL_OPENAI_CHAT,
            PROTOCOL_OPENAI_RESPONSES,
            "target",
            input.as_bytes(),
            &mut out,
        )
        .unwrap();
        let text = String::from_utf8(out).unwrap();
        assert!(text.contains("\"object\":\"chat.completion.chunk\""));
        assert!(text.contains("data: [DONE]"));
    }

    #[test]
    fn responses_stream_to_chat_stops_after_terminal_event_with_trailing_data() {
        let input = concat!(
            "event: response.created\n",
            "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt\",\"status\":\"in_progress\"}}\n\n",
            "event: response.output_text.delta\n",
            "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n",
            "event: response.completed\n",
            "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt\",\"status\":\"completed\"}}\n\n",
            "event: response.output_text.delta\n",
            "data: {\"type\":\"response.output_text.delta\",\"delta\":\"ignored\"}\n\n",
        );
        let mut out = Vec::new();
        convert_stream(
            PROTOCOL_OPENAI_CHAT,
            PROTOCOL_OPENAI_RESPONSES,
            "target",
            input.as_bytes(),
            &mut out,
        )
        .unwrap();
        let text = String::from_utf8(out).unwrap();
        assert!(text.contains("\"content\":\"hello\""));
        assert!(!text.contains("ignored"));
        assert!(text.contains("data: [DONE]"));
    }

    #[test]
    fn chat_stream_to_responses_stops_after_done_with_trailing_data() {
        let input = concat!(
            "data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hello\"},\"finish_reason\":null}]}\n\n",
            "data: [DONE]\n\n",
            "data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ignored\"},\"finish_reason\":null}]}\n\n",
        );
        let mut out = Vec::new();
        convert_stream(
            PROTOCOL_OPENAI_RESPONSES,
            PROTOCOL_OPENAI_CHAT,
            "target",
            input.as_bytes(),
            &mut out,
        )
        .unwrap();
        let text = String::from_utf8(out).unwrap();
        assert!(text.contains("\"delta\":\"hello\""));
        assert!(!text.contains("ignored"));
        assert!(text.contains("event: response.completed"));
    }

    #[test]
    fn responses_stream_to_chat_converts_tool_call_deltas() {
        let input = concat!(
            "event: response.created\n",
            "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt\",\"status\":\"in_progress\"}}\n\n",
            "event: response.output_item.added\n",
            "data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"lookup\"}}\n\n",
            "event: response.function_call_arguments.delta\n",
            "data: {\"type\":\"response.function_call_arguments.delta\",\"output_index\":0,\"delta\":\"{\\\"q\\\":\"}\n\n",
            "event: response.function_call_arguments.delta\n",
            "data: {\"type\":\"response.function_call_arguments.delta\",\"output_index\":0,\"delta\":\"\\\"rust\\\"}\"}\n\n",
            "event: response.completed\n",
            "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt\",\"status\":\"completed\"}}\n\n",
        );
        let mut out = Vec::new();
        convert_stream(
            PROTOCOL_OPENAI_CHAT,
            PROTOCOL_OPENAI_RESPONSES,
            "target",
            input.as_bytes(),
            &mut out,
        )
        .unwrap();
        let text = String::from_utf8(out).unwrap();
        assert!(text.contains("\"tool_calls\""));
        assert!(text.contains("\"id\":\"call_1\""));
        assert!(text.contains("\"name\":\"lookup\""));
        assert!(text.contains("{\\\"q\\\":"));
        assert!(text.contains("\\\"rust\\\"}"));
        assert!(text.contains("\"finish_reason\":\"tool_calls\""));
    }

    #[test]
    fn responses_stream_to_chat_converts_reasoning_deltas() {
        let input = concat!(
            "event: response.created\n",
            "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt\",\"status\":\"in_progress\"}}\n\n",
            "event: response.reasoning_summary_text.delta\n",
            "data: {\"type\":\"response.reasoning_summary_text.delta\",\"delta\":\"I checked.\"}\n\n",
            "event: response.output_text.delta\n",
            "data: {\"type\":\"response.output_text.delta\",\"delta\":\"Use lookup.\"}\n\n",
            "event: response.completed\n",
            "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt\",\"status\":\"completed\"}}\n\n",
        );
        let mut out = Vec::new();
        convert_stream(
            PROTOCOL_OPENAI_CHAT,
            PROTOCOL_OPENAI_RESPONSES,
            "target",
            input.as_bytes(),
            &mut out,
        )
        .unwrap();
        let text = String::from_utf8(out).unwrap();
        assert!(text.contains("\"reasoning_content\":\"I checked.\""));
        assert!(text.contains("\"content\":\"Use lookup.\""));
        assert!(text.contains("\"finish_reason\":\"stop\""));
    }

    #[test]
    fn chat_stream_to_responses_converts_tool_call_deltas() {
        let input = concat!(
            "data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"lookup\",\"arguments\":\"{\\\"q\\\":\"}}]},\"finish_reason\":null}]}\n\n",
            "data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"rust\\\"}\"}}]},\"finish_reason\":null}]}\n\n",
            "data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n",
            "data: [DONE]\n\n",
        );
        let mut out = Vec::new();
        convert_stream(
            PROTOCOL_OPENAI_RESPONSES,
            PROTOCOL_OPENAI_CHAT,
            "target",
            input.as_bytes(),
            &mut out,
        )
        .unwrap();
        let text = String::from_utf8(out).unwrap();
        assert!(text.contains("event: response.output_item.added"));
        assert!(text.contains("\"type\":\"function_call\""));
        assert!(text.contains("\"call_id\":\"call_1\""));
        assert!(text.contains("\"name\":\"lookup\""));
        assert!(text.contains("event: response.function_call_arguments.delta"));
        assert!(text.contains("{\\\"q\\\":"));
        assert!(text.contains("\\\"rust\\\"}"));
        assert!(text.contains("event: response.completed"));
    }

    #[test]
    fn chat_stream_to_anthropic_emits_message_stop() {
        let input = concat!(
            "data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hello\"},\"finish_reason\":null}]}\n\n",
            "data: {\"id\":\"chatcmpl_1\",\"object\":\"chat.completion.chunk\",\"model\":\"gpt\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
            "data: [DONE]\n\n",
        );
        let mut out = Vec::new();
        convert_stream(
            PROTOCOL_ANTHROPIC,
            PROTOCOL_OPENAI_CHAT,
            "target",
            input.as_bytes(),
            &mut out,
        )
        .unwrap();
        let text = String::from_utf8(out).unwrap();
        assert!(text.contains("event: message_start"));
        assert!(text.contains("\"type\":\"text_delta\""));
        assert!(text.contains("event: message_stop"));
    }

    #[test]
    fn chat_response_tool_call_to_responses_function_call() {
        let out = convert_response(
            PROTOCOL_OPENAI_RESPONSES,
            PROTOCOL_OPENAI_CHAT,
            "target-model",
            br#"{"id":"chatcmpl_1","object":"chat.completion","model":"gpt","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{\"q\":\"rust\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}"#,
        )
        .unwrap();
        let payload: Value = serde_json::from_slice(&out).unwrap();
        assert_eq!(payload["output"][1]["type"], "function_call");
        assert_eq!(payload["output"][1]["call_id"], "call_1");
        assert_eq!(payload["output"][1]["name"], "lookup");
        assert_eq!(payload["output"][1]["arguments"], "{\"q\":\"rust\"}");
    }

    #[test]
    fn chat_response_tool_call_to_anthropic_tool_use() {
        let out = convert_response(
            PROTOCOL_ANTHROPIC,
            PROTOCOL_OPENAI_CHAT,
            "target-model",
            br#"{"id":"chatcmpl_1","object":"chat.completion","model":"gpt","choices":[{"index":0,"message":{"role":"assistant","content":"checking","tool_calls":[{"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{\"q\":\"rust\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}"#,
        )
        .unwrap();
        let payload: Value = serde_json::from_slice(&out).unwrap();
        assert_eq!(payload["content"][0]["type"], "text");
        assert_eq!(payload["content"][1]["type"], "tool_use");
        assert_eq!(payload["content"][1]["id"], "call_1");
        assert_eq!(payload["content"][1]["input"]["q"], "rust");
    }

    #[test]
    fn responses_function_call_to_anthropic_tool_use() {
        let out = convert_response(
            PROTOCOL_ANTHROPIC,
            PROTOCOL_OPENAI_RESPONSES,
            "target-model",
            br#"{"id":"resp_1","object":"response","model":"gpt","status":"completed","output":[{"type":"function_call","call_id":"call_1","name":"lookup","arguments":"{\"q\":\"rust\"}"}],"usage":{"input_tokens":2,"output_tokens":3,"total_tokens":5}}"#,
        )
        .unwrap();
        let payload: Value = serde_json::from_slice(&out).unwrap();
        assert_eq!(payload["content"][0]["type"], "tool_use");
        assert_eq!(payload["content"][0]["id"], "call_1");
        assert_eq!(payload["content"][0]["input"]["q"], "rust");
    }

    #[test]
    fn chat_multimodal_request_to_responses_input_parts() {
        let out = convert_request(
            PROTOCOL_OPENAI_CHAT,
            PROTOCOL_OPENAI_RESPONSES,
            "target-model",
            br#"{"model":"gpt","messages":[{"role":"user","content":[{"type":"text","text":"look"},{"type":"image_url","image_url":{"url":"data:image/png;base64,abc"}}]}]}"#,
        )
        .unwrap();
        let payload: Value = serde_json::from_slice(&out).unwrap();
        assert_eq!(payload["input"][0]["role"], "user");
        assert_eq!(payload["input"][0]["content"][0]["type"], "input_text");
        assert_eq!(payload["input"][0]["content"][0]["text"], "look");
        assert_eq!(payload["input"][0]["content"][1]["type"], "input_image");
        assert_eq!(
            payload["input"][0]["content"][1]["image_url"],
            "data:image/png;base64,abc"
        );
    }

    #[test]
    fn chat_tools_and_reasoning_request_to_responses() {
        let out = convert_request(
            PROTOCOL_OPENAI_CHAT,
            PROTOCOL_OPENAI_RESPONSES,
            "target-model",
            br#"{"model":"gpt","messages":[{"role":"user","content":"hello"}],"reasoning_effort":"high","tools":[{"type":"function","function":{"name":"lookup","description":"Lookup docs","parameters":{"type":"object"},"strict":true}}],"tool_choice":{"type":"function","name":"lookup"}}"#,
        )
        .unwrap();
        let payload: Value = serde_json::from_slice(&out).unwrap();
        assert_eq!(payload["reasoning"]["effort"], "high");
        assert_eq!(payload["reasoning"]["summary"], "auto");
        assert_eq!(payload["tools"][0]["type"], "function");
        assert_eq!(payload["tools"][0]["name"], "lookup");
        assert_eq!(payload["tools"][0]["parameters"]["type"], "object");
        assert_eq!(payload["tools"][0]["strict"], true);
        assert_eq!(payload["tool_choice"]["name"], "lookup");
    }

    #[test]
    fn responses_request_to_anthropic_maps_image_reasoning_tools_and_choice() {
        let out = convert_request(
            PROTOCOL_OPENAI_RESPONSES,
            PROTOCOL_ANTHROPIC,
            "target-model",
            br#"{"model":"gpt","input":[{"role":"user","content":[{"type":"input_text","text":"look"},{"type":"input_image","image_url":"data:image/png;base64,abc"}]}],"max_output_tokens":256,"reasoning":{"effort":"xhigh"},"tools":[{"type":"function","name":"lookup","description":"Lookup docs","parameters":{"type":"object"}}],"tool_choice":"required"}"#,
        )
        .unwrap();
        let payload: Value = serde_json::from_slice(&out).unwrap();
        assert_eq!(payload["max_tokens"], 256);
        assert_eq!(payload["output_config"]["effort"], "max");
        assert_eq!(payload["thinking"]["type"], "enabled");
        assert_eq!(payload["messages"][0]["content"][0]["type"], "text");
        assert_eq!(payload["messages"][0]["content"][1]["type"], "image");
        assert_eq!(
            payload["messages"][0]["content"][1]["source"]["media_type"],
            "image/png"
        );
        assert_eq!(
            payload["messages"][0]["content"][1]["source"]["data"],
            "abc"
        );
        assert_eq!(payload["tools"][0]["name"], "lookup");
        assert_eq!(payload["tools"][0]["input_schema"]["type"], "object");
        assert_eq!(payload["tool_choice"]["type"], "any");
    }

    #[test]
    fn responses_reasoning_and_function_call_to_chat_message() {
        let out = convert_response(
            PROTOCOL_OPENAI_CHAT,
            PROTOCOL_OPENAI_RESPONSES,
            "target-model",
            br#"{"id":"resp_1","object":"response","model":"gpt","status":"completed","output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"I checked."}]},{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Use lookup."}]},{"type":"function_call","call_id":"call_1","name":"lookup","arguments":"{\"q\":\"rust\"}"}],"usage":{"input_tokens":2,"output_tokens":3,"total_tokens":5}}"#,
        )
        .unwrap();
        let payload: Value = serde_json::from_slice(&out).unwrap();
        let msg = &payload["choices"][0]["message"];
        assert_eq!(msg["content"], "Use lookup.");
        assert_eq!(msg["reasoning_content"], "I checked.");
        assert_eq!(msg["tool_calls"][0]["id"], "call_1");
        assert_eq!(msg["tool_calls"][0]["function"]["name"], "lookup");
        assert_eq!(payload["choices"][0]["finish_reason"], "tool_calls");
    }
}
