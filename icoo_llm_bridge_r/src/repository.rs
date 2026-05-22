use anyhow::bail;
use rusqlite::{params, OptionalExtension, Row};

use crate::{
    db::Database,
    model::{
        self, APIKey, DatabaseDiagnostics, IngressEndpoint, Provider, ProviderModel, RoutingRule,
        TrafficRecord,
    },
};

#[derive(Clone)]
pub struct Repository {
    db: Database,
}

impl Repository {
    pub fn new(db: Database) -> Self {
        Self { db }
    }

    pub fn database_diagnostics(&self) -> DatabaseDiagnostics {
        self.db.diagnostics()
    }

    pub fn list_providers(&self) -> anyhow::Result<Vec<Provider>> {
        let conn = self.db.main.lock().unwrap();
        let mut stmt = conn.prepare("SELECT * FROM providers ORDER BY name ASC")?;
        let result = rows(stmt.query_map([], provider_from_row)?);
        result
    }

    pub fn find_provider(&self, id: &str) -> anyhow::Result<Provider> {
        let conn = self.db.main.lock().unwrap();
        conn.query_row(
            "SELECT * FROM providers WHERE id = ?1",
            [id],
            provider_from_row,
        )
        .map_err(Into::into)
    }

    pub fn save_provider(&self, item: &Provider) -> anyhow::Result<()> {
        let conn = self.db.main.lock().unwrap();
        conn.execute(
            r#"
INSERT OR REPLACE INTO providers
(id, name, protocol, vendor, base_url, api_key_cipher, only_stream, user_agent, enabled, description, created_at, updated_at)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)
"#,
            params![
                item.id,
                item.name,
                item.protocol,
                item.vendor,
                item.base_url,
                item.api_key_cipher,
                item.only_stream as i64,
                item.user_agent,
                item.enabled as i64,
                item.description,
                item.created_at,
                item.updated_at
            ],
        )?;
        Ok(())
    }

    pub fn delete_provider(&self, id: &str) -> anyhow::Result<()> {
        self.db
            .main
            .lock()
            .unwrap()
            .execute("DELETE FROM providers WHERE id = ?1", [id])?;
        Ok(())
    }

    pub fn list_models_by_provider(&self, provider_id: &str) -> anyhow::Result<Vec<ProviderModel>> {
        let conn = self.db.main.lock().unwrap();
        let mut stmt =
            conn.prepare("SELECT * FROM provider_models WHERE provider_id = ?1 ORDER BY name ASC")?;
        let result = rows(stmt.query_map([provider_id], provider_model_from_row)?);
        result
    }

    pub fn save_model(&self, item: &ProviderModel) -> anyhow::Result<()> {
        let conn = self.db.main.lock().unwrap();
        conn.execute(
            r#"
INSERT OR REPLACE INTO provider_models
(id, provider_id, name, max_tokens, enabled, created_at, updated_at)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)
"#,
            params![
                item.id,
                item.provider_id,
                item.name,
                item.max_tokens,
                item.enabled as i64,
                item.created_at,
                item.updated_at
            ],
        )?;
        Ok(())
    }

    pub fn delete_model(&self, provider_id: &str, id: &str) -> anyhow::Result<()> {
        self.db.main.lock().unwrap().execute(
            "DELETE FROM provider_models WHERE provider_id = ?1 AND id = ?2",
            params![provider_id, id],
        )?;
        Ok(())
    }

    pub fn list_endpoints(&self) -> anyhow::Result<Vec<IngressEndpoint>> {
        let conn = self.db.main.lock().unwrap();
        let mut stmt =
            conn.prepare("SELECT * FROM ingress_endpoints ORDER BY built_in DESC, path ASC")?;
        let result = rows(stmt.query_map([], endpoint_from_row)?);
        result
    }

    pub fn enabled_endpoints(&self) -> anyhow::Result<Vec<IngressEndpoint>> {
        let conn = self.db.main.lock().unwrap();
        let mut stmt = conn.prepare(
            "SELECT * FROM ingress_endpoints WHERE enabled = 1 ORDER BY built_in DESC, path ASC",
        )?;
        let result = rows(stmt.query_map([], endpoint_from_row)?);
        result
    }

    pub fn find_endpoint(&self, id: &str) -> anyhow::Result<IngressEndpoint> {
        self.db
            .main
            .lock()
            .unwrap()
            .query_row(
                "SELECT * FROM ingress_endpoints WHERE id = ?1",
                [id],
                endpoint_from_row,
            )
            .map_err(Into::into)
    }

    pub fn save_endpoint(&self, item: &IngressEndpoint) -> anyhow::Result<()> {
        let conn = self.db.main.lock().unwrap();
        conn.execute(
            r#"
INSERT OR REPLACE INTO ingress_endpoints
(id, path, downstream_protocol, enabled, protected, built_in, description, created_at, updated_at)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)
"#,
            params![
                item.id,
                item.path,
                item.downstream_protocol,
                item.enabled as i64,
                item.protected as i64,
                item.built_in as i64,
                item.description,
                item.created_at,
                item.updated_at
            ],
        )?;
        Ok(())
    }

    pub fn delete_endpoint(&self, id: &str) -> anyhow::Result<()> {
        self.db
            .main
            .lock()
            .unwrap()
            .execute("DELETE FROM ingress_endpoints WHERE id = ?1", [id])?;
        Ok(())
    }

    pub fn list_rules(&self) -> anyhow::Result<Vec<RoutingRule>> {
        let conn = self.db.main.lock().unwrap();
        let upstream_protocol = optional_column_expr(&conn, "routing_rules", "upstream_protocol")?;
        let sql = format!(
            "SELECT id, name, priority, match_protocol, match_model_pattern, {upstream_protocol} AS upstream_protocol, target_provider_id, target_model, enabled, created_at, updated_at FROM routing_rules ORDER BY priority ASC, name ASC"
        );
        let mut stmt = conn.prepare(&sql)?;
        let result = rows(stmt.query_map([], rule_from_row)?);
        result
    }

    pub fn list_enabled_rules(&self) -> anyhow::Result<Vec<RoutingRule>> {
        let conn = self.db.main.lock().unwrap();
        let upstream_protocol = optional_column_expr(&conn, "routing_rules", "upstream_protocol")?;
        let sql = format!(
            "SELECT id, name, priority, match_protocol, match_model_pattern, {upstream_protocol} AS upstream_protocol, target_provider_id, target_model, enabled, created_at, updated_at FROM routing_rules WHERE enabled = 1 ORDER BY priority ASC"
        );
        let mut stmt = conn.prepare(&sql)?;
        let result = rows(stmt.query_map([], rule_from_row)?);
        result
    }

    pub fn save_rule(&self, item: &RoutingRule) -> anyhow::Result<()> {
        let conn = self.db.main.lock().unwrap();
        if table_has_column(&conn, "routing_rules", "upstream_protocol")? {
            conn.execute(
                r#"
INSERT OR REPLACE INTO routing_rules
(id, name, priority, match_protocol, match_model_pattern, upstream_protocol, target_provider_id, target_model, enabled, created_at, updated_at)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11)
"#,
                params![
                    item.id,
                    item.name,
                    item.priority,
                    item.match_protocol,
                    item.match_model_pattern,
                    item.upstream_protocol,
                    item.target_provider_id,
                    item.target_model,
                    item.enabled as i64,
                    item.created_at,
                    item.updated_at
                ],
            )?;
        } else {
            conn.execute(
                r#"
INSERT OR REPLACE INTO routing_rules
(id, name, priority, match_protocol, match_model_pattern, target_provider_id, target_model, enabled, created_at, updated_at)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10)
"#,
                params![
                    item.id,
                    item.name,
                    item.priority,
                    item.match_protocol,
                    item.match_model_pattern,
                    item.target_provider_id,
                    item.target_model,
                    item.enabled as i64,
                    item.created_at,
                    item.updated_at
                ],
            )?;
        }
        Ok(())
    }

    pub fn delete_rule(&self, id: &str) -> anyhow::Result<()> {
        self.db
            .main
            .lock()
            .unwrap()
            .execute("DELETE FROM routing_rules WHERE id = ?1", [id])?;
        Ok(())
    }

    pub fn list_keys(&self) -> anyhow::Result<Vec<APIKey>> {
        let conn = self.db.main.lock().unwrap();
        let secret_cipher = optional_column_expr(&conn, "api_keys", "secret_cipher")?;
        let sql = format!(
            "SELECT id, name, secret_hash, secret_preview, {secret_cipher} AS secret_cipher, scopes, enabled, expires_at, created_at, updated_at FROM api_keys ORDER BY created_at DESC"
        );
        let mut stmt = conn.prepare(&sql)?;
        let result = rows(stmt.query_map([], api_key_from_row)?);
        result
    }

    pub fn list_enabled_keys(&self) -> anyhow::Result<Vec<APIKey>> {
        let conn = self.db.main.lock().unwrap();
        let secret_cipher = optional_column_expr(&conn, "api_keys", "secret_cipher")?;
        let sql = format!(
            "SELECT id, name, secret_hash, secret_preview, {secret_cipher} AS secret_cipher, scopes, enabled, expires_at, created_at, updated_at FROM api_keys WHERE enabled = 1"
        );
        let mut stmt = conn.prepare(&sql)?;
        let result = rows(stmt.query_map([], api_key_from_row)?);
        result
    }

    pub fn find_key(&self, id: &str) -> anyhow::Result<APIKey> {
        let conn = self.db.main.lock().unwrap();
        let secret_cipher = optional_column_expr(&conn, "api_keys", "secret_cipher")?;
        let sql = format!(
            "SELECT id, name, secret_hash, secret_preview, {secret_cipher} AS secret_cipher, scopes, enabled, expires_at, created_at, updated_at FROM api_keys WHERE id = ?1"
        );
        conn.query_row(&sql, [id], api_key_from_row)
            .map_err(Into::into)
    }

    pub fn save_key(&self, item: &APIKey) -> anyhow::Result<()> {
        if item.secret_hash.trim().is_empty() {
            bail!("secret is required");
        }
        let conn = self.db.main.lock().unwrap();
        if table_has_column(&conn, "api_keys", "secret_cipher")? {
            conn.execute(
                r#"
INSERT OR REPLACE INTO api_keys
(id, name, secret_hash, secret_preview, secret_cipher, scopes, enabled, expires_at, created_at, updated_at)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10)
"#,
                params![
                    item.id,
                    item.name,
                    item.secret_hash,
                    item.secret_preview,
                    item.secret_cipher,
                    item.scopes,
                    item.enabled as i64,
                    item.expires_at,
                    item.created_at,
                    item.updated_at
                ],
            )?;
        } else {
            conn.execute(
                r#"
INSERT OR REPLACE INTO api_keys
(id, name, secret_hash, secret_preview, scopes, enabled, expires_at, created_at, updated_at)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)
"#,
                params![
                    item.id,
                    item.name,
                    item.secret_hash,
                    item.secret_preview,
                    item.scopes,
                    item.enabled as i64,
                    item.expires_at,
                    item.created_at,
                    item.updated_at
                ],
            )?;
        }
        Ok(())
    }

    pub fn delete_key(&self, id: &str) -> anyhow::Result<()> {
        self.db
            .main
            .lock()
            .unwrap()
            .execute("DELETE FROM api_keys WHERE id = ?1", [id])?;
        Ok(())
    }

    pub fn record_traffic(&self, item: &TrafficRecord) -> anyhow::Result<()> {
        self.db.traffic.lock().unwrap().execute(
            r#"
INSERT INTO traffic_records
(id, request_id, endpoint, method, client_ip, user_agent, content_type, upstream_protocol, downstream_protocol,
 route_name, route_source, matched_rule_id, matched_rule_name, request_model, model, request_body, request_body_bytes,
 request_body_truncated, status_code, duration_ms, input_tokens, output_tokens, total_tokens, error, created_at)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, ?16, ?17, ?18, ?19, ?20, ?21, ?22, ?23, ?24, ?25)
"#,
            params![
                item.id,
                item.request_id,
                item.endpoint,
                item.method,
                item.client_ip,
                item.user_agent,
                item.content_type,
                item.upstream_protocol,
                item.downstream_protocol,
                item.route_name,
                item.route_source,
                item.matched_rule_id,
                item.matched_rule_name,
                item.requested_model,
                item.model,
                item.request_body,
                item.request_body_bytes,
                item.request_body_truncated as i64,
                item.status_code,
                item.duration_ms,
                item.input_tokens,
                item.output_tokens,
                item.total_tokens,
                item.error,
                item.created_at
            ],
        )?;
        Ok(())
    }

    pub fn list_traffic(&self, mut limit: i64) -> anyhow::Result<Vec<TrafficRecord>> {
        if limit <= 0 || limit > 500 {
            limit = 100;
        }
        let conn = self.db.traffic.lock().unwrap();
        let mut stmt =
            conn.prepare("SELECT * FROM traffic_records ORDER BY created_at DESC LIMIT ?1")?;
        let result = rows(stmt.query_map([limit], traffic_from_row)?);
        result
    }

    pub fn clear_traffic(&self) -> anyhow::Result<()> {
        self.db
            .traffic
            .lock()
            .unwrap()
            .execute("DELETE FROM traffic_records WHERE 1 = 1", [])?;
        Ok(())
    }

    pub fn traffic_table_exists_in_main(&self) -> bool {
        self.db
            .main
            .lock()
            .unwrap()
            .query_row(
                "SELECT name FROM sqlite_master WHERE type='table' AND name='traffic_records'",
                [],
                |row| row.get::<_, String>(0),
            )
            .optional()
            .ok()
            .flatten()
            .is_some()
    }
}

