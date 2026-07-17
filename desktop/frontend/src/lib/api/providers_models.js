import { API_PREFIX, client } from "./transport";
import {
  boolOf,
  maskSecret,
  matchesKeyword,
  normalizePage,
  pageItems,
  valueOf,
} from "./normalize";

/** Process-plugin vendor: traffic goes over IPC, not HTTP BaseURL. */
export function isPluginVendor(vendor) {
  return String(vendor || "").trim().toLowerCase() === "plugin";
}

function pluginIdFromBaseURL(baseURL) {
  const value = String(baseURL || "").trim();
  const prefix = "plugin://";
  if (value.toLowerCase().startsWith(prefix)) {
    return value.slice(prefix.length).trim();
  }
  return "";
}

function normalizeProvider(raw) {
  const vendor = valueOf(raw, "vendor", "Vendor", "custom");
  let pluginId = valueOf(raw, "plugin_id", "PluginID");
  const baseURL = valueOf(raw, "base_url", "BaseURL");
  if (isPluginVendor(vendor) && !pluginId) {
    pluginId = pluginIdFromBaseURL(baseURL);
  }
  return {
    id: valueOf(raw, "id", "ID"),
    name: valueOf(raw, "name", "Name"),
    protocol: valueOf(raw, "protocol", "Protocol"),
    vendor,
    plugin_id: pluginId,
    base_url: baseURL,
    models_url: valueOf(raw, "models_url", "ModelsURL"),
    proxy_url: valueOf(raw, "proxy_url", "ProxyURL"),
    api_key_masked:
      valueOf(raw, "api_key_masked", "APIKeyMasked") ||
      maskSecret(valueOf(raw, "api_key_cipher", "APIKeyCipher")),
    only_stream: boolOf(raw, "only_stream", "OnlyStream"),
    user_agent: valueOf(raw, "user_agent", "UserAgent"),
    enabled: boolOf(raw, "enabled", "Enabled", true),
    description: valueOf(raw, "description", "Description"),
    created_at: valueOf(raw, "created_at", "CreatedAt"),
    updated_at: valueOf(raw, "updated_at", "UpdatedAt"),
    models: [],
  };
}
export function normalizeProviderModel(raw) {
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
export function normalizeCatalogModel(raw) {
  return {
    id: valueOf(raw, "id", "ID"),
    name: valueOf(raw, "name", "Name"),
    family: valueOf(raw, "family", "Family"),
    icon: valueOf(raw, "icon", "Icon", "custom"),
    max_tokens: Number(valueOf(raw, "max_tokens", "MaxTokens", 32768) || 32768),
    description: valueOf(raw, "description", "Description"),
    built_in: boolOf(raw, "built_in", "BuiltIn"),
    created_at: valueOf(raw, "created_at", "CreatedAt"),
    updated_at: valueOf(raw, "updated_at", "UpdatedAt"),
  };
}

export async function listProviderModels(providerID) {
  if (!providerID) return [];
  const raw = await client.get(`${API_PREFIX}/providers/${providerID}/models`, {
    params: { page: 1, page_size: 200 },
  });
  return normalizePage(raw, 1, 200).items.map(normalizeProviderModel);
}
export async function FetchModelsFromProvider(providerID) {
  if (!providerID) return [];
  const raw = await client.post(
    `${API_PREFIX}/providers/${providerID}/fetch-models`,
  );
  return Array.isArray(raw) ? raw : [];
}
export async function ListModelCatalog() {
  const raw = await client.get(`${API_PREFIX}/model-catalog`, {
    params: { page: 1, page_size: 200 },
  });
  return normalizePage(raw, 1, 200).items.map(normalizeCatalogModel);
}
export function SaveCatalogModel(input) {
  const payload = {
    id: input.id || undefined,
    name: String(input.name || "").trim(),
    family: String(input.family || "").trim(),
    icon: String(input.icon || "custom").trim(),
    max_tokens: Number(input.max_tokens || 32768),
    description: String(input.description || "").trim(),
  };
  const req = payload.id
    ? client.put(`${API_PREFIX}/model-catalog/${payload.id}`, payload)
    : client.post(`${API_PREFIX}/model-catalog`, payload);
  return req.then(normalizeCatalogModel);
}
export function DeleteCatalogModel(id) {
  return client.delete(`${API_PREFIX}/model-catalog/${id}`);
}

export async function listProvidersWithModels() {
  const raw = await client.get(`${API_PREFIX}/providers`, {
    params: { page: 1, page_size: 200 },
  });
  const providers = normalizePage(raw, 1, 200).items.map(normalizeProvider);
  await Promise.all(
    providers.map(async (p) => {
      p.models = await listProviderModels(p.id);
    }),
  );
  return providers;
}
function providerPayload(input) {
  const vendor = String(input.vendor || "custom").trim() || "custom";
  let pluginId = String(input.plugin_id || "").trim();
  let baseURL = String(input.base_url || "").trim();
  if (isPluginVendor(vendor)) {
    if (!pluginId) {
      pluginId = pluginIdFromBaseURL(baseURL);
    }
    if (pluginId && !baseURL) {
      baseURL = `plugin://${pluginId}`;
    }
  } else {
    pluginId = "";
  }
  return {
    id: input.id || undefined,
    name: String(input.name || "").trim(),
    protocol: input.protocol,
    vendor,
    plugin_id: pluginId,
    base_url: baseURL,
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
    if (!name) continue;
    const current = existing.find((i) => i.name === name);
    const id = current?.id || "";
    keep.add(id || name);
    const payload = {
      id: id || undefined,
      name,
      max_tokens: Number(model?.max_tokens || 32768),
      enabled: model?.enabled ?? true,
    };
    if (id)
      await client.put(
        `${API_PREFIX}/providers/${providerID}/models/${id}`,
        payload,
      );
    else
      await client.post(
        `${API_PREFIX}/providers/${providerID}/models`,
        payload,
      );
  }
  for (const model of existing)
    if (!keep.has(model.id) && !keep.has(model.name))
      await client.delete(
        `${API_PREFIX}/providers/${providerID}/models/${model.id}`,
      );
}
export async function GetSuppliersPage(page, pageSize, keyword, protocol) {
  const items = (await listProvidersWithModels()).filter(
    (i) =>
      (protocol === "all" || !protocol || i.protocol === protocol) &&
      matchesKeyword(i, keyword, [
        "name",
        "base_url",
        "description",
        "plugin_id",
        "vendor",
      ]),
  );
  const result = pageItems(items, page, pageSize);
  result.total_count = items.length;
  result.enabled_count = items.filter((i) => i.enabled).length;
  return result;
}
export function ListSuppliers() {
  return listProvidersWithModels();
}
export async function SaveSupplier(input) {
  const payload = providerPayload(input);
  const saved = payload.id
    ? await client.put(`${API_PREFIX}/providers/${payload.id}`, payload)
    : await client.post(`${API_PREFIX}/providers`, payload);
  const provider = normalizeProvider(saved);
  if (Array.isArray(input.models))
    await syncProviderModels(provider.id, input.models);
  return provider;
}

/**
 * Default display name / models for known process plugins.
 * Unknown plugins get a generic name and empty model list
 * (caller may still try FetchModelsFromProvider).
 */
export function pluginProviderDefaults(pluginId) {
  const id = String(pluginId || "").trim();
  switch (id) {
    case "grokbuild":
      return {
        name: "GrokBuild / SuperGrok",
        protocol: "openai-responses",
        description:
          "进程插件 grokbuild：凭据在插件扩展页管理，流量经 IPC 转发到 Grok Build。",
        models: [
          { name: "grok-4", max_tokens: 131072 },
          { name: "grok-4.5", max_tokens: 131072 },
          { name: "grok-build-0.1", max_tokens: 131072 },
        ],
      };
    case "mock":
      return {
        name: "Mock Plugin",
        protocol: "openai-responses",
        description: "进程插件 mock：集成测试用。",
        models: [{ name: "mock-model", max_tokens: 8192 }],
      };
    default:
      return {
        name: id ? `Plugin ${id}` : "Process Plugin",
        protocol: "openai-responses",
        description: id
          ? `进程插件 ${id}：流量经 IPC 转发。`
          : "进程插件供应商。",
        models: [],
      };
  }
}

function findPluginProvider(providers, pluginId) {
  const id = String(pluginId || "").trim();
  if (!id) return null;
  return (
    (providers || []).find(
      (p) =>
        isPluginVendor(p.vendor) &&
        (String(p.plugin_id || "").trim() === id ||
          pluginIdFromBaseURL(p.base_url) === id),
    ) || null
  );
}

/** Add models that are missing; never deletes existing models. */
async function addMissingProviderModels(providerID, models = []) {
  if (!providerID || !models.length) return 0;
  const existing = await listProviderModels(providerID);
  const have = new Set(existing.map((m) => String(m.name || "").trim()).filter(Boolean));
  let added = 0;
  for (const model of models) {
    const name = String(model?.name || "").trim();
    if (!name || have.has(name)) continue;
    await client.post(`${API_PREFIX}/providers/${providerID}/models`, {
      name,
      max_tokens: Number(model?.max_tokens || 32768),
      enabled: model?.enabled ?? true,
    });
    have.add(name);
    added += 1;
  }
  return added;
}

/**
 * Ensure a Vendor=plugin provider exists for the given plugin_id.
 * Idempotent: reuses existing provider; only appends missing default models.
 *
 * @param {{ plugin_id: string, name?: string, protocol?: string, description?: string, models?: Array<{name:string,max_tokens?:number}>, fetch_models?: boolean }} options
 * @returns {Promise<{ provider: object, created: boolean, models_added: number, fetched_models: number }>}
 */
export async function EnsurePluginProvider(options = {}) {
  const pluginId = String(options.plugin_id || options.pluginId || "").trim();
  if (!pluginId) {
    throw new Error("plugin_id is required");
  }
  const defaults = pluginProviderDefaults(pluginId);
  const name = String(options.name || defaults.name).trim() || defaults.name;
  const protocol = options.protocol || defaults.protocol;
  const description =
    options.description != null ? String(options.description) : defaults.description;
  const seedModels =
    Array.isArray(options.models) && options.models.length
      ? options.models
      : defaults.models;
  const shouldFetch = options.fetch_models !== false;

  const providers = await listProvidersWithModels();
  let provider = findPluginProvider(providers, pluginId);
  let created = false;
  let modelsAdded = 0;

  if (!provider) {
    provider = await SaveSupplier({
      name,
      protocol,
      vendor: "plugin",
      plugin_id: pluginId,
      base_url: `plugin://${pluginId}`,
      enabled: true,
      description,
      models: seedModels,
    });
    created = true;
    modelsAdded = seedModels.length;
  } else {
    modelsAdded = await addMissingProviderModels(provider.id, seedModels);
  }

  let fetchedModels = 0;
  if (shouldFetch && provider?.id) {
    try {
      const fetched = await FetchModelsFromProvider(provider.id);
      const toAdd = (fetched || [])
        .filter((m) => m?.name && !m.exists)
        .map((m) => ({
          name: m.name,
          max_tokens: Number(m.max_tokens || 32768),
        }));
      if (toAdd.length) {
        const n = await addMissingProviderModels(provider.id, toAdd);
        fetchedModels = n;
        modelsAdded += n;
      }
    } catch {
      // Plugin may be stopped or models.list unavailable — seed models are enough.
    }
  }

  // Refresh models on the returned provider snapshot.
  if (provider?.id) {
    provider.models = await listProviderModels(provider.id);
    provider.plugin_id = provider.plugin_id || pluginId;
  }

  return {
    provider,
    created,
    models_added: modelsAdded,
    fetched_models: fetchedModels,
  };
}

export function DeleteSupplier(id) {
  return client.delete(`${API_PREFIX}/providers/${id}`);
}
export function ListSupplierHealth() {
  return Promise.resolve([]);
}
export async function CheckSupplier(id) {
  return client.post(`${API_PREFIX}/providers/${id}/check`).then((raw) => ({
    supplier_id: valueOf(raw, "supplier_id", "SupplierID"),
    status: valueOf(raw, "status", "Status", "unreachable"),
    status_code: Number(valueOf(raw, "status_code", "StatusCode", 0) || 0),
    duration_ms: Number(valueOf(raw, "duration_ms", "DurationMS", 0) || 0),
    message: valueOf(raw, "message", "Message"),
    checked_at: valueOf(raw, "checked_at", "CheckedAt"),
  }));
}
export async function ChatWithSupplier(id, input = {}) {
  return client
    .post(`${API_PREFIX}/providers/${id}/chat`, {
      model: String(input.model || "").trim(),
      messages: (input.messages || []).map((m) => ({
        role: String(m?.role || "user").trim(),
        content: String(m?.content || "").trim(),
      })),
      max_tokens: Number(input.max_tokens || 1024),
      temperature: input.temperature,
    })
    .then((raw) => ({
      supplier_id: valueOf(raw, "supplier_id", "SupplierID"),
      model: valueOf(raw, "model", "Model"),
      message: {
        role: valueOf(
          valueOf(raw, "message", "Message", {}),
          "role",
          "Role",
          "assistant",
        ),
        content: valueOf(
          valueOf(raw, "message", "Message", {}),
          "content",
          "Content",
        ),
      },
      status_code: Number(valueOf(raw, "status_code", "StatusCode", 0) || 0),
      duration_ms: Number(valueOf(raw, "duration_ms", "DurationMS", 0) || 0),
    }));
}
