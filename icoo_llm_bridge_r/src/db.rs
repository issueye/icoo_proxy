use std::{
    path::Path,
    sync::{Arc, Mutex},
};

use anyhow::Context;
use rusqlite::{params, Connection};

use crate::{
    config::Config,
    model::{self, DatabaseDiagnostics},
};

#[derive(Clone)]
pub struct Database {
    pub main: Arc<Mutex<Connection>>,
    pub traffic: Arc<Mutex<Connection>>,
}

impl Database {
    pub fn open(cfg: &Config) -> anyhow::Result<Self> {
        ensure_parent(&cfg.db_path)?;
        ensure_parent(&cfg.traffic_db_path)?;
        let main = Connection::open(&cfg.db_path)
            .with_context(|| format!("open {}", cfg.db_path.display()))?;
        let traffic = Connection::open(&cfg.traffic_db_path)
            .with_context(|| format!("open {}", cfg.traffic_db_path.display()))?;
        let db = Self {
            main: Arc::new(Mutex::new(main)),
            traffic: Arc::new(Mutex::new(traffic)),
        };
        db.migrate()?;
        Ok(db)
    }

    fn migrate(&self) -> anyhow::Result<()> {
        let main = self.main.lock().unwrap();
        main.execute_batch(
            r#"
CREATE TABLE IF NOT EXISTS providers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  protocol TEXT,
  vendor TEXT,
  base_url TEXT,
  api_key_cipher TEXT,
  only_stream INTEGER,
  user_agent TEXT,
  enabled INTEGER,
  description TEXT,
  created_at TEXT,
  updated_at TEXT
);
CREATE TABLE IF NOT EXISTS provider_models (
  id TEXT PRIMARY KEY,
  provider_id TEXT NOT NULL,
  name TEXT NOT NULL,
  max_tokens INTEGER,
  enabled INTEGER,
  created_at TEXT,
  updated_at TEXT,
  UNIQUE(provider_id, name)
);
CREATE TABLE IF NOT EXISTS ingress_endpoints (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL UNIQUE,
  downstream_protocol TEXT,
  enabled INTEGER,
  protected INTEGER,
  built_in INTEGER,
  description TEXT,
  created_at TEXT,
  updated_at TEXT
);
CREATE TABLE IF NOT EXISTS routing_rules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  priority INTEGER,
  match_protocol TEXT,
  match_model_pattern TEXT,
  upstream_protocol TEXT,
  target_provider_id TEXT,
  target_model TEXT,
  enabled INTEGER,
  created_at TEXT,
  updated_at TEXT
);
CREATE TABLE IF NOT EXISTS api_keys (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  secret_hash TEXT NOT NULL UNIQUE,
  secret_preview TEXT,
  secret_cipher TEXT,
  scopes TEXT,
  enabled INTEGER,
  expires_at TEXT,
  created_at TEXT,
  updated_at TEXT
);
CREATE TABLE IF NOT EXISTS ui_preferences (
  key TEXT PRIMARY KEY,
  value_json TEXT,
  created_at TEXT,
  updated_at TEXT
);
"#,
        )?;
        drop(main);

        let traffic = self.traffic.lock().unwrap();
        traffic.execute_batch(
            r#"
CREATE TABLE IF NOT EXISTS traffic_records (
  id TEXT PRIMARY KEY,
  request_id TEXT NOT NULL UNIQUE,
  endpoint TEXT,
  method TEXT,
  client_ip TEXT,
  user_agent TEXT,
  content_type TEXT,
  upstream_protocol TEXT,
  downstream_protocol TEXT,
  route_name TEXT,
  route_source TEXT,
  matched_rule_id TEXT,
  matched_rule_name TEXT,
  request_model TEXT,
  model TEXT,
  request_body TEXT,
  request_body_bytes INTEGER,
  request_body_truncated INTEGER,
  status_code INTEGER,
  duration_ms INTEGER,
  input_tokens INTEGER,
  output_tokens INTEGER,
  total_tokens INTEGER,
  error TEXT,
  created_at TEXT
);
"#,
        )?;
        Ok(())
    }

    pub fn seed_defaults(&self) -> anyhow::Result<()> {
        let now = model::now_string();
        let endpoints = [
            (
                "endpoint-v1-messages",
                "/v1/messages",
                model::PROTOCOL_ANTHROPIC,
                "Anthropic Messages compatible endpoint.",
            ),
            (
                "endpoint-v1-chat-completions",
                "/v1/chat/completions",
                model::PROTOCOL_OPENAI_CHAT,
                "OpenAI Chat Completions compatible endpoint.",
            ),
            (
                "endpoint-v1-responses",
                "/v1/responses",
                model::PROTOCOL_OPENAI_RESPONSES,
                "OpenAI Responses compatible endpoint.",
            ),
        ];
        let conn = self.main.lock().unwrap();
        for (id, path, protocol, description) in endpoints {
            conn.execute(
                r#"
INSERT OR REPLACE INTO ingress_endpoints
(id, path, downstream_protocol, enabled, protected, built_in, description, created_at, updated_at)
VALUES (?1, ?2, ?3, 1, 1, 1, ?4, ?5, ?5)
"#,
                params![id, path, protocol, description, now],
            )?;
        }
        Ok(())
    }

    pub fn diagnostics(&self) -> DatabaseDiagnostics {
        let mut diagnostics = DatabaseDiagnostics {
            main_ok: true,
            traffic_ok: true,
            warnings: Vec::new(),
        };

        {
            let conn = self.main.lock().unwrap();
            for (table, columns) in MAIN_SCHEMA_COLUMNS {
                check_table_columns(
                    &conn,
                    table,
                    columns,
                    &mut diagnostics.main_ok,
                    &mut diagnostics.warnings,
                );
            }
        }

        {
            let conn = self.traffic.lock().unwrap();
            for (table, columns) in TRAFFIC_SCHEMA_COLUMNS {
                check_table_columns(
                    &conn,
                    table,
                    columns,
                    &mut diagnostics.traffic_ok,
                    &mut diagnostics.warnings,
                );
            }
        }

        diagnostics
    }
}

