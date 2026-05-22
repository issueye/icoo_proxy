use std::{
    net::SocketAddr,
    sync::{Arc, Mutex},
};

use axum::{
    body::Bytes,
    extract::State,
    http::{HeaderMap, StatusCode},
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use futures_util::StreamExt;
use icoo_llm_bridge::{
    auth::APIKeyCreateInput,
    config,
    db::Database,
    http_app::{self, AppState},
    model,
};
use serde_json::{json, Value};
use tokio::sync::oneshot;

#[derive(Clone, Default)]
struct CapturedUpstream {
    path: Arc<Mutex<String>>,
    authorization: Arc<Mutex<String>>,
    model: Arc<Mutex<String>>,
}

#[tokio::test]
async fn dynamic_endpoint_proxies_and_records_traffic() {
    let upstream_capture = CapturedUpstream::default();
    let upstream_addr = spawn_upstream(upstream_capture.clone()).await;

    let temp = tempfile::tempdir().unwrap();
    let mut cfg = config::defaults();
    cfg.allow_local_without_auth = false;
    cfg.apply_data_dir(temp.path().to_str().unwrap());
    let db = Database::open(&cfg).unwrap();
    db.seed_defaults().unwrap();
    let state = AppState::new(cfg, db);
    state
        .auth
        .create_key(APIKeyCreateInput {
            name: "admin".into(),
            secret: "admin-secret".into(),
            scopes: "admin".into(),
            enabled: true,
            id: String::new(),
        })
        .unwrap();
    state
        .auth
        .create_key(APIKeyCreateInput {
            name: "proxy".into(),
            secret: "proxy-secret".into(),
            scopes: "proxy".into(),
            enabled: true,
            id: String::new(),
        })
        .unwrap();
    let bridge_addr = spawn_bridge(state).await;
    let base = format!("http://{}", bridge_addr);
    let client = reqwest::Client::new();

    let admin = |req: reqwest::RequestBuilder| req.header("x-api-key", "admin-secret");
    let proxy = |req: reqwest::RequestBuilder| req.header("x-api-key", "proxy-secret");

    let resp = admin(client.post(format!("{}/api/v1/providers", base)))
        .json(&json!({
            "id": "provider-openai",
            "name": "OpenAI",
            "protocol": model::PROTOCOL_OPENAI_RESPONSES,
            "vendor": "openai",
            "base_url": format!("http://{}", upstream_addr),
            "api_key": "upstream-secret",
            "enabled": true
        }))
        .send()
        .await
        .unwrap();
    assert_eq!(resp.status(), StatusCode::OK);

    let resp = admin(client.post(format!("{}/api/v1/providers/provider-openai/models", base)))
        .json(&json!({"name":"gpt-5.4","max_tokens":32768,"enabled":true}))
        .send()
        .await
        .unwrap();
    assert_ok(resp).await;

    let resp = admin(client.post(format!("{}/api/v1/routing-rules", base)))
        .json(&json!({
            "name": "custom responses",
            "priority": 1,
            "match_protocol": model::PROTOCOL_OPENAI_RESPONSES,
            "match_model_pattern": "*",
            "target_provider_id": "provider-openai",
            "target_model": "gpt-5.4",
            "enabled": true
        }))
        .send()
        .await
        .unwrap();
    assert_ok(resp).await;

    let resp = admin(client.post(format!("{}/api/v1/ingress-endpoints", base)))
        .json(&json!({
            "path": "/responses",
            "downstream_protocol": model::PROTOCOL_OPENAI_RESPONSES,
            "enabled": true
        }))
        .send()
        .await
        .unwrap();
    assert_ok(resp).await;

    let resp = proxy(client.post(format!("{}/responses", base)))
        .json(&json!({"model":"gpt-5.4","input":"hello"}))
        .send()
        .await
        .unwrap();
    let status = resp.status();
    let body_text = resp.text().await.unwrap();
    assert_eq!(status, StatusCode::OK, "{}", body_text);
    let body: Value = serde_json::from_str(&body_text).unwrap();
    assert_eq!(body["object"], "response");

    assert_eq!(&*upstream_capture.path.lock().unwrap(), "/v1/responses");
    assert_eq!(
        &*upstream_capture.authorization.lock().unwrap(),
        "Bearer upstream-secret"
    );
    assert_eq!(&*upstream_capture.model.lock().unwrap(), "gpt-5.4");

    let traffic: Value = admin(client.get(format!("{}/api/v1/traffic?limit=10", base)))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let items = traffic["data"]["items"].as_array().unwrap();
    assert!(items.iter().any(|item| {
        item["endpoint"] == "/responses"
            || item["Endpoint"] == "/responses"
            || item["endpoint"] == Value::String("/responses".to_string())
    }));
}

#[tokio::test]
async fn provider_check_sends_minimal_real_request() {
    let upstream_capture = CapturedUpstream::default();
    let upstream_addr = spawn_upstream(upstream_capture.clone()).await;

    let temp = tempfile::tempdir().unwrap();
    let mut cfg = config::defaults();
    cfg.allow_local_without_auth = false;
    cfg.apply_data_dir(temp.path().to_str().unwrap());
    let db = Database::open(&cfg).unwrap();
    db.seed_defaults().unwrap();
    let state = AppState::new(cfg, db);
    state
        .auth
        .create_key(APIKeyCreateInput {
            name: "admin".into(),
            secret: "admin-secret".into(),
            scopes: "admin".into(),
            enabled: true,
            id: String::new(),
        })
        .unwrap();
    let bridge_addr = spawn_bridge(state).await;
    let base = format!("http://{}", bridge_addr);
    let client = reqwest::Client::new();
    let admin = |req: reqwest::RequestBuilder| req.header("x-api-key", "admin-secret");

    assert_ok(
        admin(client.post(format!("{}/api/v1/providers", base)))
            .json(&json!({
                "id": "provider-check",
                "name": "checkable",
                "protocol": model::PROTOCOL_OPENAI_RESPONSES,
                "vendor": "openai",
                "base_url": format!("http://{}", upstream_addr),
                "api_key": "upstream-secret",
                "enabled": true
            }))
            .send()
            .await
            .unwrap(),
    )
    .await;
    assert_ok(
        admin(client.post(format!("{}/api/v1/providers/provider-check/models", base)))
            .json(&json!({"name":"gpt-check","max_tokens":32768,"enabled":true}))
            .send()
            .await
            .unwrap(),
    )
    .await;

    let body: Value = admin(client.post(format!(
        "{}/api/v1/providers/provider-check/check",
        base
    )))
    .send()
    .await
    .unwrap()
    .json()
    .await
    .unwrap();
    assert_eq!(body["data"]["supplier_id"], "provider-check");
    assert_eq!(body["data"]["status"], "reachable");
    assert_eq!(body["data"]["status_code"], 200);
    assert!(
        body["data"]["duration_ms"].as_i64().unwrap() >= 0,
        "{}",
        body
    );
    assert_eq!(&*upstream_capture.path.lock().unwrap(), "/v1/responses");
    assert_eq!(
        &*upstream_capture.authorization.lock().unwrap(),
        "Bearer upstream-secret"
    );
    assert_eq!(&*upstream_capture.model.lock().unwrap(), "gpt-check");
}

#[tokio::test]
async fn streaming_passthrough_returns_first_event_before_upstream_finishes() {
    let (first_sent_tx, first_sent_rx) = oneshot::channel::<()>();
    let (finish_tx, finish_rx) = oneshot::channel::<()>();
    let upstream_addr = spawn_delayed_stream_upstream(first_sent_tx, finish_rx).await;

    let temp = tempfile::tempdir().unwrap();
    let mut cfg = config::defaults();
    cfg.allow_local_without_auth = false;
    cfg.apply_data_dir(temp.path().to_str().unwrap());
    let db = Database::open(&cfg).unwrap();
    db.seed_defaults().unwrap();
    let state = AppState::new(cfg, db);
    state
        .auth
        .create_key(APIKeyCreateInput {
            name: "proxy".into(),
            secret: "proxy-secret".into(),
            scopes: "proxy".into(),
            enabled: true,
            id: String::new(),
        })
        .unwrap();
    let now = model::now_string();
    state
        .repo
        .save_provider(&model::Provider {
            id: "provider-stream".into(),
            name: "stream".into(),
            protocol: model::PROTOCOL_OPENAI_RESPONSES.into(),
            vendor: "custom".into(),
            base_url: format!("http://{}", upstream_addr),
            api_key_cipher: "upstream-secret".into(),
            enabled: true,
            created_at: now.clone(),
            updated_at: now.clone(),
            ..Default::default()
        })
        .unwrap();
    state
        .repo
        .save_model(&model::ProviderModel {
            id: "model-stream".into(),
            provider_id: "provider-stream".into(),
            name: "gpt-stream".into(),
            max_tokens: 32768,
            enabled: true,
            created_at: now.clone(),
            updated_at: now.clone(),
        })
        .unwrap();
    state
        .repo
        .save_rule(&model::RoutingRule {
            id: "rule-stream".into(),
            name: "stream passthrough".into(),
            priority: 1,
            match_protocol: model::PROTOCOL_OPENAI_RESPONSES.into(),
            match_model_pattern: "*".into(),
            target_provider_id: "provider-stream".into(),
            target_model: "gpt-stream".into(),
            enabled: true,
            created_at: now.clone(),
            updated_at: now,
            ..Default::default()
        })
        .unwrap();
    let bridge_addr = spawn_bridge(state).await;

    let send_task = tokio::spawn(async move {
        reqwest::Client::new()
            .post(format!("http://{}/v1/responses", bridge_addr))
            .header("x-api-key", "proxy-secret")
            .json(&json!({"model":"gpt-stream","stream":true,"input":"hello"}))
            .send()
            .await
    });

    let _ = first_sent_rx.await;
    let resp = match tokio::time::timeout(std::time::Duration::from_millis(300), send_task).await {
        Ok(result) => result.unwrap().unwrap(),
        Err(_) => {
            let _ = finish_tx.send(());
            panic!("bridge did not return response headers before upstream stream finished");
        }
    };
    assert_eq!(resp.status(), StatusCode::OK);

    let mut stream = resp.bytes_stream();
    let first_chunk = tokio::time::timeout(std::time::Duration::from_millis(300), stream.next())
        .await
        .expect("bridge did not forward first stream chunk before upstream finished")
        .expect("stream ended before first chunk")
        .expect("stream chunk error");
    let first_text = String::from_utf8_lossy(&first_chunk);
    assert!(first_text.contains("response.created"), "{first_text}");

    let _ = finish_tx.send(());
}

#[tokio::test]
async fn admin_api_key_reveal_update_and_endpoint_pagination_compatibility() {
    let temp = tempfile::tempdir().unwrap();
    let mut cfg = config::defaults();
    cfg.allow_local_without_auth = false;
    cfg.apply_data_dir(temp.path().to_str().unwrap());
    let db = Database::open(&cfg).unwrap();
    db.seed_defaults().unwrap();
    let state = AppState::new(cfg, db);
    state
        .auth
        .create_key(APIKeyCreateInput {
            name: "admin".into(),
            secret: "admin-secret".into(),
            scopes: "admin".into(),
            enabled: true,
            id: String::new(),
        })
        .unwrap();
    let bridge_addr = spawn_bridge(state).await;
    let base = format!("http://{}", bridge_addr);
    let client = reqwest::Client::new();
    let admin = |req: reqwest::RequestBuilder| req.header("x-api-key", "admin-secret");

    let created: Value = admin(client.post(format!("{}/api/v1/api-keys", base)))
        .json(&json!({
            "name": "copyable",
            "secret": "copyable-secret",
            "scopes": "proxy",
            "enabled": true
        }))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    let id = created["data"]["id"].as_str().unwrap();

    let secret: Value = admin(client.get(format!("{}/api/v1/api-keys/{}/secret", base, id)))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    assert_eq!(secret["data"]["secret"], "copyable-secret");

    assert_ok(
        admin(client.post(format!("{}/api/v1/api-keys", base)))
            .json(&json!({
                "id": id,
                "name": "copyable-updated",
                "secret": "",
                "scopes": "admin,proxy",
                "enabled": true
            }))
            .send()
            .await
            .unwrap(),
    )
    .await;

    let secret_after_update: Value =
        admin(client.get(format!("{}/api/v1/api-keys/{}/secret", base, id)))
            .send()
            .await
            .unwrap()
            .json()
            .await
            .unwrap();
    assert_eq!(secret_after_update["data"]["secret"], "copyable-secret");

    let delete_builtin = admin(client.delete(format!(
        "{}/api/v1/ingress-endpoints/endpoint-v1-responses",
        base
    )))
    .send()
    .await
    .unwrap();
    assert_eq!(delete_builtin.status(), StatusCode::BAD_REQUEST);
    let delete_body: Value = delete_builtin.json().await.unwrap();
    assert!(delete_body["error"]["message"]
        .as_str()
        .unwrap()
        .contains("built-in endpoint cannot be deleted"));

    let page_size: Value = admin(client.get(format!(
        "{}/api/v1/ingress-endpoints?page=1&page_size=2",
        base
    )))
    .send()
    .await
    .unwrap()
    .json()
    .await
    .unwrap();
    assert_eq!(page_size["data"]["page_size"], 2);

    let page_size_alias: Value = admin(client.get(format!(
        "{}/api/v1/ingress-endpoints?page=1&pageSize=2",
        base
    )))
    .send()
    .await
    .unwrap()
    .json()
    .await
    .unwrap();
    assert_eq!(page_size_alias["data"]["page_size"], 2);
}

#[tokio::test]
async fn runtime_state_reports_database_diagnostics_for_current_schema() {
    let temp = tempfile::tempdir().unwrap();
    let mut cfg = config::defaults();
    cfg.allow_local_without_auth = false;
    cfg.apply_data_dir(temp.path().to_str().unwrap());
    let db = Database::open(&cfg).unwrap();
    db.seed_defaults().unwrap();
    let state = AppState::new(cfg, db);
    state
        .auth
        .create_key(APIKeyCreateInput {
            name: "admin".into(),
            secret: "admin-secret".into(),
            scopes: "admin".into(),
            enabled: true,
            id: String::new(),
        })
        .unwrap();
    let bridge_addr = spawn_bridge(state).await;
    let body: Value = reqwest::Client::new()
        .get(format!("http://{}/api/v1/runtime/state", bridge_addr))
        .header("x-api-key", "admin-secret")
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    assert_eq!(body["data"]["database"]["main_ok"], true);
    assert_eq!(body["data"]["database"]["traffic_ok"], true);
    assert_eq!(
        body["data"]["database"]["warnings"]
            .as_array()
            .unwrap()
            .len(),
        0
    );
}

#[tokio::test]
async fn runtime_state_reports_warning_for_accepted_old_schema_gap() {
    let temp = tempfile::tempdir().unwrap();
    let mut cfg = config::defaults();
    cfg.allow_local_without_auth = false;
    cfg.apply_data_dir(temp.path().to_str().unwrap());
    let db = Database::open(&cfg).unwrap();
    db.seed_defaults().unwrap();
    {
        let conn = db.main.lock().unwrap();
        conn.execute_batch(
            r#"
ALTER TABLE routing_rules RENAME TO routing_rules_new;
CREATE TABLE routing_rules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  priority INTEGER,
  match_protocol TEXT,
  match_model_pattern TEXT,
  target_provider_id TEXT,
  target_model TEXT,
  enabled INTEGER,
  created_at TEXT,
  updated_at TEXT
);
DROP TABLE routing_rules_new;
"#,
        )
        .unwrap();
    }
    let state = AppState::new(cfg, db);
    state
        .auth
        .create_key(APIKeyCreateInput {
            name: "admin".into(),
            secret: "admin-secret".into(),
            scopes: "admin".into(),
            enabled: true,
            id: String::new(),
        })
        .unwrap();
    let bridge_addr = spawn_bridge(state).await;
    let body: Value = reqwest::Client::new()
        .get(format!("http://{}/api/v1/runtime/state", bridge_addr))
        .header("x-api-key", "admin-secret")
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    assert_eq!(body["data"]["database"]["main_ok"], false);
    assert!(body["data"]["database"]["warnings"]
        .as_array()
        .unwrap()
        .iter()
        .any(|item| item
            .as_str()
            .unwrap()
            .contains("routing_rules is missing column upstream_protocol")));
}

