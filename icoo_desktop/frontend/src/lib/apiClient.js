import axios from "axios";

const API_PREFIX = "/api/v1";

const client = axios.create({
  timeout: 15000,
  headers: { "Content-Type": "application/json" },
});

client.interceptors.request.use((config) => {
  config.baseURL =
    (typeof window !== "undefined" && window.__ICOOSERVER_URL) ||
    "http://127.0.0.1:18181";
  const key = (typeof window !== "undefined" && window.__ICOOSERVER_API_KEY) || "";
  if (key) {
    config.headers.Authorization = `Bearer ${key}`;
  }
  return config;
});

client.interceptors.response.use(
  (res) => res.data?.data,
  (err) => {
    const msg = err.response?.data?.error?.message || err.message;
    throw new Error(msg);
  },
);

const protocols = ["anthropic", "openai-chat", "openai-responses"];

function valueOf(raw, snake, pascal, fallback = "") {
  return raw?.[snake] ?? raw?.[pascal] ?? raw?.[snakeToPascal(snake)] ?? fallback;
}

function snakeToPascal(value) {
  return String(value || "")
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");
}

function boolOf(raw, snake, pascal, fallback = false) {
  return Boolean(valueOf(raw, snake, pascal, fallback));
}

function normalizePage(raw, fallbackPage = 1, fallbackPageSize = 20) {
  if (Array.isArray(raw)) {
    return {
      items: raw,
      total: raw.length,
      page: fallbackPage,
      page_size: fallbackPageSize,
    };
  }
  return {
    items: raw?.items || raw?.Items || [],
    total: Number(raw?.total ?? raw?.Total ?? 0),
    page: Number(raw?.page ?? raw?.Page ?? fallbackPage),
    page_size: Number(raw?.page_size ?? raw?.PageSize ?? fallbackPageSize),
  };
}

function normalizeRuntimeState(raw) {
  return {
    service: valueOf(raw, "service", "Service", "icoo_llm_bridge"),
    version: valueOf(raw, "version", "Version"),
    running: boolOf(raw, "running", "Running"),
    listen_addr: valueOf(raw, "listen_addr", "ListenAddr"),
    paths: valueOf(raw, "paths", "Paths", []),
  };
}

function pageItems(items, page = 1, pageSize = 20) {
  const safePage = Math.max(1, Number(page || 1));
  const safePageSize = Math.max(1, Number(pageSize || 20));
  const start = (safePage - 1) * safePageSize;
  return {
    items: items.slice(start, start + safePageSize),
    total: items.length,
    page: safePage,
    page_size: safePageSize,
  };
}

function matchesKeyword(item, keyword, fields) {
  const text = String(keyword || "").trim().toLowerCase();
  if (!text) {
    return true;
  }
  return fields.some((field) =>
    String(item?.[field] || "")
      .toLowerCase()
      .includes(text),
  );
}

function maskSecret(secret) {
  const value = String(secret || "").trim();
  if (!value) {
    return "";
  }
  if (value.includes("...")) {
    return value;
  }
  if (value.length <= 8) {
    return "****";
  }
  return `${value.slice(0, 4)}...${value.slice(-4)}`;
}

function normalizeProvider(raw) {
  return {
    id: valueOf(raw, "id", "ID"),
    name: valueOf(raw, "name", "Name"),
    protocol: valueOf(raw, "protocol", "Protocol"),
    vendor: valueOf(raw, "vendor", "Vendor", "custom"),
    base_url: valueOf(raw, "base_url", "BaseURL"),
    models_url: valueOf(raw, "models_url", "ModelsURL"),
    proxy_url: valueOf(raw, "proxy_url", "ProxyURL"),
    api_key_masked: maskSecret(valueOf(raw, "api_key_cipher", "APIKeyCipher")),
    only_stream: boolOf(raw, "only_stream", "OnlyStream"),
    user_agent: valueOf(raw, "user_agent", "UserAgent"),
    enabled: boolOf(raw, "enabled", "Enabled", true),
    description: valueOf(raw, "description", "Description"),
    created_at: valueOf(raw, "created_at", "CreatedAt"),
    updated_at: valueOf(raw, "updated_at", "UpdatedAt"),
    models: [],
  };
}

