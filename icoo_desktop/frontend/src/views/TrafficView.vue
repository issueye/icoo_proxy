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
          {{ store.refreshing ? "鍒锋柊涓?.." : "鍒锋柊娴侀噺" }}
        </UButton>
        <UButton
          variant="error"
          :loading="store.clearing"
          :disabled="store.refreshing || store.clearing || !store.totalRequests"
          @click="openClearConfirm"
        >
          {{ store.clearing ? "娓呯┖涓?.." : "娓呯┖璇锋眰" }}
        </UButton>
        <USwitch v-model="store.autoRefresh" label="鑷姩鍒锋柊" />
      </div>
    </Teleport>

    <div class="stat-grid stat-grid--5">
      <StatCard icon="activity" label="鏈€杩戣姹傛暟" :value="String(store.totalRequests)" tone="info" />
      <StatCard icon="check" label="鎴愬姛璇锋眰" :value="String(store.successCount)" tone="success" />
      <StatCard icon="alert" label="閿欒璇锋眰" :value="String(store.errorCount)" tone="danger" />
      <StatCard icon="timer" label="骞冲潎鑰楁椂" :value="`${store.averageLatency} ms`" />
      <StatCard icon="layers" label="鎬昏緭鍏?Token" :value="formatTokenCount(store.tokenStats.input_tokens)"
        tone="primary" />
      <StatCard icon="server" label="鎬昏緭鍑?Token" :value="formatTokenCount(store.tokenStats.output_tokens)"
        tone="warning" />
      <StatCard icon="key" label="鎬?Token" :value="formatTokenCount(store.tokenStats.total_tokens)" tone="info" />
    </div>

      <UTable :columns="tableColumns" :rows="store.requests" row-key="request_id" fixed fixed-field="freeze" stripe
        size="sm" max-height="100%" min-width="1680px" pagination
        pagination-mode="server" :page="store.page" :page-size="store.pageSize" :total="store.total"
        :page-size-options="[8, 20, 50]" @page-change="store.changePage">
        <template #empty>
          褰撳墠娌℃湁鍖归厤鐨勮姹傝褰曘€?
        </template>
        <template #query>
          <div class="query-form">
            <USelect class="query-form__field query-form__field--protocol" label="鍗忚绛涢€? hide-label
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
          <p class="text-sm text-strong table-cell-wrap">{{ row.method || "-" }} 路 {{ row.client_ip || "-" }}</p>
          <p class="mt-0.5 table-meta table-cell-wrap">{{ row.user_agent || "鏃?User-Agent" }}</p>
        </template>
        <template #cell-route="{ row }">
          <p class="text-sm text-secondary table-cell-wrap">{{ row.downstream }}</p>
          <p class="mt-0.5 table-meta table-cell-wrap">{{ row.upstream || "-" }}</p>
          <p v-if="routeHint(row)" class="mt-0.5 table-meta table-cell-wrap">{{ routeHint(row) }}</p>
        </template>
        <template #cell-model="{ row }">
          <p class="text-sm text-strong table-cell-wrap">{{ row.requested_model || "-" }}</p>
          <p class="mt-0.5 table-meta table-cell-wrap">璺敱鍒?{{ row.model || "-" }}</p>
        </template>
        <template #cell-requestBody="{ row }">
          <p class="text-sm text-secondary table-cell-wrap">{{ requestBodyPreview(row) }}</p>
          <p class="mt-0.5 table-meta">
            {{ formatBytes(row.request_body_bytes) }}{{ row.request_body_truncated ? "锛屽凡鎴柇" : "" }}
          </p>
        </template>
        <template #cell-tokens="{ row }">
          <div class="token-cell">
            <div class="token-cell__row">
              <span class="token-cell__label">鍏?/span>
              <span class="token-cell__value">{{ formatTokenCount(row.input_tokens) }}</span>
            </div>
            <div class="token-cell__row">
              <span class="token-cell__label">鍑?/span>
              <span class="token-cell__value">{{ formatTokenCount(row.output_tokens) }}</span>
            </div>
            <div class="token-cell__row token-cell__row--total">
              <span class="token-cell__label">鎬?/span>
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
          <span v-else class="table-meta">鏃?/span>
        </template>
      </UTable>

    <UConfirmDialog v-model:open="confirmState.open" title="纭娓呯┖璇锋眰璁板綍" message="纭畾瑕佹竻绌烘祦閲忕洃鎺т腑鐨勬墍鏈夎姹備俊鎭悧锛?
      description="娓呯┖鍚庡綋鍓嶄繚瀛樼殑璇锋眰璁板綍涓庣粺璁℃暟鎹皢琚Щ闄わ紝涓旀棤娉曟仮澶嶃€? confirm-text="纭娓呯┖" cancel-text="鍙栨秷"
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
  { key: "requestId", title: "璇锋眰 ID", width: 180, freeze: "left" },
  { key: "endpoint", title: "绔偣", width: 180 },
  { key: "requestInfo", title: "璇锋眰淇℃伅", width: 240 },
  { key: "route", title: "涓嬫父 / 涓婃父", width: 220 },
  { key: "model", title: "妯″瀷", width: 190 },
  { key: "requestBody", title: "璇锋眰浣?, width: 240 },
  { key: "tokens", title: "Tokens", width: 128 },
  { key: "createdAt", title: "鍒涘缓鏃堕棿", width: 172 },
  { key: "error", title: "閿欒淇℃伅", width: 216 },
  { key: "duration", title: "鑰楁椂", width: 90, align: "center", freeze: "right" },
  { key: "status", title: "鐘舵€佺爜", width: 92, align: "center", freeze: "right" },
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
      label: rawLabel === "all" ? "鍏ㄩ儴鍗忚" : rawLabel,
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
    return "鏆傛棤";
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
    return "璇锋眰浣撴湭璁板綍锛岄渶鍦ㄩ」鐩缃腑寮€鍚褰?body";
  }
  return "鏃犺姹備綋";
}

function routeHint(row) {
  if (row.matched_rule_name) {
    return `鍛戒腑瑙勫垯锛?{row.matched_rule_name}`;
  }
  if (row.route_source === "direct") {
    return `鐩磋繛璺敱锛?{row.route_name || row.model || "-"}`;
  }
  if (row.route_name) {
    return `璺敱锛?{row.route_name}`;
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
    message.success("娴侀噺璇锋眰璁板綍宸叉竻绌恒€?);
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

</style>