#[tokio::test]
async fn old_schema_rules_and_api_keys_remain_readable() {
    let temp = tempfile::tempdir().unwrap();
    let mut cfg = config::defaults();
    cfg.allow_local_without_auth = false;
    cfg.apply_data_dir(temp.path().to_str().unwrap());
    let db = Database::open(&cfg).unwrap();
    db.seed_defaults().unwrap();
    {
        let conn = db.main.lock().unwrap();
        conn.execute_batch(
            r#"
ALTER TABLE routing_rules RENAME TO routing_rules_new;
CREATE TABLE routing_rules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  priority INTEGER,
  match_protocol TEXT,
  match_model_pattern TEXT,
  target_provider_id TEXT,
  target_model TEXT,
  enabled INTEGER,
  created_at TEXT,
  updated_at TEXT
);
INSERT INTO routing_rules (id, name, priority, match_protocol, match_model_pattern, target_provider_id, target_model, enabled, created_at, updated_at)
VALUES ('rule-old', 'old route', 1, 'openai-chat', '*', 'provider-old', 'gpt-old', 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
DROP TABLE routing_rules_new;

ALTER TABLE api_keys RENAME TO api_keys_new;
CREATE TABLE api_keys (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  secret_hash TEXT NOT NULL,
  secret_preview TEXT,
  scopes TEXT,
  enabled INTEGER,
  expires_at TEXT,
  created_at TEXT,
  updated_at TEXT
);
"#,
        )
        .unwrap();
    }
    let state = AppState::new(cfg, db);
    state
        .auth
        .create_key(APIKeyCreateInput {
            name: "admin".into(),
            secret: "admin-secret".into(),
            scopes: "admin,proxy".into(),
            enabled: true,
            id: String::new(),
        })
        .unwrap();
    let bridge_addr = spawn_bridge(state).await;
    let base = format!("http://{}", bridge_addr);
    let client = reqwest::Client::new();
    let admin = |req: reqwest::RequestBuilder| req.header("x-api-key", "admin-secret");

    let rules: Value = admin(client.get(format!("{}/api/v1/routing-rules", base)))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    assert_eq!(rules["data"]["items"][0]["Name"], "old route");
    assert_eq!(rules["data"]["items"][0]["UpstreamProtocol"], "");

    let keys: Value = admin(client.get(format!("{}/api/v1/api-keys", base)))
        .send()
        .await
        .unwrap()
        .json()
        .await
        .unwrap();
    assert_eq!(keys["data"]["items"][0]["name"], "admin");
    assert_eq!(keys["data"]["items"][0]["can_reveal"], false);
}

