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

    <section class="gateway-hero" :class="{ 'is-running': gatewayStore.running }">
      <div class="gateway-hero__content">
        <div class="gateway-hero__kicker">
          <Activity :size="14" />
          本地代理网关
        </div>
        <h2 class="gateway-hero__title">
          {{ gatewayStore.running ? "网关已就绪，正在接收请求" : "网关当前未启动" }}
        </h2>
        <p class="gateway-hero__description">
          {{ gatewayStore.running
            ? "客户端可通过统一 OpenAI 兼容入口访问已配置供应商，并按模型映射与规则完成转发。"
            : "启动网关后，应用会监听本地端口并暴露兼容 OpenAI 的 /v1 接口。"
          }}
        </p>
        <div class="gateway-hero__meta">
          <StatusBadge :status="gatewayStore.running ? 'success' : 'error'" :label="gatewayStore.running ? '运行中' : '已停止'" />
          <span class="summary-chip">健康度 {{ providerHealthPercent }}%</span>
          <span class="summary-chip">每 10 秒自动刷新状态</span>
        </div>
      </div>

      <div class="gateway-hero__console">
        <div class="console-line">
          <span class="console-label">API Base</span>
          <code>http://localhost:{{ gatewayStore.port }}/v1</code>
        </div>
        <div class="console-line">
          <span class="console-label">Listen</span>
          <code>127.0.0.1:{{ gatewayStore.port }}</code>
        </div>
        <div class="console-line">
          <span class="console-label">Auth</span>
          <code>{{ gatewayStore.gatewayConfig.authKey ? "Bearer / x-api-key" : "disabled" }}</code>
        </div>
        <div class="gateway-hero__actions">
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

    <section class="summary-grid">
      <div v-for="metric in summaryMetrics" :key="metric.label" class="summary-card">
        <div class="summary-card__icon">
          <component :is="metric.icon" :size="17" />
        </div>
        <div class="summary-card__content">
          <span class="summary-label">{{ metric.label }}</span>
          <span class="summary-value">{{ metric.value }}</span>
          <span class="summary-hint">{{ metric.hint }}</span>
        </div>
      </div>
    </section>

    <div class="gateway-grid">
      <section class="panel-card">
        <div class="panel-head">
          <div>
            <h3 class="section-title">网关控制</h3>
            <p class="panel-description">确认客户端连接地址、鉴权状态与当前启动动作。</p>
          </div>
          <span class="panel-chip">{{ readinessLabel }}</span>
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
            <div class="info-row">
              <span class="info-row-label">上游健康</span>
              <span class="info-row-value">{{ gatewayStore.healthyCount }} / {{ gatewayStore.providerCount }} 个供应商可用</span>
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
          <div>
            <h3 class="section-title">诊断提示</h3>
            <p class="panel-description">按上线前检查顺序展示当前阻塞项。</p>
          </div>
        </div>

        <div class="diagnostic-list">
          <div
            v-for="(item, index) in diagnosticItems"
            :key="item.label"
            class="diagnostic-item"
            :class="{ warning: !item.ok }"
          >
            <span class="diagnostic-step">{{ index + 1 }}</span>
            <div class="diagnostic-copy">
              <StatusBadge :status="item.ok ? 'success' : 'warning'" :label="item.label" />
              <p>{{ item.description }}</p>
            </div>
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
            <p class="auth-helper">
              设置后客户端需要携带 <code>Authorization: Bearer</code> 或 <code>x-api-key</code>，留空保存则关闭访问鉴权。
            </p>
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

          <div class="auth-aside">
            <div class="auth-aside-title">安全建议</div>
            <p>本地开发可关闭鉴权；如果暴露到局域网或被其他应用调用，建议生成随机 Key 并妥善保存。</p>
          </div>
        </div>
      </section>

      <section class="panel-card panel-card--wide">
        <div class="panel-head">
          <div>
            <h3 class="section-title">接口调用示例</h3>
            <p class="panel-description">选择常用端点后复制命令，可快速验证网关转发链路。</p>
          </div>
          <button class="btn btn-secondary" type="button" @click="copyCurlCommand">
            <Copy :size="14" />
            复制示例
          </button>
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
import { Activity, Boxes, Cpu, Network, Play, Server, ShieldCheck, Square, RefreshCw, Copy } from 'lucide-vue-next';
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

const providerHealthPercent = computed(() => {
  if (gatewayStore.providerCount === 0) return 0;
  return Math.round((gatewayStore.healthyCount / gatewayStore.providerCount) * 100);
});

const readinessLabel = computed(() => {
  if (!gatewayStore.running) return "等待启动";
  if (gatewayStore.providerCount === 0) return "需要供应商";
  if (gatewayStore.models.length === 0) return "等待模型";
  return "可接入";
});

const summaryMetrics = computed(() => [
  {
    label: "监听地址",
    value: `:${gatewayStore.port}`,
    hint: `127.0.0.1:${gatewayStore.port}`,
    icon: Network,
  },
  {
    label: "供应商",
    value: `${gatewayStore.providerCount} 个`,
    hint: `${gatewayStore.healthyCount} 个健康`,
    icon: Server,
  },
  {
    label: "可用模型",
    value: `${gatewayStore.models.length} 个`,
    hint: gatewayStore.models.length > 0 ? "模型清单已同步" : "等待刷新模型",
    icon: Boxes,
  },
  {
    label: "访问鉴权",
    value: gatewayStore.gatewayConfig.authKey ? "已启用" : "未启用",
    hint: gatewayStore.gatewayConfig.authKey ? "Bearer / x-api-key" : "本地免鉴权",
    icon: ShieldCheck,
  },
  {
    label: "运行状态",
    value: gatewayStore.running ? "运行中" : "已停止",
    hint: readinessLabel.value,
    icon: Cpu,
  },
]);

