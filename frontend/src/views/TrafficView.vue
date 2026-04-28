<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <button
          class="btn btn-primary"
          :class="{ 'is-loading': store.refreshing }"
          :disabled="store.refreshing"
          @click="store.refresh"
        >
          <span v-if="store.refreshing" class="btn__spinner" />
          {{ store.refreshing ? "刷新中..." : "刷新流量" }}
        </button>
        <label class="field-toggle rounded-md">
          <input :checked="store.autoRefresh" type="checkbox" class="field-checkbox" @change="store.toggleAutoRefresh" />
          自动刷新
        </label>
      </div>
    </Teleport>

    <div v-if="store.error" class="notice-error">
      {{ store.error }}
    </div>

    <div class="section-grid grid-cols-2 lg:grid-cols-4 xl:grid-cols-7">
      <StatCard icon="activity" label="最近请求数" :value="String(store.requests.length)" tone="info" />
      <StatCard icon="check" label="成功请求" :value="String(store.successCount)" tone="success" />
      <StatCard icon="alert" label="错误请求" :value="String(store.errorCount)" tone="danger" />
      <StatCard icon="timer" label="平均耗时" :value="`${store.averageLatency} ms`" />
      <StatCard icon="layers" label="总输入 Token" :value="formatTokenCount(store.tokenStats.input_tokens)" tone="primary" />
      <StatCard icon="server" label="总输出 Token" :value="formatTokenCount(store.tokenStats.output_tokens)" tone="warning" />
      <StatCard icon="key" label="总 Token" :value="formatTokenCount(store.tokenStats.total_tokens)" tone="info" />
    </div>

    <div class="section-grid lg:grid-cols-[280px_minmax(0,1fr)]">
      <PanelBlock title="筛选条件">
        <div class="space-y-3">
          <USelect
            label="协议"
            :model-value="store.filter"
            :options="store.protocolOptions"
            @update:model-value="store.setFilter"
          />

          <div class="divide-y divide-[#f0f0f0]">
            <div class="flex items-center justify-between gap-3 py-2">
              <p class="table-meta">最近刷新</p>
              <p class="text-right text-sm font-medium text-[#262626]">{{ formatDateTime(store.lastUpdatedAt) }}</p>
            </div>
            <div class="flex items-center justify-between gap-3 py-2">
              <p class="table-meta">筛选结果</p>
              <p class="text-right text-sm font-medium text-[#262626]">{{ store.filteredRequests.length }} 条</p>
            </div>
          </div>
        </div>
      </PanelBlock>

      <PanelBlock title="最近请求明细">
        <div v-if="store.loading" class="empty-state">
          正在加载流量数据...
        </div>
        <div v-else-if="store.filteredRequests.length === 0" class="empty-state">
          当前没有匹配的请求记录。
        </div>
        <UTable
          v-else
          :columns="tableColumns"
          :rows="store.filteredRequests"
          row-key="request_id"
          fixed
          table-class="traffic-table"
          max-height="calc(100vh - 310px)"
        >
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
      </PanelBlock>
    </div>
  </section>
</template>

<script setup>
import { onBeforeUnmount, onMounted, watch } from "vue";
import { useTrafficStore } from "../stores/traffic";

import PanelBlock from "../components/PanelBlock.vue";
import StatCard from "../components/StatCard.vue";
import USelect from "../components/ued/USelect.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";

const store = useTrafficStore();
let refreshTimer = null;
const tableColumns = [
  { key: "requestId", title: "请求 ID", width: "16%" },
  { key: "route", title: "下游 / 上游", width: "14%" },
  { key: "model", title: "模型", width: "14%" },
  { key: "tokens", title: "Tokens", width: "15%" },
  { key: "status", title: "状态码", width: "9%" },
  { key: "duration", title: "耗时", width: "9%" },
  { key: "createdAt", title: "创建时间", width: "13%" },
  { key: "error", title: "错误信息", width: "10%" },
];

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
</style>