const MAIN_SCHEMA_COLUMNS: &[(&str, &[&str])] = &[
    (
        "providers",
        &[
            "id",
            "name",
            "protocol",
            "vendor",
            "base_url",
            "api_key_cipher",
            "only_stream",
            "user_agent",
            "enabled",
            "description",
            "created_at",
            "updated_at",
        ],
    ),
    (
        "provider_models",
        &[
            "id",
            "provider_id",
            "name",
            "max_tokens",
            "enabled",
            "created_at",
            "updated_at",
        ],
    ),
    (
        "ingress_endpoints",
        &[
            "id",
            "path",
            "downstream_protocol",
            "enabled",
            "protected",
            "built_in",
            "description",
            "created_at",
            "updated_at",
        ],
    ),
    (
        "routing_rules",
        &[
            "id",
            "name",
            "priority",
            "match_protocol",
            "match_model_pattern",
            "upstream_protocol",
            "target_provider_id",
            "target_model",
            "enabled",
            "created_at",
            "updated_at",
        ],
    ),
    (
        "api_keys",
        &[
            "id",
            "name",
            "secret_hash",
            "secret_preview",
            "secret_cipher",
            "scopes",
            "enabled",
            "expires_at",
            "created_at",
            "updated_at",
        ],
    ),
    (
        "ui_preferences",
        &["key", "value_json", "created_at", "updated_at"],
    ),
];

const TRAFFIC_SCHEMA_COLUMNS: &[(&str, &[&str])] = &[(
    "traffic_records",
    &[
        "id",
        "request_id",
        "endpoint",
        "method",
        "client_ip",
        "user_agent",
        "content_type",
        "upstream_protocol",
        "downstream_protocol",
        "route_name",
        "route_source",
        "matched_rule_id",
        "matched_rule_name",
        "request_model",
        "model",
        "request_body",
        "request_body_bytes",
        "request_body_truncated",
        "status_code",
        "duration_ms",
        "input_tokens",
        "output_tokens",
        "total_tokens",
        "error",
        "created_at",
    ],
)];

fn check_table_columns(
    conn: &Connection,
    table: &str,
    expected_columns: &[&str],
    ok: &mut bool,
    warnings: &mut Vec<String>,
) {
    let columns = match table_columns(conn, table) {
        Ok(columns) if !columns.is_empty() => columns,
        Ok(_) => {
            *ok = false;
            warnings.push(format!("database table {table} is missing"));
            return;
        }
        Err(err) => {
            *ok = false;
            warnings.push(format!("database table {table} cannot be inspected: {err}"));
            return;
        }
    };
    for column in expected_columns {
        if !columns.iter().any(|item| item == column) {
            *ok = false;
            warnings.push(format!("database table {table} is missing column {column}"));
        }
    }
}

fn table_columns(conn: &Connection, table: &str) -> rusqlite::Result<Vec<String>> {
    let mut stmt = conn.prepare(&format!("PRAGMA table_info({})", quote_identifier(table)))?;
    let rows = stmt.query_map([], |row| row.get::<_, String>(1))?;
    rows.collect()
}

fn quote_identifier(value: &str) -> String {
    format!("\"{}\"", value.replace('"', "\"\""))
}

fn ensure_parent(path: &Path) -> anyhow::Result<()> {
    if let Some(parent) = path.parent() {
        if !parent.as_os_str().is_empty() {
            std::fs::create_dir_all(parent)?;
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{config, repository::Repository};

    #[test]
    fn initializes_separate_main_and_traffic_databases() {
        let dir = tempfile::tempdir().unwrap();
        let mut cfg = config::defaults();
        cfg.apply_data_dir(dir.path().to_str().unwrap());
        let db = Database::open(&cfg).unwrap();
        db.seed_defaults().unwrap();
        let repo = Repository::new(db);

        assert!(!repo.traffic_table_exists_in_main());
        let endpoints = repo.enabled_endpoints().unwrap();
        assert!(endpoints.iter().any(|e| e.path == "/v1/messages"));
        assert!(endpoints.iter().any(|e| e.path == "/v1/chat/completions"));
        assert!(endpoints.iter().any(|e| e.path == "/v1/responses"));
    }
}
