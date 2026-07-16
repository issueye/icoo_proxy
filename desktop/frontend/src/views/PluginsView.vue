<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton variant="secondary" size="sm" :loading="loading" @click="reload">
          {{ loading ? "刷新中..." : "刷新" }}
        </UButton>
      </div>
    </Teleport>

    <PanelBlock title="进程插件" description="管理 IPC 插件进程；扩展页由插件 handshake 声明后自动出现在侧栏。">
      <div v-if="error" class="plugins-error">{{ error }}</div>
      <UTable :columns="tableColumns" :rows="rows" action-width="280px">
        <template #empty>
          <div class="plugins-empty">
            <p>当前没有已配置/运行中的插件。</p>
            <p>
              1. 确认 bridge 为<strong>本仓库最新构建</strong>（含
              <code>/api/v1/plugins</code>）
            </p>
            <p>
              2. 在配置中启用
              <code>plugins.entries.grokbuild</code>，并把
              <code>plugin-grokbuild.exe</code> 放在 bridge 同目录
            </p>
            <p>3. 重启 bridge / 桌面端后再刷新本页</p>
          </div>
        </template>
        <template #cell-status="{ row }">
          <UTag :variant="statusVariant(row.status)" size="xs">{{ row.status || "-" }}</UTag>
        </template>
        <template #cell-meta="{ row }">
          <p class="text-sm">{{ row.plugin_version || "-" }}</p>
          <p class="text-sm text-secondary" :title="row.executable">{{ row.executable || "-" }}</p>
          <p v-if="row.last_error" class="text-sm plugins-error" :title="row.last_error">{{ row.last_error }}</p>
        </template>
        <template #actions="{ row }">
          <div class="plugins-actions">
            <UButton size="xs" variant="secondary" :disabled="busyId === row.id" @click="doStart(row.id)">启动</UButton>
            <UButton size="xs" variant="secondary" :disabled="busyId === row.id" @click="doRestart(row.id)">重启</UButton>
            <UButton size="xs" variant="ghost" :disabled="busyId === row.id" @click="doStop(row.id)">停止</UButton>
            <UButton v-if="row.ui_pages?.length" size="xs" @click="openPage(row)">扩展页</UButton>
          </div>
        </template>
      </UTable>
    </PanelBlock>

    <PanelBlock
      v-if="uiPages.length"
      title="扩展页面"
      description="来自运行中插件的桌面扩展入口（iframe 经 bridge 反代）。"
      class="mt-4"
    >
      <div class="plugins-pages">
        <button
          v-for="p in uiPages"
          :key="p.plugin_id + ':' + p.id"
          type="button"
          class="plugins-page-card"
          @click="goExt(p)"
        >
          <strong>{{ p.title }}</strong>
          <span>{{ p.plugin_id }} · {{ p.group || "插件" }}</span>
          <span class="plugins-page-card__desc">{{ p.description || p.embed_url }}</span>
        </button>
      </div>
    </PanelBlock>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import PanelBlock from "../components/PanelBlock.vue";
import UButton from "../components/ued/UButton.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";
import {
  ListPlugins,
  ListPluginUIPages,
  RestartPlugin,
  StartPlugin,
  StopPlugin,
} from "../lib/apiClient";

const router = useRouter();
const loading = ref(false);
const busyId = ref("");
const error = ref("");
const items = ref([]);
const uiPages = ref([]);

const tableColumns = [
  { key: "id", title: "插件 ID", width: "18%" },
  { key: "status", title: "状态", width: "12%" },
  { key: "meta", title: "版本 / 路径", width: "40%" },
];

const rows = computed(() => {
  const list = Array.isArray(items.value) ? items.value : items.value?.items || [];
  return list;
});

function statusVariant(status) {
  switch (status) {
    case "running":
      return "success";
    case "unhealthy":
      return "warning";
    case "error":
      return "error";
    default:
      return "neutral";
  }
}

async function reload() {
  loading.value = true;
  error.value = "";
  try {
    const [plugins, pages] = await Promise.all([ListPlugins(), ListPluginUIPages()]);
    items.value = Array.isArray(plugins) ? plugins : plugins?.items || [];
    uiPages.value = Array.isArray(pages) ? pages : pages?.items || [];
  } catch (e) {
    error.value = e.message || String(e);
  } finally {
    loading.value = false;
  }
}

async function runAction(id, fn, okMsg) {
  busyId.value = id;
  try {
    await fn(id);
    message.success(okMsg);
    await reload();
  } catch (e) {
    message.error(e.message || String(e));
  } finally {
    busyId.value = "";
  }
}

function doStart(id) {
  return runAction(id, StartPlugin, "已请求启动");
}
function doStop(id) {
  return runAction(id, StopPlugin, "已请求停止");
}
function doRestart(id) {
  return runAction(id, RestartPlugin, "已请求重启");
}

function openPage(row) {
  const page = row.ui_pages?.[0];
  if (!page) return;
  goExt(page);
}

function goExt(p) {
  router.push(`/ext/${encodeURIComponent(p.plugin_id)}/${encodeURIComponent(p.id || "home")}`);
}

onMounted(reload);
</script>

<style scoped>
.plugins-error {
  color: #f87171;
  margin-bottom: 12px;
}
.plugins-empty {
  color: #9aa7b8;
  font-size: 14px;
  line-height: 1.6;
  padding: 12px;
}
.plugins-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.plugins-pages {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}
.plugins-page-card {
  text-align: left;
  border: 1px solid var(--border-color, #2a3544);
  background: var(--panel-bg, #121820);
  border-radius: 12px;
  padding: 14px;
  cursor: pointer;
  color: inherit;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.plugins-page-card:hover {
  border-color: #3b82f6;
}
.plugins-page-card__desc {
  font-size: 12px;
  color: #9aa7b8;
  word-break: break-all;
}
.mt-4 {
  margin-top: 16px;
}
</style>
