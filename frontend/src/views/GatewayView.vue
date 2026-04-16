<template>
  <div class="gateway-view app-page">
    <UEDPageHeader
      title="网关总览"
      divided
    >
      <template #actions>
        <button class="btn btn-secondary" @click="handleRefresh" :disabled="gatewayStore.loading">
          <RefreshCw :size="14" :class="{ spinning: gatewayStore.loading }" />
          刷新状态
        </button>
      </template>
    </UEDPageHeader>

    <section class="gateway-hero" :class="{ 'is-running': gatewayStore.running }">
      <div class="gateway-hero__content">
        <div class="gateway-hero__kicker">
          <Activity :size="14" />
          本地代理网关
        </div>
        <div class="gateway-hero__meta">
          <StatusBadge :status="gatewayStore.running ? 'success' : 'error'" :label="gatewayStore.running ? '运行中' : '已停止'" />
          <span class="summary-chip">健康度 {{ providerHealthPercent }}%</span>
          <span class="summary-chip">每 10 秒自动刷新状态</span>
        </div>
      </div>

      <div class="gateway-hero__console">
        <div class="console-line">
          <span class="console-label">API Base</span>
          <code>http://{{ gatewayStore.host }}:{{ gatewayStore.port }}/v1</code>
        </div>
        <div class="console-line">
          <span class="console-label">Listen</span>
          <code>{{ gatewayStore.host }}:{{ gatewayStore.port }}</code>
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
      <div v-for="metric in summaryMetrics" :key="metric.label" class="summary-card" :class="metric.tone ? `is-${metric.tone}` : ''">
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
      <UEDPageSection class="panel-card panel-card--wide" title="访问鉴权">
        <template #actions>
          <StatusBadge
            :status="gatewayStore.gatewayConfig.authKey ? 'info' : 'neutral'"
            :label="gatewayStore.gatewayConfig.authKey ? '已启用鉴权' : '未启用鉴权'"
          />
        </template>

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
              <button class="btn btn-secondary btn-sm" type="button" @click="copyAuthKey" :disabled="!authKeyDraft">
                <Copy :size="13" />
                复制
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
      </UEDPageSection>

      <UEDPageSection class="panel-card panel-card--wide" title="接口调用示例">
        <template #actions>
          <button class="btn btn-secondary" type="button" @click="copyCurlCommand">
            <Copy :size="14" />
            复制示例
          </button>
        </template>

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
      </UEDPageSection>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useGatewayStore } from '@/stores/gateway';
import { useProviderStore } from '@/stores/provider';
import { Activity, Boxes, Cpu, Network, Play, Server, ShieldCheck, Square, RefreshCw, Copy } from 'lucide-vue-next';
import { UEDPageHeader, UEDPageSection } from '@/components/layout';
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
  if (enabledProviderCount.value === 0) return 0;
  return Math.round((healthyProviderCount.value / enabledProviderCount.value) * 100);
});

const enabledProviderCount = computed(() =>
  providerStore.providers.filter((item) => item.enabled).length
);

const healthyProviderCount = computed(() =>
  providerStore.providers.filter((item) => item.enabled && item.healthy).length
);

const configuredModelNames = computed(() => {
  const names = new Set();

  providerStore.providers
    .filter((item) => item.enabled)
    .forEach((item) => {
      const llms = Array.isArray(item.llms) ? item.llms : [];
      llms.forEach((model) => {
        const name = (model?.model || '').trim();
        if (name) names.add(name);
      });
    });

  return Array.from(names);
});

const syncedModelNames = computed(() => {
  const names = new Set();

  gatewayStore.models.forEach((model) => {
    const name = (model?.id || model?.name || '').trim();
    if (name) names.add(name);
  });

  return Array.from(names);
});

const availableModelCount = computed(() =>
  syncedModelNames.value.length > 0 ? syncedModelNames.value.length : configuredModelNames.value.length
);

const modelReadinessLabel = computed(() => {
  if (syncedModelNames.value.length > 0) return '模型清单已同步';
  if (configuredModelNames.value.length > 0) return '已配置待刷新';
  return '等待配置模型';
});

