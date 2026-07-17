<template>
  <section class="page-section page-section--scroll">
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
      description="安装并启用后可一键创建 Vendor=plugin 供应商。凭据在扩展页管理，路由在「规则设置」绑定。"
    >
      <div v-if="error" class="plugins-error">{{ error }}</div>
      <UTable :columns="tableColumns" :rows="rows" action-width="360px" size="sm" fixed>
        <template #empty>
          <div class="empty-action">
            <p class="empty-action__title">当前没有已注册的插件</p>
            <p class="empty-action__desc">
              将插件放到 plugins/&lt;插件ID&gt;/（含 info.toml 与可执行文件），点「发现插件」扫描后安装。
            </p>
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
          <span
            class="table-cell-wrap text-xs"
            :title="pluginMetaTitle(row)"
          >{{ pluginMetaLine(row) }}</span>
        </template>
        <template #actions="{ row }">
          <div class="plugins-actions">
            <UButton size="xs" variant="secondary" :disabled="busyId === row.id" @click="doStart(row.id)">启动</UButton>
            <UButton size="xs" variant="secondary" :disabled="busyId === row.id" @click="doRestart(row.id)">重启</UButton>
            <UButton size="xs" variant="ghost" :disabled="busyId === row.id" @click="doStop(row.id)">停止</UButton>
            <UButton
              size="xs"
              variant="primary"
              :disabled="busyId === row.id"
              :loading="busyId === `ensure:${row.id}`"
              @click="doEnsureProvider(row)"
            >
              接入路由
            </UButton>
            <UButton v-if="row.ui_pages?.length" size="xs" @click="openPage(row)">扩展页</UButton>
            <UButton size="xs" variant="ghost" :disabled="busyId === row.id" @click="openUnregisterConfirm(row)">卸载</UButton>
          </div>
        </template>
      </UTable>
    </PanelBlock>

    <PanelBlock
      v-if="candidates.length"
      title="可安装插件"
      description="优先扫描 plugins/&lt;id&gt;/info.toml；安装后写入 data 目录 registry.json。"
    >
      <UTable :columns="candidateColumns" :rows="candidates" action-width="180px" size="sm" fixed>
        <template #cell-registered="{ row }">
          <UTag :variant="row.registered ? 'success' : 'neutral'" size="xs">
            {{ row.registered ? "已注册" : "未注册" }}
          </UTag>
        </template>
        <template #cell-meta="{ row }">
          <span
            class="table-cell-wrap text-xs"
            :title="candidateMetaTitle(row)"
          >{{ candidateMetaLine(row) }}</span>
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
      description="运行中插件的桌面扩展入口（iframe 经 bridge 反代）。"
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

    <UModal v-model:open="registerOpen" title="手动安装插件" width="480px">
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
        <div class="flex justify-end gap-1.5">
          <UButton size="sm" variant="secondary" @click="registerOpen = false">取消</UButton>
          <UButton size="sm" :loading="registering" @click="doRegister">安装</UButton>
        </div>
      </template>
    </UModal>

    <UConfirmDialog
      v-model:open="unregisterConfirm.open"
      title="确认卸载插件"
      :message="unregisterConfirm.message"
      description="将停止进程并从目录中移除（不会删除 exe 文件，也不会删除已创建的供应商）。"
      confirm-text="确认卸载"
      cancel-text="取消"
      :loading="Boolean(busyId) && busyId === unregisterConfirm.id"
      danger
      @confirm="confirmUnregister"
    />
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import PanelBlock from "../components/PanelBlock.vue";
import UButton from "../components/ued/UButton.vue";
import UConfirmDialog from "../components/ued/UConfirmDialog.vue";
import UInput from "../components/ued/UInput.vue";
import UModal from "../components/ued/UModal.vue";
import USwitch from "../components/ued/USwitch.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";
import {
  DiscoverPlugins,
  EnsurePluginProvider,
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
const unregisterConfirm = reactive({
  open: false,
  id: "",
  message: "",
});

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

function pluginMetaLine(row) {
  const parts = [row.plugin_version || "-", row.executable || ""].filter(Boolean);
  if (row.last_error) parts.push(row.last_error);
  return parts.join(" · ");
}

function pluginMetaTitle(row) {
  return pluginMetaLine(row);
}

function candidateMetaLine(row) {
  const head = `${row.name || row.id || ""}${row.version ? ` · v${row.version}` : ""}`;
  const bits = [head];
  if (row.description) bits.push(row.description);
  if (row.executable) bits.push(row.executable);
  return bits.filter(Boolean).join(" · ");
}

function candidateMetaTitle(row) {
  const line = candidateMetaLine(row);
  const extra = [row.source, row.manifest_path].filter(Boolean).join(" · ");
  return extra ? `${line}\n${extra}` : line;
}

/**
 * Create or refresh Vendor=plugin provider + default models.
 * @param {string} pluginId
 * @param {{ quiet?: boolean, navigate?: boolean }} opts
 */
async function ensureProviderForPlugin(pluginId, opts = {}) {
  const id = String(pluginId || "").trim();
  if (!id) return null;
  const result = await EnsurePluginProvider({
    plugin_id: id,
    fetch_models: true,
  });
  const name = result.provider?.name || id;
  if (!opts.quiet) {
    if (result.created) {
      message.success(
        `已创建供应商「${name}」（plugin://${id}）。请在扩展页配置凭据，并在规则设置中绑定路由。`,
      );
    } else if (result.models_added > 0) {
      message.success(`供应商「${name}」已存在，已补充 ${result.models_added} 个模型。`);
    } else {
      message.info(`供应商「${name}」已就绪（plugin://${id}）。`);
    }
  }
  if (opts.navigate) {
    router.push({ name: "suppliers" });
  }
  return result;
}

async function doEnsureProvider(row) {
  const id = row?.id;
  if (!id) return;
  busyId.value = `ensure:${id}`;
  try {
    await ensureProviderForPlugin(id, { navigate: false });
  } catch (e) {
    message.error(e.message || String(e));
  } finally {
    busyId.value = "";
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
      message.info("未发现插件（请将插件放到 plugins/<id>/ 并包含 info.toml 与可执行文件）");
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
    if (enabled) {
      try {
        const result = await ensureProviderForPlugin(id, { quiet: true });
        const name = result?.provider?.name || id;
        message.success(
          result?.created
            ? `已启用并启动；已创建供应商「${name}」。`
            : `已启用并启动；供应商「${name}」已同步。`,
        );
      } catch (ensureErr) {
        message.success("已启用并启动");
        message.error(`接入路由失败：${ensureErr.message || ensureErr}`);
      }
    } else {
      message.success("已停用并停止");
    }
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
    await reload();
    await refreshCandidatesQuiet();
    if (enabled) {
      try {
        const result = await ensureProviderForPlugin(id, { quiet: true });
        const name = result?.provider?.name || id;
        message.success(
          result?.created
            ? `已安装并启用；已创建供应商「${name}」。请打开扩展页配置凭据，并在规则设置绑定路由。`
            : `已安装并启用；供应商「${name}」已就绪。`,
        );
      } catch (ensureErr) {
        message.success("已安装并启用");
        message.error(`接入路由失败：${ensureErr.message || ensureErr}（可稍后点「接入路由」重试）`);
      }
    } else {
      message.success("已注册（未启动）。启用后可点「接入路由」创建供应商。");
    }
  } catch (e) {
    message.error(e.message || String(e));
  } finally {
    busyId.value = "";
  }
}

function openUnregisterConfirm(row) {
  const id = row?.id || "";
  if (!id) return;
  unregisterConfirm.open = true;
  unregisterConfirm.id = id;
  unregisterConfirm.message = `确定要卸载插件「${id}」吗？`;
}

async function confirmUnregister() {
  const id = unregisterConfirm.id;
  if (!id) return;
  busyId.value = id;
  try {
    await UnregisterPlugin(id);
    message.success("已卸载");
    unregisterConfirm.open = false;
    unregisterConfirm.id = "";
    unregisterConfirm.message = "";
    await reload();
    await refreshCandidatesQuiet();
  } catch (e) {
    message.error(e.message || String(e));
  } finally {
    busyId.value = "";
  }
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
  const shouldEnsure = !!registerForm.value.enabled;
  registering.value = true;
  try {
    await RegisterPlugin({
      id,
      executable,
      enabled: shouldEnsure,
      auto_start: shouldEnsure,
    });
    registerOpen.value = false;
    await reload();
    await refreshCandidatesQuiet();
    if (shouldEnsure) {
      try {
        const result = await ensureProviderForPlugin(id, { quiet: true });
        const name = result?.provider?.name || id;
        message.success(
          result?.created
            ? `安装成功；已创建供应商「${name}」。请配置凭据并绑定路由。`
            : `安装成功；供应商「${name}」已就绪。`,
        );
      } catch (ensureErr) {
        message.success("安装成功");
        message.error(`接入路由失败：${ensureErr.message || ensureErr}`);
      }
    } else {
      message.success("安装成功（未启动）");
    }
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
  color: var(--ued-color-destructive, #dc2626);
  margin-bottom: var(--ued-space-4, 6px);
}
.plugins-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ued-space-2, 3px);
}
.plugins-pages {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--ued-space-5, 8px);
}
.plugins-page-card {
  text-align: left;
  border: 1px solid var(--ued-color-border, #d4d7dc);
  background: var(--ued-color-bg-card, #ffffff);
  border-radius: var(--ued-radius-lg, 6px);
  padding: var(--ued-space-5, 8px);
  cursor: pointer;
  color: inherit;
  display: flex;
  flex-direction: column;
  gap: var(--ued-space-2, 3px);
}
.plugins-page-card:hover {
  border-color: color-mix(in srgb, var(--ued-color-primary, #2563eb) 45%, transparent);
}
.plugins-page-card__desc {
  font-size: var(--ued-font-size-xs, 11px);
  color: var(--ued-color-text-muted, #6b7280);
  word-break: break-all;
}
.register-form {
  display: flex;
  flex-direction: column;
  gap: var(--ued-space-stack, 6px);
}
</style>
