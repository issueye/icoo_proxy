use axum::{
    extract::{ConnectInfo, Path, Query, State},
    http::{HeaderMap, StatusCode},
    Json,
};
use serde::Deserialize;
use serde_json::{json, Value};
use std::net::SocketAddr;

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
