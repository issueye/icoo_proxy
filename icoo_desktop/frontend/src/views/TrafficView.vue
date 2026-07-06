<template>
  <section class="page-section">
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
          variant="secondary"
          class="ued-button--danger-subtle"
          :loading="store.clearing"
          :disabled="store.refreshing || store.clearing || !store.totalRequests"
          @click="openClearConfirm"
        >
          {{ store.clearing ? "清空中..." : "清空请求" }}
        </UButton>
        <USwitch v-model="store.autoRefresh" label="自动刷新" />
      </div>
    </Teleport>

    <div class="stat-grid stat-grid--4">
      <StatCard icon="activity" label="最近请求数" :value="String(store.totalRequests)" tone="info" />
      <StatCard icon="alert" label="错误请求" :value="String(store.errorCount)" tone="danger" />
      <StatCard icon="timer" label="平均耗时" :value="`${store.averageLatency} ms`" />
      <StatCard icon="key" label="总 Token" :value="formatTokenCount(store.tokenStats.total_tokens)" tone="info" />
    </div>

    <div class="stat-grid stat-grid--3">
      <StatCard icon="check" label="成功请求" :value="String(store.successCount)" tone="success" />
      <StatCard icon="layers" label="总输入 Token" :value="formatTokenCount(store.tokenStats.input_tokens)"
        tone="primary" />
      <StatCard icon="server" label="总输出 Token" :value="formatTokenCount(store.tokenStats.output_tokens)"
        tone="warning" />
    </div>

    <UTable
      :columns="tableColumns"
      :rows="store.requests"
      row-key="request_id"
      fixed
      fixed-field="freeze"
      stripe
      size="sm"
      max-height="100%"
      min-width="1680px"
      pagination
      class="grow"
      pagination-mode="server"
      :page="store.page"
      :page-size="store.pageSize"
      :total="store.total"
      @page-change="store.changePage"
    >
        <template #empty>
          <div class="empty-action">
            <p class="empty-action__title">当前没有匹配的请求记录</p>
            <div class="empty-action__actions">
              <UButton size="sm" variant="primary" :loading="store.refreshing" @click="store.refresh">刷新流量</UButton>
            </div>
          </div>
        </template>
        <template #query>
          <div class="query-form">
            <USelect class="query-form__field query-form__field--protocol" label="协议筛选" hide-label
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
          <p class="table-cell-wrap text-sm text-strong" :title="requestInfoTitle(row)">
            {{ row.method || "-" }} · {{ row.client_ip || "-" }} · {{ row.user_agent || "无 User-Agent" }}
          </p>
        </template>
        <template #cell-route="{ row }">
          <p class="table-cell-wrap text-sm text-secondary" :title="routeTitle(row)">
            {{ row.downstream || "-" }} → {{ row.upstream || "-" }}<span v-if="routeHint(row)"> · {{ routeHint(row) }}</span>
          </p>
        </template>
        <template #cell-model="{ row }">
          <p class="table-cell-wrap text-sm text-strong" :title="modelTitle(row)">
            {{ row.requested_model || "-" }} → {{ row.model || "-" }}
          </p>
        </template>
        <template #cell-requestBody="{ row }">
          <p class="table-cell-wrap text-sm text-secondary" :title="requestBodyTitle(row)">
            {{ requestBodyPreview(row) }} · {{ formatBytes(row.request_body_bytes) }}{{ row.request_body_truncated ? "，已截断" : "" }}
          </p>
        </template>
        <template #cell-tokens="{ row }">
          <div class="token-cell token-cell--inline" :title="tokenTitle(row)">
            <span class="token-cell__item">入 {{ formatTokenCount(row.input_tokens) }}</span>
            <span class="token-cell__item">出 {{ formatTokenCount(row.output_tokens) }}</span>
            <span class="token-cell__item token-cell__item--total">总 {{ formatTokenCount(row.total_tokens) }}</span>
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
          <p v-if="row.error" class="table-cell-wrap text-sm text-error" :title="row.error">{{ row.error }}</p>
          <span v-else class="table-meta">无</span>
        </template>
    </UTable>

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

function requestInfoTitle(row) {
  return `${row.method || "-"} · ${row.client_ip || "-"} · ${row.user_agent || "无 User-Agent"}`;
}

function routeTitle(row) {
  return [row.downstream || "-", row.upstream || "-", routeHint(row)].filter(Boolean).join(" · ");
}

function modelTitle(row) {
  return `${row.requested_model || "-"} → ${row.model || "-"}`;
}

function requestBodyTitle(row) {
  return `${requestBodyPreview(row)} · ${formatBytes(row.request_body_bytes)}${row.request_body_truncated ? "，已截断" : ""}`;
}

function tokenTitle(row) {
  return `输入 ${formatTokenCount(row.input_tokens)} · 输出 ${formatTokenCount(row.output_tokens)} · 总 ${formatTokenCount(row.total_tokens)}`;
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






