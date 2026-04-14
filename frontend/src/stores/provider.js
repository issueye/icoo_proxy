// Provider Store - 管理供应商信息（通过 Wails 绑定）
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

function isWailsEnv() {
  return typeof window !== 'undefined' && window.go !== undefined;
}

export const useProviderStore = defineStore('provider', () => {
  const providers = ref([]);
  const loading = ref(false);
  const error = ref(null);

  const providerCount = computed(() => providers.value.length);
  const enabledProviders = computed(() => providers.value.filter(p => p.enabled));

  async function fetchProviders() {
    if (!isWailsEnv()) return;
    loading.value = true;
    error.value = null;
    try {
      const result = await window.go.services.App.GetProviders();
      providers.value = JSON.parse(result);
    } catch (e) {
      error.value = e.message;
      providers.value = [];
    } finally {
      loading.value = false;
    }
  }

  async function addProvider({ id, name, type, apiBase, apiKey, enabled, priority }) {
    loading.value = true;
    error.value = null;
    try {
      await window.go.services.App.AddProvider(
        id || '', name, type, apiBase, apiKey || '', enabled !== false, priority || 0
      );
      await fetchProviders();
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function updateProvider({ id, name, type, apiBase, apiKey, enabled, priority }) {
    loading.value = true;
    error.value = null;
    try {
      await window.go.services.App.UpdateProvider(
        id, name, type, apiBase, apiKey || '', enabled, priority || 0
      );
      await fetchProviders();
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function deleteProvider(id) {
    loading.value = true;
    error.value = null;
    try {
      await window.go.services.App.DeleteProvider(id);
      providers.value = providers.value.filter(p => p.id !== id);
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function testProvider({ id, name, type, apiBase, apiKey }) {
    if (!isWailsEnv()) return { success: false, error: 'Not in Wails environment' };
    try {
      const result = await window.go.services.App.TestProvider(
        id || '', name, type, apiBase, apiKey || ''
      );
      return JSON.parse(result);
    } catch (e) {
      return { success: false, error: e.message };
    }
  }

  async function getProviderModels(providerId) {
    if (!isWailsEnv()) return { llms: [], defaultModel: '' };
    const result = await window.go.services.App.GetProviderModels(providerId);
    return JSON.parse(result);
  }

  async function setProviderModels(providerId, llms, defaultModel) {
    if (!isWailsEnv()) return;
    await window.go.services.App.SetProviderModels(providerId, llms, defaultModel);
    await fetchProviders();
  }

  return {
    providers, loading, error,
    providerCount, enabledProviders,
    fetchProviders, addProvider, updateProvider, deleteProvider, testProvider,
    getProviderModels, setProviderModels,
  };
});
