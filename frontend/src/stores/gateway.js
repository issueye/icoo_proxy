// Gateway Store - 管理网关运行状态
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

function isWailsEnv() {
  return typeof window !== 'undefined' && window.go !== undefined;
}

export const useGatewayStore = defineStore('gateway', () => {
  const running = ref(false);
  const host = ref('127.0.0.1');
  const port = ref(16790);
  const providerCount = ref(0);
  const healthyCount = ref(0);
  const apiKeyCount = ref(0);
  const models = ref([]);
  const requestLogs = ref([]);
  const logsLoading = ref(false);
  const gatewayConfig = ref({
    listenHost: '127.0.0.1',
    listenPort: 16790,
    defaultProvider: "",
    logLevel: "info",
    retryCount: 2,
    retryIntervalMs: 500,
  });
  const loading = ref(false);
  const error = ref(null);

  const statusText = computed(() => running.value ? '运行中' : '已停止');
  const statusColor = computed(() => running.value ? 'success' : 'error');

  async function fetchStatus() {
    if (!isWailsEnv()) return;
    try {
      const result = await window.go.services.App.GetGatewayStatus();
      const data = JSON.parse(result);
      running.value = data.running;
      host.value = data.host || data.listenHost || '127.0.0.1';
      port.value = data.port;
      providerCount.value = data.providerCount;
      healthyCount.value = data.healthyCount;
    } catch (e) {
      console.error('Failed to fetch gateway status:', e);
    }
  }

  async function fetchModels() {
    if (!isWailsEnv()) return;
    loading.value = true;
    try {
      const result = await window.go.services.App.GetModels();
      models.value = JSON.parse(result);
    } catch (e) {
      error.value = e.message;
    } finally {
      loading.value = false;
    }
  }

  async function fetchAPIKeys() {
    if (!isWailsEnv()) return;
    try {
      const result = await window.go.services.App.GetAPIKeys();
      const data = JSON.parse(result);
      apiKeyCount.value = Array.isArray(data) ? data.length : 0;
    } catch (e) {
      error.value = e.message;
      apiKeyCount.value = 0;
    }
  }

  async function fetchConfig() {
    if (!isWailsEnv()) return;
    try {
      const result = await window.go.services.App.GetGatewayConfig();
      const data = JSON.parse(result);
      gatewayConfig.value = {
        listenHost: data.listenHost ?? '127.0.0.1',
        listenPort: data.listenPort ?? 16790,
        defaultProvider: data.defaultProvider ?? "",
        logLevel: data.logLevel ?? "info",
        retryCount: data.retryCount ?? 2,
        retryIntervalMs: data.retryIntervalMs ?? 500,
      };
      host.value = gatewayConfig.value.listenHost;
      port.value = gatewayConfig.value.listenPort;
      await fetchAPIKeys();
    } catch (e) {
      error.value = e.message;
    }
  }

  async function saveConfig(patch = {}) {
    if (!isWailsEnv()) return;
    const nextConfig = {
      ...gatewayConfig.value,
      ...patch,
    };
    loading.value = true;
    try {
      await window.go.services.App.SetGatewayConfig(
        nextConfig.listenHost || '127.0.0.1',
        Number(nextConfig.listenPort) || 16790,
        nextConfig.defaultProvider || "",
        nextConfig.logLevel || "info",
        Number(nextConfig.retryCount) || 2,
        Number(nextConfig.retryIntervalMs) || 500,
      );
      gatewayConfig.value = nextConfig;
      host.value = nextConfig.listenHost || '127.0.0.1';
      port.value = Number(nextConfig.listenPort) || 16790;
      await fetchStatus();
      await fetchAPIKeys();
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function fetchRequestLogs(limit = 20) {
    if (!isWailsEnv()) return;
    logsLoading.value = true;
    try {
      const result = await window.go.services.App.GetGatewayRequestLogs(limit);
      const data = JSON.parse(result);
      requestLogs.value = Array.isArray(data) ? data : [];
    } catch (e) {
      error.value = e.message;
    } finally {
      logsLoading.value = false;
    }
  }

  async function refreshModels() {
    if (!isWailsEnv()) return;
    loading.value = true;
    try {
      const result = await window.go.services.App.RefreshModels();
      const data = JSON.parse(result);
      providerCount.value = data.length;
      await fetchModels();
    } catch (e) {
      error.value = e.message;
    } finally {
      loading.value = false;
    }
  }

  async function startGateway() {
    if (!isWailsEnv()) return;
    try {
      await window.go.services.App.StartGateway();
      await fetchStatus();
    } catch (e) {
      error.value = e.message;
    }
  }

  async function stopGateway() {
    if (!isWailsEnv()) return;
    try {
      await window.go.services.App.StopGateway();
      await fetchStatus();
    } catch (e) {
      error.value = e.message;
    }
  }

  return {
    running, host, port, providerCount, healthyCount, apiKeyCount, models, requestLogs, logsLoading, gatewayConfig, loading, error,
    statusText, statusColor,
    fetchStatus, fetchModels, fetchAPIKeys, fetchConfig, saveConfig, fetchRequestLogs, refreshModels, startGateway, stopGateway,
  };
});