async fn spawn_bridge(state: AppState) -> SocketAddr {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    tokio::spawn(async move {
        axum::serve(
            listener,
            http_app::router(state).into_make_service_with_connect_info::<SocketAddr>(),
        )
        .await
        .unwrap();
    });
    addr
}

async fn assert_ok(resp: reqwest::Response) {
    let status = resp.status();
    let body = resp.text().await.unwrap();
    assert_eq!(status, StatusCode::OK, "{}", body);
}

async fn spawn_upstream(captured: CapturedUpstream) -> SocketAddr {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    let app = Router::new()
        .route("/v1/responses", post(upstream_responses))
        .with_state(captured);
    tokio::spawn(async move {
        axum::serve(listener, app).await.unwrap();
    });
    addr
}

async fn spawn_delayed_stream_upstream(
    first_sent: oneshot::Sender<()>,
    finish: oneshot::Receiver<()>,
) -> SocketAddr {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:0").await.unwrap();
    let addr = listener.local_addr().unwrap();
    let first_sent = Arc::new(Mutex::new(Some(first_sent)));
    let finish = Arc::new(Mutex::new(Some(finish)));
    let app = Router::new()
        .route(
            "/v1/responses",
            post({
                let first_sent = first_sent.clone();
                let finish = finish.clone();
                move || delayed_stream(first_sent.clone(), finish.clone())
            }),
        )
        .route("/healthz", get(|| async { "ok" }));
    tokio::spawn(async move {
        axum::serve(listener, app).await.unwrap();
    });
    addr
}

