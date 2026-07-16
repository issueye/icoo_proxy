import { API_PREFIX, client } from "./transport";
import {
  boolOf,
  matchesKeyword,
  normalizePage,
  pageItems,
  valueOf,
} from "./normalize";
import { listProvidersWithModels } from "./providers_models";

export function normalizeEndpoint(raw) {
  return {
    id: valueOf(raw, "id", "ID"),
    path: valueOf(raw, "path", "Path"),
    protocol: valueOf(raw, "downstream_protocol", "DownstreamProtocol"),
    downstream_protocol: valueOf(
      raw,
      "downstream_protocol",
      "DownstreamProtocol",
    ),
    enabled: boolOf(raw, "enabled", "Enabled", true),
    protected: boolOf(raw, "protected", "Protected", true),
    built_in: boolOf(raw, "built_in", "BuiltIn"),
    description: valueOf(raw, "description", "Description"),
    created_at: valueOf(raw, "created_at", "CreatedAt"),
    updated_at: valueOf(raw, "updated_at", "UpdatedAt"),
  };
}
export function normalizeRoutingRule(raw) {
  const rule = {
    id: valueOf(raw, "id", "ID"),
    name: valueOf(raw, "name", "Name"),
    priority: Number(valueOf(raw, "priority", "Priority", 100) || 100),
    match_protocol: valueOf(raw, "match_protocol", "MatchProtocol"),
    match_model_pattern: valueOf(
      raw,
      "match_model_pattern",
      "MatchModelPattern",
      "*",
    ),
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
export function normalizeAPIKey(raw) {
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
export async function listRoutingRules() {
  const raw = await client.get(`${API_PREFIX}/routing-rules`, {
    params: { page: 1, page_size: 200 },
  });
  const rules = normalizePage(raw, 1, 200).items.map(normalizeRoutingRule);
  const providers = await listProvidersWithModels();
  const lookup = Object.fromEntries(providers.map((i) => [i.id, i]));
  return rules.map((rule) => ({
    ...rule,
    supplier_name: lookup[rule.target_provider_id]?.name || "",
    upstream_protocol:
      rule.upstream_protocol || lookup[rule.target_provider_id]?.protocol || "",
  }));
}
export async function GetEndpointsPage(page, pageSize, keyword, protocol) {
  const raw = await client.get(`${API_PREFIX}/ingress-endpoints`, {
    params: { page: 1, page_size: 200 },
  });
  const items = normalizePage(raw, 1, 200)
    .items.map(normalizeEndpoint)
    .filter(
      (i) =>
        (protocol === "all" || !protocol || i.protocol === protocol) &&
        matchesKeyword(i, keyword, ["path", "description"]),
    );
  const result = pageItems(items, page, pageSize);
  result.total_count = items.length;
  result.enabled_count = items.filter((i) => i.enabled).length;
  result.custom_count = items.filter((i) => !i.built_in).length;
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
  return payload.id
    ? client.put(`${API_PREFIX}/ingress-endpoints/${payload.id}`, payload)
    : client.post(`${API_PREFIX}/ingress-endpoints`, payload);
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
      (i) =>
        (status === "all" ||
          !status ||
          (status === "enabled" ? i.enabled : !i.enabled)) &&
        matchesKeyword(i, keyword, ["name", "scopes"]),
    );
  const result = pageItems(items, page, pageSize);
  result.total_count = items.length;
  result.enabled_count = items.filter((i) => i.enabled).length;
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
export function GetAuthKeySecret(id) {
  return client.get(`${API_PREFIX}/api-keys/${id}/secret`);
}
export function ListRoutePolicies() {
  return listRoutingRules().then((r) =>
    r.filter(
      (i) => i.match_model_pattern === "*" || i.match_model_pattern === "",
    ),
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
  if (payload.id)
    await client.put(`${API_PREFIX}/routing-rules/${payload.id}`, payload);
  else await client.post(`${API_PREFIX}/routing-rules`, payload);
  return ListRoutePolicies();
}
export function ListModelAliases() {
  return listRoutingRules().then((r) =>
    r
      .filter((i) => i.match_model_pattern && i.match_model_pattern !== "*")
      .map((i) => ({
        id: i.id,
        name: i.match_model_pattern,
        supplier_id: i.target_provider_id,
        supplier_name: i.supplier_name,
        upstream_protocol: i.upstream_protocol,
        model: i.target_model,
        enabled: i.enabled,
        updated_at: i.updated_at,
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
  if (payload.id)
    await client.put(`${API_PREFIX}/routing-rules/${payload.id}`, payload);
  else await client.post(`${API_PREFIX}/routing-rules`, payload);
  return ListModelAliases();
}
export async function DeleteModelAlias(id) {
  await client.delete(`${API_PREFIX}/routing-rules/${id}`);
  return ListModelAliases();
}
