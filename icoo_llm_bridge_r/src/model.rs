use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

pub const PROTOCOL_ANTHROPIC: &str = "anthropic";
pub const PROTOCOL_OPENAI_CHAT: &str = "openai-chat";
pub const PROTOCOL_OPENAI_RESPONSES: &str = "openai-responses";

pub fn valid_protocol(value: &str) -> bool {
    matches!(
        value,
        PROTOCOL_ANTHROPIC | PROTOCOL_OPENAI_CHAT | PROTOCOL_OPENAI_RESPONSES
    )
}

#[derive(Clone, Debug, Serialize, Deserialize, Default)]
#[serde(rename_all = "PascalCase")]
pub struct Provider {
    pub id: String,
    pub name: String,
    pub protocol: String,
    pub vendor: String,
    #[serde(rename = "BaseURL")]
    pub base_url: String,
    #[serde(rename = "APIKeyCipher")]
    pub api_key_cipher: String,
    pub only_stream: bool,
    pub user_agent: String,
    pub enabled: bool,
    pub description: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize, Default)]
pub struct ProviderModel {
    pub id: String,
    pub provider_id: String,
    pub name: String,
    pub max_tokens: i64,
    pub enabled: bool,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize, Default)]
#[serde(rename_all = "PascalCase")]
pub struct IngressEndpoint {
    #[serde(rename = "ID")]
    pub id: String,
    pub path: String,
    pub downstream_protocol: String,
    pub enabled: bool,
    pub protected: bool,
    pub built_in: bool,
    pub description: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize, Default)]
#[serde(rename_all = "PascalCase")]
pub struct RoutingRule {
    #[serde(rename = "ID")]
    pub id: String,
    pub name: String,
    pub priority: i64,
    pub match_protocol: String,
    pub match_model_pattern: String,
    pub upstream_protocol: String,
    pub target_provider_id: String,
    pub target_model: String,
    pub enabled: bool,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize, Deserialize, Default)]
#[serde(rename_all = "PascalCase")]
pub struct TrafficRecord {
    #[serde(rename = "ID")]
    pub id: String,
    pub request_id: String,
    pub endpoint: String,
    pub method: String,
    #[serde(rename = "ClientIP")]
    pub client_ip: String,
    pub user_agent: String,
    pub content_type: String,
    pub upstream_protocol: String,
    pub downstream_protocol: String,
    pub route_name: String,
    pub route_source: String,
    pub matched_rule_id: String,
    pub matched_rule_name: String,
    pub requested_model: String,
    pub model: String,
    pub request_body: String,
    pub request_body_bytes: i64,
    pub request_body_truncated: bool,
    pub status_code: i64,
    #[serde(rename = "DurationMS")]
    pub duration_ms: i64,
    pub input_tokens: i64,
    pub output_tokens: i64,
    pub total_tokens: i64,
    pub error: String,
    pub created_at: String,
}

#[derive(Clone, Debug, Default)]
pub struct APIKey {
    pub id: String,
    pub name: String,
    pub secret_hash: String,
    pub secret_preview: String,
    pub secret_cipher: String,
    pub scopes: String,
    pub enabled: bool,
    pub expires_at: Option<String>,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize, Default)]
pub struct APIKeyView {
    pub id: String,
    pub name: String,
    pub secret_preview: String,
    pub can_reveal: bool,
    pub scopes: String,
    pub enabled: bool,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Clone, Debug, Serialize)]
pub struct Page<T: Serialize> {
    pub items: Vec<T>,
    pub total: usize,
    pub page: usize,
    pub page_size: usize,
}

#[derive(Clone, Debug, Serialize)]
pub struct Response<T: Serialize> {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<T>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<ResponseError>,
}

#[derive(Clone, Debug, Serialize)]
pub struct ResponseError {
    pub code: String,
    pub message: String,
}

#[derive(Clone, Debug, Serialize, Default)]
pub struct State {
    pub service: String,
    pub version: String,
    pub running: bool,
    pub listen_addr: String,
    pub paths: Vec<String>,
    pub database: DatabaseDiagnostics,
}

#[derive(Clone, Debug, Serialize, Default)]
pub struct DatabaseDiagnostics {
    pub main_ok: bool,
    pub traffic_ok: bool,
    pub warnings: Vec<String>,
}

#[derive(Clone, Debug, Default)]
pub struct ProviderSnapshot {
    pub id: String,
    pub name: String,
    pub protocol: String,
    pub vendor: String,
    pub base_url: String,
    pub api_key: String,
    pub only_stream: bool,
    pub user_agent: String,
    pub enabled: bool,
    pub description: String,
    pub models: Vec<ProviderModelSnapshot>,
}

#[derive(Clone, Debug, Default)]
pub struct ProviderModelSnapshot {
    pub name: String,
    pub max_tokens: i64,
    pub enabled: bool,
}

#[derive(Clone, Debug, Default)]
pub struct Route {
    pub name: String,
    pub upstream_protocol: String,
    pub model: String,
    pub default_max_tokens: i64,
    pub source: String,
    pub priority: i64,
    pub provider: ProviderSnapshot,
}

#[derive(Clone, Debug, Default)]
pub struct TokenUsage {
    pub input_tokens: i64,
    pub output_tokens: i64,
    pub total_tokens: i64,
}

impl TokenUsage {
    pub fn normalize(mut self) -> Self {
        if self.total_tokens == 0 && (self.input_tokens > 0 || self.output_tokens > 0) {
            self.total_tokens = self.input_tokens + self.output_tokens;
        }
        self
    }
}

pub fn now_string() -> String {
    Utc::now().format("%Y-%m-%dT%H:%M:%S%.9fZ").to_string()
}

pub fn parse_time_rfc3339(value: &str) -> Option<DateTime<Utc>> {
    DateTime::parse_from_rfc3339(value)
        .ok()
        .map(|v| v.with_timezone(&Utc))
}
