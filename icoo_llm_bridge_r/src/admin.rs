use axum::{
    extract::{ConnectInfo, Path, Query, State},
    http::{HeaderMap, StatusCode},
    Json,
};
use serde::Deserialize;
use serde_json::{json, Value};
use std::{net::SocketAddr, time::Instant};

use crate::{
    auth::{new_id, APIKeyCreateInput},
    http_app::{admin_authorized, AppState},
    model::{
        self, IngressEndpoint, Page, Provider, ProviderModel, Response, ResponseError, RoutingRule,
    },
};

#[derive(Deserialize)]
pub struct PageQuery {
    page: Option<usize>,
    page_size: Option<usize>,
    #[serde(rename = "pageSize")]
    page_size_alt: Option<usize>,
    limit: Option<i64>,
}

#[derive(Deserialize)]
pub struct ProviderInput {
    #[serde(default)]
    pub id: String,
    pub name: String,
    pub protocol: String,
    #[serde(default)]
    pub vendor: String,
    #[serde(default)]
    pub base_url: String,
    #[serde(default)]
    pub api_key: String,
    #[serde(default)]
    pub only_stream: bool,
    #[serde(default)]
    pub user_agent: String,
    #[serde(default)]
    pub enabled: bool,
    #[serde(default)]
    pub description: String,
}

#[derive(Deserialize)]
pub struct ModelInput {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub provider_id: String,
    pub name: String,
    #[serde(default)]
    pub max_tokens: i64,
    #[serde(default)]
    pub enabled: bool,
}

#[derive(Deserialize)]
pub struct EndpointInput {
    #[serde(default)]
    pub id: String,
    pub path: String,
    pub downstream_protocol: String,
    #[serde(default)]
    pub enabled: bool,
    #[serde(default)]
    pub protected: bool,
    #[serde(default)]
    pub description: String,
}

#[derive(Deserialize)]
pub struct RuleInput {
    #[serde(default)]
    pub id: String,
    pub name: String,
    #[serde(default)]
    pub priority: i64,
    #[serde(default)]
    pub match_protocol: String,
    #[serde(default)]
    pub match_model_pattern: String,
    #[serde(default)]
    pub upstream_protocol: String,
    #[serde(default)]
    pub target_provider_id: String,
    #[serde(default)]
    pub target_model: String,
    #[serde(default)]
    pub enabled: bool,
}

pub type ApiResult = (StatusCode, Json<Value>);

pub async fn runtime_state(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    ok(state.runtime_state())
}

pub async fn providers(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Query(q): Query<PageQuery>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.repo.list_providers() {
        Ok(items) => ok(paginate(items, &q)),
        Err(e) => bad(e),
    }
}

pub async fn create_provider(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Json(input): Json<ProviderInput>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    let id = input.id.clone();
    match upsert_provider(&state, id, input) {
        Ok(item) => ok(item),
        Err(e) => bad(e),
    }
}

pub async fn update_provider(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(path_id): Path<String>,
    Json(input): Json<ProviderInput>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    let id = if input.id.trim().is_empty() {
        path_id
    } else {
        input.id.clone()
    };
    match upsert_provider(&state, id, input) {
        Ok(item) => ok(item),
        Err(e) => bad(e),
    }
}

pub async fn delete_provider(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(id): Path<String>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.repo.delete_provider(id.trim()) {
        Ok(_) => ok(json!({"deleted": true})),
        Err(e) => bad(e),
    }
}

pub async fn check_provider(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(id): Path<String>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match run_provider_check(&state, id.trim()).await {
        Ok(item) => ok(item),
        Err(e) => bad(e),
    }
}

pub async fn provider_models(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(provider_id): Path<String>,
    Query(q): Query<PageQuery>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.repo.list_models_by_provider(provider_id.trim()) {
        Ok(items) => ok(paginate(items, &q)),
        Err(e) => bad(e),
    }
}

pub async fn create_model(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(provider_id): Path<String>,
    Json(mut input): Json<ModelInput>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    input.provider_id = provider_id;
    match upsert_model(&state, input) {
        Ok(item) => ok(item),
        Err(e) => bad(e),
    }
}

pub async fn update_model(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path((provider_id, id)): Path<(String, String)>,
    Json(mut input): Json<ModelInput>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    input.provider_id = provider_id;
    if input.id.trim().is_empty() {
        input.id = id;
    }
    match upsert_model(&state, input) {
        Ok(item) => ok(item),
        Err(e) => bad(e),
    }
}

pub async fn delete_model(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path((provider_id, id)): Path<(String, String)>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.repo.delete_model(provider_id.trim(), id.trim()) {
        Ok(_) => ok(json!({"deleted": true})),
        Err(e) => bad(e),
    }
}