function normalizeProviderModel(raw) {
  return {
    id: valueOf(raw, "id", "ID"),
    provider_id: valueOf(raw, "provider_id", "ProviderID"),
    name: valueOf(raw, "name", "Name"),
    max_tokens: Number(valueOf(raw, "max_tokens", "MaxTokens", 32768) || 32768),
    enabled: boolOf(raw, "enabled", "Enabled", true),
    created_at: valueOf(raw, "created_at", "CreatedAt"),
    updated_at: valueOf(raw, "updated_at", "UpdatedAt"),
  };
}

function normalizeEndpoint(raw) {
  return {
    id: valueOf(raw, "id", "ID"),
    path: valueOf(raw, "path", "Path"),
    protocol: valueOf(raw, "downstream_protocol", "DownstreamProtocol"),
    downstream_protocol: valueOf(raw, "downstream_protocol", "DownstreamProtocol"),
    enabled: boolOf(raw, "enabled", "Enabled", true),
    protected: boolOf(raw, "protected", "Protected", true),
    built_in: boolOf(raw, "built_in", "BuiltIn"),
    description: valueOf(raw, "description", "Description"),
    created_at: valueOf(raw, "created_at", "CreatedAt"),
    updated_at: valueOf(raw, "updated_at", "UpdatedAt"),
  };
}

function normalizeRoutingRule(raw) {
  const rule = {
    id: valueOf(raw, "id", "ID"),
    name: valueOf(raw, "name", "Name"),
    priority: Number(valueOf(raw, "priority", "Priority", 100) || 100),
    match_protocol: valueOf(raw, "match_protocol", "MatchProtocol"),
    match_model_pattern: valueOf(raw, "match_model_pattern", "MatchModelPattern", "*"),
    upstream_protocol: valueOf(raw, "upstream_protocol", "UpstreamProtocol"),
    target_provider_id: valueOf(raw, "target_provider_id", "TargetProviderID"),
    target_model: valueOf(raw, "target_model", "TargetModel"),
    enabled: boolOf(raw, "enabled", "Enabled", true),
    created_at: valueOf(raw, "created_at", "CreatedAt"),
    updated_at: valueOf(raw, "updated_at", "UpdatedAt"),
  };
  return {
    ...rule,
    downstream_protocol: rule.match_protocol,
    supplier_id: rule.target_provider_id,
    supplier_name: "",
    model: rule.target_model,
  };
}

function normalizeAPIKey(raw) {
  const preview = raw?.secret_preview || raw?.SecretPreview || "";
  return {
    id: raw?.id || raw?.ID || "",
    name: raw?.name || raw?.Name || "",
    secret_preview: preview,
    secret_masked: preview,
    can_reveal: Boolean(raw?.can_reveal ?? raw?.CanReveal ?? false),
    scopes: raw?.scopes || raw?.Scopes || "",
    enabled: Boolean(raw?.enabled ?? raw?.Enabled ?? true),
    description: "",
    created_at: raw?.created_at || raw?.CreatedAt || "",
    updated_at: raw?.updated_at || raw?.UpdatedAt || "",
  };
}

