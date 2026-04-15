<template>
  <div class="app-container">
    <header class="custom-header">
      <div class="header-drag-region">
        <div class="app-brand">
          <div class="brand-mark">IC</div>
          <span class="app-title">icoo_proxy</span>
        </div>
        <div class="header-tools">
          <div class="header-divider"></div>

          <!-- 主题切换 -->
          <div class="relative theme-menu-wrap" ref="themeMenuRef">
            <HeaderToolButton
              @click="showThemeMenu = !showThemeMenu"
              :title="`切换主题 (${themeStore.theme === 'light' ? '浅色' : '深色'})`"
            >
              <Sun v-if="themeStore.theme === 'light'" :size="14" />
              <Moon v-else :size="14" />
            </HeaderToolButton>

            <Transition name="fade">
              <div v-if="showThemeMenu" class="theme-menu">
                <div class="theme-menu-section">
                  <div class="theme-menu-label">主题模式</div>
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
                  <div class="theme-menu-label">颜色主题</div>
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

          <!-- 刷新按钮 -->
          <HeaderToolButton
            @click="handleRefresh"
            title="刷新页面"
          >
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
        <nav class="sidebar-nav">
          <button
            v-for="item in menuItems"
            :key="item.path"
            @click="navigateTo(item.path)"
            class="nav-item"
            :class="{ active: isActive(item.path) }"
            :title="item.label"
          >
            <component :is="item.icon" :size="22" />
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
            <Settings :size="22" />
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
          :label="gatewayStore.running ? `网关 :${gatewayStore.port}` : '网关未启动'"
          title="网关状态"
        />
      </div>
      <div class="footer-right">
        <span class="footer-label">AI Gateway</span>
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
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--color-accent) 4%, transparent), transparent 200px),
    var(--color-bg-primary);
}

.custom-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--header-height);
  background: hsl(var(--vscode-titlebar));
  border-bottom: 1px solid hsl(var(--vscode-chrome-border));
  color: hsl(var(--vscode-chrome-foreground));
  user-select: none;
}

.header-drag-region {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px;
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

.header-divider {
  width: 1px;
  height: 16px;
  background: hsl(var(--vscode-chrome-border));
  margin: 0 4px;
}

.app-brand {
  display: flex;
  align-items: center;
  gap: 10px;
}

.brand-mark {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--color-accent), #0ea5e9);
  color: white;
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.14);
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 0.06em;
}

.app-title {
  font-size: 13px;
  font-weight: 800;
  color: hsl(var(--vscode-chrome-foreground));
  letter-spacing: -0.02em;
  white-space: nowrap;
}

.theme-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 8px);
  width: 228px;
  padding: 12px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-primary);
  box-shadow: var(--shadow-lg);
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
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.theme-mode-switch button.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
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
  transition: transform 0.15s ease, border-color 0.15s ease;
}

.theme-color-dot:hover {
  transform: scale(1.05);
}

.theme-color-dot.active {
  border-color: var(--color-text-primary);
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
  color: hsl(var(--vscode-chrome-muted));
  cursor: pointer;
  transition: background-color 0.12s, color 0.12s;
}

.control-btn:hover {
  background: hsl(var(--vscode-chrome-hover));
  color: hsl(var(--vscode-chrome-foreground));
}

.close-btn:hover {
  background: #e81123;
  color: white;
}

.app-body {
  display: flex;
  flex: 1;
  overflow: hidden;
  background: transparent;
}

.sidebar {
  width: var(--sidebar-width);
  background: hsl(var(--vscode-activitybar));
  border-right: 1px solid hsl(var(--vscode-chrome-border));
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 8px 8px 10px;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  padding: 2px 0 4px;
  gap: 4px;
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  gap: 5px;
  width: 100%;
  min-height: 60px;
  margin: 0 auto;
  color: hsl(var(--vscode-chrome-muted));
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all 0.15s ease;
}

.nav-item:hover {
  color: hsl(var(--vscode-chrome-foreground));
  background: hsl(var(--vscode-chrome-hover));
  border-color: color-mix(in srgb, hsl(var(--vscode-chrome-border)) 80%, transparent);
}

.nav-item.active {
  color: hsl(var(--vscode-chrome-foreground));
  background: hsl(var(--vscode-chrome-active));
  border-color: color-mix(in srgb, var(--color-accent) 16%, hsl(var(--vscode-chrome-border)));
  box-shadow: var(--shadow-sm);
}

.nav-item.active::before {
  content: '';
  position: absolute;
  left: 50%;
  top: 0;
  transform: translateX(-50%);
  width: 20px;
  height: 2px;
  background: var(--color-accent);
  border-radius: 999px;
}

.sidebar-foot {
  padding-top: 10px;
  display: flex;
  justify-content: center;
  border-top: 1px solid hsl(var(--vscode-chrome-border));
}

.nav-item-label {
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
  letter-spacing: 0.01em;
}

.nav-item:focus-visible {
  outline: 1px solid var(--color-accent);
  outline-offset: -1px;
}

.sidebar-foot .nav-item:focus-visible {
  outline: 1px solid var(--color-accent);
  outline-offset: -1px;
}

.main-content {
  flex: 1;
  overflow: hidden;
  min-width: 0;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--color-accent) 3%, transparent), transparent 180px),
    var(--color-bg-primary);
}

/* 底部状态栏 */
.app-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 24px;
  padding: 0 8px;
  background: hsl(var(--vscode-statusbar));
  color: hsl(var(--vscode-statusbar-foreground));
  border-top: 1px solid hsl(var(--vscode-chrome-border));
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
  font-weight: 700;
  opacity: 0.8;
}

/* 底部状态栏徽章适配 */
.app-footer :deep(div[role="status"]) {
  height: 18px !important;
  padding: 0 4px !important;
  font-size: 10px !important;
  color: inherit;
}

.app-footer :deep(div[role="status"] > div:first-child) {
  background: currentColor !important;
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
  color: hsl(var(--vscode-chrome-muted));
}

.custom-header :deep(.header-tool-btn:hover) {
  color: hsl(var(--vscode-chrome-foreground));
  background: hsl(var(--vscode-chrome-hover));
}

.custom-header :deep(.header-tool-btn:active) {
  background: hsl(var(--vscode-chrome-active));
}

.custom-header :deep(.header-tool-btn:focus-visible) {
  box-shadow: 0 0 0 1px hsl(var(--ring));
}
</style>
