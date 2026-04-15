<template>
  <div class="gateway-view app-page">
    <PageHeader title="网关总览">
      <template #actions>
        <button class="btn btn-secondary" @click="handleRefresh" :disabled="gatewayStore.loading">
          <RefreshCw :size="14" :class="{ spinning: gatewayStore.loading }" />
          刷新状态
        </button>
      </template>
    </PageHeader>

    <section class="summary-strip">
      <div class="summary-item">
        <span class="summary-label">运行状态</span>
        <StatusBadge :status="gatewayStore.running ? 'success' : 'error'" :label="gatewayStore.running ? '运行中' : '已停止'" />
      </div>
      <div class="summary-item">
        <span class="summary-label">监听地址</span>
        <code class="summary-value">127.0.0.1:{{ gatewayStore.port }}</code>
      </div>
      <div class="summary-item">
        <span class="summary-label">供应商</span>
        <span class="summary-value">{{ gatewayStore.providerCount }} 个</span>
      </div>
      <div class="summary-item">
        <span class="summary-label">可用模型</span>
        <span class="summary-value">{{ gatewayStore.models.length }} 个</span>
      </div>
      <div class="summary-item">
        <span class="summary-label">健康供应商</span>
        <span class="summary-value">{{ gatewayStore.healthyCount }} 个</span>
      </div>
    </section>

    <div class="gateway-grid">
      <section class="panel-card">
        <div class="panel-head">
          <h3 class="section-title">网关控制</h3>
        </div>

        <div class="control-layout">
          <div class="info-list">
            <div class="info-row">
              <span class="info-row-label">API Base</span>
              <code class="summary-value">http://localhost:{{ gatewayStore.port }}/v1</code>
            </div>
            <div class="info-row">
              <span class="info-row-label">鉴权模式</span>
              <span class="info-row-value">{{ gatewayStore.gatewayConfig.authKey ? "Bearer / x-api-key" : "未启用" }}</span>
            </div>
          </div>

          <div class="control-actions">
            <button v-if="!gatewayStore.running" class="btn btn-success" @click="handleStart" :disabled="gatewayStore.loading">
              <Play :size="14" />
              启动网关
            </button>
            <button v-else class="btn btn-danger" @click="handleStop" :disabled="gatewayStore.loading">
              <Square :size="14" />
              停止网关
            </button>
            <button class="btn btn-secondary" type="button" @click="copyApiBase">
              <Copy :size="14" />
              复制地址
            </button>
          </div>
        </div>
      </section>

      <section class="panel-card">
        <div class="panel-head">
          <h3 class="section-title">诊断提示</h3>
        </div>

        <div class="diagnostic-list">
          <div class="diagnostic-item" :class="{ warning: !gatewayStore.running }">
            <StatusBadge :status="gatewayStore.running ? 'success' : 'warning'" :label="gatewayStore.running ? '网关正在监听请求' : '网关尚未启动'" />
          </div>
          <div class="diagnostic-item" :class="{ warning: gatewayStore.providerCount === 0 }">
            <StatusBadge :status="gatewayStore.providerCount > 0 ? 'success' : 'warning'" :label="gatewayStore.providerCount > 0 ? '已配置供应商' : '尚未配置供应商'" />
          </div>
          <div class="diagnostic-item" :class="{ warning: gatewayStore.models.length === 0 }">
            <StatusBadge :status="gatewayStore.models.length > 0 ? 'success' : 'warning'" :label="gatewayStore.models.length > 0 ? '模型列表已同步' : '模型尚未刷新'" />
          </div>
        </div>
      </section>

      <section class="panel-card panel-card--wide">
        <div class="panel-head">
          <h3 class="section-title">访问鉴权</h3>
          <StatusBadge
            :status="gatewayStore.gatewayConfig.authKey ? 'info' : 'neutral'"
            :label="gatewayStore.gatewayConfig.authKey ? '已启用鉴权' : '未启用鉴权'"
          />
        </div>

        <div class="auth-layout">
          <div class="auth-form">
            <div class="auth-input-shell">
              <input
                v-model="authKeyDraft"
                :type="showAuthKey ? 'text' : 'password'"
                class="auth-input"
                placeholder="留空表示关闭鉴权"
              >
              <button class="btn btn-secondary btn-sm" type="button" @click="showAuthKey = !showAuthKey">
                {{ showAuthKey ? "隐藏" : "显示" }}
              </button>
            </div>

            <div class="auth-actions">
              <button class="btn btn-secondary" type="button" @click="handleGenerateAuthKey">
                生成随机 Key
              </button>
              <button class="btn btn-primary" type="button" @click="handleSaveAuthKey" :disabled="gatewayStore.loading">
                保存设置
              </button>
            </div>
          </div>
        </div>
      </section>

      <section class="panel-card panel-card--wide">
        <div class="panel-head">
          <h3 class="section-title">接口调用示例</h3>
        </div>

        <div class="example-tabs">
          <button
            v-for="item in requestModes"
            :key="item.key"
            class="settings-tab-button"
            :class="{ 'is-active': activeExample === item.key }"
            @click="activeExample = item.key"
          >
            {{ item.name }}
          </button>
        </div>

        <div class="example-panel">
          <div class="example-meta">
            <div>
              <div class="example-endpoint">{{ activeRequestMode.endpoint }}</div>
              <p class="panel-description">{{ activeRequestMode.description }}</p>
            </div>
            <span class="panel-chip">{{ activeRequestMode.payload }}</span>
          </div>

          <pre class="code-block"><code>{{ activeCurlCommand }}</code></pre>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useGatewayStore } from '@/stores/gateway';