pub async fn endpoints(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Query(q): Query<PageQuery>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.repo.list_endpoints() {
        Ok(items) => ok(paginate(items, &q)),
        Err(e) => bad(e),
    }
}

pub async fn create_endpoint(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Json(input): Json<EndpointInput>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    let id = input.id.clone();
    match upsert_endpoint(&state, id, input) {
        Ok(item) => ok(item),
        Err(e) => bad(e),
    }
}

pub async fn update_endpoint(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(path_id): Path<String>,
    Json(input): Json<EndpointInput>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    let id = if input.id.trim().is_empty() {
        path_id
    } else {
        input.id.clone()
    };
    match upsert_endpoint(&state, id, input) {
        Ok(item) => ok(item),
        Err(e) => bad(e),
    }
}

pub async fn delete_endpoint(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(id): Path<String>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.repo.find_endpoint(id.trim()).and_then(|item| {
        if item.built_in {
            anyhow::bail!("built-in endpoint cannot be deleted");
        }
        state.repo.delete_endpoint(&item.id)
    }) {
        Ok(_) => ok(json!({"deleted": true})),
        Err(e) => bad(e),
    }
}

pub async fn rules(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Query(q): Query<PageQuery>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.repo.list_rules() {
        Ok(items) => ok(paginate(items, &q)),
        Err(e) => bad(e),
    }
}

pub async fn create_rule(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Json(input): Json<RuleInput>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    let id = input.id.clone();
    if !id.is_empty() && state.tracker.active_count(&id) > 0 {
        return bad(anyhow::anyhow!(
            "routing rule is currently handling active requests, cannot modify"
        ));
    }
    match upsert_rule(&state, id, input) {
        Ok(item) => ok(item),
        Err(e) => bad(e),
    }
}

pub async fn update_rule(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(path_id): Path<String>,
    Json(input): Json<RuleInput>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    let id = if input.id.trim().is_empty() {
        path_id
    } else {
        input.id.clone()
    };
    if !id.is_empty() && state.tracker.active_count(&id) > 0 {
        return bad(anyhow::anyhow!(
            "routing rule is currently handling active requests, cannot modify"
        ));
    }
    match upsert_rule(&state, id, input) {
        Ok(item) => ok(item),
        Err(e) => bad(e),
    }
}

pub async fn delete_rule(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(id): Path<String>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    if state.tracker.active_count(id.trim()) > 0 {
        return bad(anyhow::anyhow!(
            "routing rule is currently handling active requests, cannot delete"
        ));
    }
    match state.repo.delete_rule(id.trim()) {
        Ok(_) => ok(json!({"deleted": true})),
        Err(e) => bad(e),
    }
}

pub async fn api_keys(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Query(q): Query<PageQuery>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.auth.list_keys() {
        Ok(items) => ok(paginate(items, &q)),
        Err(e) => bad(e),
    }
}

pub async fn api_key_secret(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(id): Path<String>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.auth.get_secret(&id) {
        Ok(secret) => ok(json!({"secret": secret})),
        Err(e) => bad(e),
    }
}

pub async fn create_api_key(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Json(input): Json<APIKeyCreateInput>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.auth.create_key(input) {
        Ok(item) => ok(item),
        Err(e) => bad(e),
    }
}

pub async fn delete_api_key(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Path(id): Path<String>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.auth.delete_key(&id) {
        Ok(_) => ok(json!({"deleted": true})),
        Err(e) => bad(e),
    }
}

pub async fn traffic(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    Query(q): Query<PageQuery>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.repo.list_traffic(q.limit.unwrap_or(500)) {
        Ok(items) => ok(paginate(items, &q)),
        Err(e) => bad(e),
    }
}

pub async fn clear_traffic(
    State(state): State<AppState>,
    headers: HeaderMap,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
) -> ApiResult {
    if let Err(resp) = require_admin(&state, &headers, addr) {
        return resp;
    }
    match state.repo.clear_traffic() {
        Ok(_) => ok(json!({"cleared": true})),
        Err(e) => bad(e),
    }
}

fn upsert_provider(state: &AppState, id: String, input: ProviderInput) -> anyhow::Result<Provider> {
    if input.name.trim().is_empty() {
        anyhow::bail!("name is required");
    }
    if !model::valid_protocol(&input.protocol) {
        anyhow::bail!("protocol is invalid");
    }
    let now = model::now_string();
    let item = Provider {
        id: if id.trim().is_empty() {
            new_id("provider")
        } else {
            id
        },
        name: input.name.trim().to_string(),
        protocol: input.protocol,
        vendor: input.vendor,
        base_url: input.base_url.trim().to_string(),
        api_key_cipher: input.api_key.trim().to_string(),
        only_stream: input.only_stream,
        user_agent: input.user_agent.trim().to_string(),
        enabled: input.enabled,
        description: input.description.trim().to_string(),
        created_at: now.clone(),
        updated_at: now,
    };
    state.repo.save_provider(&item)?;
    Ok(item)
}

