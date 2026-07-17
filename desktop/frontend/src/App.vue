<template>
  <div class="app-shell" :class="{ 'app-shell--sidebar-collapsed': sidebarCollapsed }">
    <header class="app-titlebar">
      <div class="app-titlebar__brand">
        <img class="app-titlebar__logo" src="./assets/images/appicon.png" alt="" />
        <span class="app-titlebar__name">icoo_proxy</span>
        <span class="app-titlebar__divider" aria-hidden="true"></span>
        <span class="app-titlebar__caption">本地 AI 协议转换网关</span>
      </div>

      <div class="app-titlebar__actions">
        <button class="app-service-info-btn" type="button" title="服务信息" @click="openServerInfo">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.1" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="2" y="3" width="20" height="14" rx="2" /><path d="M8 21h8" /><path d="M12 17v4" /><path d="M7 8h.01" /><path d="M11 8h6" /><path d="M7 12h.01" /><path d="M11 12h6" /></svg>
          <span>服务信息</span>
        </button>
      </div>

      <div class="app-window-controls" aria-label="窗口控制">
        <button class="app-window-control" type="button" aria-label="最小化" title="最小化" @click="minimizeWindow">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14" /></svg>
        </button>
        <button class="app-window-control app-window-control--close" type="button" aria-label="隐藏到托盘" title="隐藏到托盘" @click="closeWindow">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg>
        </button>
      </div>
    </header>

    <div class="app-workbench">
      <aside class="app-sidebar">
        <div class="app-sidebar__header">
          <button
            class="app-sidebar-toggle"
            type="button"
            :aria-label="sidebarCollapsed ? '展开导航' : '收起导航'"
            :title="sidebarCollapsed ? '展开导航' : '收起导航'"
            @click="toggleSidebar"
          >
            <svg v-if="sidebarCollapsed" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" x2="20" y1="12" y2="12" /><line x1="4" x2="20" y1="6" y2="6" /><line x1="4" x2="20" y1="18" y2="18" /></svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6" /></svg>
          </button>
          <div class="app-sidebar__heading">
            <span>控制台</span>
          </div>
        </div>

        <nav class="app-sidebar-nav" aria-label="主导航">
          <section v-for="group in navGroups" :key="group.name" class="app-nav-group">
            <p class="app-nav-group__label">{{ group.name }}</p>
            <RouterLink
              v-for="item in group.items"
              :key="item.to"
              :to="item.to"
              class="app-nav-item"
              :class="{ 'app-nav-item--active': route.path === item.to }"
              :title="item.label"
            >
              <span class="app-nav-icon" aria-hidden="true">
                <svg v-if="item.icon === 'overview'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" /><rect x="14" y="14" width="7" height="7" /><rect x="3" y="14" width="7" height="7" /></svg>
                <svg v-else-if="item.icon === 'chat'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a4 4 0 0 1-4 4H8l-5 3V7a4 4 0 0 1 4-4h10a4 4 0 0 1 4 4z" /><path d="M8 9h8" /><path d="M8 13h5" /></svg>
                <svg v-else-if="item.icon === 'supplier'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="8" rx="2" /><rect x="2" y="14" width="20" height="8" rx="2" /><line x1="6" y1="6" x2="6.01" y2="6" /><line x1="6" y1="18" x2="6.01" y2="18" /></svg>
                <svg v-else-if="item.icon === 'rules'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 6h11" /><path d="M9 12h11" /><path d="M9 18h11" /><path d="M5 6h.01" /><path d="M5 12h.01" /><path d="M5 18h.01" /></svg>
                <svg v-else-if="item.icon === 'model'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2H2v10l9.29 9.29c.94.94 2.48.94 3.42 0l6.58-6.58c.94-.94.94-2.48 0-3.42L12 2Z" /><path d="M7 7h.01" /></svg>
                <svg v-else-if="item.icon === 'models'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7" rx="1" /><rect x="14" y="3" width="7" height="7" rx="1" /><rect x="3" y="14" width="7" height="7" rx="1" /><rect x="14" y="14" width="7" height="7" rx="1" /></svg>
                <svg v-else-if="item.icon === 'endpoint'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10" /><path d="M2 12h20" /><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" /></svg>
                <svg v-else-if="item.icon === 'key'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21 2-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4" /></svg>
                <svg v-else-if="item.icon === 'traffic'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12" /></svg>
                <svg v-else-if="item.icon === 'settings'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" /><circle cx="12" cy="12" r="3" /></svg>
                <svg v-else-if="item.icon === 'ued'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 4h16v16H4z" /><path d="M4 9h16" /><path d="M9 20V9" /></svg>
                <svg v-else-if="item.icon === 'plugin' || item.icon === 'key'" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v4" /><path d="M12 18v4" /><path d="m4.93 4.93 2.83 2.83" /><path d="m16.24 16.24 2.83 2.83" /><path d="M2 12h4" /><path d="M18 12h4" /><path d="m4.93 19.07 2.83-2.83" /><path d="m16.24 7.76 2.83-2.83" /></svg>
              </span>
              <span class="app-nav-text">{{ item.label }}</span>
            </RouterLink>
          </section>
        </nav>

      </aside>

      <main class="app-main">
        <header class="app-topbar">
          <div class="app-page-identity">
            <h1>{{ currentTitle }}</h1>
            <div class="app-breadcrumb">
              <span>{{ currentGroupName }}</span>
              <span>/</span>
              <span>{{ currentTitle }}</span>
            </div>
          </div>
          <div id="app-topbar-actions" class="app-topbar-actions" />
        </header>

        <div class="app-content">
          <RouterView />
        </div>
      </main>
    </div>

    <footer class="app-statusbar">
      <span>{{ statusText }}</span>
      <span class="app-statusbar__item app-statusbar__item--state">
        <span class="app-status-dot" :class="statusDotClass" aria-hidden="true"></span>
        <span>Server {{ serverStatusLabel }}</span>
        <UButton
          v-if="serverStatus === 'disconnected' || serverStatus === 'error'"
          size="xs"
          :loading="waking"
          :disabled="waking"
          @click="wake"
        >
          {{ waking ? "唤醒中..." : "唤醒" }}
        </UButton>
      </span>
      <span class="app-statusbar__item" :title="serverUrl">{{ serverUrl }}</span>
      <span v-if="serverError" class="app-statusbar__item app-statusbar__item--error" :title="serverError">{{ serverError }}</span>
      <span class="app-statusbar__item app-statusbar__item--right">icoo_desktop</span>
    </footer>

    <UMessage />
    <UModal v-model:open="serverInfoOpen" title="唤醒程序信息" width="640px">
      <div class="server-info-panel">
        <div class="server-info-summary">
          <span class="app-status-dot" :class="serverInfo.running ? 'app-status-dot--running' : 'app-status-dot--stopped'" aria-hidden="true"></span>
          <div class="server-info-summary__copy">
            <strong>{{ serverInfo.running ? "icoo_llm_bridge 正在运行" : "icoo_llm_bridge 未由桌面端托管运行" }}</strong>
            <span>{{ serverInfo.listen_addr || serverUrl }}</span>
          </div>
        </div>

        <dl class="server-info-list">
          <template v-for="item in serverInfoRows" :key="item.label">
            <dt>{{ item.label }}</dt>
            <dd :class="{ 'server-info-list__mono': item.mono }">{{ item.value }}</dd>
          </template>
        </dl>
      </div>
      <template #footer>
        <div class="server-info-footer">
          <UButton variant="secondary" size="sm" :loading="serverInfoLoading" @click="refreshServerInfo">刷新</UButton>
          <UButton size="sm" @click="serverInfoOpen = false">关闭</UButton>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router";