const diagnosticItems = computed(() => [
  {
    ok: gatewayStore.running,
    label: gatewayStore.running ? "网关正在监听请求" : "网关尚未启动",
    description: gatewayStore.running
      ? `当前监听 127.0.0.1:${gatewayStore.port}，客户端可以接入。`
      : "启动网关后才会开放本地 /v1 兼容接口。",
  },
  {
    ok: gatewayStore.providerCount > 0,
    label: gatewayStore.providerCount > 0 ? "已配置供应商" : "尚未配置供应商",
    description: gatewayStore.providerCount > 0
      ? `${gatewayStore.healthyCount} / ${gatewayStore.providerCount} 个供应商处于健康状态。`
      : "请先在供应商管理中添加至少一个上游服务。",
  },
  {
    ok: gatewayStore.models.length > 0,
    label: gatewayStore.models.length > 0 ? "模型列表已同步" : "模型尚未刷新",
    description: gatewayStore.models.length > 0
      ? `当前向客户端暴露 ${gatewayStore.models.length} 个模型。`
      : "刷新模型后，客户端才能通过 /v1/models 看到可用模型。",
  },
]);

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

async function copyCurlCommand() {
  try {
    await navigator.clipboard.writeText(activeCurlCommand.value);
    toast("调用示例已复制", "success");
  } catch {
    toast("复制失败，请手动复制示例", "error");
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

.gateway-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(320px, 0.8fr);
  gap: 18px;
  padding: 18px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-lg);
  background:
    radial-gradient(circle at 12% 12%, color-mix(in srgb, var(--color-accent) 18%, transparent), transparent 32%),
    linear-gradient(135deg, var(--ui-bg-surface), var(--ui-bg-surface-muted));
  box-shadow: var(--shadow-rest);
  overflow: hidden;
  position: relative;
}

.gateway-hero::after {
  content: "";
  position: absolute;
  right: -72px;
  bottom: -92px;
  width: 230px;
  height: 230px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-accent) 10%, transparent);
  pointer-events: none;
}

.gateway-hero.is-running {
  border-color: color-mix(in srgb, var(--color-success) 28%, var(--ui-border-default));
}

.gateway-hero__content,
.gateway-hero__console {
  position: relative;
  z-index: 1;
}

.gateway-hero__content {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
}

.gateway-hero__kicker {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  width: fit-content;
  min-height: 26px;
  padding: 0 10px;
  border: 1px solid var(--ui-border-default);
  border-radius: 999px;
  background: color-mix(in srgb, var(--ui-bg-surface) 82%, transparent);
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 700;
}

.gateway-hero__title {
  margin: 14px 0 0;
  color: var(--color-text-primary);
  font-size: clamp(24px, 3.6vw, 38px);
  line-height: 1.12;
  letter-spacing: -0.04em;
}

.gateway-hero__description {
  max-width: 680px;
  margin: 12px 0 0;
  color: var(--color-text-secondary);
  font-size: 14px;
  line-height: 1.75;
}

.gateway-hero__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
}

.gateway-hero__console {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--ui-bg-surface) 88%, transparent);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.38);
}

.console-line {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
  min-height: 32px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
}

.console-label {
  color: var(--color-text-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.console-line code {
  overflow: hidden;
  color: var(--color-text-primary);
  font-family: var(--font-mono);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gateway-hero__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 2px;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.summary-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 16px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-rest);
}

.summary-card__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 1px solid color-mix(in srgb, var(--color-accent) 20%, var(--ui-border-default));
  border-radius: var(--radius-sm);
  background: var(--color-accent-soft);
  color: var(--color-accent);
  flex-shrink: 0;
}

.summary-card__content {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
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

.summary-hint {
  overflow: hidden;
  color: var(--color-text-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.gateway-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.panel-card {
  padding: 16px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-lg);
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
  align-content: flex-start;
  padding: 12px;
  border: 1px solid var(--ui-border-subtle);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface-muted);
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
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--ui-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
}

.info-row {
  flex-direction: column;
  gap: 6px;
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

.diagnostic-step {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 999px;
  background: var(--ui-bg-surface);
  color: var(--color-text-muted);
  font-size: 12px;
  font-weight: 800;
  flex-shrink: 0;
}

.diagnostic-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 8px;
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
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface-muted);
}

.auth-aside p,
.auth-helper {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.65;
}

.auth-helper code {
  color: var(--color-text-primary);
  font-family: var(--font-mono);
  font-size: 12px;
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
  border-radius: var(--radius-md);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--ui-bg-surface-muted) 86%, transparent), var(--ui-bg-surface-muted));
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
  .gateway-hero,
  .summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .gateway-grid {
    grid-template-columns: 1fr;
  }

  .panel-card--wide {
    grid-column: auto;
  }
}

@media (max-width: 820px) {
  .gateway-hero,
  .summary-grid {
    grid-template-columns: 1fr;
  }

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
  .gateway-hero {
    padding: 14px;
  }

  .console-line {
    grid-template-columns: 1fr;
    gap: 4px;
  }

  .auth-input-shell {
    grid-template-columns: 1fr;
  }

  .gateway-hero__actions,
  .control-actions,
  .auth-actions {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
