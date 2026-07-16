import { client, API_PREFIX } from "./transport";

function withPluginApiHint(err) {
  const status = err?.response?.status;
  const msg = err?.response?.data?.error?.message || err?.message || String(err);
  if (status === 404 || /not found|404/i.test(msg)) {
    return new Error(
      "插件 API 不存在（404）。当前连接的 bridge 可能是旧版本，请停止旧进程后使用本仓库构建的 bridge.exe 重启（确保包含 /api/v1/plugins）。",
    );
  }
  return err instanceof Error ? err : new Error(msg);
}

export async function ListPlugins() {
  try {
    return await client.get(`${API_PREFIX}/plugins`);
  } catch (err) {
    throw withPluginApiHint(err);
  }
}

export async function ListPluginUIPages() {
  try {
    return await client.get(`${API_PREFIX}/plugins/ui-pages`);
  } catch (err) {
    throw withPluginApiHint(err);
  }
}

export async function DiscoverPlugins() {
  try {
    return await client.get(`${API_PREFIX}/plugins/discover`);
  } catch (err) {
    throw withPluginApiHint(err);
  }
}

export async function InstallPlugin(payload) {
  try {
    return await client.post(`${API_PREFIX}/plugins/install`, payload);
  } catch (err) {
    throw withPluginApiHint(err);
  }
}

export async function RegisterPlugin(payload) {
  try {
    return await client.post(`${API_PREFIX}/plugins`, payload);
  } catch (err) {
    throw withPluginApiHint(err);
  }
}

export async function UnregisterPlugin(id) {
  try {
    return await client.delete(`${API_PREFIX}/plugins/${encodeURIComponent(id)}`);
  } catch (err) {
    throw withPluginApiHint(err);
  }
}

export async function SetPluginEnabled(id, enabled) {
  try {
    return await client.put(`${API_PREFIX}/plugins/${encodeURIComponent(id)}/enabled`, { enabled });
  } catch (err) {
    throw withPluginApiHint(err);
  }
}

export function StartPlugin(id) {
  return client.post(`${API_PREFIX}/plugins/${encodeURIComponent(id)}/start`);
}

export function StopPlugin(id) {
  return client.post(`${API_PREFIX}/plugins/${encodeURIComponent(id)}/stop`);
}

export function RestartPlugin(id) {
  return client.post(`${API_PREFIX}/plugins/${encodeURIComponent(id)}/restart`);
}

export function GetPluginHealth(id) {
  return client.get(`${API_PREFIX}/plugins/${encodeURIComponent(id)}/health`);
}
