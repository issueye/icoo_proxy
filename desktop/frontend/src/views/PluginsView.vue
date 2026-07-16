<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton variant="secondary" size="sm" :loading="discovering" @click="doDiscover">
          {{ discovering ? "扫描中..." : "发现插件" }}
        </UButton>
        <UButton variant="secondary" size="sm" @click="openRegister">手动安装</UButton>
        <UButton variant="secondary" size="sm" :loading="loading" @click="reload">
          {{ loading ? "刷新中..." : "刷新" }}
        </UButton>
      </div>
    </Teleport>

    <PanelBlock
      title="进程插件"
      description="通过桌面端动态安装 / 启停插件，无需改 TOML 或重启 bridge。扩展页由 handshake 声明后自动出现在侧栏。"
    >
      <div v-if="error" class="plugins-error">{{ error }}</div>
      <UTable :columns="tableColumns" :rows="rows" action-width="360px">
        <template #empty>
          <div class="plugins-empty">
            <p>当前没有已注册的插件。</p>
            <p>1. 把 <code>plugin-*.exe</code> 放到 bridge 同目录</p>
            <p>2. 点击「发现插件」扫描，再「安装并启用」</p>
            <p>3. 或用「手动安装」填写可执行文件路径</p>
          </div>
        </template>
        <template #cell-enabled="{ row }">
          <USwitch
            :model-value="!!row.enabled"
            :label="row.enabled ? '已启用' : '已停用'"
            :disabled="busyId === row.id"
            @update:model-value="(v) => doSetEnabled(row.id, v)"
          />
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
            <UButton size="xs" variant="ghost" :disabled="busyId === row.id" @click="doUnregister(row.id)">卸载</UButton>
          </div>
        </template>
      </UTable>
    </PanelBlock>

    <PanelBlock
      v-if="candidates.length"
      title="可安装插件"
      description="扫描 bridge 目录、当前工作目录与 data_dir/plugins 得到的候选；安装后写入 registry.json，无需改配置文件。"
      class="mt-4"
    >
      <UTable :columns="candidateColumns" :rows="candidates" action-width="200px">
        <template #cell-registered="{ row }">
          <UTag :variant="row.registered ? 'success' : 'neutral'" size="xs">
            {{ row.registered ? "已注册" : "未注册" }}
          </UTag>
        </template>
        <template #cell-meta="{ row }">
          <p class="text-sm">{{ row.name || row.id }}{{ row.version ? ` · v${row.version}` : "" }}</p>
          <p class="text-sm text-secondary" :title="row.executable">{{ row.executable }}</p>
          <p class="text-sm text-secondary">来源: {{ row.source }}</p>
        </template>
        <template #actions="{ row }">
          <div class="plugins-actions">
            <UButton
              size="xs"
              :disabled="busyId === row.id || row.registered"
              @click="doInstall(row.id, true)"
            >
              安装并启用
            </UButton>
            <UButton
              size="xs"
              variant="secondary"
              :disabled="busyId === row.id || row.registered"
              @click="doInstall(row.id, false)"
            >
              仅注册
            </UButton>
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

    <UModal v-model:open="registerOpen" title="手动安装插件" width="520px">
      <div class="register-form">
        <UInput v-model="registerForm.id" label="插件 ID" placeholder="例如：grokbuild" required />
        <UInput
          v-model="registerForm.executable"
          label="可执行文件路径"
          placeholder="plugin-grokbuild.exe 或绝对路径"
          required
        />
        <USwitch v-model="registerForm.enabled" label="安装后启用并启动" />
      </div>
      <template #footer>
        <UButton variant="secondary" @click="registerOpen = false">取消</UButton>
        <UButton :loading="registering" @click="doRegister">安装</UButton>
      </template>
    </UModal>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import PanelBlock from "../components/PanelBlock.vue";
import UButton from "../components/ued/UButton.vue";
import UInput from "../components/ued/UInput.vue";
import UModal from "../components/ued/UModal.vue";
import USwitch from "../components/ued/USwitch.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";
import {
  DiscoverPlugins,
  InstallPlugin,
  ListPlugins,
  ListPluginUIPages,
  RegisterPlugin,
  RestartPlugin,
  SetPluginEnabled,
  StartPlugin,
  StopPlugin,
  UnregisterPlugin,
} from "../lib/apiClient";

