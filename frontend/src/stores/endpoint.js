// Endpoint Store - 管理上游端点资源
import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

function isWailsEnv() {
  return typeof window !== 'undefined' && window.go !== undefined;
}

function normalizeEndpoint(item = {}) {
  return {
    id: item.id || '',
    name: item.name || '',
    providerId: item.providerId || '',
    path: item.path || '',
    method: item.method || 'POST',
    capability: item.capability || '',
    requestProtocol: item.requestProtocol || '',
    responseProtocol: item.responseProtocol || '',
    enabled: item.enabled !== false,
    priority: Number(item.priority) || 0,
    isDefault: item.isDefault === true,
    remark: item.remark || '',
  };
}

export const useEndpointStore = defineStore('endpoint', () => {
  const endpoints = ref([]);
  const loading = ref(false);
  const error = ref(null);

  const enabledEndpoints = computed(() => endpoints.value.filter(item => item.enabled));
  const endpointCount = computed(() => endpoints.value.length);

  async function fetchEndpoints() {
    if (!isWailsEnv()) return;
    loading.value = true;
    error.value = null;
    try {
      const result = await window.go.services.App.GetEndpoints();
      const data = JSON.parse(result);
      endpoints.value = Array.isArray(data) ? data.map(normalizeEndpoint) : [];
    } catch (e) {
      error.value = e.message;
      endpoints.value = [];
    } finally {
      loading.value = false;
    }
  }

  async function addEndpoint(payload) {
    if (!isWailsEnv()) return;
    const item = normalizeEndpoint(payload);
    loading.value = true;
    error.value = null;
    try {
      await window.go.services.App.AddEndpoint(
        item.id,
        item.name,
        item.providerId,
        item.path,
        item.method,
        item.capability,
        item.requestProtocol,
        item.responseProtocol,
        item.enabled,
        item.priority,
        item.isDefault,
        item.remark,
      );
      await fetchEndpoints();
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function updateEndpoint(payload) {
    if (!isWailsEnv()) return;
    const item = normalizeEndpoint(payload);
    loading.value = true;
    error.value = null;
    try {
      await window.go.services.App.UpdateEndpoint(
        item.id,
        item.name,
        item.providerId,
        item.path,
        item.method,
        item.capability,
        item.requestProtocol,
        item.responseProtocol,
        item.enabled,
        item.priority,
        item.isDefault,
        item.remark,
      );
      await fetchEndpoints();
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function deleteEndpoint(id) {
    if (!isWailsEnv()) return;
    loading.value = true;
    error.value = null;
    try {
      await window.go.services.App.DeleteEndpoint(id);
      endpoints.value = endpoints.value.filter(item => item.id !== id);
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  return {
    endpoints,
    loading,
    error,
    enabledEndpoints,
    endpointCount,
    fetchEndpoints,
    addEndpoint,
    updateEndpoint,
    deleteEndpoint,
  };
});