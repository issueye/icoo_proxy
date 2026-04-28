<template>
  <section class="page-section traffic-page">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <button class="btn btn-primary" :class="{ 'is-loading': store.refreshing }" :disabled="store.refreshing"
          @click="store.refresh">
          <span v-if="store.refreshing" class="btn__spinner" />
          {{ store.refreshing ? "刷新中..." : "刷新流量" }}
        </button>
        <label class="field-toggle rounded-md">
          <input :checked="store.autoRefresh" type="checkbox" class="field-checkbox"
            @change="store.toggleAutoRefresh" />
          自动刷新
        </label>
      </div>
    </Teleport>

    <div v-if="store.error" class="notice-error">
      {{ store.error }}
    </div>

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
      <UTable :columns="tableColumns" :rows="store.requests" row-key="request_id" fixed stripe size="small"
        table-class="traffic-table" max-height="100%" min-width="1240px" pagination pagination-mode="server"
        :page="store.page" :page-size="store.pageSize" :total="store.total" :page-size-options="[8, 20, 50]"
        @page-change="store.changePage">
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
          <p class="font-medium text-[#262626] table-cell-wrap">{{ row.request_id }}</p>
        </template>
        <template #cell-route="{ row }">
          <p class="text-sm text-[#595959] table-cell-wrap">{{ row.downstream }}</p>
          <p class="mt-0.5 table-meta table-cell-wrap">{{ row.upstream || "-" }}</p>
        </template>
        <template #cell-model="{ row }">
          <UTag code size="xs">{{ row.model || "-" }}</UTag>
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
          <p v-if="row.error" class="text-sm text-[#cf1322] table-cell-wrap">{{ row.error }}</p>
          <span v-else class="table-meta">无</span>
        </template>
      </UTable>
    </div>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, watch } from "vue";
import { useTrafficStore } from "../stores/traffic";

import StatCard from "../components/StatCard.vue";
import USelect from "../components/ued/USelect.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";

const store = useTrafficStore();
let refreshTimer = null;
const tableColumns = [
  { key: "requestId", title: "请求 ID", width: 180 },
  { key: "route", title: "下游 / 上游", width: 220 },
  { key: "model", title: "模型", width: 140 },
  { key: "tokens", title: "Tokens", width: 128 },
  { key: "status", title: "状态码", width: 92, align: "center" },
  { key: "duration", title: "耗时", width: 92, align: "center" },
  { key: "createdAt", title: "创建时间", width: 172 },
  { key: "error", title: "错误信息", width: 216 },
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
  min-height: calc(100vh - 142px);
  flex-direction: column;
}

.traffic-stats {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
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
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 12px;
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
  border: 1px solid #dbe7ff;
  background: #f8fbff;
  color: #2448bd;
}

.traffic-header-note {
  font-size: 12px;
  color: #6b7a90;
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
  gap: 12px;
  flex-wrap: wrap;
}

.traffic-query-form__field {
  min-width: 0;
}

.traffic-query-form__field--protocol {
  width: 180px;
}

.traffic-query-form__field--protocol :deep(.ued-field) {
  display: inline-flex;
  align-items: center;
}

.traffic-query-form__field--protocol :deep(.ued-select__control) {
  min-width: 160px;
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
  height: 24px;
  padding: 0 10px;
  border: 1px solid #dbe7ff;
  border-radius: 999px;
  background: #f8fbff;
  color: #2448bd;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.traffic-stats :deep(.stat-card) {
  min-height: 84px;
  gap: 10px;
  padding: 12px;
  border-radius: 12px;
}

.traffic-stats :deep(.stat-card__icon) {
  width: 40px;
  height: 40px;
  border-radius: 10px;
}

.traffic-stats :deep(.stat-card__label) {
  line-height: 1.35;
}

.traffic-stats :deep(.stat-card__value) {
  margin-top: 6px;
  font-size: 18px;
  line-height: 1.1;
  word-break: keep-all;
}

.token-cell {
  display: grid;
  gap: 4px;
  min-width: 92px;
  padding: 6px 8px;
  border: 1px solid #e6ecfb;
  border-radius: 10px;
  background: linear-gradient(180deg, #fbfdff 0%, #f4f8ff 100%);
}

.token-cell__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  line-height: 1.2;
}

.token-cell__row--total {
  padding-top: 4px;
  border-top: 1px dashed #d7e3ff;
}

.token-cell__label {
  color: #8c8c8c;
}

.token-cell__value {
  font-weight: 600;
  color: #1f1f1f;
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
