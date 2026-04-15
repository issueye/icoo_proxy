<template>
  <div class="logs-view app-page">
    <PageHeader
      title="请求日志"
      description="集中查看请求入口、转发目标、错误回包和参数明细，便于排障和追踪。"
    >
      <template #actions>
        <button class="btn btn-secondary" @click="handleRefreshLogs" :disabled="gatewayStore.logsLoading">
          <RefreshCw :size="14" :class="{ spinning: gatewayStore.logsLoading }" /> 刷新日志
        </button>
      </template>
    </PageHeader>

    <div class="logs-toolbar">
      <div class="toolbar-left">
        <label class="toolbar-label">
          查询条数
          <select v-model.number="logLimit" class="toolbar-select">
            <option :value="20">20</option>
            <option :value="50">50</option>
            <option :value="100">100</option>
          </select>
        </label>

        <label class="toggle-chip">
          <input v-model="showErrorsOnly" type="checkbox">
          <span>仅看失败</span>
        </label>
      </div>

      <div class="toolbar-summary">
        <span class="toolbar-chip">{{ filteredLogs.length }} 条</span>
        <span class="toolbar-chip">{{ errorCount }} 失败</span>
        <span class="toolbar-chip">网关 {{ gatewayStore.running ? '运行中' : '未启动' }}</span>
        <span class="toolbar-chip">端口 {{ gatewayStore.port }}</span>
      </div>
    </div>

    <div v-if="filteredLogs.length === 0" class="empty-state">
      {{ gatewayStore.logsLoading ? '正在加载请求日志...' : '暂时还没有请求记录' }}
    </div>

    <div v-else class="request-log-list">
      <div
        v-for="log in filteredLogs"
        :key="log.id"
        class="request-log-card"
        :class="{ 'request-log-card--error': log.statusCode >= 400 }"
      >
        <div class="request-log-top">
          <div class="request-log-main">
            <StatusBadge
              :status="log.statusCode >= 500 ? 'error' : log.statusCode >= 400 ? 'warning' : 'success'"
              :label="`${log.statusCode}`"
            />
            <div class="request-log-heading">
              <div class="request-log-model-row">
                <span class="request-log-model">{{ log.model || '未识别模型' }}</span>
                <span v-if="log.targetModel && log.targetModel !== log.model" class="request-log-target">
                  → {{ log.targetModel }}
                </span>
              </div>
              <div class="request-log-subline">
                <span>{{ log.method }} {{ log.path }}</span>
                <span v-if="log.providerName">· {{ log.providerName }}</span>
                <span v-if="log.providerType">· {{ log.providerType }}</span>
                <span v-if="log.endpointMode">· {{ log.endpointMode }}</span>
                <span v-if="log.clientIP">· {{ log.clientIP }}</span>
              </div>
            </div>
          </div>

          <div class="request-log-meta">
            <span>{{ formatTimestamp(log.createdAt) }}</span>
            <span>{{ log.durationMs }}ms</span>
          </div>
        </div>

        <div class="request-log-tags">
          <span class="request-log-tag request-log-tag--mode">{{ getRequestMode(log) }}</span>
          <span v-if="log.streaming" class="request-log-tag request-log-tag--accent">stream</span>
          <span class="request-log-tag">{{ log.statusCode >= 400 ? '失败' : '成功' }}</span>
          <span v-if="log.targetModel && log.targetModel !== log.model" class="request-log-tag">
            已改写模型
          </span>
        </div>

        <div v-if="log.errorMessage" class="request-log-error">
          {{ log.errorMessage }}
        </div>

        <div v-if="log.upstreamBase || log.upstreamPath" class="request-log-upstream">
          <span class="request-log-upstream-label">上游目标</span>
          <code>{{ [log.upstreamBase, log.upstreamPath].filter(Boolean).join('') }}</code>
        </div>

        <details v-if="log.requestPayload" class="request-log-body">
          <summary>查看请求参数</summary>
          <pre>{{ formatPayload(log.requestPayload) }}</pre>
        </details>

        <details v-if="log.responseHeaders" class="request-log-body">
          <summary>查看响应头</summary>
          <pre>{{ formatPayload(log.responseHeaders) }}</pre>
        </details>

        <details v-if="log.responsePayload" class="request-log-body">
          <summary>查看响应数据</summary>
          <pre>{{ formatPayload(log.responsePayload) }}</pre>
        </details>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue';
