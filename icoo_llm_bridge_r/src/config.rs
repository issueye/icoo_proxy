use std::{net::ToSocketAddrs, path::PathBuf, time::Duration};

use anyhow::{bail, Context};
use serde::Deserialize;

pub const DEFAULT_MAX_TOKENS: i64 = 32768;

#[derive(Clone, Debug)]
pub struct Config {
    pub host: String,
    pub port: u16,
    pub read_timeout: Duration,
    pub write_timeout: Duration,
    pub shutdown_timeout: Duration,
    pub allow_local_without_auth: bool,
    pub default_max_tokens: i64,
    pub data_dir: PathBuf,
    pub db_path: PathBuf,
    pub traffic_db_path: PathBuf,
    pub log: LogConfig,
    pub archive: ArchiveConfig,
}

#[derive(Clone, Debug)]
pub struct LogConfig {
    pub chain_log_path: PathBuf,
    pub chain_log_bodies: bool,
    pub chain_log_max_body_bytes: usize,
}

#[derive(Clone, Debug)]
pub struct ArchiveConfig {
    pub enabled: bool,
    pub down_request_dir: PathBuf,
    pub up_request_dir: PathBuf,
}

#[derive(Clone, Default, Debug)]
pub struct Options {
    pub config_path: String,
    pub data_dir: String,
    pub addr_override: String,
}

#[derive(Default, Deserialize)]
struct FileConfig {
    host: Option<String>,
    port: Option<u16>,
    read_timeout_seconds: Option<u64>,
    write_timeout_seconds: Option<u64>,
    shutdown_timeout_seconds: Option<u64>,
    allow_local_without_auth: Option<bool>,
    allow_unauthenticated_local: Option<bool>,
    default_max_tokens: Option<i64>,
    data_dir: Option<String>,
    db_path: Option<String>,
    traffic_db_path: Option<String>,
    log: Option<FileLogConfig>,
    archive: Option<FileArchiveConfig>,
}

#[derive(Default, Deserialize)]
struct FileLogConfig {
    chain_log_path: Option<String>,
    chain_log_bodies: Option<bool>,
    chain_log_max_body_bytes: Option<usize>,
}

#[derive(Default, Deserialize)]
struct FileArchiveConfig {
    enabled: Option<bool>,
    down_request_dir: Option<String>,
    up_request_dir: Option<String>,
}

pub fn load(options: &Options) -> anyhow::Result<Config> {
    let mut cfg = defaults();
    let config_path = if options.config_path.trim().is_empty() {
        PathBuf::from("config.toml")
    } else {
        PathBuf::from(options.config_path.trim())
    };
    if config_path.exists() {
        let text = std::fs::read_to_string(&config_path)
            .with_context(|| format!("read config {}", config_path.display()))?;
        let file: FileConfig = toml::from_str(&text)
            .with_context(|| format!("parse config {}", config_path.display()))?;
        apply_file_config(&mut cfg, file);
    }
    cfg.apply_data_dir(options.data_dir.trim());
    cfg.apply_addr_override(options.addr_override.trim())?;
    Ok(cfg)
}

pub fn defaults() -> Config {
    let data_dir = PathBuf::from(".data");
    Config {
        host: "127.0.0.1".to_string(),
        port: 18181,
        read_timeout: Duration::from_secs(15),
        write_timeout: Duration::from_secs(300),
        shutdown_timeout: Duration::from_secs(10),
        allow_local_without_auth: true,
        default_max_tokens: DEFAULT_MAX_TOKENS,
        db_path: data_dir.join("icoo_llm_bridge.db"),
        traffic_db_path: data_dir.join("icoo_llm_bridge_traffic.db"),
        log: LogConfig {
            chain_log_path: data_dir.join("bridge-chain.log"),
            chain_log_bodies: false,
            chain_log_max_body_bytes: 8192,
        },
        archive: ArchiveConfig {
            enabled: false,
            down_request_dir: data_dir.join("down_request"),
            up_request_dir: data_dir.join("up_request"),
        },
        data_dir,
    }
}

impl Config {
    pub fn addr(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }

