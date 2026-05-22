use std::{net::SocketAddr, sync::Arc};

use axum::{
    extract::{ConnectInfo, State},
    http::{HeaderMap, Method, StatusCode},
    response::IntoResponse,
    routing::{delete, get, post, put},
    Json, Router,
};
use serde_json::json;
use tower_http::cors::{Any, CorsLayer};

use crate::{
    admin,
    auth::{extract_api_key, AuthService},
    config::Config,
    db::Database,
    model::{Response, ResponseError, State as RuntimeState},
    proxy,
    repository::Repository,
    routing::RouteResolver,
    tracker::RequestTracker,
    VERSION,
};

#[derive(Clone)]
pub struct AppState(Arc<AppStateInner>);

pub struct AppStateInner {
    pub cfg: Config,
    pub repo: Repository,
    pub auth: AuthService,
    pub resolver: RouteResolver,
    pub tracker: RequestTracker,
    pub client: reqwest::Client,
    pub stream_client: reqwest::Client,
}

impl std::ops::Deref for AppState {
    type Target = AppStateInner;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl AppState {
    pub fn new(cfg: Config, db: Database) -> Self {
        let repo = Repository::new(db);
        Self(Arc::new(AppStateInner {
            cfg: cfg.clone(),
            auth: AuthService::new(repo.clone()),
            resolver: RouteResolver::new(repo.clone()),
            repo,
            tracker: RequestTracker::default(),
            client: reqwest::Client::builder()
                .timeout(cfg.write_timeout)
                .build()
                .unwrap(),
            stream_client: reqwest::Client::new(),
        }))
    }

    pub fn runtime_state(&self) -> RuntimeState {
        let mut paths = vec![
            "/healthz".to_string(),
            "/readyz".to_string(),
            "/api/v1/runtime/state".to_string(),
        ];
        if let Ok(items) = self.repo.enabled_endpoints() {
            paths.extend(items.into_iter().map(|item| item.path));
        }
        RuntimeState {
            service: "icoo_llm_bridge".to_string(),
            version: VERSION.to_string(),
            running: true,
            listen_addr: self.cfg.addr(),
            paths,
            database: self.repo.database_diagnostics(),
        }
    }
}

pub fn router(state: AppState) -> Router {
    Router::new()
        .route("/", get(index))
        .route("/healthz", get(healthz))
        .route("/readyz", get(readyz))
        .route("/v1/messages", post(proxy::anthropic))
        .route("/v1/chat/completions", post(proxy::openai_chat))
        .route("/v1/responses", post(proxy::openai_responses))
        .route("/api/v1/runtime/state", get(admin::runtime_state))
        .route(
            "/api/v1/providers",
            get(admin::providers).post(admin::create_provider),
        )
        .route(
            "/api/v1/providers/:id",
            put(admin::update_provider).delete(admin::delete_provider),
        )
        .route("/api/v1/providers/:id/check", post(admin::check_provider))
        .route(
            "/api/v1/providers/:provider_id/models",
            get(admin::provider_models).post(admin::create_model),
        )
        .route(
            "/api/v1/providers/:provider_id/models/:id",
            put(admin::update_model).delete(admin::delete_model),
        )
        .route(
            "/api/v1/ingress-endpoints",
            get(admin::endpoints).post(admin::create_endpoint),
        )
        .route(
            "/api/v1/ingress-endpoints/:id",
            put(admin::update_endpoint).delete(admin::delete_endpoint),
        )
        .route(
            "/api/v1/routing-rules",
            get(admin::rules).post(admin::create_rule),
        )
        .route(
            "/api/v1/routing-rules/:id",
            put(admin::update_rule).delete(admin::delete_rule),
        )
        .route(
            "/api/v1/api-keys",
            get(admin::api_keys).post(admin::create_api_key),
        )
        .route("/api/v1/api-keys/:id/secret", get(admin::api_key_secret))
        .route("/api/v1/api-keys/:id", delete(admin::delete_api_key))
        .route(
            "/api/v1/traffic",
            get(admin::traffic).delete(admin::clear_traffic),
        )
        .fallback(proxy::dynamic)
        .layer(
            CorsLayer::new()
                .allow_origin(Any)
                .allow_methods([
                    Method::GET,
                    Method::POST,
                    Method::PUT,
                    Method::DELETE,
                    Method::OPTIONS,
                ])
                .allow_headers(Any),
        )
        .with_state(state)
}

async fn index(State(state): State<AppState>) -> impl IntoResponse {
    Json(state.runtime_state())
}

async fn healthz() -> impl IntoResponse {
    Json(json!({"service":"icoo_llm_bridge","status":"ok"}))
}

async fn readyz() -> impl IntoResponse {
    Json(json!({"service":"icoo_llm_bridge","ready":true}))
}

pub fn admin_authorized(state: &AppState, headers: &HeaderMap, addr: SocketAddr) -> bool {
    if state.cfg.allow_local_without_auth && addr.ip().is_loopback() {
        return true;
    }
    let key = extract_api_key(headers);
    !key.is_empty() && state.auth.verify(&key, "admin")
}

pub fn proxy_authorized(state: &AppState, headers: &HeaderMap, addr: SocketAddr) -> bool {
    if state.cfg.allow_local_without_auth && addr.ip().is_loopback() {
        return true;
    }
    let key = extract_api_key(headers);
    !key.is_empty() && state.auth.verify(&key, "proxy")
}

pub fn error_response(status: StatusCode, code: &str, message: &str) -> impl IntoResponse {
    (
        status,
        Json(Response::<serde_json::Value> {
            data: None,
            error: Some(ResponseError {
                code: code.to_string(),
                message: message.to_string(),
            }),
        }),
    )
}

pub fn request_id() -> String {
    let data: [u8; 8] = rand::random();
    format!("req-{}", hex_lower(&data))
}

fn hex_lower(data: &[u8]) -> String {
    data.iter().map(|b| format!("{:02x}", b)).collect()
}

#[allow(dead_code)]
pub fn _connect_info(addr: SocketAddr) -> ConnectInfo<SocketAddr> {
    ConnectInfo(addr)
}