import { RefreshCw } from 'lucide-vue-next';
import { useGatewayStore } from '@/stores/gateway';
import PageHeader from '@/components/layout/PageHeader.vue';
import StatusBadge from '@/components/ui/StatusBadge.vue';

const gatewayStore = useGatewayStore();
const showErrorsOnly = ref(false);
const logLimit = ref(50);

const filteredLogs = computed(() => {
  if (!showErrorsOnly.value) return gatewayStore.requestLogs;
  return gatewayStore.requestLogs.filter((item) => item.statusCode >= 400);
});

const errorCount = computed(() =>
  gatewayStore.requestLogs.filter((item) => item.statusCode >= 400).length
);

async function handleRefreshLogs() {
  await gatewayStore.fetchRequestLogs(logLimit.value);
}

function formatTimestamp(value) {
  if (!value) return '--';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function formatPayload(value) {
  if (!value) return '';
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

function getRequestMode(log) {
  if (log?.path?.includes('/v1/responses')) return 'responses';
  if (log?.path?.includes('/v1/chat/completions')) return 'chat.completions';
  return 'gateway';
}

onMounted(async () => {
  await gatewayStore.fetchStatus();
  await handleRefreshLogs();
});
</script>

<style scoped>
.logs-toolbar,
.request-log-card,
.empty-state {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-secondary);
}

.logs-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
  padding: 14px 16px;
}

.toolbar-left,
.toolbar-summary {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.empty-state {
  padding: 28px;
  text-align: center;
  color: var(--color-text-muted);
}

.request-log-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.request-log-card {
  padding: 16px;
}

.request-log-card--error {
  border-color: color-mix(in srgb, #ef4444 40%, var(--color-border));
  background: color-mix(in srgb, #ef4444 4%, var(--color-bg-secondary));
}

.request-log-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.request-log-main {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.request-log-heading {
  min-width: 0;
}

.request-log-model-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.request-log-model {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.request-log-target {
  font-size: 12px;
  color: var(--color-text-muted);
}

.request-log-subline {
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-text-muted);
}

.request-log-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.request-log-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.request-log-tag {
  padding: 4px 8px;
  border-radius: 999px;
  background: var(--color-bg-tertiary, var(--color-bg-primary));
  color: var(--color-text-secondary);
  font-size: 11px;
}

.request-log-tag--accent {
  color: var(--color-accent);
}

.request-log-tag--mode {
  background: color-mix(in srgb, var(--color-accent) 10%, var(--color-bg-tertiary, var(--color-bg-primary)));
  color: var(--color-accent);
  font-weight: 700;
}

.request-log-error {
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  background: color-mix(in srgb, #ef4444 8%, var(--color-bg-tertiary, var(--color-bg-primary)));
  color: #b91c1c;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.request-log-upstream {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.request-log-upstream-label {
  color: var(--color-text-muted);
}

.request-log-upstream code {
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  background: var(--color-bg-tertiary, var(--color-bg-primary));
  color: var(--color-text-primary);
}

.request-log-body {
  margin-top: 12px;
  border-top: 1px solid var(--color-border);
  padding-top: 12px;
}

.request-log-body summary {
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.request-log-body pre {
  margin: 10px 0 0;
  padding: 12px;
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary, var(--color-bg-primary));
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

@media (max-width: 720px) {
  .logs-view {
    padding: 16px;
  }

  .logs-toolbar,
  .request-log-top {
    flex-direction: column;
    align-items: stretch;
  }

  .request-log-meta {
    white-space: normal;
  }
}
</style>
