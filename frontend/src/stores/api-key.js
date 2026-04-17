// API Key Store - 管理网关访问密钥
import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

function isWailsEnv() {
  return typeof window !== 'undefined' && window.go !== undefined;
}

function normalizeApiKey(item = {}) {
  return {
    id: item.id || '',
    name: item.name || '',
    key: item.key || '',
    description: item.description || '',
    enabled: item.enabled !== false,
    scopeMode: item.scopeMode || 'all',
    providerIds: Array.isArray(item.providerIds) ? item.providerIds : [],
    endpointIds: Array.isArray(item.endpointIds) ? item.endpointIds : [],
    lastUsedAt: item.lastUsedAt || '',
    createdAt: item.createdAt || '',
    updatedAt: item.updatedAt || '',
  };
}

export const useApiKeyStore = defineStore('apiKey', () => {
  const apiKeys = ref([]);
  const loading = ref(false);
  const error = ref(null);

  const enabledApiKeys = computed(() => apiKeys.value.filter(item => item.enabled));
  const apiKeyCount = computed(() => apiKeys.value.length);

  async function fetchAPIKeys() {
    if (!isWailsEnv()) return;
    loading.value = true;
    error.value = null;
    try {
      const result = await window.go.services.App.GetAPIKeys();
      const data = JSON.parse(result);
      apiKeys.value = Array.isArray(data) ? data.map(normalizeApiKey) : [];
    } catch (e) {
      error.value = e.message;
      apiKeys.value = [];
    } finally {
      loading.value = false;
    }
  }

  async function addAPIKey(payload) {
    if (!isWailsEnv()) return;
    const item = normalizeApiKey(payload);
    loading.value = true;
    error.value = null;
    try {
      await window.go.services.App.AddAPIKey(
        item.id,
        item.name,
        item.key,
        item.description,
        item.enabled,
        item.scopeMode,
        item.providerIds,
        item.endpointIds,
      );
      await fetchAPIKeys();
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function updateAPIKey(payload) {
    if (!isWailsEnv()) return;
    const item = normalizeApiKey(payload);
    loading.value = true;
    error.value = null;
    try {
      await window.go.services.App.UpdateAPIKey(
        item.id,
        item.name,
        item.key,
        item.description,
        item.enabled,
        item.scopeMode,
        item.providerIds,
        item.endpointIds,
      );
      await fetchAPIKeys();
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function deleteAPIKey(id) {
    if (!isWailsEnv()) return;
    loading.value = true;
    error.value = null;
    try {
      await window.go.services.App.DeleteAPIKey(id);
      apiKeys.value = apiKeys.value.filter(item => item.id !== id);
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return {
    apiKeys,
    loading,
    error,
    enabledApiKeys,
    apiKeyCount,
    fetchAPIKeys,
    addAPIKey,
    updateAPIKey,
    deleteAPIKey,
  };
});