    pub fn apply_data_dir(&mut self, data_dir: &str) {
        if data_dir.trim().is_empty() {
            return;
        }
        self.data_dir = PathBuf::from(data_dir);
        self.db_path = self.data_dir.join("icoo_llm_bridge.db");
        self.traffic_db_path = self.data_dir.join("icoo_llm_bridge_traffic.db");
        if self.log.chain_log_path.as_os_str().is_empty() {
            self.log.chain_log_path = self.data_dir.join("bridge-chain.log");
        }
        if self.archive.down_request_dir.as_os_str().is_empty() {
            self.archive.down_request_dir = self.data_dir.join("down_request");
        }
        if self.archive.up_request_dir.as_os_str().is_empty() {
            self.archive.up_request_dir = self.data_dir.join("up_request");
        }
    }

    pub fn apply_addr_override(&mut self, addr: &str) -> anyhow::Result<()> {
        if addr.trim().is_empty() {
            return Ok(());
        }
        let parsed = addr
            .to_socket_addrs()
            .with_context(|| format!("parse listen address: {}", addr))?
            .next();
        if parsed.is_none() {
            bail!("parse listen address: invalid address");
        }
        let Some((host, port)) = addr.rsplit_once(':') else {
            bail!("parse listen address: missing port");
        };
        let port: u16 = port.parse().context("parse listen address: invalid port")?;
        if port == 0 {
            bail!("parse listen address: invalid port");
        }
        self.host = host.trim_matches(['[', ']']).to_string();
        self.port = port;
        Ok(())
    }
}

fn apply_file_config(cfg: &mut Config, fc: FileConfig) {
    if let Some(host) = fc.host.filter(|v| !v.is_empty()) {
        cfg.host = host;
    }
    if let Some(port) = fc.port.filter(|v| *v > 0) {
        cfg.port = port;
    }
    if let Some(v) = fc.read_timeout_seconds.filter(|v| *v > 0) {
        cfg.read_timeout = Duration::from_secs(v);
    }
    if let Some(v) = fc.write_timeout_seconds.filter(|v| *v > 0) {
        cfg.write_timeout = Duration::from_secs(v);
    }
    if let Some(v) = fc.shutdown_timeout_seconds.filter(|v| *v > 0) {
        cfg.shutdown_timeout = Duration::from_secs(v);
    }
    if let Some(v) = fc.allow_local_without_auth {
        cfg.allow_local_without_auth = v;
    } else if let Some(v) = fc.allow_unauthenticated_local {
        cfg.allow_local_without_auth = v;
    }
    if let Some(v) = fc.default_max_tokens.filter(|v| *v > 0) {
        cfg.default_max_tokens = v;
    }
    if let Some(v) = fc.data_dir.filter(|v| !v.is_empty()) {
        cfg.apply_data_dir(&v);
    }
    if let Some(v) = fc.db_path.filter(|v| !v.is_empty()) {
        cfg.db_path = PathBuf::from(v);
    }
    if let Some(v) = fc.traffic_db_path.filter(|v| !v.is_empty()) {
        cfg.traffic_db_path = PathBuf::from(v);
    }
    if let Some(log) = fc.log {
        if let Some(v) = log.chain_log_path.filter(|v| !v.is_empty()) {
            cfg.log.chain_log_path = PathBuf::from(v);
        }
        if let Some(v) = log.chain_log_bodies {
            cfg.log.chain_log_bodies = v;
        }
        if let Some(v) = log.chain_log_max_body_bytes.filter(|v| *v > 0) {
            cfg.log.chain_log_max_body_bytes = v;
        }
    }
    if let Some(archive) = fc.archive {
        if let Some(v) = archive.enabled {
            cfg.archive.enabled = v;
        }
        if let Some(v) = archive.down_request_dir.filter(|v| !v.is_empty()) {
            cfg.archive.down_request_dir = PathBuf::from(v);
        }
        if let Some(v) = archive.up_request_dir.filter(|v| !v.is_empty()) {
            cfg.archive.up_request_dir = PathBuf::from(v);
        }
    }
}