import { WindowHide, WindowMinimise } from "./lib/wailsRuntime";
import { useServerConnection } from "./composables/useServerConnection";
import { ListPluginUIPages } from "./lib/apiClient";
import UButton from "./components/ued/UButton.vue";
import UMessage from "./components/ued/UMessage.vue";
import UModal from "./components/ued/UModal.vue";

const route = useRoute();
const router = useRouter();
const sidebarCollapsed = ref(false);
const serverInfoOpen = ref(false);
const serverInfoLoading = ref(false);
const serverInfo = ref({});
const pluginPages = ref([]);
let pluginPollTimer = null;

const {
  status: serverStatus,
  statusText,
  statusDotClass,
  serverUrl,
  waking,
  error: serverError,
  wake,
} = useServerConnection();

const baseNavGroups = [
  {
    name: "运行",
    items: [
      { to: "/", label: "网关概览", icon: "overview" },
      { to: "/chat", label: "聊天", icon: "chat" },
      { to: "/traffic", label: "流量监控", icon: "traffic" },
    ],
  },
  {
    name: "配置",
    items: [
      { to: "/suppliers", label: "供应商", icon: "supplier" },
      { to: "/models", label: "模型管理", icon: "models" },
      { to: "/routing-rules", label: "路由规则", icon: "rules" },
      { to: "/model-aliases", label: "模型路由", icon: "model" },
      { to: "/endpoints", label: "端点", icon: "endpoint" },
      { to: "/auth-keys", label: "授权 Key", icon: "key" },
    ],
  },
  {
    name: "系统",
    items: [
      { to: "/plugins", label: "插件", icon: "plugin" },
      { to: "/settings", label: "项目设置", icon: "settings" },
      { to: "/ued", label: "组件规范", icon: "ued" },
    ],
  },
];