const readinessLabel = computed(() => {
  if (!gatewayStore.running) return "等待启动";
  if (enabledProviderCount.value === 0) return "需要供应商";
  if (configuredModelNames.value.length === 0) return "需要模型映射";
  if (syncedModelNames.value.length === 0) return "等待模型";
  return "可接入";
});

const gatewayAddress = computed(() => `${gatewayStore.host}:${gatewayStore.port}`);

const summaryMetrics = computed(() => [
  {
    label: "监听地址",
    value: `:${gatewayStore.port}`,
    hint: `127.0.0.1:${gatewayStore.port}`,
    icon: Network,
    tone: gatewayStore.running ? "info" : "warning",
  },
  {
    label: "供应商",
    value: `${enabledProviderCount.value} 个`,
    hint: enabledProviderCount.value > 0
      ? `${healthyProviderCount.value} 个健康，${providerStore.providers.length - enabledProviderCount.value} 个禁用`
      : providerStore.providers.length > 0
        ? '当前供应商均为禁用状态'
        : '尚未配置供应商',
    icon: Server,
    tone: enabledProviderCount.value === 0 ? "warning" : healthyProviderCount.value > 0 ? "success" : "danger",
  },
  {
    label: "可用模型",
    value: `${availableModelCount.value} 个`,
    hint: modelReadinessLabel.value,
    icon: Boxes,
    tone: syncedModelNames.value.length > 0 ? "success" : configuredModelNames.value.length > 0 ? "info" : "warning",
  },
  {
    label: "访问鉴权",
    value: gatewayStore.gatewayConfig.authKey ? "已启用" : "未启用",
    hint: gatewayStore.gatewayConfig.authKey ? "Bearer / x-api-key" : "本地免鉴权",
    icon: ShieldCheck,
    tone: gatewayStore.gatewayConfig.authKey ? "success" : "info",
  },
  {
    label: "运行状态",
    value: gatewayStore.running ? "运行中" : "已停止",
    hint: readinessLabel.value,
    icon: Cpu,
    tone: gatewayStore.running ? "success" : "danger",
  },
]);

const nextStep = computed(() => {
  if (!gatewayStore.running) {
    return {
      title: "先启动网关",
      description: "网关尚未监听本地端口，启动后客户端才能通过统一 /v1 入口接入。",
    };
  }

  if (enabledProviderCount.value === 0) {
    return {
      title: "补充供应商配置",
      description: "当前没有已启用供应商，先启用并验证至少一个上游连接。",
    };
  }

  if (configuredModelNames.value.length === 0) {
    return {
      title: "配置模型映射",
      description: "供应商已启用，但还没有可供网关暴露的模型映射，请先在供应商中补齐模型配置。",
    };
  }

  if (syncedModelNames.value.length === 0) {
    return {
      title: "刷新模型列表",
      description: "模型映射已存在，但网关尚未同步公开模型。刷新后再验证 /v1/models 返回。",
    };
  }

  if (!gatewayStore.gatewayConfig.authKey) {
    return {
      title: "建议启用访问鉴权",
      description: "当前已具备调用条件；如果网关会被其他应用或局域网设备访问，建议立即配置认证 Key。",
    };
  }

  return {
    title: "可以接入客户端",
    description: "网关、模型与鉴权均已就绪，可直接复制 API Base 和调试示例进行联调。",
  };
});

const authPreviewValue = computed(() => {
  if (!gatewayStore.gatewayConfig.authKey) {
    return "Authorization: disabled";
  }

  return `Authorization: Bearer ${gatewayStore.gatewayConfig.authKey}`;
});

