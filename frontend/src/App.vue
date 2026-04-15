<template>
  <div class="app-container">
    <header class="custom-header">
      <div class="header-drag-region">
        <div class="app-brand">
          <div class="brand-mark">IC</div>
          <div class="brand-copy">
            <span class="app-title">icoo_proxy</span>
            <span class="app-subtitle">{{ currentModule }}</span>
          </div>
        </div>
        <div class="header-tools">
          <div class="relative theme-menu-wrap" ref="themeMenuRef">
            <HeaderToolButton
              @click="showThemeMenu = !showThemeMenu"
              :title="`外观设置 (${themeStore.theme === 'light' ? '浅色' : '深色'})`"
            >
              <Sun v-if="themeStore.theme === 'light'" :size="14" />
              <Moon v-else :size="14" />
            </HeaderToolButton>

            <Transition name="fade">
              <div v-if="showThemeMenu" class="theme-menu">
                <div class="theme-menu-section">
                  <div class="theme-menu-label">显示模式</div>
                  <div class="theme-mode-switch">
                    <button
                      @click="themeStore.setTheme('light')"
                      :class="themeStore.theme === 'light' ? 'active' : ''"
                    >
                      <Sun :size="12" />
                      浅色
                    </button>
                    <button
                      @click="themeStore.setTheme('dark')"
                      :class="themeStore.theme === 'dark' ? 'active' : ''"
                    >
                      <Moon :size="12" />
                      深色
                    </button>
                  </div>
                </div>

                <div class="theme-menu-section">
                  <div class="theme-menu-label">强调色</div>
                  <div class="theme-color-grid">
                    <button
                      v-for="color in colorList"
                      :key="color.key"
                      @click="themeStore.setColorTheme(color.key)"
                      class="theme-color-dot"
                      :class="themeStore.colorTheme === color.key ? 'active' : ''"
                      :style="{ backgroundColor: color.color }"
                      :title="color.name"
                    ></button>
                  </div>
                </div>
              </div>
            </Transition>
          </div>

          <HeaderToolButton @click="handleRefresh" title="刷新页面">
            <RefreshCw :size="14" />
          </HeaderToolButton>
        </div>
      </div>
      <div class="window-controls">
        <button class="control-btn minimize-btn" @click="handleMinimize" title="最小化">
          <svg width="12" height="12" viewBox="0 0 12 12">
            <rect x="1" y="5.5" width="10" height="1" fill="currentColor" />
          </svg>
        </button>
        <button class="control-btn close-btn" @click="handleClose" title="关闭">
          <svg width="12" height="12" viewBox="0 0 12 12">
            <path
              d="M1 1L11 11M11 1L1 11"
              stroke="currentColor"
              stroke-width="1.5"
              fill="none"
            />
          </svg>
        </button>
      </div>
    </header>
    <div class="app-body">
      <aside class="sidebar">
        <div class="sidebar-section-title">工作区</div>
        <nav class="sidebar-nav">
          <button
            v-for="item in menuItems"
            :key="item.path"
            @click="navigateTo(item.path)"
            class="nav-item"
            :class="{ active: isActive(item.path) }"
            :title="item.label"
          >
            <component :is="item.icon" :size="16" />
            <span class="nav-item-label">{{ item.label }}</span>
          </button>
        </nav>
        <div class="sidebar-foot">
          <button
            @click="navigateTo('/settings')"
            class="nav-item"
            :class="{ active: isActive('/settings') }"
            title="设置"
          >
            <Settings :size="16" />
            <span class="nav-item-label">设置</span>
          </button>
        </div>
      </aside>
      <main class="main-content">
        <RouterView v-slot="{ Component, route }">
          <transition name="fade-slide" mode="out-in">
            <component :is="Component" :key="route.path" />
          </transition>
        </RouterView>
      </main>
    </div>

    <!-- 底部状态栏 -->
    <footer class="app-footer">
      <div class="footer-left">
        <StatusBadge
          :status="gatewayStore.running ? 'success' : 'error'"
          :label="gatewayStore.running ? `网关运行中 · :${gatewayStore.port}` : '网关未启动'"
          title="网关状态"
        />
        <span class="footer-meta">监听地址 127.0.0.1:{{ gatewayStore.port }}</span>
      </div>
      <div class="footer-right">
        <span class="footer-label">icoo_proxy Desktop</span>
      </div>
    </footer>

    <!-- 全局确认弹窗 -->
    <ConfirmDialog />
    <!-- 全局 Toast 通知 -->
    <ToastContainer />
    
  </div>