import { useProviderStore } from '@/stores/provider';
import { Play, Square, RefreshCw, Copy } from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';
import StatusBadge from '@/components/ui/StatusBadge.vue';
import { useToast } from '@/composables/useToast';

const gatewayStore = useGatewayStore();
const providerStore = useProviderStore();
const { toast } = useToast();

let timer = null;
const showAuthKey = ref(false);
const authKeyDraft = ref("");
const activeExample = ref("chat");

const requestModes = [
  {
    key: "chat",
    name: "Chat Completions",
    endpoint: "POST /v1/chat/completions",
    payload: "messages",
    description: "兼容传统 OpenAI Chat Completions 请求结构，适合现有 SDK 直接迁移。",
  },
  {
    key: "responses",
    name: "Responses",
    endpoint: "POST /v1/responses",
    payload: "input",
    description: "兼容 Responses API，适合统一输入、工具调用和多模态扩展场景。",
  },
  {
    key: "models",
    name: "Models",
    endpoint: "GET /v1/models",
    payload: "list",
    description: "用于检查当前暴露给客户端的模型清单与同步状态。",
  },
];

const activeRequestMode = computed(
  () => requestModes.find((item) => item.key === activeExample.value) || requestModes[0]
);

const authHeader = computed(() =>
  gatewayStore.gatewayConfig.authKey
    ? `-H "Authorization: Bearer ${gatewayStore.gatewayConfig.authKey}" \\\n  `
    : ""
);

const activeCurlCommand = computed(() => {
  const port = gatewayStore.port;
  if (activeExample.value === "responses") {
    return `curl http://localhost:${port}/v1/responses \\\n  ${authHeader.value}-H "Content-Type: application/json" \\\n  -d '{"model":"gpt-4.1","input":"hello"}'`;
  }
  if (activeExample.value === "models") {
    return `curl http://localhost:${port}/v1/models${gatewayStore.gatewayConfig.authKey ? ` \\\n  -H "Authorization: Bearer ${gatewayStore.gatewayConfig.authKey}"` : ""}`;
  }
  return `curl http://localhost:${port}/v1/chat/completions \\\n  ${authHeader.value}-H "Content-Type: application/json" \\\n  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}'`;
});

async function handleStart() {
  await gatewayStore.startGateway();
}

async function handleStop() {
  await gatewayStore.stopGateway();
}

async function handleRefresh() {
  await gatewayStore.fetchStatus();
  await gatewayStore.fetchModels();
  await providerStore.fetchProviders();
}