const navGroups = computed(() => {
  const groups = baseNavGroups.map((g) => ({
    name: g.name,
    items: g.items.map((item) => ({ ...item })),
  }));
  if (!pluginPages.value.length) return groups;

  const byGroup = new Map();
  for (const page of pluginPages.value) {
    const groupName = page.group || "插件";
    if (!byGroup.has(groupName)) byGroup.set(groupName, []);
    const pageId = page.id || "home";
    const to = `/ext/${encodeURIComponent(page.plugin_id)}/${encodeURIComponent(pageId)}`;
    byGroup.get(groupName).push({
      to,
      label: page.title || page.plugin_id,
      icon: page.icon || "plugin",
      embedPath: page.embed_url,
      pluginId: page.plugin_id,
      pageId,
    });
  }
  for (const [name, items] of byGroup.entries()) {
    const existing = groups.find((g) => g.name === name);
    if (existing) existing.items.push(...items);
    else groups.push({ name, items });
  }
  return groups;
});

const navItems = computed(() => navGroups.value.flatMap((group) => group.items));

const currentNavItem = computed(() => {
  const exact = navItems.value.find((item) => item.to === route.path);
  if (exact) return exact;
  if (route.name === "plugin-extension") {
    return navItems.value.find(
      (item) =>
        item.pluginId === route.params.pluginId &&
        (item.pageId === route.params.pageId || (!route.params.pageId && item.pageId === "home")),
    );
  }
  return null;
});

const currentTitle = computed(() => currentNavItem.value?.label || route.meta?.title || "本地 AI 网关管理台");

const currentGroupName = computed(() => (
  navGroups.value.find((group) => group.items.some((item) => item.to === route.path || item === currentNavItem.value))?.name || "控制台"
));

async function refreshPluginPages() {
  if (serverStatus.value !== "connected") {
    pluginPages.value = [];
    return;
  }
  try {
    const pages = await ListPluginUIPages();
    pluginPages.value = Array.isArray(pages) ? pages : pages?.items || [];
    // Keep route meta in sync for iframe src.
    for (const page of pluginPages.value) {
      const pageId = page.id || "home";
      const namePath = `/ext/${page.plugin_id}/${pageId}`;
      const matched = router.getRoutes().find((r) => r.path === "/ext/:pluginId/:pageId?");
      if (matched && route.path === namePath) {
        route.meta.title = page.title;
        route.meta.embedPath = page.embed_url;
      }
    }
  } catch {
    pluginPages.value = [];
  }
}

