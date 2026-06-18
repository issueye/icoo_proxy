<template>
  <section class="page-section traffic-page">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton
          variant="primary"
          :loading="store.refreshing"
          :disabled="store.refreshing || store.clearing"
          @click="store.refresh"
        >
          {{ store.refreshing ? "刷新中..." : "刷新流量" }}
        </UButton>
        <UButton
          variant="error"
          :loading="store.clearing"
          :disabled="store.refreshing || store.clearing || !store.totalRequests"
          @click="openClearConfirm"
        >
          {{ store.clearing ? "清空中..." : "清空请求" }}
        </UButton>
        <USwitch v-model="store.autoRefresh" label="自动刷新" />
      </div>
    </Teleport>

    <div class="traffic-stats">
      <StatCard icon="activity" label="最近请求数" :value="String(store.totalRequests)" tone="info" />
      <StatCard icon="check" label="成功请求" :value="String(store.successCount)" tone="success" />
      <StatCard icon="alert" label="错误请求" :value="String(store.errorCount)" tone="danger" />
      <StatCard icon="timer" label="平均耗时" :value="`${store.averageLatency} ms`" />
      <StatCard icon="layers" label="总输入 Token" :value="formatTokenCount(store.tokenStats.input_tokens)"
        tone="primary" />
      <StatCard icon="server" label="总输出 Token" :value="formatTokenCount(store.tokenStats.output_tokens)"
        tone="warning" />
      <StatCard icon="key" label="总 Token" :value="formatTokenCount(store.tokenStats.total_tokens)" tone="info" />
    </div>

    <div class="traffic-layout">
      <UTable :columns="tableColumns" :rows="store.requests" row-key="request_id" fixed fixed-field="freeze" stripe
        size="sm" table-class="traffic-table" max-height="100%" min-width="1680px" pagination
        pagination-mode="server" :page="store.page" :page-size="store.pageSize" :total="store.total"
        :page-size-options="[8, 20, 50]" @page-change="store.changePage">
        <template #empty>
          当前没有匹配的请求记录。
        </template>
        <template #query>
          <div class="traffic-query-form">
            <USelect class="traffic-query-form__field traffic-query-form__field--protocol" label="协议筛选" hide-label
              :model-value="store.filter" :options="normalizedProtocolOptions" @update:model-value="store.setFilter" />
          </div>
        </template>
        <template #cell-requestId="{ row }">
          <p class="font-medium text-strong table-cell-wrap">{{ row.request_id }}</p>
        </template>
        <template #cell-endpoint="{ row }">
          <UTag code size="xs">{{ row.endpoint || "-" }}</UTag>
        </template>
        <template #cell-requestInfo="{ row }">
          <p class="text-sm text-strong table-cell-wrap">{{ row.method || "-" }} · {{ row.client_ip || "-" }}</p>
          <p class="mt-0.5 table-meta table-cell-wrap">{{ row.user_agent || "无 User-Agent" }}</p>
        </template>
        <template #cell-route="{ row }">
          <p class="text-sm text-secondary table-cell-wrap">{{ row.downstream }}</p>
          <p class="mt-0.5 table-meta table-cell-wrap">{{ row.upstream || "-" }}</p>
          <p v-if="routeHint(row)" class="mt-0.5 table-meta table-cell-wrap">{{ routeHint(row) }}</p>
        </template>
        <template #cell-model="{ row }">
          <p class="text-sm text-strong table-cell-wrap">{{ row.requested_model || "-" }}</p>
          <p class="mt-0.5 table-meta table-cell-wrap">路由到 {{ row.model || "-" }}</p>
        </template>
        <template #cell-requestBody="{ row }">
          <p class="text-sm text-secondary table-cell-wrap">{{ requestBodyPreview(row) }}</p>
          <p class="mt-0.5 table-meta">
            {{ formatBytes(row.request_body_bytes) }}{{ row.request_body_truncated ? "，已截断" : "" }}
          </p>
        </template>
        <template #cell-tokens="{ row }">
          <div class="token-cell">
            <div class="token-cell__row">
              <span class="token-cell__label">入</span>
              <span class="token-cell__value">{{ formatTokenCount(row.input_tokens) }}</span>
            </div>
            <div class="token-cell__row">
              <span class="token-cell__label">出</span>
              <span class="token-cell__value">{{ formatTokenCount(row.output_tokens) }}</span>
            </div>
            <div class="token-cell__row token-cell__row--total">
              <span class="token-cell__label">总</span>
              <span class="token-cell__value">{{ formatTokenCount(row.total_tokens) }}</span>
            </div>
          </div>
        </template>
        <template #cell-status="{ row }">
          <UTag :variant="row.status_code >= 400 ? 'error' : 'success'" size="xs">
            {{ row.status_code || "-" }}
          </UTag>
        </template>
        <template #cell-duration="{ row }">
          <UTag size="xs">{{ row.duration_ms }} ms</UTag>
        </template>
        <template #cell-createdAt="{ row }">
          <span class="table-meta">{{ formatDateTime(row.created_at) }}</span>
        </template>
        <template #cell-error="{ row }">
          <p v-if="row.error" class="text-sm text-error table-cell-wrap">{{ row.error }}</p>
          <span v-else class="table-meta">无</span>
        </template>
      </UTable>
    </div>

    <UConfirmDialog v-model:open="confirmState.open" title="确认清空请求记录" message="确定要清空流量监控中的所有请求信息吗？"
      description="清空后当前保存的请求记录与统计数据将被移除，且无法恢复。" confirm-text="确认清空" cancel-text="取消"
      :loading="store.clearing" danger @confirm="confirmClear" />
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, watch } from "vue";
import { useTrafficStore } from "../stores/traffic";
import { useStoreError } from "../composables/useStoreError";