async fn run_provider_check(state: &AppState, provider_id: &str) -> anyhow::Result<Value> {
    let provider = state.repo.find_provider(provider_id)?;
    let selected_model = state
        .repo
        .list_models_by_provider(&provider.id)?
        .into_iter()
        .find(|item| item.enabled && !item.name.trim().is_empty());
    let checked_at = model::now_string();

    if !provider.enabled {
        return Ok(provider_health(
            &provider.id,
            "warning",
            0,
            0,
            "供应商未启用。",
            &checked_at,
        ));
    }
    if provider.base_url.trim().is_empty() {
        return Ok(provider_health(
            &provider.id,
            "unreachable",
            0,
            0,
            "基础地址为空。",
            &checked_at,
        ));
    }
    if provider.api_key_cipher.trim().is_empty() {
        return Ok(provider_health(
            &provider.id,
            "unreachable",
            0,
            0,
            "API Key 为空。",
            &checked_at,
        ));
    }

    let Some(selected_model) = selected_model else {
        return Ok(provider_health(
            &provider.id,
            "warning",
            0,
            0,
            "暂无已启用模型，无法发送真实检查请求。",
            &checked_at,
        ));
    };

    let url = join_upstream_url(&provider.base_url, &provider.protocol);
    if url.trim().is_empty() {
        return Ok(provider_health(
            &provider.id,
            "unreachable",
            0,
            0,
            "上游协议或基础地址无效。",
            &checked_at,
        ));
    }

    let body = provider_check_body(&provider.protocol, &selected_model.name)?;
    let start = Instant::now();
    let mut req = state
        .client
        .post(url)
        .header("content-type", "application/json")
        .body(body);
    match provider.protocol.as_str() {
        model::PROTOCOL_ANTHROPIC => {
            req = req
                .header("x-api-key", provider.api_key_cipher.trim())
                .header("anthropic-version", "2023-06-01");
        }
        model::PROTOCOL_OPENAI_CHAT | model::PROTOCOL_OPENAI_RESPONSES => {
            req = req.header(
                "authorization",
                format!("Bearer {}", provider.api_key_cipher.trim()),
            );
        }
        _ => {}
    }
    if !provider.user_agent.trim().is_empty() {
        req = req.header("user-agent", provider.user_agent.trim());
    }

    match req.send().await {
        Ok(resp) => {
            let status = resp.status().as_u16();
            let ok = resp.status().is_success();
            let body = resp.bytes().await.unwrap_or_default();
            let message = if ok {
                format!("连接成功，模型 {} 可用。", selected_model.name)
            } else {
                let detail = upstream_error_message(status, &body);
                format!("连接失败：{}", detail)
            };
            Ok(provider_health(
                &provider.id,
                if ok { "reachable" } else { "unreachable" },
                status,
                start.elapsed().as_millis() as i64,
                &message,
                &checked_at,
            ))
        }
        Err(err) => Ok(provider_health(
            &provider.id,
            "unreachable",
            0,
            start.elapsed().as_millis() as i64,
            &format!("连接失败：{}", err),
            &checked_at,
        )),
    }
}

fn provider_check_body(protocol: &str, model_name: &str) -> anyhow::Result<String> {
    let body = match protocol {
        model::PROTOCOL_ANTHROPIC => json!({
            "model": model_name,
            "messages": [{"role": "user", "content": "Reply with exactly: ok"}],
            "max_tokens": 8,
            "stream": false
        }),
        model::PROTOCOL_OPENAI_CHAT => json!({
            "model": model_name,
            "messages": [{"role": "user", "content": "Reply with exactly: ok"}],
            "max_tokens": 8,
            "stream": false
        }),
        model::PROTOCOL_OPENAI_RESPONSES => json!({
            "model": model_name,
            "input": "Reply with exactly: ok",
            "max_output_tokens": 8,
            "stream": false
        }),
        _ => anyhow::bail!("protocol is invalid"),
    };
    Ok(body.to_string())
}