watch(
  () => serverStatus.value,
  () => {
    refreshPluginPages();
  },
);

watch(
  () => route.fullPath,
  () => {
    if (route.name === "plugin-extension") {
      const item = currentNavItem.value;
      if (item?.embedPath) {
        route.meta.embedPath = item.embedPath;
        route.meta.title = item.label;
      } else {
        const page = pluginPages.value.find(
          (p) => p.plugin_id === route.params.pluginId && (p.id === route.params.pageId || (!route.params.pageId && (p.id === "home" || p.id === "credentials"))),
        );
        if (page) {
          route.meta.embedPath = page.embed_url;
          route.meta.title = page.title;
        }
      }
    }
  },
  { immediate: true },
);

onMounted(() => {
  refreshPluginPages();
  pluginPollTimer = setInterval(refreshPluginPages, 15000);
});

onUnmounted(() => {
  if (pluginPollTimer) clearInterval(pluginPollTimer);
});

const serverStatusLabel = computed(() => {
  switch (serverStatus.value) {
    case "connected": return "已连接"
    case "connecting": return "连接中"
    case "disconnected": return "未连接"
    case "error": return "异常"
    default: return "未知"
  }
});

const serverInfoRows = computed(() => [
  { label: "状态", value: serverInfo.value.status || serverStatusLabel.value },
  { label: "PID", value: serverInfo.value.pid || "-", mono: true },
  { label: "监听地址", value: serverInfo.value.listen_addr || serverUrl.value, mono: true },
  { label: "启动时间", value: serverInfo.value.started_at || "-" },
  { label: "程序路径", value: serverInfo.value.executable || "-", mono: true },
  { label: "工作目录", value: serverInfo.value.working_directory || "-", mono: true },
  { label: "数据目录", value: serverInfo.value.data_dir || "-", mono: true },
  { label: "启动参数", value: Array.isArray(serverInfo.value.args) && serverInfo.value.args.length ? serverInfo.value.args.join(" ") : "-", mono: true },
  { label: "日志路径", value: serverInfo.value.log_path || "-", mono: true },
  { label: "最近错误", value: serverInfo.value.last_error || "-" },
]);

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value;
}

function minimizeWindow() {
  WindowMinimise();
}

function closeWindow() {
  WindowHide();
}

async function openServerInfo() {
  serverInfoOpen.value = true;
  await refreshServerInfo();
}

async function refreshServerInfo() {
  serverInfoLoading.value = true;
  try {
    if (typeof window !== "undefined" && window.go?.main?.App?.GetServerProcessInfo) {
      serverInfo.value = normalizeServerInfo(await window.go.main.App.GetServerProcessInfo());
    } else {
      serverInfo.value = {
        running: serverStatus.value === "connected",
        status: serverStatus.value,
        listen_addr: serverUrl.value.replace(/^https?:\/\//, ""),
      };
    }
  } finally {
    serverInfoLoading.value = false;
  }
}

function normalizeServerInfo(raw = {}) {
  return {
    running: raw.running ?? raw.Running ?? false,
    status: raw.status ?? raw.Status ?? "",
    pid: raw.pid ?? raw.PID ?? 0,
    executable: raw.executable ?? raw.Executable ?? "",
    working_directory: raw.working_directory ?? raw.WorkingDirectory ?? "",
    data_dir: raw.data_dir ?? raw.DataDir ?? "",
    listen_addr: raw.listen_addr ?? raw.ListenAddr ?? "",
    started_at: raw.started_at ?? raw.StartedAt ?? "",
    args: raw.args ?? raw.Args ?? [],
    log_path: raw.log_path ?? raw.LogPath ?? "",
    last_error: raw.last_error ?? raw.LastError ?? "",
  };
}
</script>
