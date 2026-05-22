use std::{net::SocketAddr, time::Instant};

use axum::{
    body::{Body, Bytes},
    extract::{ConnectInfo, State},
    http::{HeaderMap, HeaderName, HeaderValue, Method, StatusCode},
    response::{IntoResponse, Response},
};
use reqwest::header::CONTENT_TYPE;
use serde_json::{json, Value};

use crate::{
    http_app::{proxy_authorized, request_id, AppState},
    model::{self, Route, TokenUsage, TrafficRecord},
    protocol,
};

enum ProxyBody {
    Buffered(Vec<u8>),
    Streaming(Body),
}

pub async fn anthropic(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    body: Bytes,
) -> Response {
    handle(
        state,
        headers,
        addr,
        "/v1/messages",
        model::PROTOCOL_ANTHROPIC,
        body,
    )
    .await
}

pub async fn openai_chat(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    body: Bytes,
) -> Response {
    handle(
        state,
        headers,
        addr,
        "/v1/chat/completions",
        model::PROTOCOL_OPENAI_CHAT,
        body,
    )
    .await
}

pub async fn openai_responses(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    body: Bytes,
) -> Response {
    handle(
        state,
        headers,
        addr,
        "/v1/responses",
        model::PROTOCOL_OPENAI_RESPONSES,
        body,
    )
    .await
}

pub async fn dynamic(
    State(state): State<AppState>,
    method: Method,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    uri: axum::http::Uri,
    body: Bytes,
) -> Response {
    if method != Method::POST {
        return StatusCode::NOT_FOUND.into_response();
    }
    let path = normalize_path(uri.path());
    let Ok(endpoints) = state.repo.enabled_endpoints() else {
        return StatusCode::NOT_FOUND.into_response();
    };
    for endpoint in endpoints {
        if normalize_path(&endpoint.path) == path {
            return handle(
                state,
                headers,
                addr,
                &path,
                &endpoint.downstream_protocol,
                body,
            )
            .await;
        }
    }
    StatusCode::NOT_FOUND.into_response()
}

async fn handle(
    state: AppState,
    headers: HeaderMap,
    addr: SocketAddr,
    endpoint: &str,
    downstream: &str,
    body: Bytes,
) -> Response {
    let start = Instant::now();
    let req_id = request_id();
    if !proxy_authorized(&state, &headers, addr) {
        let resp = proxy_error(
            downstream,
            StatusCode::UNAUTHORIZED,
            "invalid proxy api key",
        );
        record(
            &state,
            endpoint,
            "POST",
            &headers,
            addr,
            &req_id,
            downstream,
            &Route::default(),
            StatusCode::UNAUTHORIZED.as_u16() as i64,
            start,
            "invalid proxy api key",
            TokenUsage::default(),
            "",
            &body,
        );
        return with_request_id(resp, &req_id);
    }
    let requested_model = match extract_request_model(&body) {
        Ok(model) => model,
        Err(err) => {
            let resp = proxy_error(downstream, StatusCode::BAD_REQUEST, &err.to_string());
            record(
                &state,
                endpoint,
                "POST",
                &headers,
                addr,
                &req_id,
                downstream,
                &Route::default(),
                StatusCode::BAD_REQUEST.as_u16() as i64,
                start,
                &err.to_string(),
                TokenUsage::default(),
                "",
                &body,
            );
            return with_request_id(resp, &req_id);
        }
    };
    let route = match state.resolver.resolve(downstream, &requested_model) {
        Ok(route) => route,
        Err(err) => {
            let resp = proxy_error(downstream, StatusCode::BAD_REQUEST, &err.to_string());
            record(
                &state,
                endpoint,
                "POST",
                &headers,
                addr,
                &req_id,
                downstream,
                &Route::default(),
                StatusCode::BAD_REQUEST.as_u16() as i64,
                start,
                &err.to_string(),
                TokenUsage::default(),
                &requested_model,
                &body,
            );
            return with_request_id(resp, &req_id);
        }
    };
    let rule_id = extract_rule_id(&route.source);
    if !rule_id.is_empty() {
        state.tracker.acquire(rule_id);
    }
    let result = forward(&state, &headers, downstream, &route, &body).await;
    if !rule_id.is_empty() {
        state.tracker.release(rule_id);
    }
    match result {
        Ok((status, out_headers, out_body, usage)) => {
            let mut resp = match out_body {
                ProxyBody::Buffered(bytes) => (status, bytes).into_response(),
                ProxyBody::Streaming(body) => {
                    let mut resp = Response::new(body);
                    *resp.status_mut() = status;
                    resp
                }
            };
            *resp.headers_mut() = HeaderMap::new();
            copy_headers(resp.headers_mut(), &out_headers);
            resp.headers_mut().insert(
                HeaderName::from_static("x-icoo-request-id"),
                HeaderValue::from_str(&req_id).unwrap(),
            );
            record(
                &state,
                endpoint,
                "POST",
                &headers,
                addr,
                &req_id,
                downstream,
                &route,
                status.as_u16() as i64,
                start,
                "",
                usage,
                &requested_model,
                &body,
            );
            resp
        }
        Err((status, message)) => {
            let resp = proxy_error(downstream, status, &message);
            record(
                &state,
                endpoint,
                "POST",
                &headers,
                addr,
                &req_id,
                downstream,
                &route,
                status.as_u16() as i64,
                start,
                &message,
                TokenUsage::default(),
                &requested_model,
                &body,
            );
            with_request_id(resp, &req_id)
        }
    }
}