fn rows<T>(iter: impl Iterator<Item = rusqlite::Result<T>>) -> anyhow::Result<Vec<T>> {
    iter.collect::<rusqlite::Result<Vec<T>>>()
        .map_err(Into::into)
}

fn optional_column_expr(
    conn: &rusqlite::Connection,
    table: &str,
    column: &str,
) -> anyhow::Result<String> {
    if table_has_column(conn, table, column)? {
        Ok(column.to_string())
    } else {
        Ok("''".to_string())
    }
}

fn table_has_column(
    conn: &rusqlite::Connection,
    table: &str,
    column: &str,
) -> anyhow::Result<bool> {
    let sql = format!("PRAGMA table_info({})", quote_identifier(table));
    let mut stmt = conn.prepare(&sql)?;
    let mut rows = stmt.query([])?;
    while let Some(row) = rows.next()? {
        let name: String = row.get(1)?;
        if name == column {
            return Ok(true);
        }
    }
    Ok(false)
}

fn quote_identifier(value: &str) -> String {
    format!("\"{}\"", value.replace('"', "\"\""))
}

fn provider_from_row(row: &Row<'_>) -> rusqlite::Result<Provider> {
    Ok(Provider {
        id: row.get("id")?,
        name: row.get("name")?,
        protocol: row.get("protocol")?,
        vendor: row.get("vendor")?,
        base_url: row.get("base_url")?,
        api_key_cipher: row.get("api_key_cipher")?,
        only_stream: int_bool(row.get("only_stream")?),
        user_agent: row.get("user_agent")?,
        enabled: int_bool(row.get("enabled")?),
        description: row.get("description")?,
        created_at: row.get("created_at")?,
        updated_at: row.get("updated_at")?,
    })
}