async function handleSaveAuthKey() {
  await gatewayStore.saveConfig({ authKey: authKeyDraft.value.trim() });
}

async function copyApiBase() {
  const value = `http://localhost:${gatewayStore.port}/v1`;
  try {
    await navigator.clipboard.writeText(value);
    toast("API 地址已复制", "success");
  } catch {
    toast("复制失败，请手动复制地址", "error");
  }
}

function handleGenerateAuthKey() {
  authKeyDraft.value = generateGatewayKey();
  showAuthKey.value = true;
}

function generateGatewayKey() {
  const prefix = 'gw_';
  const size = 24;

  if (typeof globalThis !== 'undefined' && globalThis.crypto?.getRandomValues) {
    const bytes = new Uint8Array(size);
    globalThis.crypto.getRandomValues(bytes);
    return prefix + Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
  }

  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  let result = prefix;
  for (let i = 0; i < size * 2; i += 1) {
    result += chars[Math.floor(Math.random() * chars.length)];
  }
  return result;
}

onMounted(async () => {
  await gatewayStore.fetchStatus();
  await gatewayStore.fetchModels();
  await gatewayStore.fetchConfig();
  await providerStore.fetchProviders();
  authKeyDraft.value = gatewayStore.gatewayConfig.authKey || "";
  timer = window.setInterval(() => {
    gatewayStore.fetchStatus();
  }, 10000);
});

onUnmounted(() => {
  if (timer) window.clearInterval(timer);
});
</script>

<style scoped>
.gateway-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.summary-strip {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 0;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-rest);
  overflow: hidden;
}

.summary-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;
  border-right: 1px solid var(--ui-border-subtle);
}

.summary-item:last-child {
  border-right: 0;
}

.summary-label {
  font-size: 12px;
  color: var(--color-text-muted);
}

.summary-value {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.gateway-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.panel-card {
  padding: 16px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-rest);
}

.panel-card--wide {
  grid-column: 1 / -1;
}

.control-layout,
.auth-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(280px, 0.9fr);
  gap: 16px;
}

.control-actions,
.auth-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.control-actions {
  justify-content: flex-end;
}

.info-list,
.diagnostic-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.info-row,
.diagnostic-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
  border: 1px solid var(--ui-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
}

.diagnostic-item.warning {
  border-color: color-mix(in srgb, var(--color-warning) 24%, var(--ui-border-default));
}

.diagnostic-item p {
  margin: 0;
  font-size: 12px;
  line-height: 1.55;
  color: var(--color-text-secondary);
}

.info-row-label,
.auth-aside-title,
.example-endpoint {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.info-row-value {
  font-size: 13px;
  color: var(--color-text-primary);
}

.auth-aside {
  padding: 12px;
  border: 1px solid var(--ui-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.auth-input-shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  padding: 8px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface);
}

.auth-input-shell:focus-within {
  border-color: var(--color-accent);
  box-shadow: var(--shadow-focus);
}

.auth-input {
  width: 100%;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--color-text-primary);
  font-family: var(--font-mono);
  font-size: 13px;
}

.example-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.example-panel {
  padding: 14px;
  border: 1px solid var(--ui-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
}

.example-meta {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.code-block {
  margin: 0;
  padding: 14px 16px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface);
  overflow-x: auto;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-text-secondary);
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (max-width: 1180px) {
  .summary-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .summary-item:nth-child(2n) {
    border-right: 0;
  }

  .gateway-grid {
    grid-template-columns: 1fr;
  }

  .panel-card--wide {
    grid-column: auto;
  }
}

@media (max-width: 820px) {
  .control-layout,
  .auth-layout {
    grid-template-columns: 1fr;
  }

  .control-actions {
    justify-content: flex-start;
  }

  .example-meta {
    flex-direction: column;
  }
}

@media (max-width: 640px) {
  .summary-strip {
    grid-template-columns: 1fr;
  }

  .summary-item {
    border-right: 0;
    border-bottom: 1px solid var(--ui-border-subtle);
  }

  .summary-item:last-child {
    border-bottom: 0;
  }

  .auth-input-shell {
    grid-template-columns: 1fr;
  }
}
</style>
