<template>
  <div class="gateway-view app-page">
    <PageHeader
      title="网关总览"
      description="统一管理网关状态、入口鉴权、可用模型和对外接入方式。"
    />

    <!-- 状态卡片 -->
    <div class="stats-grid">
      <div class="stat-card" :class="{ 'stat-ok': gatewayStore.running }">
        <div class="stat-icon">
          <Server :size="24" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ gatewayStore.running ? '运行中' : '已停止' }}</div>
          <div class="stat-label">网关状态</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon">
          <Cpu :size="24" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ gatewayStore.providerCount }}</div>
          <div class="stat-label">供应商数量</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon">
          <Activity :size="24" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ gatewayStore.models.length }}</div>
          <div class="stat-label">可用模型</div>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon">
          <HeartPulse :size="24" />
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ gatewayStore.healthyCount }}</div>
          <div class="stat-label">健康供应商</div>
        </div>
      </div>
    </div>

    <!-- 网关控制 -->
    <div class="section">
      <h3 class="section-title">网关控制</h3>
      <div class="control-bar">
        <div class="control-info">
          <span class="control-label">监听端口:</span>
          <code class="control-value">127.0.0.1:{{ gatewayStore.port }}</code>
        </div>
        <div class="control-info">
          <span class="control-label">API 地址:</span>
          <code class="control-value">http://localhost:{{ gatewayStore.port }}/v1</code>
        </div>
        <div class="control-actions">
          <button v-if="!gatewayStore.running" class="btn btn-success" @click="handleStart" :disabled="gatewayStore.loading">
            <Play :size="14" /> 启动
          </button>
          <button v-else class="btn btn-danger" @click="handleStop" :disabled="gatewayStore.loading">
            <Square :size="14" /> 停止
          </button>
          <button class="btn btn-secondary" @click="handleRefresh" :disabled="gatewayStore.loading">
            <RefreshCw :size="14" :class="{ spinning: gatewayStore.loading }" /> 刷新模型
          </button>
        </div>
      </div>
    </div>

    <!-- 快速测试 -->
    <div class="section">
      <h3 class="section-title">快速测试</h3>
      <div class="mode-grid">
        <div v-for="item in requestModes" :key="item.name" class="mode-card">
          <div class="mode-card-head">
            <div>
              <div class="mode-card-title">{{ item.name }}</div>
              <div class="mode-card-endpoint">{{ item.endpoint }}</div>
            </div>
            <span class="mode-chip">{{ item.payload }}</span>
          </div>
          <p class="mode-card-description">{{ item.description }}</p>
        </div>
      </div>
      <div class="test-hint">
        <pre class="code-block"><code>curl http://localhost:{{ gatewayStore.port }}/v1/chat/completions \
  {{ gatewayStore.gatewayConfig.authKey ? `-H "Authorization: Bearer ${gatewayStore.gatewayConfig.authKey}" \\` : '' }}
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}'</code></pre>
        <pre class="code-block"><code>curl http://localhost:{{ gatewayStore.port }}/v1/responses \
  {{ gatewayStore.gatewayConfig.authKey ? `-H "Authorization: Bearer ${gatewayStore.gatewayConfig.authKey}" \\` : '' }}
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4.1","input":"hello"}'</code></pre>
        <pre class="code-block"><code>curl http://localhost:{{ gatewayStore.port }}/v1/models{{ gatewayStore.gatewayConfig.authKey ? ` \
  -H "Authorization: Bearer ${gatewayStore.gatewayConfig.authKey}"` : '' }}</code></pre>
      </div>
    </div>

    <div class="section">
      <div class="section-heading">
        <h3 class="section-title">访问鉴权</h3>
      </div>
      <div class="auth-card">
        <div class="auth-copy">
          <div class="auth-copy-title">访问密钥</div>
          <div class="auth-copy-meta">
            <span class="auth-meta-chip">Bearer Token</span>
            <span class="auth-meta-chip">x-api-key</span>
            <span class="auth-meta-chip auth-meta-chip--muted">重启后保留</span>
          </div>
        </div>
        <div class="auth-panel">
          <div class="auth-status-row">
            <div>
              <div class="auth-status-label">当前状态</div>
              <div :class="gatewayStore.gatewayConfig.authKey ? 'auth-enabled' : 'auth-disabled'">
                {{ gatewayStore.gatewayConfig.authKey ? '已启用鉴权' : '未启用鉴权' }}
              </div>
            </div>
          </div>

          <div class="auth-form">
            <label class="control-label">鉴权 Key</label>
            <div class="auth-input-shell">
              <input
                v-model="authKeyDraft"
                :type="showAuthKey ? 'text' : 'password'"
                class="auth-input"
                placeholder="留空表示关闭鉴权"
              >
            </div>
            <div class="auth-actions">
              <button class="btn btn-secondary auth-action-btn" type="button" @click="showAuthKey = !showAuthKey">
                {{ showAuthKey ? '隐藏' : '显示' }}
              </button>
              <button class="btn btn-secondary auth-action-btn" type="button" @click="handleGenerateAuthKey">
                生成
              </button>
              <button class="btn btn-primary auth-action-btn auth-action-btn--primary" type="button" @click="handleSaveAuthKey" :disabled="gatewayStore.loading">
                保存
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue';
import { useGatewayStore } from '@/stores/gateway';
import { useProviderStore } from '@/stores/provider';
import {
  Server, Cpu, Activity, HeartPulse, Play, Square, RefreshCw,
} from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';

const gatewayStore = useGatewayStore();
const providerStore = useProviderStore();
let timer = null;
const showAuthKey = ref(false);
const authKeyDraft = ref("");
const requestModes = [
  {
    name: "Chat Completions",
    endpoint: "POST /v1/chat/completions",
    payload: "messages",
    description: "兼容传统 OpenAI Chat Completions 结构，使用 messages 数组组织上下文。",
  },
  {
    name: "Responses",
    endpoint: "POST /v1/responses",
    payload: "input",
    description: "兼容 OpenAI Responses API，支持 input、instructions 和函数工具定义。",
  },
];

async function handleStart() {
  await gatewayStore.startGateway();
}

async function handleStop() {
  await gatewayStore.stopGateway();
}

async function handleRefresh() {
  await gatewayStore.refreshModels();
  await providerStore.fetchProviders();
}

async function handleSaveAuthKey() {
  await gatewayStore.saveConfig({ authKey: authKeyDraft.value.trim() });
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
  authKeyDraft.value = gatewayStore.gatewayConfig.authKey || "";
  await providerStore.fetchProviders();
  timer = window.setInterval(() => {
    gatewayStore.fetchStatus();
  }, 10000);
});

onUnmounted(() => {
  if (timer) window.clearInterval(timer);
});
</script>

<style scoped>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}
.stat-card.stat-ok {
  border-color: var(--color-success, #22c55e);
  background: color-mix(in srgb, var(--color-success, #22c55e) 8%, var(--color-bg-secondary));
}

.stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary, var(--color-bg-primary));
  color: var(--color-text-muted);
}

.stat-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-primary);
}
.stat-label {
  font-size: 12px;
  color: var(--color-text-muted);
}

.section {
  margin-bottom: 24px;
}
.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}
.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 0;
}
.section-tools {
  display: flex;
  align-items: center;
  gap: 10px;
}

.control-bar {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  flex-wrap: wrap;
}
.control-info {
  display: flex;
  align-items: center;
  gap: 6px;
}
.control-label {
  font-size: 13px;
  color: var(--color-text-muted);
}
.control-value {
  font-size: 13px;
  color: var(--color-accent);
  background: var(--color-bg-tertiary, var(--color-bg-primary));
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}
.control-actions {
  display: flex;
  gap: 8px;
  margin-left: auto;
}
.auth-card {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) minmax(420px, 1.15fr);
  gap: 18px;
  padding: 18px 20px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  align-items: stretch;
}
.auth-copy-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-primary);
}
.auth-copy-meta {
  margin-top: 14px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.auth-meta-chip {
  padding: 5px 10px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-accent) 10%, var(--color-bg-tertiary, var(--color-bg-primary)));
  color: var(--color-accent);
  font-size: 11px;
  font-weight: 600;
}
.auth-meta-chip--muted {
  background: var(--color-bg-tertiary, var(--color-bg-primary));
  color: var(--color-text-muted);
}
.auth-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 14px;
  border-radius: var(--radius-lg);
  background: var(--color-bg-tertiary, var(--color-bg-primary));
  border: 1px solid color-mix(in srgb, var(--color-border) 85%, transparent);
}
.auth-status-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}
.auth-status-label {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 4px;
}
.auth-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.auth-input-shell {
  padding: 10px 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: #fff;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.3);
}
.auth-input {
  width: 100%;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-text-primary);
  font-size: 13px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
.auth-input:focus {
  outline: none;
}
.auth-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.auth-action-btn {
  min-width: 72px;
  justify-content: center;
  padding: 7px 14px;
}
.auth-action-btn--primary {
  min-width: 88px;
}
.auth-enabled {
  color: var(--color-success, #16a34a);
  font-weight: 600;
  font-size: 14px;
}
.auth-disabled {
  color: #b45309;
  font-weight: 600;
  font-size: 14px;
}
.spinning {
  animation: spin 1s linear infinite;
}
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.test-hint {
  font-size: 13px;
  color: var(--color-text-muted);
  line-height: 1.6;
}
.mode-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}
.mode-card {
  padding: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--color-accent) 8%, transparent), transparent 55%),
    var(--color-bg-secondary);
}
.mode-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.mode-card-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-primary);
}
.mode-card-endpoint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
.mode-card-description {
  margin: 10px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-text-secondary);
}
.mode-chip {
  display: inline-flex;
  align-items: center;
  padding: 5px 10px;
  border-radius: 999px;
  background: var(--color-bg-tertiary, var(--color-bg-primary));
  color: var(--color-accent);
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
}
.code-block {
  margin-top: 8px;
  padding: 12px 16px;
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary, #1e1e2e);
  overflow-x: auto;
  font-size: 12px;
  line-height: 1.5;
}
.code-block code {
  color: var(--color-text-secondary);
}

.provider-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.provider-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}
.provider-left {
  display: flex;
  align-items: center;
  gap: 10px;
}
.provider-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.provider-type {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--color-bg-tertiary);
  color: var(--color-text-muted);
  text-transform: uppercase;
}
.provider-models {
  font-size: 12px;
  color: var(--color-text-muted);
}

.empty-hint {
  font-size: 13px;
  color: var(--color-text-muted);
  padding: 24px;
  text-align: center;
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-lg);
}
.empty-hint a {
  color: var(--color-accent);
  text-decoration: underline;
}

.section-link {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-accent);
  text-decoration: none;
}

.log-module-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background:
    linear-gradient(135deg, rgba(14, 165, 233, 0.06), rgba(14, 165, 233, 0.01)),
    var(--color-bg-secondary);
}

.log-module-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.log-module-actions {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 10px;
  flex-shrink: 0;
}

.log-module-chip {
  display: inline-flex;
  align-items: center;
  padding: 6px 10px;
  border-radius: 999px;
  background: var(--color-bg-tertiary, var(--color-bg-primary));
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.route-rules-panel {
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}
.route-rule-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.route-rule-card {
  padding: 14px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-tertiary, var(--color-bg-primary));
}
.route-rule-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}
.route-rule-actions {
  margin-top: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

@media (max-width: 1100px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .route-rule-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .gateway-view {
    padding: 16px;
  }
  .stats-grid {
    grid-template-columns: 1fr;
  }
  .mode-grid {
    grid-template-columns: 1fr;
  }
  .section-heading,
  .log-module-card,
  .auth-card,
  .auth-status-row {
    flex-direction: column;
    align-items: flex-start;
  }
  .route-rule-grid {
    grid-template-columns: 1fr;
  }
  .route-rule-actions {
    flex-direction: column;
    align-items: flex-start;
  }
  .auth-actions {
    width: 100%;
  }
  .auth-actions .btn {
    width: auto;
    justify-content: center;
  }
  .log-module-actions {
    width: 100%;
    align-items: stretch;
  }
}
</style>