fn provider_model_from_row(row: &Row<'_>) -> rusqlite::Result<ProviderModel> {
    Ok(ProviderModel {
        id: row.get("id")?,
        provider_id: row.get("provider_id")?,
        name: row.get("name")?,
        max_tokens: row.get("max_tokens")?,
        enabled: int_bool(row.get("enabled")?),
        created_at: row.get("created_at")?,
        updated_at: row.get("updated_at")?,
    })
}

fn endpoint_from_row(row: &Row<'_>) -> rusqlite::Result<IngressEndpoint> {
    Ok(IngressEndpoint {
        id: row.get("id")?,
        path: row.get("path")?,
        downstream_protocol: row.get("downstream_protocol")?,
        enabled: int_bool(row.get("enabled")?),
        protected: int_bool(row.get("protected")?),
        built_in: int_bool(row.get("built_in")?),
        description: row.get("description")?,
        created_at: row.get("created_at")?,
        updated_at: row.get("updated_at")?,
    })
}

fn rule_from_row(row: &Row<'_>) -> rusqlite::Result<RoutingRule> {
    Ok(RoutingRule {
        id: row.get("id")?,
        name: row.get("name")?,
        priority: row.get("priority")?,
        match_protocol: row.get("match_protocol")?,
        match_model_pattern: row.get("match_model_pattern")?,
        upstream_protocol: row.get("upstream_protocol")?,
        target_provider_id: row.get("target_provider_id")?,
        target_model: row.get("target_model")?,
        enabled: int_bool(row.get("enabled")?),
        created_at: row.get("created_at")?,
        updated_at: row.get("updated_at")?,
    })
}