import StatCard from "../components/StatCard.vue";
import UButton from "../components/ued/UButton.vue";
import UConfirmDialog from "../components/ued/UConfirmDialog.vue";
import USelect from "../components/ued/USelect.vue";
import USwitch from "../components/ued/USwitch.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";

const store = useTrafficStore();
useStoreError(store);
const confirmState = reactive({
  open: false,
});
let refreshTimer = null;
const tableColumns = [
  { key: "requestId", title: "请求 ID", width: 180, freeze: "left" },
  { key: "endpoint", title: "端点", width: 180 },
  { key: "requestInfo", title: "请求信息", width: 240 },
  { key: "route", title: "下游 / 上游", width: 220 },
  { key: "model", title: "模型", width: 190 },
  { key: "requestBody", title: "请求体", width: 240 },
  { key: "tokens", title: "Tokens", width: 128 },
  { key: "createdAt", title: "创建时间", width: 172 },
  { key: "error", title: "错误信息", width: 216 },
  { key: "duration", title: "耗时", width: 90, align: "center", freeze: "right" },
  { key: "status", title: "状态码", width: 92, align: "center", freeze: "right" },
];

const normalizedProtocolOptions = computed(() =>
  store.protocolOptions.map((option) => {
    const value = typeof option === "string" || typeof option === "number"
      ? String(option)
      : String(option?.value ?? "");

    const rawLabel = typeof option === "string" || typeof option === "number"
      ? String(option)
      : option?.label || value;

    return {
      value,
      label: rawLabel === "all" ? "全部协议" : rawLabel,
    };
  }),
);

function stopTimer() {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
}

function startTimer() {
  stopTimer();
  if (!store.autoRefresh) {
    return;
  }
  refreshTimer = setInterval(() => {
    store.refresh();
  }, 6000);
}

function formatDateTime(value) {
  if (!value) {
    return "暂无";
  }
  return new Date(value).toLocaleString();
}

function formatTokenCount(value) {
  const amount = Number(value || 0);
  return new Intl.NumberFormat("zh-CN").format(amount);
}

function formatBytes(value) {
  const bytes = Number(value || 0);
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  return `${(bytes / 1024).toFixed(1)} KB`;
}

