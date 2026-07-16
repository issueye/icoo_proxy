import { API_PREFIX, client } from "./transport";
import { boolOf, normalizePage, pageItems, valueOf } from "./normalize";
import { ListSuppliers } from "./providers_models";
import { listRoutingRules } from "./routes_endpoints_keys";

export function normalizeRuntimeState(raw) {
  return {
    service: valueOf(raw, "service", "Service", "icoo_llm_bridge"),
    version: valueOf(raw, "version", "Version"),
    running: boolOf(raw, "running", "Running"),
    listen_addr: valueOf(raw, "listen_addr", "ListenAddr"),
    paths: valueOf(raw, "paths", "Paths", []),
  };
}
export function normalizeTraffic(raw) {
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
    downstream_protocol: valueOf(
      raw,
      "downstream_protocol",
      "DownstreamProtocol",
    ),
    upstream_protocol: valueOf(raw, "upstream_protocol", "UpstreamProtocol"),
    route_name: valueOf(raw, "route_name", "RouteName"),
    route_source: valueOf(raw, "route_source", "RouteSource"),
    matched_rule_id: valueOf(raw, "matched_rule_id", "MatchedRuleID"),
    matched_rule_name: valueOf(raw, "matched_rule_name", "MatchedRuleName"),
    requested_model: valueOf(raw, "requested_model", "RequestedModel"),
    model: valueOf(raw, "model", "Model"),
    request_body: valueOf(raw, "request_body", "RequestBody"),
    request_body_bytes: Number(
      valueOf(raw, "request_body_bytes", "RequestBodyBytes", 0) || 0,
    ),
    request_body_truncated: boolOf(
      raw,
      "request_body_truncated",
      "RequestBodyTruncated",
    ),
    status_code: Number(valueOf(raw, "status_code", "StatusCode", 0) || 0),
    duration_ms: Number(valueOf(raw, "duration_ms", "DurationMS", 0) || 0),
    input_tokens: Number(valueOf(raw, "input_tokens", "InputTokens", 0) || 0),
    output_tokens: Number(
      valueOf(raw, "output_tokens", "OutputTokens", 0) || 0,
    ),
    total_tokens: Number(valueOf(raw, "total_tokens", "TotalTokens", 0) || 0),
    error: valueOf(raw, "error", "Error"),
    created_at: valueOf(raw, "created_at", "CreatedAt"),
  };
}
export function State() {
  return client.get(`${API_PREFIX}/runtime/state`);
}
export function ReloadProxy() {
  return GetOverview();
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
      providers: suppliers.filter((i) => i.enabled).length,
      routing_rules: rules.filter((i) => i.enabled).length,
    },
    supported_paths: runtime.paths,
    recent_requests: traffic.items || [],
    route_policies: rules,
  };
}
export async function GetTrafficPage(page, pageSize, filter) {
  const raw = await client.get(`${API_PREFIX}/traffic`, {
    params: { limit: 500, page: 1, page_size: 500 },
  });
  let items = normalizePage(raw, 1, 500).items.map(normalizeTraffic);
  if (filter && filter !== "all")
    items = items.filter(
      (i) => i.downstream_protocol === filter || i.upstream_protocol === filter,
    );
  const token_stats = items.reduce(
    (a, i) => ({
      input_tokens: a.input_tokens + i.input_tokens,
      output_tokens: a.output_tokens + i.output_tokens,
      total_tokens: a.total_tokens + i.total_tokens,
    }),
    { input_tokens: 0, output_tokens: 0, total_tokens: 0 },
  );
  const result = pageItems(items, page, pageSize);
  result.protocol_options = [
    "all",
    "anthropic",
    "openai-chat",
    "openai-responses",
  ];
  result.token_stats = token_stats;
  result.total_requests = items.length;
  result.success_count = items.filter((i) => i.status_code < 400).length;
  result.canceled_count = items.filter((i) => i.status_code === 499).length;
  result.error_count = items.filter(
    (i) => i.status_code >= 400 && i.status_code !== 499,
  ).length;
  result.average_latency = items.length
    ? Math.round(items.reduce((s, i) => s + i.duration_ms, 0) / items.length)
    : 0;
  result.last_updated_at = new Date().toISOString();
  return result;
}
export function ClearTrafficRequests() {
  return client.delete(`${API_PREFIX}/traffic`);
}
export function GetUiPrefs() {
  return client.get(`${API_PREFIX}/ui-prefs`);
}
export function SaveUiPrefs(input) {
  return client.put(`${API_PREFIX}/ui-prefs`, input || {});
}
