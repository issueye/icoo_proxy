use rand::{distributions::Alphanumeric, Rng};
use serde::Deserialize;
use sha2::{Digest, Sha256};

use crate::{
    model::{self, APIKey, APIKeyView},
    repository::{to_api_key_view, Repository},
};

#[derive(Clone)]
pub struct AuthService {
    repo: Repository,
}

#[derive(Debug, Deserialize)]
pub struct APIKeyCreateInput {
    #[serde(default)]
    pub id: String,
    pub name: String,
    #[serde(default)]
    pub secret: String,
    #[serde(default)]
    pub scopes: String,
    #[serde(default)]
    pub enabled: bool,
}

impl AuthService {
    pub fn new(repo: Repository) -> Self {
        Self { repo }
    }

    pub fn verify(&self, secret: &str, scope: &str) -> bool {
        let secret = secret.trim();
        if secret.is_empty() {
            return false;
        }
        let sum = hash_secret(secret);
        let Ok(keys) = self.repo.list_enabled_keys() else {
            return false;
        };
        keys.into_iter().any(|key| {
            scope_allowed(&key.scopes, scope)
                && !is_expired(key.expires_at.as_deref())
                && constant_time_eq(key.secret_hash.as_bytes(), sum.as_bytes())
        })
    }

    pub fn list_keys(&self) -> anyhow::Result<Vec<APIKeyView>> {
        Ok(self.repo.list_keys()?.iter().map(to_api_key_view).collect())
    }

    pub fn get_secret(&self, id: &str) -> anyhow::Result<String> {
        let key = self.repo.find_key(id.trim())?;
        let secret = key.secret_cipher.trim();
        if secret.is_empty() {
            anyhow::bail!("该 Key 创建于旧版本，明文不可恢复，请重新生成后再复制");
        }
        Ok(secret.to_string())
    }

    pub fn create_key(&self, input: APIKeyCreateInput) -> anyhow::Result<APIKeyView> {
        let name = input.name.trim();
        if name.is_empty() {
            anyhow::bail!("name is required");
        }
        let id = input.id.trim();
        let scopes = if input.scopes.trim().is_empty() {
            "proxy".to_string()
        } else {
            input.scopes.trim().to_string()
        };

        let mut item = if id.is_empty() {
            let now = model::now_string();
            APIKey {
                id: new_id("key"),
                created_at: now,
                ..Default::default()
            }
        } else {
            self.repo.find_key(id)?
        };

        let mut secret = input.secret.trim().to_string();
        let generated = secret.is_empty() && id.is_empty();
        if generated {
            secret = new_id("sk");
        }
        if !secret.is_empty() {
            item.secret_hash = hash_secret(&secret);
            item.secret_preview = preview_secret(&secret);
            item.secret_cipher = secret.clone();
        } else if item.secret_hash.is_empty() {
            anyhow::bail!("secret is required");
        }
        item.name = name.to_string();
        item.scopes = scopes;
        item.enabled = input.enabled;
        item.updated_at = model::now_string();
        self.repo.save_key(&item)?;
        let mut view = to_api_key_view(&item);
        if generated || !input.secret.trim().is_empty() {
            view.secret_preview = secret;
        }
        Ok(view)
    }

    pub fn delete_key(&self, id: &str) -> anyhow::Result<()> {
        self.repo.delete_key(id.trim())
    }
}

pub fn extract_api_key(headers: &axum::http::HeaderMap) -> String {
    if let Some(key) = headers.get("x-api-key").and_then(|v| v.to_str().ok()) {
        let key = key.trim();
        if !key.is_empty() {
            return key.to_string();
        }
    }
    let auth = headers
        .get("authorization")
        .and_then(|v| v.to_str().ok())
        .unwrap_or("")
        .trim();
    if auth.len() > 7 && auth[..7].eq_ignore_ascii_case("Bearer ") {
        return auth[7..].trim().to_string();
    }
    String::new()
}

pub fn hash_secret(secret: &str) -> String {
    let mut hasher = Sha256::new();
    hasher.update(secret.as_bytes());
    format!("{:x}", hasher.finalize())
}

fn preview_secret(secret: &str) -> String {
    let secret = secret.trim();
    if secret.len() <= 8 {
        "****".to_string()
    } else {
        format!("{}...{}", &secret[..4], &secret[secret.len() - 4..])
    }
}

fn scope_allowed(scopes: &str, scope: &str) -> bool {
    scopes
        .split(',')
        .map(str::trim)
        .any(|part| part == "*" || part == scope)
}

fn is_expired(expires_at: Option<&str>) -> bool {
    expires_at
        .and_then(model::parse_time_rfc3339)
        .map(|t| chrono::Utc::now() > t)
        .unwrap_or(false)
}

fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
    if a.len() != b.len() {
        return false;
    }
    let mut diff = 0u8;
    for (x, y) in a.iter().zip(b.iter()) {
        diff |= x ^ y;
    }
    diff == 0
}

pub fn new_id(prefix: &str) -> String {
    let suffix: String = rand::thread_rng()
        .sample_iter(&Alphanumeric)
        .take(16)
        .map(char::from)
        .collect();
    format!("{}-{}", prefix, suffix.to_lowercase())
}