async fn delayed_stream(
    first_sent: Arc<Mutex<Option<oneshot::Sender<()>>>>,
    finish: Arc<Mutex<Option<oneshot::Receiver<()>>>>,
) -> impl IntoResponse {
    let receiver = finish.lock().unwrap().take().unwrap();
    let stream = async_stream::stream! {
        yield Ok::<Bytes, std::io::Error>(Bytes::from_static(
            b"event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-stream\",\"status\":\"in_progress\"}}\n\n",
        ));
        if let Some(sender) = first_sent.lock().unwrap().take() {
            let _ = sender.send(());
        }
        let _ = receiver.await;
        yield Ok::<Bytes, std::io::Error>(Bytes::from_static(
            b"event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-stream\",\"status\":\"completed\",\"output\":[]}}\n\n",
        ));
    };
    (
        [("content-type", "text/event-stream")],
        axum::body::Body::from_stream(stream),
    )
}

async fn upstream_responses(
    State(captured): State<CapturedUpstream>,
    headers: HeaderMap,
    uri: axum::http::Uri,
    body: Bytes,
) -> impl IntoResponse {
    let payload: Value = serde_json::from_slice(&body).unwrap();
    *captured.path.lock().unwrap() = uri.path().to_string();
    *captured.authorization.lock().unwrap() = headers
        .get("authorization")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .to_string();
    *captured.model.lock().unwrap() = payload["model"].as_str().unwrap_or("").to_string();
    Json(json!({
        "id": "resp_test",
        "object": "response",
        "model": payload["model"],
        "status": "completed",
        "output": [{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],
        "usage": {"input_tokens":1,"output_tokens":2,"total_tokens":3}
    }))
}