function normalizeTraffic(raw) {
  return {
    id: valueOf(raw, "id", "ID"),
    request_id: valueOf(raw, "request_id", "RequestID"),
    endpoint: valueOf(raw, "endpoint", "Endpoint"),
    method: valueOf(raw, "method", "Method"),
    client_ip: valueOf(raw, "client_ip", "ClientIP"),
    user_agent: valueOf(raw, "user_agent", "UserAgent"),
    content_type: valueOf(raw, "content_type", "ContentType"),
    downstream: valueOf(raw, "downstream_protocol", "DownstreamProtocol"),
    upstream: valueOf(raw, "upstream_protocol", "UpstreamProtocol"),
    downstream_protocol: valueOf(raw, "downstream_protocol", "DownstreamProtocol"),
    upstream_protocol: valueOf(raw, "upstream_protocol", "UpstreamProtocol"),
    route_name: valueOf(raw, "route_name", "RouteName"),
    route_source: valueOf(raw, "route_source", "RouteSource"),
    matched_rule_id: valueOf(raw, "matched_rule_id", "MatchedRuleID"),
    matched_rule_name: valueOf(raw, "matched_rule_name", "MatchedRuleName"),
    requested_model: valueOf(raw, "requested_model", "RequestedModel"),
    model: valueOf(raw, "model", "Model"),
    request_body: valueOf(raw, "request_body", "RequestBody"),
    request_body_bytes: Number(valueOf(raw, "request_body_bytes", "RequestBodyBytes", 0) || 0),
    request_body_truncated: boolOf(raw, "request_body_truncated", "RequestBodyTruncated"),
    status_code: Number(valueOf(raw, "status_code", "StatusCode", 0) || 0),
    duration_ms: Number(valueOf(raw, "duration_ms", "DurationMS", 0) || 0),
    input_tokens: Number(valueOf(raw, "input_tokens", "InputTokens", 0) || 0),
    output_tokens: Number(valueOf(raw, "output_tokens", "OutputTokens", 0) || 0),
    total_tokens: Number(valueOf(raw, "total_tokens", "TotalTokens", 0) || 0),
    error: valueOf(raw, "error", "Error"),
    created_at: valueOf(raw, "created_at", "CreatedAt"),
  };
}

function normalizeSupplierHealth(raw) {
  return {
    supplier_id: valueOf(raw, "supplier_id", "SupplierID"),
    status: valueOf(raw, "status", "Status", "unreachable"),
    status_code: Number(valueOf(raw, "status_code", "StatusCode", 0) || 0),
    duration_ms: Number(valueOf(raw, "duration_ms", "DurationMS", 0) || 0),
    message: valueOf(raw, "message", "Message"),
    checked_at: valueOf(raw, "checked_at", "CheckedAt"),
  };
}

async function listProviderModels(providerID) {
  if (!providerID) {
    return [];
  }
  const raw = await client.get(`${API_PREFIX}/providers/${providerID}/models`, {
    params: { page: 1, page_size: 200 },
  });
  return normalizePage(raw, 1, 200).items.map(normalizeProviderModel);
}

export async function FetchModelsFromProvider(providerID) {
  if (!providerID) {
    return [];
  }
  const raw = await client.post(`${API_PREFIX}/providers/${providerID}/fetch-models`);
  return raw?.data || [];
}

async function listProvidersWithModels() {
  const raw = await client.get(`${API_PREFIX}/providers`, {
    params: { page: 1, page_size: 200 },
  });
  const providers = normalizePage(raw, 1, 200).items.map(normalizeProvider);
  await Promise.all(
    providers.map(async (provider) => {
      provider.models = await listProviderModels(provider.id);
    }),
  );
  return providers;
}

async function listRoutingRules() {
  const raw = await client.get(`${API_PREFIX}/routing-rules`, {
    params: { page: 1, page_size: 200 },
  });
  const rules = normalizePage(raw, 1, 200).items.map(normalizeRoutingRule);
  const providers = await listProvidersWithModels();
  const lookup = Object.fromEntries(providers.map((item) => [item.id, item]));
  return rules.map((rule) => ({
    ...rule,
    supplier_name: lookup[rule.target_provider_id]?.name || "",
    upstream_protocol: rule.upstream_protocol || lookup[rule.target_provider_id]?.protocol || "",
  }));
}

function providerPayload(input) {
  return {
    id: input.id || undefined,
    name: String(input.name || "").trim(),
    protocol: input.protocol,
    vendor: input.vendor || "custom",
    base_url: String(input.base_url || "").trim(),
    models_url: String(input.models_url || "").trim(),
    proxy_url: String(input.proxy_url || "").trim(),
    api_key: String(input.api_key || "").trim(),
    only_stream: Boolean(input.only_stream),
    user_agent: String(input.user_agent || "").trim(),
    enabled: Boolean(input.enabled),
    description: String(input.description || "").trim(),
  };
}

