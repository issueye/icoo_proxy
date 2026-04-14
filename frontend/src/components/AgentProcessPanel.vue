<template>
  <div ref="rootRef" class="relative">
    <button
      class="agent-trigger"
      :class="statusClass"
      :disabled="busy"
      :title="buttonTitle"
      @click="togglePanel"
    >
      <LoaderIcon v-if="busy" :size="14" class="animate-spin" />
      <RocketIcon v-else :size="14" />
    </button>

    <transition name="fade">
      <div
        v-if="panelVisible"
        class="agent-panel"
      >
        <!-- 头部 -->
        <div class="agent-panel__header">
          <div class="flex-1 min-w-0">
            <p class="text-sm font-semibold text-foreground">icoo_agent 进程</p>
            <p class="mt-0.5 text-xs text-muted-foreground">
              查看状态并唤醒 Agent
            </p>
          </div>
          <Badge :variant="statusBadge.variant" class="flex-shrink-0">
            {{ statusText }}
          </Badge>
          <IconButton
            variant="ghost"
            size="sm"
            @click="panelVisible = false"
            title="关闭"
          >
            <X :size="14" />
          </IconButton>
        </div>

        <Separator />

        <!-- 主体 -->
        <div class="agent-panel__body">
          <!-- 状态指标 -->
          <div class="mt-3 grid grid-cols-2 gap-2 text-xs">
            <div class="status-card">
              <p class="text-muted-foreground">PID</p>
              <p class="mt-1 font-mono text-foreground">{{ status.pid || "-" }}</p>
            </div>
            <div class="status-card">
              <p class="text-muted-foreground">模式</p>
              <p class="mt-1 text-foreground">{{ status.managed ? "托管" : (status.healthy ? "外部" : "未启动") }}</p>
            </div>
            <div class="status-card">
              <p class="text-muted-foreground">健康检查</p>
              <p class="mt-1 text-foreground">{{ status.healthy ? "可达" : "不可达" }}</p>
            </div>
            <div class="status-card">
              <p class="text-muted-foreground">启动时间</p>
              <p class="mt-1 text-foreground">{{ startedAtLabel }}</p>
            </div>
          </div>

          <Separator class="my-3" />

          <!-- 路径信息 -->
          <div class="space-y-2 text-xs">
            <div class="info-row">
              <p class="text-muted-foreground">API Base</p>
              <p class="mt-0.5 break-all font-mono text-foreground">{{ status.apiBase || "-" }}</p>
            </div>
            <div class="info-row">
              <p class="text-muted-foreground">可执行文件</p>
              <p class="mt-0.5 break-all font-mono text-foreground">{{ status.binaryPath || "-" }}</p>
            </div>
            <div class="info-row">
              <p class="text-muted-foreground">工作目录</p>
              <p class="mt-0.5 break-all font-mono text-foreground">{{ status.workingDir || "-" }}</p>
            </div>
            <div class="info-row">
              <p class="text-muted-foreground">工作区</p>
              <p class="mt-0.5 break-all font-mono text-foreground">{{ status.workspacePath || "-" }}</p>
            </div>
            <div v-if="status.configPath" class="info-row">
              <p class="text-muted-foreground">配置文件</p>
              <p class="mt-0.5 break-all font-mono text-foreground">{{ status.configPath }}</p>
            </div>
          </div>

          <!-- 错误信息 -->
          <div v-if="status.lastError" class="mt-3 rounded-md border border-destructive/20 bg-destructive/10 px-3 py-2">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-destructive">最近错误</p>
            <p class="mt-1 break-all text-xs text-destructive">{{ status.lastError }}</p>
          </div>

          <div v-if="status.lastExit" class="mt-3 rounded-md border border-amber-500/20 bg-amber-500/10 px-3 py-2 dark:border-amber-400/20 dark:bg-amber-400/10">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-amber-600 dark:text-amber-400">最近退出</p>
            <p class="mt-1 break-all text-xs text-amber-700 dark:text-amber-300">{{ status.lastExit }}</p>
          </div>

          <div v-if="status.outputPreview" class="mt-3 rounded-md border border-border bg-muted px-3 py-2">
            <p class="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">最近输出</p>
            <pre class="mt-1 max-h-28 overflow-auto whitespace-pre-wrap break-all text-[11px] leading-5 text-muted-foreground font-mono">{{ status.outputPreview }}</pre>
          </div>

          <p v-if="!desktopReady" class="mt-3 text-[11px] text-muted-foreground">
            当前不是 Wails 桌面环境，无法直接管理本地进程。
          </p>
        </div>

        <Separator />

        <!-- 底部操作 -->
        <div class="agent-panel__footer">
          <Button
            variant="default"
            size="sm"
            :disabled="busy || !desktopReady"
            @click="wakeAgent"
          >
            <PowerIcon :size="14" />
            唤醒
          </Button>
          <Button
            variant="outline"
            size="sm"
            :disabled="busy || !desktopReady"
            @click="restartAgent"
          >
            <RotateCcwIcon :size="14" />
            重启
          </Button>
          <Button
            variant="outline"
            size="sm"
            :disabled="busy || !status.managed"
            @click="stopAgent"
          >
            <SquareIcon :size="14" />
            停止
          </Button>
          <IconButton
            variant="ghost"
            size="sm"
            :disabled="busy"
            @click="refreshStatus"
            title="刷新状态"
          >
            <RefreshCwIcon :size="14" :class="{ 'animate-spin': busy }" />
          </IconButton>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import {
  Loader as LoaderIcon,
  Power as PowerIcon,
  RefreshCw as RefreshCwIcon,
  Rocket as RocketIcon,
  RotateCcw as RotateCcwIcon,
  Square as SquareIcon,
  X,
} from "lucide-vue-next";
import { useToast } from "@/composables/useToast";
import { isWailsEnv, wailsService } from "@/services/wails";
import { Button, Badge, IconButton, Separator } from "@/components/ui";

