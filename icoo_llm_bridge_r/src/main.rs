use std::net::SocketAddr;

use anyhow::Context;
use icoo_llm_bridge::{config::Options, db::Database, http_app};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let options = parse_args();
    let cfg = icoo_llm_bridge::config::load(&options).context("failed to load config")?;
    let db = Database::open(&cfg).context("failed to initialize database")?;
    db.seed_defaults().context("failed to seed defaults")?;

    let addr: SocketAddr = cfg
        .addr()
        .parse()
        .with_context(|| format!("invalid listen addr {}", cfg.addr()))?;
    let state = http_app::AppState::new(cfg.clone(), db);
    let app = http_app::router(state);
    let listener = tokio::net::TcpListener::bind(addr).await?;

    println!("icoo_llm_bridge started addr={}", cfg.addr());
    axum::serve(
        listener,
        app.into_make_service_with_connect_info::<SocketAddr>(),
    )
    .with_graceful_shutdown(async {
        let _ = tokio::signal::ctrl_c().await;
    })
    .await?;

    Ok(())
}

fn parse_args() -> Options {
    let mut options = Options::default();
    let mut args = std::env::args().skip(1);
    while let Some(arg) = args.next() {
        match arg.as_str() {
            "-config" | "--config" => options.config_path = args.next().unwrap_or_default(),
            "-data-dir" | "--data-dir" => options.data_dir = args.next().unwrap_or_default(),
            "-addr" | "--addr" => options.addr_override = args.next().unwrap_or_default(),
            _ => {}
        }
    }
    options
}