const diagnosticItems = computed(() => [
  {
    ok: gatewayStore.running,
    label: gatewayStore.running ? "网关正在监听请求" : "网关尚未启动",
    description: gatewayStore.running
      ? `当前监听 127.0.0.1:${gatewayStore.port}，客户端可以接入。`
      : "启动网关后才会开放本地 /v1 兼容接口。",
  },
  {
    ok: enabledProviderCount.value > 0,
    label: enabledProviderCount.value > 0 ? "已启用供应商" : "尚未启用供应商",
    description: enabledProviderCount.value > 0
      ? `${healthyProviderCount.value} / ${enabledProviderCount.value} 个启用供应商处于健康状态。`
      : "请先在供应商管理中启用至少一个上游服务。",
  },
  {
    ok: configuredModelNames.value.length > 0,
    label: configuredModelNames.value.length > 0 ? "已配置模型映射" : "模型映射尚未配置",
    description: configuredModelNames.value.length > 0
      ? syncedModelNames.value.length > 0
        ? `当前向客户端暴露 ${availableModelCount.value} 个模型。`
        : `已配置 ${configuredModelNames.value.length} 个模型映射，等待刷新到 /v1/models。`
      : "请先在供应商中配置至少一个模型映射。",
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
  await providerStore.fetchProviders();
  await gatewayStore.refreshModels();
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

async function copyAuthKey() {
  const value = authKeyDraft.value.trim();
  if (!value) return;

  try {
    await navigator.clipboard.writeText(value);
    toast("Auth Key 已复制", "success");
  } catch {
    toast("复制失败，请手动复制 Key", "error");
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
  min-height: 0;
  flex-direction: column;
  gap: 16px;
  overflow: auto;
  overscroll-behavior: contain;
}

.gateway-view > * {
  flex-shrink: 0;
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
  align-items: center;
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

.summary-card.is-success {
  border-color: color-mix(in srgb, var(--color-success) 28%, var(--ui-border-default));
  background: color-mix(in srgb, var(--color-success) 7%, var(--ui-bg-surface));
}

.summary-card.is-success .summary-card__icon {
  border-color: color-mix(in srgb, var(--color-success) 26%, var(--ui-border-default));
  background: color-mix(in srgb, var(--color-success) 14%, var(--ui-bg-surface));
  color: var(--color-success);
}

.summary-card.is-warning {
  border-color: color-mix(in srgb, var(--color-warning) 28%, var(--ui-border-default));
  background: color-mix(in srgb, var(--color-warning) 8%, var(--ui-bg-surface));
}

.summary-card.is-warning .summary-card__icon {
  border-color: color-mix(in srgb, var(--color-warning) 28%, var(--ui-border-default));
  background: color-mix(in srgb, var(--color-warning) 14%, var(--ui-bg-surface));
  color: var(--color-warning);
}

.summary-card.is-danger {
  border-color: color-mix(in srgb, var(--color-danger) 26%, var(--ui-border-default));
  background: color-mix(in srgb, var(--color-danger) 7%, var(--ui-bg-surface));
}

.summary-card.is-danger .summary-card__icon {
  border-color: color-mix(in srgb, var(--color-danger) 24%, var(--ui-border-default));
  background: color-mix(in srgb, var(--color-danger) 12%, var(--ui-bg-surface));
  color: var(--color-danger);
}

.summary-card.is-info {
  border-color: color-mix(in srgb, var(--color-accent) 26%, var(--ui-border-default));
  background: color-mix(in srgb, var(--color-accent) 7%, var(--ui-bg-surface));
}

.summary-card.is-info .summary-card__icon {
  border-color: color-mix(in srgb, var(--color-accent) 24%, var(--ui-border-default));
  background: color-mix(in srgb, var(--color-accent) 14%, var(--ui-bg-surface));
  color: var(--color-accent);
}

.overview-strip {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.overview-strip__card {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--ui-bg-surface) 95%, transparent), var(--ui-bg-surface-muted));
  box-shadow: var(--shadow-rest);
}

.overview-strip__card--actions {
  justify-content: space-between;
}

.overview-strip__label {
  color: var(--color-text-muted);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.overview-strip__value {
  color: var(--color-text-primary);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.35;
}

.overview-strip__value--mono {
  overflow: hidden;
  font-family: var(--font-mono);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-strip__description {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.65;
}

.overview-strip__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 4px;
}

.gateway-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.panel-card {
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-lg);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-rest);
}

.panel-card--wide {
  grid-column: 1 / -1;
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
  grid-template-columns: minmax(0, 1fr) auto auto;
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

  .overview-strip {
    grid-template-columns: 1fr;
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