</template>

<script setup>
import { RouterView, useRoute, useRouter } from "vue-router";
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useThemeStore } from "./stores/theme";
import { useGatewayStore } from "./stores/gateway";
import {
  Server,
  MessageSquareText,
  Cpu,
  ScrollText,
  Settings,
  RefreshCw,
  Sun,
  Moon,
} from "lucide-vue-next";
import ConfirmDialog from "@/components/ConfirmDialog.vue";
import ToastContainer from "@/components/ToastContainer.vue";
import HeaderToolButton from "@/components/ui/HeaderToolButton.vue";
import StatusBadge from "@/components/ui/StatusBadge.vue";

const themeStore = useThemeStore();
themeStore.initTheme();
const gatewayStore = useGatewayStore();
const showThemeMenu = ref(false);
const themeMenuRef = ref(null);
const colorList = computed(() => themeStore.getColorThemeList());
let statusTimer = null;

const currentModule = computed(() => {
  const current = [...menuItems, { path: "/settings", label: "设置" }].find((item) =>
    item.path === "/" ? route.path === "/" : route.path.startsWith(item.path)
  );
  return current?.label || "工作区";
});

function handleRefresh() {
  window.location.reload();
}

const menuItems = [
  { path: "/", label: "网关", icon: Server },
  { path: "/dialog-rules", label: "对话规则", icon: MessageSquareText },
  { path: "/providers", label: "供应商", icon: Cpu },
  { path: "/logs", label: "日志", icon: ScrollText },
];

const route = useRoute();
const router = useRouter();

function navigateTo(path) {
  router.push(path);
}

function isWailsEnv() {
  return typeof window !== 'undefined' && window.go !== undefined;
}

function handleClickOutside(event) {
  if (themeMenuRef.value && !themeMenuRef.value.contains(event.target)) {
    showThemeMenu.value = false;
  }
}

onMounted(async () => {
  if (isWailsEnv()) {
    await gatewayStore.fetchStatus();
  }
  statusTimer = window.setInterval(() => {
    if (isWailsEnv()) {
      gatewayStore.fetchStatus();
    }
  }, 10000);
  document.addEventListener("click", handleClickOutside);
});

onUnmounted(() => {
  if (statusTimer) {
    window.clearInterval(statusTimer);
  }
  document.removeEventListener("click", handleClickOutside);
});

function isActive(path) {
  if (path === "/") {
    return route.path === "/";
  }
  return route.path.startsWith(path);
}

function handleMinimize() {
  if (isWailsEnv()) {
    window.go.services.App.MinimizeWindow();
  }
}

function handleClose() {
  if (isWailsEnv()) {
    window.go.services.App.CloseWindow();
  }
}
</script>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  overflow: hidden;
  background: var(--ui-bg-window);
}

.custom-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--header-height);
  background: var(--ui-bg-toolbar);
  border-bottom: 1px solid var(--ui-border-default);
  color: var(--color-text-primary);
  user-select: none;
}

.header-drag-region {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  padding-left: 16px;
  height: 100%;
  --wails-draggable: drag;
  gap: 12px;
}

.header-tools {
  display: flex;
  align-items: center;
  gap: 6px;
  --wails-draggable: no-drag;
}

.app-brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-mark {
  width: 26px;
  height: 26px;
  border-radius: 7px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(180deg, color-mix(in srgb, var(--color-accent) 92%, white), var(--color-accent));
  color: white;
  box-shadow: var(--shadow-rest);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.06em;
}

.brand-copy {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.app-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-primary);
  white-space: nowrap;
}

.app-subtitle {
  font-size: 12px;
  color: var(--color-text-muted);
}

.theme-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 8px);
  width: 228px;
  padding: 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--ui-border-default);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-dialog);
  z-index: 50;
}

.theme-menu-section + .theme-menu-section {
  margin-top: 10px;
}

.theme-menu-label {
  margin-bottom: 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-muted);
}

.theme-mode-switch {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 4px;
}