function requestBodyPreview(row) {
  if (row.request_body) {
    return row.request_body;
  }
  if (row.request_body_bytes > 0) {
    return "请求体未记录，需在项目设置中开启记录 body";
  }
  return "无请求体";
}

function routeHint(row) {
  if (row.matched_rule_name) {
    return `命中规则：${row.matched_rule_name}`;
  }
  if (row.route_source === "direct") {
    return `直连路由：${row.route_name || row.model || "-"}`;
  }
  if (row.route_name) {
    return `路由：${row.route_name}`;
  }
  return "";
}

function openClearConfirm() {
  confirmState.open = true;
}

async function confirmClear() {
  await store.clear();
  if (!store.error) {
    confirmState.open = false;
    message.success("流量请求记录已清空。");
  }
}

watch(
  () => store.autoRefresh,
  () => {
    startTimer();
  },
);

onMounted(() => {
  store.load();
  startTimer();
});

onBeforeUnmount(() => {
  stopTimer();
});
</script>

<style scoped>
.traffic-page {
  display: flex;
  height: 100%;
  flex-direction: column;
}

.traffic-stats {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.traffic-layout {
  display: flex;
  flex: 1;
  min-height: 0;
  width: 100%;
}

.traffic-panel {
  min-height: 0;
}

.traffic-header-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 22px;
  padding: 0 8px;
  border-radius: var(--ued-radius-pill);
  font-size: var(--ued-font-size-sm);
  font-weight: 600;
  white-space: nowrap;
}

.traffic-header-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-left: auto;
  justify-content: flex-end;
  min-width: 0;
}

.traffic-header-badge {
  border: 1px solid color-mix(in srgb, var(--ued-color-primary) 30%, transparent);
  background: var(--ued-color-primary-soft);
  color: var(--ued-color-primary);
}

.traffic-header-note {
  font-size: var(--ued-font-size-sm);
  color: var(--ued-color-text-muted);
  white-space: nowrap;
}

.traffic-panel--table {
  display: flex;
  flex: 1 1 auto;
  width: 100%;
  min-width: 0;
  min-height: 460px;
  flex-direction: column;
}

.traffic-query-form {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}

.traffic-query-form__field {
  min-width: 0;
}

.traffic-query-form__field--protocol {
  width: 170px;
}

.traffic-query-form__field--protocol :deep(.ued-field) {
  display: inline-flex;
  align-items: center;
}

.traffic-query-form__field--protocol :deep(.ued-select__control) {
  min-width: 150px;
}

.traffic-query-form__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-left: auto;
}

.traffic-query-form__chip {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border: 1px solid color-mix(in srgb, var(--ued-color-primary) 30%, transparent);
  border-radius: var(--ued-radius-pill);
  background: var(--ued-color-primary-soft);
  color: var(--ued-color-primary);
  font-size: var(--ued-font-size-sm);
  font-weight: 600;
  white-space: nowrap;
}

/* Flat token cell — no gradient, hairline border + muted surface. */
.token-cell {
  display: grid;
  gap: 3px;
  min-width: 92px;
  padding: 5px 7px;
  border: 1px solid var(--ued-color-border);
  border-radius: var(--ued-radius-md);
  background: var(--ued-color-muted);
}

.token-cell__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: var(--ued-font-size-sm);
  line-height: 1.2;
}

.token-cell__row--total {
  padding-top: 3px;
  border-top: 1px solid var(--ued-color-divider);
}

.token-cell__label {
  color: var(--ued-color-text-muted);
}

.token-cell__value {
  font-weight: 600;
  color: var(--ued-color-text);
}

@media (max-width: 1180px) {
  .traffic-stats {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 960px) {
  .traffic-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .traffic-header-note {
    display: none;
  }
}

@media (max-width: 760px) {
  .traffic-page {
    min-height: auto;
  }

  .traffic-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .traffic-header-meta {
    width: 100%;
    margin-left: 0;
    justify-content: flex-start;
  }

  .traffic-query-form {
    align-items: stretch;
  }

  .traffic-query-form__field--protocol,
  .traffic-query-form__meta {
    width: 100%;
    margin-left: 0;
  }
}
</style>