async fn forward(
    state: &AppState,
    headers: &HeaderMap,
    downstream: &str,
    route: &Route,
    body: &[u8],
) -> Result<(StatusCode, HeaderMap, ProxyBody, TokenUsage), (StatusCode, String)> {
    let upstream_body =
        protocol::convert_request(downstream, &route.upstream_protocol, &route.model, body)
            .map_err(|e| (StatusCode::BAD_REQUEST, e.to_string()))?;
    let url = join_upstream_url(&route.provider.base_url, &route.upstream_protocol);
    if url.trim().is_empty() {
        return Err((
            StatusCode::BAD_GATEWAY,
            "upstream base_url is required".to_string(),
        ));
    }
    let streaming = protocol::request_wants_stream(&upstream_body);
    let client = if streaming {
        &state.stream_client
    } else {
        &state.client
    };
    let mut req = client
        .post(url)
        .body(upstream_body.clone())
        .header(CONTENT_TYPE, "application/json");
    if let Some(accept) = headers.get("accept").and_then(|v| v.to_str().ok()) {
        if !accept.trim().is_empty() {
            req = req.header("accept", accept);
        }
    }
    match route.upstream_protocol.as_str() {
        model::PROTOCOL_ANTHROPIC => {
            req = req
                .header("x-api-key", &route.provider.api_key)
                .header("anthropic-version", "2023-06-01");
        }
        model::PROTOCOL_OPENAI_CHAT | model::PROTOCOL_OPENAI_RESPONSES => {
            req = req.header(
                "authorization",
                format!("Bearer {}", route.provider.api_key),
            );
        }
        _ => {}
    }
    if !route.provider.user_agent.is_empty() {
        req = req.header("user-agent", &route.provider.user_agent);
    }
    let resp = req.send().await.map_err(|e| {
        (
            StatusCode::BAD_GATEWAY,
            format!("upstream request failed: {}", e),
        )
    })?;
    let status = StatusCode::from_u16(resp.status().as_u16()).unwrap_or(StatusCode::BAD_GATEWAY);
    let headers_out = to_axum_headers(resp.headers());
    let content_type = resp
        .headers()
        .get(CONTENT_TYPE)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_ascii_lowercase();
    if status.is_success()
        && content_type.contains("text/event-stream")
        && downstream == route.upstream_protocol
    {
        let mut h = headers_out;
        h.insert(
            HeaderName::from_static("content-type"),
            HeaderValue::from_static("text/event-stream"),
        );
        h.insert(
            HeaderName::from_static("cache-control"),
            HeaderValue::from_static("no-cache"),
        );
        let body = Body::from_stream(resp.bytes_stream());
        return Ok((status, h, ProxyBody::Streaming(body), TokenUsage::default()));
    }
    let bytes = resp
        .bytes()
        .await
        .map_err(|_| {
            (
                StatusCode::BAD_GATEWAY,
                "read upstream response failed".to_string(),
            )
        })?
        .to_vec();
    if !status.is_success() {
        let message = upstream_error_message(status.as_u16(), &bytes);
        return Err((status, message));
    }
    if content_type.contains("text/event-stream") {
        if let Err(err) = preflight_stream(&bytes) {
            return Err((StatusCode::BAD_GATEWAY, err));
        }
        let mut out = Vec::new();
        let usage = protocol::convert_stream(
            downstream,
            &route.upstream_protocol,
            &route.model,
            bytes.as_slice(),
            &mut out,
        )
        .map_err(|e| (StatusCode::BAD_GATEWAY, e.to_string()))?;
        let mut h = headers_out;
        h.insert(
            HeaderName::from_static("content-type"),
            HeaderValue::from_static("text/event-stream"),
        );
        h.insert(
            HeaderName::from_static("cache-control"),
            HeaderValue::from_static("no-cache"),
        );
        return Ok((status, h, ProxyBody::Buffered(out), usage));
    }
    let converted =
        protocol::convert_response(downstream, &route.upstream_protocol, &route.model, &bytes)
            .map_err(|e| (StatusCode::BAD_GATEWAY, e.to_string()))?;
    let usage = protocol::extract_usage(&bytes);
    if downstream == model::PROTOCOL_OPENAI_CHAT && protocol::request_wants_stream(body) {
        let mut out = Vec::new();
        protocol::write_chat_completion_as_stream(
            &converted,
            protocol::chat_include_usage(body),
            &mut out,
        )
        .map_err(|e| (StatusCode::BAD_GATEWAY, e.to_string()))?;
        let mut h = headers_out;
        h.insert(
            HeaderName::from_static("content-type"),
            HeaderValue::from_static("text/event-stream"),
        );
        h.insert(
            HeaderName::from_static("cache-control"),
            HeaderValue::from_static("no-cache"),
        );
        return Ok((status, h, ProxyBody::Buffered(out), usage));
    }
    Ok((status, headers_out, ProxyBody::Buffered(converted), usage))
}