const router = useRouter();
const loading = ref(false);
const discovering = ref(false);
const registering = ref(false);
const busyId = ref("");
const error = ref("");
const items = ref([]);
const uiPages = ref([]);
const candidates = ref([]);
const registerOpen = ref(false);
const registerForm = ref({ id: "", executable: "", enabled: true });

const tableColumns = [
  { key: "id", title: "插件 ID", width: "14%" },
  { key: "enabled", title: "启用", width: "14%" },
  { key: "status", title: "状态", width: "10%" },
  { key: "meta", title: "版本 / 路径", width: "32%" },
];

const candidateColumns = [
  { key: "id", title: "ID", width: "16%" },
  { key: "registered", title: "状态", width: "12%" },
  { key: "meta", title: "信息", width: "48%" },
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

async function doDiscover() {
  discovering.value = true;
  try {
    const res = await DiscoverPlugins();
    candidates.value = Array.isArray(res) ? res : res?.items || [];
    if (!candidates.value.length) {
      message.info("未发现插件二进制（请把 plugin-*.exe 放到 bridge 同目录）");
    } else {
      message.success(`发现 ${candidates.value.length} 个候选`);
    }
  } catch (e) {
    message.error(e.message || String(e));
  } finally {
    discovering.value = false;
  }
}

async function runAction(id, fn, okMsg) {
  busyId.value = id;
  try {
    await fn(id);
    message.success(okMsg);
    await reload();
    await refreshCandidatesQuiet();
  } catch (e) {
    message.error(e.message || String(e));
  } finally {
    busyId.value = "";
  }
}

async function refreshCandidatesQuiet() {
  if (!candidates.value.length) return;
  try {
    const res = await DiscoverPlugins();
    candidates.value = Array.isArray(res) ? res : res?.items || [];
  } catch {
    /* ignore */
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

async function doSetEnabled(id, enabled) {
  busyId.value = id;
  try {
    await SetPluginEnabled(id, enabled);
    message.success(enabled ? "已启用并启动" : "已停用并停止");
    await reload();
  } catch (e) {
    message.error(e.message || String(e));
    await reload();
  } finally {
    busyId.value = "";
  }
}

async function doInstall(id, enabled) {
  busyId.value = id;
  try {
    await InstallPlugin({ id, enabled });
    message.success(enabled ? "已安装并启用" : "已注册（未启动）");
    await reload();
    await refreshCandidatesQuiet();
  } catch (e) {
    message.error(e.message || String(e));
  } finally {
    busyId.value = "";
  }
}

async function doUnregister(id) {
  if (!window.confirm(`确定卸载插件「${id}」？将停止进程并从目录中移除（不会删除 exe 文件）。`)) {
    return;
  }
  return runAction(id, UnregisterPlugin, "已卸载");
}

function openRegister() {
  registerForm.value = { id: "", executable: "", enabled: true };
  registerOpen.value = true;
}

async function doRegister() {
  const id = registerForm.value.id?.trim();
  const executable = registerForm.value.executable?.trim();
  if (!id || !executable) {
    message.error("请填写插件 ID 与可执行文件路径");
    return;
  }
  registering.value = true;
  try {
    await RegisterPlugin({
      id,
      executable,
      enabled: !!registerForm.value.enabled,
      auto_start: !!registerForm.value.enabled,
    });
    message.success("安装成功");
    registerOpen.value = false;
    await reload();
    await refreshCandidatesQuiet();
  } catch (e) {
    message.error(e.message || String(e));
  } finally {
    registering.value = false;
  }
}

function openPage(row) {
  const page = row.ui_pages?.[0];
  if (!page) return;
  goExt(page);
}

function goExt(p) {
  router.push(`/ext/${encodeURIComponent(p.plugin_id)}/${encodeURIComponent(p.id || "home")}`);
}

onMounted(async () => {
  await reload();
  // Best-effort: auto-scan once so users see installable plugins immediately.
  try {
    const res = await DiscoverPlugins();
    candidates.value = Array.isArray(res) ? res : res?.items || [];
  } catch {
    /* old bridge without discover — ignore */
  }
});
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
.register-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.mt-4 {
  margin-top: 16px;
}
</style>
