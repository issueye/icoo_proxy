const defaults = {
  host: "127.0.0.1",
  port: 18181,
  read_timeout_seconds: 15,
  write_timeout_seconds: 300,
  shutdown_timeout_seconds: 10,
  api_keys: [],
  allow_unauthenticated_local: true,
  chain_log_path: "",
  chain_log_bodies: false,
  chain_log_max_body_bytes: 8192,
  default_max_tokens: 32768,
};
function app() {
  return typeof window !== "undefined" ? window.go?.main?.App : null;
}
export async function GetProjectSettings() {
  const a = app();
  if (a?.GetServerConfig) {
    const c = await a.GetServerConfig();
    return {
      proxy_host: c.Host || "127.0.0.1",
      proxy_port: c.Port || 18181,
      proxy_read_timeout_seconds: c.ReadTimeoutSeconds || 15,
      proxy_write_timeout_seconds: c.WriteTimeoutSeconds || 300,
      proxy_shutdown_timeout_seconds: c.ShutdownTimeoutSeconds || 10,
      default_max_tokens: c.DefaultMaxTokens || 32768,
      proxy_chain_log_path: c.ChainLogPath || ".data/bridge-chain.log",
      proxy_chain_log_bodies: c.ChainLogBodies ?? false,
      proxy_chain_log_max_body_bytes: c.ChainLogMaxBodyBytes ?? 8192,
    };
  }
  return {
    proxy_host: defaults.host,
    proxy_port: defaults.port,
    proxy_read_timeout_seconds: defaults.read_timeout_seconds,
    proxy_write_timeout_seconds: defaults.write_timeout_seconds,
    proxy_shutdown_timeout_seconds: defaults.shutdown_timeout_seconds,
    default_max_tokens: defaults.default_max_tokens,
    proxy_chain_log_path: ".data/bridge-chain.log",
    proxy_chain_log_bodies: false,
    proxy_chain_log_max_body_bytes: 8192,
  };
}
export async function SaveProjectSettings(v) {
  const a = app();
  if (a?.SaveServerConfig) {
    const c = await a.GetServerConfig();
    await a.SaveServerConfig({
      ...c,
      Host: v.proxy_host,
      Port: v.proxy_port,
      ReadTimeoutSeconds: v.proxy_read_timeout_seconds,
      WriteTimeoutSeconds: v.proxy_write_timeout_seconds,
      ShutdownTimeoutSeconds: v.proxy_shutdown_timeout_seconds,
      DefaultMaxTokens: v.default_max_tokens,
      ChainLogPath: v.proxy_chain_log_path,
      ChainLogBodies: v.proxy_chain_log_bodies,
      ChainLogMaxBodyBytes: v.proxy_chain_log_max_body_bytes,
    });
  }
}
export async function getServerConfig() {
  const a = app();
  if (!a?.GetServerConfig) return { ...defaults, api_keys: [] };
  const c = await a.GetServerConfig();
  return {
    ...defaults,
    host: c.Host || defaults.host,
    port: c.Port || defaults.port,
    read_timeout_seconds: c.ReadTimeoutSeconds || 15,
    write_timeout_seconds: c.WriteTimeoutSeconds || 300,
    shutdown_timeout_seconds: c.ShutdownTimeoutSeconds || 10,
    api_keys: c.APIKeys || [],
    allow_unauthenticated_local: c.AllowUnauthenticatedLocal ?? true,
    chain_log_path: c.ChainLogPath || "",
    chain_log_bodies: c.ChainLogBodies ?? false,
    chain_log_max_body_bytes: c.ChainLogMaxBodyBytes ?? 8192,
    default_max_tokens: c.DefaultMaxTokens || 32768,
  };
}
export async function saveServerConfig(partial) {
  const a = app();
  if (!a?.SaveServerConfig) return;
  const c = await a.GetServerConfig();
  const next = { ...c };
  const map = {
    host: "Host",
    port: "Port",
    read_timeout_seconds: "ReadTimeoutSeconds",
    write_timeout_seconds: "WriteTimeoutSeconds",
    shutdown_timeout_seconds: "ShutdownTimeoutSeconds",
    default_max_tokens: "DefaultMaxTokens",
    chain_log_path: "ChainLogPath",
    chain_log_bodies: "ChainLogBodies",
    chain_log_max_body_bytes: "ChainLogMaxBodyBytes",
  };
  Object.entries(map).forEach(([k, p]) => {
    if (partial[k] !== undefined) next[p] = partial[k];
  });
  await a.SaveServerConfig(next);
}
export function getServerURL() {
  const a = app();
  return a?.GetServerConfig
    ? a
        .GetServerConfig()
        .then((c) => `http://${c.Host || "127.0.0.1"}:${c.Port || 18181}`)
    : Promise.resolve("http://127.0.0.1:18181");
}
export function getAppVersion() {
  return app()?.GetAppVersion ? app().GetAppVersion() : "0.0.0-dev";
}
export function serverStatus() {
  return app()?.ServerStatus ? app().ServerStatus() : "stopped";
}
export function wakeServer() {
  if (app()?.StartServer) return app().StartServer();
  throw new Error("Wails runtime not available");
}