fn proxy_error(protocol_name: &str, status: StatusCode, message: &str) -> Response {
    let body = if protocol_name == model::PROTOCOL_ANTHROPIC {
        json!({"type":"error","error":{"type":"invalid_request_error","message":message}})
    } else {
        json!({"error":{"type":"invalid_request_error","message":message}})
    };
    let mut resp = (status, body.to_string()).into_response();
    resp.headers_mut().insert(
        HeaderName::from_static("content-type"),
        HeaderValue::from_static("application/json; charset=utf-8"),
    );
    resp
}

fn with_request_id(mut resp: Response, req_id: &str) -> Response {
    resp.headers_mut().insert(
        HeaderName::from_static("x-icoo-request-id"),
        HeaderValue::from_str(req_id).unwrap(),
    );
    resp
}

fn extract_request_model(body: &[u8]) -> anyhow::Result<String> {
    let payload: Value =
        serde_json::from_slice(body).map_err(|_| anyhow::anyhow!("invalid json body"))?;
    Ok(payload
        .get("model")
        .and_then(Value::as_str)
        .unwrap_or("")
        .trim()
        .to_string())
}

fn join_upstream_url(base_url: &str, protocol_name: &str) -> String {
    let base = base_url.trim().trim_end_matches('/');
    if base.is_empty() {
        return String::new();
    }
    let endpoint = match protocol_name {
        model::PROTOCOL_ANTHROPIC => "/v1/messages",
        model::PROTOCOL_OPENAI_CHAT => "/v1/chat/completions",
        model::PROTOCOL_OPENAI_RESPONSES => "/v1/responses",
        _ => "",
    };
    if base.ends_with(endpoint) {
        base.to_string()
    } else if base.ends_with("/v1") {
        format!("{}{}", base, endpoint.trim_start_matches("/v1"))
    } else {
        format!("{}{}", base, endpoint)
    }
}

fn copy_headers(dst: &mut HeaderMap, src: &HeaderMap) {
    for (key, value) in src {
        let lower = key.as_str().to_ascii_lowercase();
        if matches!(
            lower.as_str(),
            "connection"
                | "keep-alive"
                | "proxy-authenticate"
                | "proxy-authorization"
                | "te"
                | "trailer"
                | "trailers"
                | "transfer-encoding"
                | "upgrade"
                | "content-encoding"
                | "content-length"
                | "content-range"
        ) {
            continue;
        }
        dst.append(key.clone(), value.clone());
    }
    if !dst.contains_key("content-type") {
        dst.insert(
            HeaderName::from_static("content-type"),
            HeaderValue::from_static("application/json; charset=utf-8"),
        );
    }
}

fn to_axum_headers(headers: &reqwest::header::HeaderMap) -> HeaderMap {
    let mut out = HeaderMap::new();
    for (key, value) in headers {
        if let Ok(name) = HeaderName::from_bytes(key.as_str().as_bytes()) {
            if let Ok(v) = HeaderValue::from_bytes(value.as_bytes()) {
                out.append(name, v);
            }
        }
    }
    out
}