fn provider_health(
    supplier_id: &str,
    status: &str,
    status_code: u16,
    duration_ms: i64,
    message: &str,
    checked_at: &str,
) -> Value {
    json!({
        "supplier_id": supplier_id,
        "status": status,
        "status_code": status_code,
        "duration_ms": duration_ms,
        "message": message,
        "checked_at": checked_at,
    })
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

fn upstream_error_message(status: u16, body: &[u8]) -> String {
    let fallback = format!("上游返回 HTTP {}", status);
    if body.iter().all(|b| b.is_ascii_whitespace()) {
        return fallback;
    }
    if let Ok(payload) = serde_json::from_slice::<Value>(body) {
        if let Some(message) = payload.pointer("/error/message").and_then(Value::as_str) {
            return format!("{}：{}", fallback, message.trim());
        }
        if let Some(message) = payload.get("message").and_then(Value::as_str) {
            return format!("{}：{}", fallback, message.trim());
        }
    }
    fallback
}

fn upsert_model(state: &AppState, input: ModelInput) -> anyhow::Result<ProviderModel> {
    if input.provider_id.trim().is_empty() {
        anyhow::bail!("provider_id is required");
    }
    if input.name.trim().is_empty() {
        anyhow::bail!("name is required");
    }
    let now = model::now_string();
    let item = ProviderModel {
        id: if input.id.trim().is_empty() {
            new_id("model")
        } else {
            input.id
        },
        provider_id: input.provider_id.trim().to_string(),
        name: input.name.trim().to_string(),
        max_tokens: input.max_tokens,
        enabled: input.enabled,
        created_at: now.clone(),
        updated_at: now,
    };
    state.repo.save_model(&item)?;
    Ok(item)
}

fn upsert_endpoint(
    state: &AppState,
    id: String,
    input: EndpointInput,
) -> anyhow::Result<IngressEndpoint> {
    let mut path = input.path.trim().to_string();
    if path.is_empty() {
        anyhow::bail!("path is required");
    }
    if !path.starts_with('/') {
        path = format!("/{}", path);
    }
    if !model::valid_protocol(&input.downstream_protocol) {
        anyhow::bail!("downstream_protocol is invalid");
    }
    let now = model::now_string();
    let item = IngressEndpoint {
        id: if id.trim().is_empty() {
            new_id("endpoint")
        } else {
            id
        },
        path,
        downstream_protocol: input.downstream_protocol,
        enabled: input.enabled,
        protected: input.protected,
        built_in: false,
        description: input.description.trim().to_string(),
        created_at: now.clone(),
        updated_at: now,
    };
    state.repo.save_endpoint(&item)?;
    Ok(item)
}

fn upsert_rule(state: &AppState, id: String, input: RuleInput) -> anyhow::Result<RoutingRule> {
    if input.name.trim().is_empty() {
        anyhow::bail!("name is required");
    }
    let now = model::now_string();
    let item = RoutingRule {
        id: if id.trim().is_empty() {
            new_id("rule")
        } else {
            id
        },
        name: input.name.trim().to_string(),
        priority: input.priority,
        match_protocol: input.match_protocol,
        match_model_pattern: input.match_model_pattern.trim().to_string(),
        upstream_protocol: input.upstream_protocol,
        target_provider_id: input.target_provider_id.trim().to_string(),
        target_model: input.target_model.trim().to_string(),
        enabled: input.enabled,
        created_at: now.clone(),
        updated_at: now,
    };
    state.repo.save_rule(&item)?;
    Ok(item)
}

fn paginate<T: serde::Serialize + Clone>(items: Vec<T>, q: &PageQuery) -> Page<T> {
    let mut page = q.page.unwrap_or(1);
    let mut page_size = q.page_size.or(q.page_size_alt).unwrap_or(20);
    if page == 0 {
        page = 1;
    }
    if page_size == 0 {
        page_size = 20;
    }
    if page_size > 200 {
        page_size = 200;
    }
    let total = items.len();
    let start = ((page - 1) * page_size).min(total);
    let end = (start + page_size).min(total);
    Page {
        items: items[start..end].to_vec(),
        total,
        page,
        page_size,
    }
}

fn require_admin(state: &AppState, headers: &HeaderMap, addr: SocketAddr) -> Result<(), ApiResult> {
    if admin_authorized(state, headers, addr) {
        Ok(())
    } else {
        Err((
            StatusCode::UNAUTHORIZED,
            Json(json!({"error":{"code":"UNAUTHORIZED","message":"invalid admin api key"}})),
        ))
    }
}

pub fn ok<T: serde::Serialize>(data: T) -> ApiResult {
    (
        StatusCode::OK,
        Json(
            serde_json::to_value(Response {
                data: Some(data),
                error: None::<ResponseError>,
            })
            .unwrap(),
        ),
    )
}

pub fn bad(error: anyhow::Error) -> ApiResult {
    (
        StatusCode::BAD_REQUEST,
        Json(json!({"error":{"code":"BAD_REQUEST","message":error.to_string()}})),
    )
}