async function syncProviderModels(providerID, models = []) {
  const existing = await listProviderModels(providerID);
  const keep = new Set();
  for (const model of models) {
    const name = String(model?.name || "").trim();
    if (!name) {
      continue;
    }
    const current = existing.find((item) => item.name === name);
    const id = current?.id || "";
    keep.add(id || name);
    const payload = {
      id: id || undefined,
      name,
      max_tokens: Number(model?.max_tokens || 32768),
      enabled: model?.enabled ?? true,
    };
    if (id) {
      await client.put(`${API_PREFIX}/providers/${providerID}/models/${id}`, payload);
    } else {
      await client.post(`${API_PREFIX}/providers/${providerID}/models`, payload);
    }
  }
  for (const model of existing) {
    if (!keep.has(model.id) && !keep.has(model.name)) {
      await client.delete(`${API_PREFIX}/providers/${providerID}/models/${model.id}`);
    }
  }
}

export async function GetOverview() {
  const [state, suppliers, rules, traffic] = await Promise.all([
    State(),
    ListSuppliers(),
    listRoutingRules(),
    GetTrafficPage(1, 8, "all"),
  ]);
  const runtime = normalizeRuntimeState(state);
  return {
    service: runtime.service,
    version: runtime.version,
    running: runtime.running,
    listen_addr: runtime.listen_addr,
    checks: {
      service: runtime.running ? "ok" : "stopped",
      providers: suppliers.filter((item) => item.enabled).length,
      routing_rules: rules.filter((item) => item.enabled).length,
    },
    supported_paths: runtime.paths,
    recent_requests: traffic.items || [],
    route_policies: rules,
  };
}

export function State() {
  return client.get(`${API_PREFIX}/runtime/state`);
}

export function ReloadProxy() {
  return GetOverview();
}

export async function GetSuppliersPage(page, pageSize, keyword, protocol) {
  const items = (await listProvidersWithModels()).filter(
    (item) =>
      (protocol === "all" || !protocol || item.protocol === protocol) &&
      matchesKeyword(item, keyword, ["name", "base_url", "description"]),
  );
  const result = pageItems(items, page, pageSize);
  result.total_count = items.length;
  result.enabled_count = items.filter((item) => item.enabled).length;
  return result;
}

export function ListSuppliers() {
  return listProvidersWithModels();
}

export async function SaveSupplier(input) {
  const payload = providerPayload(input);
  if (payload.id && !payload.api_key) {
    const raw = await client.get(`${API_PREFIX}/providers`, {
      params: { page: 1, page_size: 200 },
    });
    const existing = normalizePage(raw, 1, 200).items.find(
      (item) => valueOf(item, "id", "ID") === payload.id,
    );
    payload.api_key = valueOf(existing, "api_key_cipher", "APIKeyCipher");
  }
  let saved;
  if (payload.id) {
    saved = await client.put(`${API_PREFIX}/providers/${payload.id}`, payload);
  } else {
    saved = await client.post(`${API_PREFIX}/providers`, payload);
  }
  const provider = normalizeProvider(saved);
  if (Array.isArray(input.models)) {
    await syncProviderModels(provider.id, input.models);
  }
  return provider;
}

export function DeleteSupplier(id) {
  return client.delete(`${API_PREFIX}/providers/${id}`);
}

export function ListSupplierHealth() {
  return Promise.resolve([]);
}

export async function CheckSupplier(id) {
  return client.post(`${API_PREFIX}/providers/${id}/check`).then(normalizeSupplierHealth);
}

export async function GetEndpointsPage(page, pageSize, keyword, protocol) {
  const raw = await client.get(`${API_PREFIX}/ingress-endpoints`, {
    params: { page: 1, page_size: 200 },
  });
  const items = normalizePage(raw, 1, 200)
    .items.map(normalizeEndpoint)
    .filter(
      (item) =>
        (protocol === "all" || !protocol || item.protocol === protocol) &&
        matchesKeyword(item, keyword, ["path", "description"]),
    );
  const result = pageItems(items, page, pageSize);
  result.total_count = items.length;
  result.enabled_count = items.filter((item) => item.enabled).length;
  result.custom_count = items.filter((item) => !item.built_in).length;
  return result;
}

export async function ListEndpoints() {
  const raw = await client.get(`${API_PREFIX}/ingress-endpoints`, {
    params: { page: 1, page_size: 200 },
  });
  return normalizePage(raw, 1, 200).items.map(normalizeEndpoint);
}