.theme-mode-switch button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 30px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--ui-border-default);
  background: var(--ui-bg-surface);
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.theme-mode-switch button.active {
  background: var(--color-accent-soft);
  border-color: color-mix(in srgb, var(--color-accent) 24%, var(--ui-border-default));
  color: var(--color-accent);
}

.theme-mode-switch button:hover {
  background: var(--ui-bg-surface-hover);
}

.theme-mode-switch button.active:hover {
  background: var(--color-accent-soft);
  color: var(--color-accent);
}

.theme-color-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
}

.theme-color-dot {
  width: 32px;
  height: 32px;
  border-radius: 999px;
  border: 2px solid transparent;
  transition: transform 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
}

.theme-color-dot:hover {
  transform: scale(1.05);
}

.theme-color-dot.active {
  border-color: var(--ui-bg-surface);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 22%, transparent);
}

.window-controls {
  display: flex;
  -webkit-app-region: no-drag;
  height: 100%;
}

.control-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: var(--header-height);
  border: none;
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: background-color 0.12s, color 0.12s;
}

.control-btn:hover {
  background: var(--ui-bg-surface-hover);
  color: var(--color-text-primary);
}

.close-btn:hover {
  background: #e81123;
  color: white;
}

.app-body {
  display: flex;
  flex: 1;
  overflow: hidden;
  background: var(--ui-bg-window);
}

.sidebar {
  width: var(--sidebar-width);
  background: var(--ui-bg-sidebar);
  border-right: 1px solid var(--ui-border-default);
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  padding: 14px 10px 12px;
}

.sidebar-section-title {
  padding: 0 10px 8px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-text-muted);
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  padding: 0;
  gap: 4px;
}

.nav-item {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  position: relative;
  gap: 10px;
  width: 100%;
  min-height: 38px;
  padding: 0 10px;
  margin: 0 auto;
  color: var(--color-text-secondary);
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.nav-item:hover {
  color: var(--color-text-primary);
  background: var(--ui-bg-surface-hover);
  border-color: var(--ui-border-subtle);
}

.nav-item.active {
  color: var(--color-accent);
  background: var(--color-accent-soft);
  border-color: color-mix(in srgb, var(--color-accent) 22%, var(--ui-border-default));
  box-shadow: var(--shadow-rest);
}

.nav-item.active::before {
  content: '';
  position: absolute;
  left: -10px;
  top: 6px;
  bottom: 6px;
  width: 3px;
  background: var(--color-accent);
  border-radius: 999px;
}

.sidebar-foot {
  margin-top: 8px;
  padding-top: 10px;
  display: flex;
  justify-content: stretch;
  border-top: 1px solid var(--ui-border-default);
}

.nav-item-label {
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
}

.nav-item:focus-visible {
  outline: none;
  box-shadow: var(--shadow-focus);
}

.main-content {
  flex: 1;
  overflow: hidden;
  min-width: 0;
  background: var(--ui-bg-app);
}

/* 底部状态栏 */
.app-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 26px;
  padding: 0 10px;
  background: var(--ui-bg-statusbar);
  color: var(--color-text-secondary);
  border-top: 1px solid var(--ui-border-default);
  user-select: none;
}

.footer-left,
.footer-right {
  display: flex;
  align-items: center;
  gap: 6px;
}

.footer-label {
  font-size: 11px;
  font-weight: 600;
}

.footer-meta {
  font-size: 11px;
  color: var(--color-text-muted);
}

/* 底部状态栏徽章适配 */
.app-footer :deep(div[role="status"]) {
  height: 20px !important;
  padding: 0 6px !important;
  font-size: 10px !important;
}

.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.2s ease-out;
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateX(8px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateX(-8px);
}

.custom-header :deep(.header-tool-btn) {
  color: var(--color-text-muted);
}

.custom-header :deep(.header-tool-btn:hover) {
  color: var(--color-text-primary);
  background: var(--ui-bg-surface-hover);
}

.custom-header :deep(.header-tool-btn:active) {
  background: var(--ui-bg-surface-active);
}

.custom-header :deep(.header-tool-btn:focus-visible) {
  box-shadow: var(--shadow-focus);
}

@media (max-width: 860px) {
  .sidebar {
    width: 76px;
  }

  .sidebar-section-title,
  .app-subtitle,
  .footer-meta {
    display: none;
  }

  .nav-item {
    justify-content: center;
    padding: 0;
  }

  .nav-item-label {
    display: none;
  }
}
</style>