fn upstream_error_message(status: u16, body: &[u8]) -> String {
    let fallback = format!("upstream returned status {}", status);
    if body.iter().all(|b| b.is_ascii_whitespace()) {
        return fallback;
    }
    if let Ok(payload) = serde_json::from_slice::<Value>(body) {
        if let Some(message) = payload.pointer("/error/message").and_then(Value::as_str) {
            return format!("{}: {}", fallback, message.trim());
        }
        if let Some(message) = payload.get("message").and_then(Value::as_str) {
            return format!("{}: {}", fallback, message.trim());
        }
    }
    fallback
}

fn preflight_stream(body: &[u8]) -> Result<(), String> {
    if body.is_empty() {
        return Err("upstream stream was empty".to_string());
    }
    let text = String::from_utf8_lossy(body);
    let mut event_name = "";
    for line in text.lines().take(20) {
        if let Some(rest) = line.strip_prefix("event:") {
            event_name = rest.trim();
        }
        if let Some(rest) = line.strip_prefix("data:") {
            let data = rest.trim();
            if data == "[DONE]" || data.is_empty() {
                continue;
            }
            if let Ok(payload) = serde_json::from_str::<Value>(data) {
                let typ = payload
                    .get("type")
                    .and_then(Value::as_str)
                    .unwrap_or(event_name);
                if typ == "error" || typ.ends_with(".failed") || event_name.contains("error") {
                    if let Some(msg) = payload.pointer("/error/message").and_then(Value::as_str) {
                        return Err(format!("upstream stream error: {}", msg));
                    }
                    return Err("upstream stream error".to_string());
                }
            }
            return Ok(());
        }
    }
    Err("upstream stream ended before first event".to_string())
}

fn record(
    state: &AppState,
    endpoint: &str,
    method: &str,
    headers: &HeaderMap,
    addr: SocketAddr,
    req_id: &str,
    downstream: &str,
    route: &Route,
    status: i64,
    start: Instant,
    error: &str,
    usage: TokenUsage,
    requested_model: &str,
    body: &[u8],
) {
    let (preview, body_bytes, truncated) = body_preview(state, body);
    let rule_id = extract_rule_id(&route.source).to_string();
    let record = TrafficRecord {
        id: req_id.to_string(),
        request_id: req_id.to_string(),
        endpoint: endpoint.to_string(),
        method: method.to_string(),
        client_ip: addr.ip().to_string(),
        user_agent: header(headers, "user-agent"),
        content_type: header(headers, "content-type"),
        downstream_protocol: downstream.to_string(),
        upstream_protocol: route.upstream_protocol.clone(),
        route_name: route.name.clone(),
        route_source: route.source.clone(),
        matched_rule_id: rule_id.clone(),
        matched_rule_name: if rule_id.is_empty() {
            String::new()
        } else {
            route.name.clone()
        },
        requested_model: requested_model.to_string(),
        model: route.model.clone(),
        request_body: preview,
        request_body_bytes: body_bytes,
        request_body_truncated: truncated,
        status_code: status,
        duration_ms: start.elapsed().as_millis() as i64,
        input_tokens: usage.input_tokens,
        output_tokens: usage.output_tokens,
        total_tokens: usage.normalize().total_tokens,
        error: error.to_string(),
        created_at: model::now_string(),
    };
    let _ = state.repo.record_traffic(&record);
}

fn body_preview(state: &AppState, body: &[u8]) -> (String, i64, bool) {
    let body_bytes = body.len() as i64;
    if !state.cfg.log.chain_log_bodies || body.is_empty() {
        return (String::new(), body_bytes, false);
    }
    let limit = state.cfg.log.chain_log_max_body_bytes;
    if limit == 0 {
        return (String::new(), body_bytes, body_bytes > 0);
    }
    if body.len() > limit {
        (
            String::from_utf8_lossy(&body[..limit]).to_string(),
            body_bytes,
            true,
        )
    } else {
        (String::from_utf8_lossy(body).to_string(), body_bytes, false)
    }
}

fn header(headers: &HeaderMap, name: &str) -> String {
    headers
        .get(name)
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_string()
}

fn extract_rule_id(source: &str) -> &str {
    source.strip_prefix("routing_rule:").unwrap_or("")
}

fn normalize_path(path: &str) -> String {
    let mut path = path.trim().to_string();
    if path.is_empty() {
        return "/".to_string();
    }
    if !path.starts_with('/') {
        path = format!("/{}", path);
    }
    while path.len() > 1 && path.ends_with('/') {
        path.pop();
    }
    path
}