export function SaveEndpoint(input) {
  const payload = {
    id: input.id || undefined,
    path: String(input.path || "").trim(),
    downstream_protocol: input.protocol || input.downstream_protocol,
    enabled: Boolean(input.enabled),
    protected: input.protected ?? true,
    description: String(input.description || "").trim(),
  };
  if (payload.id) {
    return client.put(`${API_PREFIX}/ingress-endpoints/${payload.id}`, payload);
  }
  return client.post(`${API_PREFIX}/ingress-endpoints`, payload);
}

export function DeleteEndpoint(id) {
  return client.delete(`${API_PREFIX}/ingress-endpoints/${id}`);
}

export async function GetAuthKeysPage(page, pageSize, keyword, status) {
  const raw = await client.get(`${API_PREFIX}/api-keys`, {
    params: { page: 1, page_size: 200 },
  });
  const items = normalizePage(raw, 1, 200)
    .items.map(normalizeAPIKey)
    .filter(
      (item) =>
        (status === "all" ||
          !status ||
          (status === "enabled" ? item.enabled : !item.enabled)) &&
        matchesKeyword(item, keyword, ["name", "scopes"]),
    );
  const result = pageItems(items, page, pageSize);
  result.total_count = items.length;
  result.enabled_count = items.filter((item) => item.enabled).length;
  return result;
}

export async function ListAuthKeys() {
  const raw = await client.get(`${API_PREFIX}/api-keys`, {
    params: { page: 1, page_size: 200 },
  });
  return normalizePage(raw, 1, 200).items.map(normalizeAPIKey);
}

export async function SaveAuthKey(input) {
  return client
    .post(`${API_PREFIX}/api-keys`, {
      id: input.id || undefined,
      name: String(input.name || "").trim(),
      secret: String(input.secret || "").trim(),
      scopes: String(input.scopes || "admin,proxy").trim(),
      enabled: Boolean(input.enabled),
    })
    .then(normalizeAPIKey);
}

export function DeleteAuthKey(id) {
  return client.delete(`${API_PREFIX}/api-keys/${id}`);
}

export async function GetAuthKeySecret(id) {
  return client.get(`${API_PREFIX}/api-keys/${id}/secret`);
}

export function ListRoutePolicies() {
  return listRoutingRules().then((rules) =>
    rules.filter((item) => item.match_model_pattern === "*" || item.match_model_pattern === ""),
  );
}

export function ListRoutingRules() {
  return listRoutingRules();
}

export async function SaveRoutePolicy(input) {
  const payload = {
    id: input.id || undefined,
    name: `${input.downstream_protocol} default route`,
    priority: 100,
    match_protocol: input.downstream_protocol,
    match_model_pattern: "*",
    upstream_protocol: input.upstream_protocol || "",
    target_provider_id: input.supplier_id,
    target_model: "",
    enabled: Boolean(input.enabled),
    force: Boolean(input.force),
  };
  if (payload.id) {
    await client.put(`${API_PREFIX}/routing-rules/${payload.id}`, payload);
  } else {
    await client.post(`${API_PREFIX}/routing-rules`, payload);
  }
  return ListRoutePolicies();
}

export function ListModelAliases() {
  return listRoutingRules().then((rules) =>
    rules
      .filter((item) => item.match_model_pattern && item.match_model_pattern !== "*")
      .map((item) => ({
        id: item.id,
        name: item.match_model_pattern,
        supplier_id: item.target_provider_id,
        supplier_name: item.supplier_name,
        upstream_protocol: item.upstream_protocol,
        model: item.target_model,
        enabled: item.enabled,
        updated_at: item.updated_at,
      })),
  );
}

export async function SaveModelAlias(input) {
  const payload = {
    id: input.id || undefined,
    name: `alias ${input.name}`,
    priority: 50,
    match_protocol: "openai-chat",
    match_model_pattern: String(input.name || "").trim(),
    upstream_protocol: input.upstream_protocol || "",
    target_provider_id: String(input.supplier_id || "").trim(),
    target_model: String(input.model || "").trim(),
    enabled: Boolean(input.enabled),
  };
  if (payload.id) {
    await client.put(`${API_PREFIX}/routing-rules/${payload.id}`, payload);
  } else {
    await client.post(`${API_PREFIX}/routing-rules`, payload);
  }
  return ListModelAliases();
}