fn api_key_from_row(row: &Row<'_>) -> rusqlite::Result<APIKey> {
    Ok(APIKey {
        id: row.get("id")?,
        name: row.get("name")?,
        secret_hash: row.get("secret_hash")?,
        secret_preview: row.get("secret_preview")?,
        secret_cipher: row.get("secret_cipher")?,
        scopes: row.get("scopes")?,
        enabled: int_bool(row.get("enabled")?),
        expires_at: row.get("expires_at")?,
        created_at: row.get("created_at")?,
        updated_at: row.get("updated_at")?,
    })
}

fn traffic_from_row(row: &Row<'_>) -> rusqlite::Result<TrafficRecord> {
    Ok(TrafficRecord {
        id: row.get("id")?,
        request_id: row.get("request_id")?,
        endpoint: row.get("endpoint")?,
        method: row.get("method")?,
        client_ip: row.get("client_ip")?,
        user_agent: row.get("user_agent")?,
        content_type: row.get("content_type")?,
        upstream_protocol: row.get("upstream_protocol")?,
        downstream_protocol: row.get("downstream_protocol")?,
        route_name: row.get("route_name")?,
        route_source: row.get("route_source")?,
        matched_rule_id: row.get("matched_rule_id")?,
        matched_rule_name: row.get("matched_rule_name")?,
        requested_model: row.get("request_model")?,
        model: row.get("model")?,
        request_body: row.get("request_body")?,
        request_body_bytes: row.get("request_body_bytes")?,
        request_body_truncated: int_bool(row.get("request_body_truncated")?),
        status_code: row.get("status_code")?,
        duration_ms: row.get("duration_ms")?,
        input_tokens: row.get("input_tokens")?,
        output_tokens: row.get("output_tokens")?,
        total_tokens: row.get("total_tokens")?,
        error: row.get("error")?,
        created_at: row.get("created_at")?,
    })
}

fn int_bool(value: i64) -> bool {
    value != 0
}

pub fn to_api_key_view(item: &APIKey) -> model::APIKeyView {
    model::APIKeyView {
        id: item.id.clone(),
        name: item.name.clone(),
        secret_preview: item.secret_preview.clone(),
        can_reveal: !item.secret_cipher.trim().is_empty(),
        scopes: item.scopes.clone(),
        enabled: item.enabled,
        created_at: item.created_at.clone(),
        updated_at: item.updated_at.clone(),
    }
}