const { toast } = useToast();

const rootRef = ref(null);
const panelVisible = ref(false);
const busy = ref(false);
const desktopReady = isWailsEnv();
const status = ref({
  managed: false,
  running: false,
  healthy: false,
  pid: 0,
  startedAt: "",
  binaryPath: "",
  configPath: "",
  workingDir: "",
  workspacePath: "",
  apiBase: "",
  lastError: "",
  lastExit: "",
  outputPreview: "",
});

let refreshTimer = null;

const statusText = computed(() => {
  if (status.value.healthy) {
    return status.value.managed ? "运行中" : "外部运行中";
  }
  if (status.value.running) {
    return "启动中";
  }
  return "未运行";
});

const statusTone = computed(() => {
  if (status.value.healthy) {
    return "success";
  }
  if (status.value.running) {
    return "warning";
  }
  return "neutral";
});

const statusClass = computed(() => {
  if (status.value.healthy) {
    return "agent-status--healthy";
  }
  if (status.value.running) {
    return "agent-status--running";
  }
  return "agent-status--idle";
});

const statusBadge = computed(() => {
  if (status.value.healthy) {
    return { variant: "default" };
  }
  if (status.value.running) {
    return { variant: "secondary" };
  }
  return { variant: "outline" };
});

const buttonLabel = computed(() => {
  if (status.value.healthy) {
    return "Agent 已就绪";
  }
  if (status.value.running) {
    return "Agent 启动中";
  }
  return "唤醒 Agent";
});

const buttonTitle = computed(() => {
  if (!desktopReady) {
    return "仅桌面版支持本地 Agent 进程管理";
  }
  return "查看并管理 icoo_agent 进程";
});

const startedAtLabel = computed(() => {
  if (!status.value.startedAt) {
    return "-";
  }
  const date = new Date(status.value.startedAt);
  if (Number.isNaN(date.getTime())) {
    return status.value.startedAt;
  }
  return date.toLocaleString();
});

function togglePanel() {
  panelVisible.value = !panelVisible.value;
  if (panelVisible.value) {
    refreshStatus();
  }
}

async function refreshStatus(silent = true) {
  if (!desktopReady) {
    return;
  }
  try {
    status.value = await wailsService.getAgentProcessStatus();
  } catch (error) {
    if (!silent) {
      toast("获取 Agent 状态失败: " + error.message, "error");
    }
  }
}

async function runAction(action, successMessage) {
  if (!desktopReady || busy.value) {
    return;
  }
  busy.value = true;
  try {
    status.value = await action();
    if (successMessage) {
      toast(successMessage, "success");
    }
  } catch (error) {
    await refreshStatus();
    toast(error.message || "操作失败", "error");
  }
  busy.value = false;
}

function wakeAgent() {
  return runAction(() => wailsService.wakeAgent(), "icoo_agent 已唤醒");
}

function stopAgent() {
  return runAction(() => wailsService.stopAgent(), "icoo_agent 已停止");
}

function restartAgent() {
  return runAction(() => wailsService.restartAgent(), "icoo_agent 已重启");
}

function handleDocumentClick(event) {
  if (!panelVisible.value || !rootRef.value) {
    return;
  }
  if (!rootRef.value.contains(event.target)) {
    panelVisible.value = false;
  }
}

onMounted(() => {
  refreshStatus();
  refreshTimer = window.setInterval(() => {
    refreshStatus();
  }, 5000);
  document.addEventListener("mousedown", handleDocumentClick);
});

onBeforeUnmount(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer);
  }
  document.removeEventListener("mousedown", handleDocumentClick);
});
</script>

<style scoped>
/* 触发器按钮 */
.agent-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 24px;
  width: 28px;
  padding: 0;
  background: transparent;
  color: var(--color-text-muted);
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
  transition: all 0.12s;
}

.agent-trigger:hover:not(:disabled) {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.agent-trigger:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.agent-trigger.agent-status--healthy {
  color: var(--color-success);
}

.agent-trigger.agent-status--running {
  color: var(--color-warning);
}

.agent-trigger.agent-status--idle {
  color: var(--color-text-muted);
}

/* 面板 */
.agent-panel {
  position: absolute;
  right: 0;
  top: calc(100% + 6px);
  z-index: 50;
  width: 320px;
  max-height: min(540px, calc(100vh - 70px));
  display: flex;
  flex-direction: column;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
}

.agent-panel__header {
  flex-shrink: 0;
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 8px 10px 6px;
}

.agent-panel__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 0 10px 10px 10px;
}

.agent-panel__footer {
  flex-shrink: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 6px 10px;
  background: var(--color-bg-tertiary);
  border-radius: 0 0 var(--radius) var(--radius);
}

/* 状态卡片 */
.status-card {
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  background: var(--color-bg-tertiary);
  padding: 5px 8px;
}

/* 信息行 */
.info-row {
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  background: var(--color-bg-tertiary);
  padding: 5px 8px;
}

/* 底部按钮 */
.agent-panel__footer :deep(button) {
  height: 26px !important;
  padding: 0 8px !important;
  font-size: 11px !important;
}

.agent-panel__footer :deep(svg) {
  width: 12px !important;
  height: 12px !important;
}

/* 过渡动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>