export async function DeleteModelAlias(id) {
  await client.delete(`${API_PREFIX}/routing-rules/${id}`);
  return ListModelAliases();
}

export async function GetProjectSettings() {
  // IMPORTANT: Wails v2 exposes Go struct fields to JS using their exported
  // (PascalCase) names when no `json:` tag is present. The generated binding
  // (wailsjs/go/models.ts) confirms ServerConfig uses PascalCase:
  // Host, Port, ReadTimeoutSeconds, …, ChainLogMaxBodyBytes, DefaultMaxTokens.
  // Reading snake_case here (the old code) returned undefined for every field
  // and silently fell back to defaults — so edits never round-tripped.
  if (typeof window !== "undefined" && window.go?.main?.App?.GetServerConfig) {
    const cfg = await window.go.main.App.GetServerConfig();
    return {
      proxy_host: cfg.Host || "127.0.0.1",
      proxy_port: cfg.Port || 18181,
      proxy_read_timeout_seconds: cfg.ReadTimeoutSeconds || 15,
      proxy_write_timeout_seconds: cfg.WriteTimeoutSeconds || 300,
      proxy_shutdown_timeout_seconds: cfg.ShutdownTimeoutSeconds || 10,
      default_max_tokens: cfg.DefaultMaxTokens || 32768,
      proxy_chain_log_path: cfg.ChainLogPath || ".data/bridge-chain.log",
      proxy_chain_log_bodies: cfg.ChainLogBodies ?? false,
      proxy_chain_log_max_body_bytes: cfg.ChainLogMaxBodyBytes ?? 8192,
    };
  }
  return {
    proxy_host: "127.0.0.1",
    proxy_port: 18181,
    proxy_read_timeout_seconds: 15,
    proxy_write_timeout_seconds: 300,
    proxy_shutdown_timeout_seconds: 10,
    default_max_tokens: 32768,
    proxy_chain_log_path: ".data/bridge-chain.log",
    proxy_chain_log_bodies: false,
    proxy_chain_log_max_body_bytes: 8192,
  };
}

export async function SaveProjectSettings(values) {
  // Spread the current config first so untouched fields (APIKeys,
  // AllowUnauthenticatedLocal, …) are preserved instead of zeroed out,
  // then overlay the edited fields using the PascalCase names Wails expects.
  if (typeof window !== "undefined" && window.go?.main?.App?.SaveServerConfig) {
    const current = await window.go.main.App.GetServerConfig();
    await window.go.main.App.SaveServerConfig({
      ...current,
      Host: values.proxy_host,
      Port: values.proxy_port,
      ReadTimeoutSeconds: values.proxy_read_timeout_seconds,
      WriteTimeoutSeconds: values.proxy_write_timeout_seconds,
      ShutdownTimeoutSeconds: values.proxy_shutdown_timeout_seconds,
      DefaultMaxTokens: values.default_max_tokens,
      ChainLogPath: values.proxy_chain_log_path,
      ChainLogBodies: values.proxy_chain_log_bodies,
      ChainLogMaxBodyBytes: values.proxy_chain_log_max_body_bytes,
    });
  }
}

export function GetUiPrefs() {
  return client.get(`${API_PREFIX}/ui-prefs`);
}

export function SaveUiPrefs(input) {
  return client.put(`${API_PREFIX}/ui-prefs`, input || {});
}

export async function GetTrafficPage(page, pageSize, filter) {
  const raw = await client.get(`${API_PREFIX}/traffic`, {
    params: { limit: 500, page: 1, page_size: 500 },
  });
  let items = normalizePage(raw, 1, 500).items.map(normalizeTraffic);
  if (filter && filter !== "all") {
    items = items.filter(
      (item) => item.downstream_protocol === filter || item.upstream_protocol === filter,
    );
  }
  const tokenStats = items.reduce(
    (acc, item) => {
      acc.input_tokens += item.input_tokens;
      acc.output_tokens += item.output_tokens;
      acc.total_tokens += item.total_tokens;
      return acc;
    },
    { input_tokens: 0, output_tokens: 0, total_tokens: 0 },
  );
  const result = pageItems(items, page, pageSize);
  result.protocol_options = ["all", ...protocols];
  result.token_stats = tokenStats;
  result.total_requests = items.length;
  result.success_count = items.filter((item) => item.status_code < 400).length;
  result.error_count = items.filter((item) => item.status_code >= 400).length;
  result.average_latency = items.length
    ? Math.round(items.reduce((sum, item) => sum + item.duration_ms, 0) / items.length)
    : 0;
  result.last_updated_at = new Date().toISOString();
  return result;
}

