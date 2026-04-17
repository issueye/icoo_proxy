<template>
  <div class="logs-view app-page">
    <UEDPageHeader title="请求日志" description="集中查看最近请求、上游路由结果与响应状态，便于联调和排障。" divided>
      <template #actions>
        <button class="btn btn-secondary" @click="handleRefreshLogs" :disabled="gatewayStore.logsLoading">
          <RefreshCw :size="14" :class="{ spinning: gatewayStore.logsLoading }" />
          刷新日志
        </button>
      </template>
    </UEDPageHeader>

    <section class="toolbar-surface logs-toolbar">
      <div class="toolbar-group">
        <div class="toolbar-field">
          <label class="toolbar-label">查询条数</label>
          <Select v-model="logLimit" :options="logLimitOptions" class="toolbar-select" @change="handleRefreshLogs" />
        </div>
        <button
          class="btn btn-secondary logs-error-toggle"
          :class="{ 'is-active': showErrorsOnly }"
          type="button"
          @click="showErrorsOnly = !showErrorsOnly"
        >
          仅看错误
        </button>
      </div>
      <div class="toolbar-summary logs-summary">
        <span class="toolbar-chip">总计 {{ gatewayStore.requestLogs.length }} 条</span>
        <span class="toolbar-chip">错误 {{ errorCount }} 条</span>
        <span class="toolbar-chip">当前显示 {{ filteredLogs.length }} 条</span>
      </div>
    </section>

    <div class="logs-workspace">
      <section class="logs-table-panel">
        <UEDTable
          :columns="columns"
          :data="filteredLogs"
          :loading="gatewayStore.logsLoading"
          row-key="id"
          clickable
          empty-title="暂无数据"
          @row-click="handleSelectLog"
        >
          <template #cell-createdAt="{ value }">
            <span class="table-mono">{{ formatTimestamp(value) }}</span>
          </template>

          <template #cell-path="{ row }">
            <div class="path-cell">
              <span class="path-method">{{ row.method }}</span>
              <span class="path-value">{{ row.path }}</span>
            </div>
          </template>

          <template #cell-providerName="{ row }">
            <div class="provider-cell">
              <span class="provider-primary">{{ row.providerName || '未知供应商' }}</span>
              <span class="provider-secondary">{{ row.providerType || '未标识类型' }}</span>
            </div>
          </template>

          <template #cell-model="{ row }">
            <div class="provider-cell">
              <span class="provider-primary">{{ row.model || '未识别模型' }}</span>
              <span v-if="row.targetModel && row.targetModel !== row.model" class="provider-secondary">
                → {{ row.targetModel }}
              </span>
            </div>
          </template>

          <template #cell-statusCode="{ value }">
            <StatusBadge :status="value >= 500 ? 'error' : value >= 400 ? 'warning' : 'success'"
              :label="String(value)" />
          </template>

          <template #cell-durationMs="{ value }">
            <span class="table-mono">{{ value }} ms</span>
          </template>
        </UEDTable>
      </section>

      <aside class="logs-detail-panel">
        <div v-if="!selectedLog" class="empty-state logs-detail-empty">
          <div class="empty-title">请选择一条日志</div>
          <p>点击左侧表格中的任意一行，查看请求参数、上游目标和响应详情。</p>
        </div>

        <template v-else>
          <div class="panel-head">
            <div>
              <h3 class="section-title">日志详情</h3>
              <p class="panel-description">{{ selectedLog.method }} {{ selectedLog.path }}</p>
            </div>
            <StatusBadge
              :status="selectedLog.statusCode >= 500 ? 'error' : selectedLog.statusCode >= 400 ? 'warning' : 'success'"
              :label="`${selectedLog.statusCode}`" />
          </div>

          <div class="detail-summary">
            <div class="detail-item">
              <span class="detail-label">时间</span>
              <span class="detail-value">{{ formatTimestamp(selectedLog.createdAt) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">供应商</span>
              <span class="detail-value">{{ selectedLog.providerName || '未知' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">模型</span>
              <span class="detail-value">{{ selectedLog.model || '未识别模型' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">延迟</span>
              <span class="detail-value">{{ selectedLog.durationMs }} ms</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">来源 IP</span>
              <span class="detail-value">{{ selectedLog.clientIP || '--' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">模式</span>
              <span class="detail-value">{{ getRequestMode(selectedLog) }}</span>
            </div>
          </div>

          <div v-if="selectedLog.errorMessage" class="detail-alert">
            {{ selectedLog.errorMessage }}
          </div>

          <div v-if="selectedLog.upstreamBase || selectedLog.upstreamPath" class="settings-note">
            <div class="field-label">上游目标</div>
            <div class="settings-help table-mono">{{ [selectedLog.upstreamBase,
            selectedLog.upstreamPath].filter(Boolean).join('') }}</div>
          </div>

          <details v-if="selectedLog.requestPayload" class="detail-section" open>
            <summary>请求参数</summary>
            <pre>{{ formatPayload(selectedLog.requestPayload) }}</pre>
          </details>

          <details v-if="selectedLog.responseHeaders" class="detail-section">
            <summary>响应头</summary>
            <pre>{{ formatPayload(selectedLog.responseHeaders) }}</pre>
          </details>

          <details v-if="selectedLog.responsePayload" class="detail-section">
            <summary>响应数据</summary>
            <pre>{{ formatPayload(selectedLog.responsePayload) }}</pre>
          </details>
        </template>
      </aside>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { RefreshCw } from 'lucide-vue-next';
import { useGatewayStore } from '@/stores/gateway';
import UEDPageHeader from '@/components/layout/UEDPageHeader.vue';
import StatusBadge from '@/components/ui/StatusBadge.vue';
import UEDTable from '@/components/layout/UEDTable.vue';
import Select from '@/components/ui/Select.vue';

const gatewayStore = useGatewayStore();
const showErrorsOnly = ref(false);
const logLimit = ref(50);
const selectedLogId = ref(null);
const logLimitOptions = [
  { label: '20', value: 20 },
  { label: '50', value: 50 },
  { label: '100', value: 100 },
];

const columns = [
  { key: 'createdAt', title: '时间', class: 'w-[148px]' },
  { key: 'path', title: '请求' },
  { key: 'providerName', title: '供应商' },
  { key: 'model', title: '模型' },
  { key: 'statusCode', title: '状态', class: 'w-[90px]' },
  { key: 'durationMs', title: '延迟', class: 'w-[90px]' },
];

const filteredLogs = computed(() => {
  if (!showErrorsOnly.value) return gatewayStore.requestLogs;
  return gatewayStore.requestLogs.filter((item) => item.statusCode >= 400);
});

const selectedLog = computed(() =>
  filteredLogs.value.find((item) => item.id === selectedLogId.value) || filteredLogs.value[0] || null
);

const errorCount = computed(() =>
  gatewayStore.requestLogs.filter((item) => item.statusCode >= 400).length
);

async function handleRefreshLogs() {
  await gatewayStore.fetchRequestLogs(logLimit.value);
  if (!selectedLogId.value && gatewayStore.requestLogs.length > 0) {
    selectedLogId.value = gatewayStore.requestLogs[0].id;
  }
}

function handleSelectLog(log) {
  selectedLogId.value = log.id;
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

watch(filteredLogs, (logs) => {
  if (logs.length === 0) {
    selectedLogId.value = null;
    return;
  }
  if (!logs.some((item) => item.id === selectedLogId.value)) {
    selectedLogId.value = logs[0].id;
  }
});

onMounted(async () => {
  await gatewayStore.fetchStatus();
  await handleRefreshLogs();
});
</script>

<style scoped>
.logs-view {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  gap: 16px;
}

.logs-toolbar {
  align-items: flex-end;
}

.toolbar-field {
  min-width: 120px;
}

.toolbar-label {
  display: block;
  margin-bottom: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.logs-error-toggle.is-active {
  border-color: color-mix(in srgb, var(--color-error) 24%, var(--ui-border-default));
  background: color-mix(in srgb, var(--color-error) 8%, var(--ui-bg-surface));
  color: var(--color-error);
}

.logs-summary {
  justify-content: flex-end;
}

.logs-workspace {
  display: grid;
  grid-template-columns: minmax(0, 1.35fr) 380px;
  gap: 16px;
  min-height: 0;
  flex: 1;
}

.logs-table-panel,
.logs-detail-panel {
  min-height: 0;
}

.logs-detail-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-rest);
  overflow-y: auto;
}

.logs-detail-empty {
  min-height: 260px;
}

.empty-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.empty-state p {
  margin: 0;
  font-size: 12px;
  line-height: 1.55;
}

.table-mono {
  font-family: var(--font-mono);
  font-size: 12px;
}

.path-cell,
.provider-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.path-method,
.provider-primary,
.detail-value {
  color: var(--color-text-primary);
  font-weight: 600;
}

.path-value,
.provider-secondary,
.detail-label {
  font-size: 12px;
  color: var(--color-text-muted);
}

.detail-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  border: 1px solid var(--ui-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
}

.detail-alert {
  padding: 12px;
  border: 1px solid color-mix(in srgb, var(--color-error) 20%, var(--ui-border-default));
  border-radius: var(--radius-sm);
  background: var(--ui-danger-soft);
  color: var(--color-error);
  font-size: 12px;
  line-height: 1.55;
  white-space: pre-wrap;
}

.detail-section {
  border-top: 1px solid var(--ui-border-subtle);
  padding-top: 12px;
}

.detail-section summary {
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.detail-section pre {
  margin: 10px 0 0;
  padding: 12px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
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
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1080px) {
  .logs-workspace {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .logs-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .detail-summary {
    grid-template-columns: 1fr;
  }
}
</style>