export function ClearTrafficRequests() {
  return client.delete(`${API_PREFIX}/traffic`);
}

export async function getServerConfig() {
  // Wails exposes ServerConfig fields as PascalCase (see generated
  // wailsjs/go/models.ts). Reading snake_case silently returned undefined.
  if (typeof window !== "undefined" && window.go?.main?.App?.GetServerConfig) {
    const cfg = await window.go.main.App.GetServerConfig();
    return {
      host: cfg.Host || "127.0.0.1",
      port: cfg.Port || 18181,
      read_timeout_seconds: cfg.ReadTimeoutSeconds || 15,
      write_timeout_seconds: cfg.WriteTimeoutSeconds || 300,
      shutdown_timeout_seconds: cfg.ShutdownTimeoutSeconds || 10,
      api_keys: cfg.APIKeys || [],
      allow_unauthenticated_local: cfg.AllowUnauthenticatedLocal ?? true,
      chain_log_path: cfg.ChainLogPath || "",
      chain_log_bodies: cfg.ChainLogBodies ?? false,
      chain_log_max_body_bytes: cfg.ChainLogMaxBodyBytes ?? 8192,
      default_max_tokens: cfg.DefaultMaxTokens || 32768,
    };
  }
  return {
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
}

export async function saveServerConfig(partial) {
  // `partial` uses snake_case from the rest of the frontend; translate to the
  // PascalCase Wails expects and merge over the current config so we don't
  // wipe unrelated fields (APIKeys, AllowUnauthenticatedLocal, …).
  if (typeof window !== "undefined" && window.go?.main?.App?.SaveServerConfig) {
    const current = await window.go.main.App.GetServerConfig();
    const next = { ...current };
    if (partial.host !== undefined) next.Host = partial.host;
    if (partial.port !== undefined) next.Port = partial.port;
    if (partial.read_timeout_seconds !== undefined) next.ReadTimeoutSeconds = partial.read_timeout_seconds;
    if (partial.write_timeout_seconds !== undefined) next.WriteTimeoutSeconds = partial.write_timeout_seconds;
    if (partial.shutdown_timeout_seconds !== undefined) next.ShutdownTimeoutSeconds = partial.shutdown_timeout_seconds;
    if (partial.default_max_tokens !== undefined) next.DefaultMaxTokens = partial.default_max_tokens;
    if (partial.chain_log_path !== undefined) next.ChainLogPath = partial.chain_log_path;
    if (partial.chain_log_bodies !== undefined) next.ChainLogBodies = partial.chain_log_bodies;
    if (partial.chain_log_max_body_bytes !== undefined) next.ChainLogMaxBodyBytes = partial.chain_log_max_body_bytes;
    await window.go.main.App.SaveServerConfig(next);
  }
}

export function getServerURL() {
  if (typeof window !== "undefined" && window.go?.main?.App?.GetServerConfig) {
    return window.go.main.App.GetServerConfig().then((cfg) => `http://${cfg.Host || "127.0.0.1"}:${cfg.Port || 18181}`);
  }
  return Promise.resolve("http://127.0.0.1:18181");
}

export function getAppVersion() {
  if (typeof window !== "undefined" && window.go?.main?.App?.GetAppVersion) {
    return window.go.main.App.GetAppVersion();
  }
  return "0.1.0";
}

export function serverStatus() {
  if (typeof window !== "undefined" && window.go?.main?.App?.ServerStatus) {
    return window.go.main.App.ServerStatus();
  }
  return "stopped";
}

export function wakeServer() {
  if (typeof window !== "undefined" && window.go?.main?.App?.StartServer) {
    return window.go.main.App.StartServer();
  }
  throw new Error("Wails runtime not available");
}
