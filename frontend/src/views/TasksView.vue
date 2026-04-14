<template>
  <div class="page-shell">
    <div class="page-frame">
      <section class="surface-panel page-panel tasks-page flex flex-col w-full min-w-0">
          <ManagePage
            title="定时任务"
            description="统一管理周期执行的定时任务，不展示立即执行任务。"
            :icon="ClockIcon"
            :columns="columns"
            :data="filteredTasks"
            :loading="loading"
            :metrics="metrics"
            :filters="filterConfig"
            :primary-action="{ key: 'add', label: '新建任务' }"
            @search="handleSearch"
            @filter-change="handleFilterChange"
            @action="handleAction"
            @refresh="loadTasks"
          >
            <template #cell-name="{ row: task }">
              <div class="flex items-center gap-3">
                <div :class="[
                  'w-10 h-10 rounded-md flex items-center justify-center transition-all',
                  task.enabled
                    ? 'bg-green-500/15 text-green-500'
                    : 'bg-gray-500/10 text-gray-500',
                ]">
                  <component :is="task.enabled ? PlayIcon : PauseIcon" :size="20" />
                </div>
                <div>
                  <div class="flex items-center gap-2">
                    <span class="font-semibold text-foreground">{{ task.name }}</span>
                  </div>
                  <p v-if="task.description" class="text-xs text-muted-foreground mt-1 line-clamp-1">
                    {{ task.description }}
                  </p>
                  <p v-if="task.content" class="text-xs text-primary/80 mt-1 flex items-start gap-1">
                    <MessageSquareIcon :size="10" class="mt-0.5 flex-shrink-0" />
                    <span class="line-clamp-1">{{ task.content }}</span>
                  </p>
                </div>
              </div>
            </template>

            <template #cell-status="{ row: task }">
              <Badge :variant="task.enabled ? 'default' : 'secondary'"
                :class="task.enabled ? 'bg-green-500/15 text-green-500 border-transparent' : 'bg-gray-500/15 text-gray-500 border-transparent'">
                {{ task.enabled ? '运行中' : '已暂停' }}
              </Badge>
            </template>

            <template #cell-cron="{ row: task }">
              <code class="bg-secondary px-1.5 py-0.5 rounded font-mono text-xs">{{ task.cron_expr }}</code>
            </template>

            <template #cell-channel="{ row: task }">
              <div class="space-y-1 text-xs text-muted-foreground">
                <div class="flex items-center gap-1">
                  <component :is="getChannelIcon(task.channel)" :size="10" />
                  {{ getChannelLabel(task.channel) }}
                </div>
                <div v-if="task.session_id" class="flex items-center gap-1">
                  <HashIcon :size="10" />
                  {{ task.session_id.substring(0, 8) }}...
                </div>
                <div v-else class="text-yellow-500 flex items-center gap-1">
                  <HashIcon :size="10" />
                  未绑定会话
                </div>
              </div>
            </template>

            <template #cell-warning="{ row: task }">
              <Badge v-if="!task.session_id && task.channel !== 'webhook'" variant="secondary"
                class="bg-yellow-500/15 text-yellow-500 border-transparent">
                <AlertTriangleIcon :size="10" />
                缺少会话ID
              </Badge>
            </template>

            <template #cell-last-run="{ row: task }">
              <span v-if="task.last_run_at" class="text-xs text-muted-foreground">
                {{ formatLastRun(task.last_run_at) }}
                <span v-if="task.last_run_status === 'success'" class="text-green-500">成功</span>
                <span v-else-if="task.last_run_status === 'failed'" class="text-red-500">失败</span>
              </span>
              <span v-else class="text-xs text-muted-foreground">从未</span>
            </template>

            <template #cell-actions="{ row: task }">
              <div class="flex items-center gap-1 justify-end">
                <IconButton
                  @click="toggleTask(task)"
                  :variant="task.enabled ? 'status-warning' : 'status-success'"
                  size="sm"
                  :title="task.enabled ? '暂停' : '启用'">
                  <component :is="task.enabled ? PauseIcon : PlayIcon" :size="14" />
                </IconButton>
                <IconButton
                  @click="executeTaskHandler(task.id)"
                  variant="primary"
                  size="sm"
                  title="立即执行">
                  <ZapIcon :size="14" />
                </IconButton>
                <IconButton
                  @click="editTask(task)"
                  variant="default"
                  size="sm"
                  title="编辑">
                  <EditIcon :size="14" />
                </IconButton>
                <IconButton
                  @click="deleteTask(task.id)"
                  variant="destructive"
                  size="sm"
                  title="删除">
                  <TrashIcon :size="14" />
                </IconButton>
              </div>
            </template>
          </ManagePage>
      </section>
    </div>

    <!-- 新建/编辑任务弹窗 -->
    <ModalDialog v-model:visible="dialogVisible" :title="editingTask ? '编辑任务' : '新建任务'" size="md" :loading="saving"
      :confirm-disabled="!taskForm.name || !taskForm.cron_expr || !taskForm.channel || (taskForm.channel !== 'webhook' && !taskForm.session_id)" confirm-text="保存" loading-text="保存中..."
      @confirm="saveTask">
      <div class="dialog-body">
        <section class="dialog-section">
          <h4 class="dialog-section-title">基本信息</h4>
          <Input v-model="taskForm.name" label="任务名称" type="text" placeholder="请输入任务名称" />

          <Textarea
            v-model="taskForm.content"
            label="任务内容"
            rows="3"
            placeholder="请输入任务内容（消息文本）"
            description="任务执行时发送的消息内容"
          />

          <Input
            v-model="taskForm.description"
            label="任务描述"
            type="text"
            placeholder="请输入任务描述（可选）"
          />
        </section>

        <section class="dialog-section">
          <h4 class="dialog-section-title">调度与投递</h4>
          <div>
            <label class="block text-sm text-muted-foreground mb-2">Cron 表达式</label>
            <Input v-model="taskForm.cron_expr" type="text" placeholder="* * * * * (分 时 日 月 周)" class="font-mono" />
            <div class="flex flex-wrap gap-2 mt-2">
              <Button v-for="preset in cronPresets" :key="preset.value"
                @click="taskForm.cron_expr = preset.value"
                variant="outline" size="sm">
                {{ preset.label }}
              </Button>
            </div>
          </div>

          <div>
            <label class="block text-sm text-muted-foreground mb-2">渠道类型</label>
            <div class="grid grid-cols-3 gap-2 sm:grid-cols-4 xl:grid-cols-7">
              <button v-for="ch in channels" :key="ch.value" @click="taskForm.channel = ch.value" :class="getTaskChannelButtonClass(ch)">
                <div :class="getTaskChannelIconWrapperClass(ch)">
                  <component :is="ch.icon" :size="18" :class="getTaskChannelIconClass(ch)" />
                </div>
                <span class="text-xs">{{ ch.label }}</span>
              </button>
            </div>
          </div>

          <div>
            <label class="block text-sm text-muted-foreground mb-2">会话ID</label>
            <Input v-model="taskForm.session_id" type="text"
              :placeholder="taskForm.channel && taskForm.channel !== 'webhook' ? '请输入会话ID（必填）' : '请输入会话ID（可选）'"
              class="font-mono"
              :class="!taskForm.session_id && taskForm.channel && taskForm.channel !== 'webhook'
                ? 'border-yellow-500/50 focus:border-yellow-500'
                : ''" />
            <p class="dialog-helper mt-2" :class="taskForm.channel && taskForm.channel !== 'webhook' && !taskForm.session_id ? 'text-yellow-500' : ''">
              {{ taskForm.channel && taskForm.channel !== 'webhook' && !taskForm.session_id ? '必填：聊天渠道需要绑定会话ID才能发送消息' : '绑定到指定会话，任务执行时将在该会话中进行' }}
            </p>
          </div>
        </section>

        <section class="dialog-section">
          <h4 class="dialog-section-title">扩展参数</h4>
          <div>
            <label class="block text-sm text-muted-foreground mb-2">参数 (JSON格式)</label>
            <Textarea v-model="taskForm.params" rows="3" placeholder='{"key": "value"}' class="font-mono" />
          </div>

          <label class="flex items-center gap-3 rounded-md border border-border bg-background/70 p-3">
            <input v-model="taskForm.enabled" type="checkbox" id="enabled" class="w-4 h-4 rounded border-border bg-background text-primary focus:ring-primary" />
            <span class="text-sm text-muted-foreground">创建后立即启用</span>
          </label>
        </section>
      </div>
    </ModalDialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from "vue";
import {
  Clock as ClockIcon,
  Plus as PlusIcon,
  Edit as EditIcon,
  Trash as TrashIcon,
  Play as PlayIcon,
  Pause as PauseIcon,
  Zap as ZapIcon,
  Terminal as TerminalIcon,
  Calendar as CalendarIcon,
  MessageSquare as MessageSquareIcon,
  Send as SendIcon,
  Hash as HashIcon,
  AlertTriangle as AlertTriangleIcon,
} from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";
import { ManagePage } from "@/components/layout";
import { Input, Textarea, Select, Button, Badge, IconButton } from "@/components/ui";
import {
  getTasks,
  createTask,
  updateTask,
  deleteTask as apiDeleteTask,
  toggleTask as apiToggleTask,
  executeTask as apiExecuteTask,
} from "@/services/api.js";
import { useConfirm } from "@/composables/useConfirm.js";
import { useToast } from "@/composables/useToast.js";

const { confirm } = useConfirm();
const { toast } = useToast();

const loading = ref(true);
const tasks = ref([]);
const showAddDialog = ref(false);
const editingTask = ref(null);
const saving = ref(false);
const searchQuery = ref("");
const filterStatus = ref("");
const filterChannel = ref("");

const cronPresets = [
  { label: "每分钟", value: "* * * * *" },
  { label: "每5分钟", value: "*/5 * * * *" },
  { label: "每15分钟", value: "*/15 * * * *" },
  { label: "每小时", value: "0 * * * *" },
  { label: "每天凌晨", value: "0 0 * * *" },
  { label: "每天8点", value: "0 8 * * *" },
  { label: "每周一", value: "0 0 * * 1" },
  { label: "每月1号", value: "0 0 1 * *" },
];

const channels = [
  { label: "WebSocket", value: "websocket", icon: MessageSquareIcon },
  { label: "QQ", value: "qq", icon: MessageSquareIcon },
  { label: "icoo_proxy", value: "icoo_proxy", icon: SendIcon },
  { label: "飞书", value: "feishu", icon: SendIcon },
  { label: "钉钉", value: "dingtalk", icon: SendIcon },
  { label: "Webhook", value: "webhook", icon: HashIcon },
  { label: "Telegram", value: "telegram", icon: SendIcon },
];
const taskStatusOptions = [
  { label: "全部状态", value: "" },
  { label: "运行中", value: "enabled" },
  { label: "已暂停", value: "disabled" },
];
const taskChannelOptions = [
  { label: "全部渠道", value: "" },
  { label: "WebSocket", value: "websocket" },
  { label: "QQ", value: "qq" },
  { label: "icoo_proxy", value: "icoo_proxy" },
  { label: "飞书", value: "feishu" },
  { label: "钉钉", value: "dingtalk" },
  { label: "Webhook", value: "webhook" },
  { label: "Telegram", value: "telegram" },
];

const columns = [
  { key: "name", title: "任务名称", width: "280px" },
  { key: "status", title: "状态", width: "90px" },
  { key: "cron", title: "Cron", width: "120px" },
  { key: "channel", title: "渠道", width: "140px" },
  { key: "warning", title: "警告", width: "100px" },
  { key: "last-run", title: "上次执行", width: "120px" },
  { key: "actions", title: "操作", align: "right", width: "160px" },
];

const metrics = computed(() => [
  {
    icon: CalendarIcon,
    iconColor: "text-primary",
    iconBg: "bg-primary/10",
    value: scheduledTasks.value.length,
    label: "总任务数",
  },
  {
    icon: PlayIcon,
    iconColor: "text-green-500",
    iconBg: "bg-green-500/10",
    value: enabledCount.value,
    label: "运行中",
  },
  {
    icon: PauseIcon,
    iconColor: "text-gray-500",
    iconBg: "bg-gray-500/10",
    value: scheduledTasks.value.length - enabledCount.value,
    label: "已暂停",
  },
  {
    icon: ZapIcon,
    iconColor: "text-purple-500",
    iconBg: "bg-purple-500/10",
    value: channelStats.value,
    label: "活跃渠道",
  },
]);

const filterConfig = [
  {
    key: "status",
    placeholder: "全部状态",
    options: [
      { label: "全部状态", value: "" },
      { label: "运行中", value: "enabled" },
      { label: "已暂停", value: "disabled" },
    ],
  },
  {
    key: "channel",
    placeholder: "全部渠道",
    options: [{ label: "全部渠道", value: "" }, ...taskChannelOptions.slice(1)],
  },
];

const scheduledTasks = computed(() =>
  tasks.value.filter((task) => normalizeTaskType(task.task_type) === "scheduled"),
);
const enabledCount = computed(() => scheduledTasks.value.filter((t) => t.enabled).length);
const channelStats = computed(() =>
  new Set(scheduledTasks.value.filter((t) => t.enabled).map((t) => t.channel)).size,
);

const filteredTasks = computed(() => {
  let result = scheduledTasks.value;
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    result = result.filter(
      (t) =>
        t.name?.toLowerCase().includes(query) ||
        t.description?.toLowerCase().includes(query)
    );
  }
  if (filterStatus.value) {
    result =
      filterStatus.value === "enabled"
        ? result.filter((t) => t.enabled)
        : result.filter((t) => !t.enabled);
  }
  if (filterChannel.value) {
    result = result.filter((t) => t.channel === filterChannel.value);
  }
  return result;
});

function handleSearch(value) {
  searchQuery.value = value;
}

function handleFilterChange({ key, value }) {
  if (key === "status") {
    filterStatus.value = value;
  } else if (key === "channel") {
    filterChannel.value = value;
  }
}

function handleAction({ action, row }) {
  if (action === "add") {
    openAddDialog();
  }
}

function getChannelIcon(channel) {
  return channels.find((c) => c.value === normalizeTaskChannel(channel))?.icon || SendIcon;
}

function getChannelLabel(channel) {
  return channels.find((c) => c.value === normalizeTaskChannel(channel))?.label || channel;
}

function getTaskChannelTheme(channel) {
  const value = normalizeTaskChannel(channel);
  const themes = {
    websocket: {
      selected: "border-cyan-500/50 bg-cyan-500/10 text-cyan-300",
      idle: "border-border bg-secondary text-muted-foreground hover:border-cyan-500/35 hover:bg-cyan-500/5 hover:text-cyan-200",
      iconWrapSelected: "bg-cyan-500/15",
      iconWrapIdle: "bg-background",
      iconSelected: "text-cyan-300",
      iconIdle: "text-cyan-400/80",
    },
    qq: {
      selected: "border-green-500/45 bg-green-500/10 text-green-300",
      idle: "border-border bg-secondary text-muted-foreground hover:border-green-500/35 hover:bg-green-500/5 hover:text-green-200",
      iconWrapSelected: "bg-green-500/15",
      iconWrapIdle: "bg-background",
      iconSelected: "text-green-300",
      iconIdle: "text-green-400/80",
    },
    icoo_proxy: {
      selected: "border-emerald-500/45 bg-emerald-500/10 text-emerald-300",
      idle: "border-border bg-secondary text-muted-foreground hover:border-emerald-500/35 hover:bg-emerald-500/5 hover:text-emerald-200",
      iconWrapSelected: "bg-emerald-500/15",
      iconWrapIdle: "bg-background",
      iconSelected: "text-emerald-300",
      iconIdle: "text-emerald-400/80",
    },
    feishu: {
      selected: "border-blue-500/45 bg-blue-500/10 text-blue-300",
      idle: "border-border bg-secondary text-muted-foreground hover:border-blue-500/35 hover:bg-blue-500/5 hover:text-blue-200",
      iconWrapSelected: "bg-blue-500/15",
      iconWrapIdle: "bg-background",
      iconSelected: "text-blue-300",
      iconIdle: "text-blue-400/80",
    },
    dingtalk: {
      selected: "border-sky-500/45 bg-sky-500/10 text-sky-300",
      idle: "border-border bg-secondary text-muted-foreground hover:border-sky-500/35 hover:bg-sky-500/5 hover:text-sky-200",
      iconWrapSelected: "bg-sky-500/15",
      iconWrapIdle: "bg-background",
      iconSelected: "text-sky-300",
      iconIdle: "text-sky-400/80",
    },
    webhook: {
      selected: "border-purple-500/45 bg-purple-500/10 text-purple-300",
      idle: "border-border bg-secondary text-muted-foreground hover:border-purple-500/35 hover:bg-purple-500/5 hover:text-purple-200",
      iconWrapSelected: "bg-purple-500/15",
      iconWrapIdle: "bg-background",
      iconSelected: "text-purple-300",
      iconIdle: "text-purple-400/80",
    },
    telegram: {
      selected: "border-indigo-500/45 bg-indigo-500/10 text-indigo-300",
      idle: "border-border bg-secondary text-muted-foreground hover:border-indigo-500/35 hover:bg-indigo-500/5 hover:text-indigo-200",
      iconWrapSelected: "bg-indigo-500/15",
      iconWrapIdle: "bg-background",
      iconSelected: "text-indigo-300",
      iconIdle: "text-indigo-400/80",
    },
  };
  return themes[value] || {
    selected: "border-primary/45 bg-primary/10 text-primary",
    idle: "border-border bg-secondary text-muted-foreground hover:border-primary/35 hover:bg-primary/5 hover:text-foreground",
    iconWrapSelected: "bg-primary/15",
    iconWrapIdle: "bg-background",
    iconSelected: "text-primary",
    iconIdle: "text-muted-foreground",
  };
}

function getTaskChannelButtonClass(channel) {
  const selected = normalizeTaskChannel(taskForm.channel) === normalizeTaskChannel(channel.value);
  const theme = getTaskChannelTheme(channel.value);
  return [
    "rounded-md border p-3 transition-all duration-200 flex flex-col items-center justify-center gap-2 min-h-[88px]",
    selected ? theme.selected : theme.idle,
  ];
}

function getTaskChannelIconWrapperClass(channel) {
  const selected = normalizeTaskChannel(taskForm.channel) === normalizeTaskChannel(channel.value);
  const theme = getTaskChannelTheme(channel.value);
  return [
    "flex h-9 w-9 items-center justify-center rounded-md transition-colors duration-200",
    selected ? theme.iconWrapSelected : theme.iconWrapIdle,
  ];
}

function getTaskChannelIconClass(channel) {
  const selected = normalizeTaskChannel(taskForm.channel) === normalizeTaskChannel(channel.value);
  const theme = getTaskChannelTheme(channel.value);
  return selected ? theme.iconSelected : theme.iconIdle;
}

function normalizeTaskChannel(channel) {
  const value = String(channel || "").trim().toLowerCase();
  if (!value) return "";
  if (value === "飞书") return "feishu";
  if (value === "钉钉") return "dingtalk";
  return value;
}

function normalizeTaskType(taskType) {
  return taskType === "immediate" ? "immediate" : "scheduled";
}

function formatLastRun(timestamp) {
  if (!timestamp) return "从未";
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now - date;
  if (diff < 60000) return "刚刚";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`;
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}天前`;
  return date.toLocaleDateString("zh-CN");
}

const taskForm = reactive({
  name: "",
  content: "",
  description: "",
  cron_expr: "",
  channel: "",
  session_id: "",
  params: "",
  enabled: true,
});

// 计算属性：控制弹窗显示
const dialogVisible = computed({
  get: () => showAddDialog.value || !!editingTask.value,
  set: (val) => {
    if (!val) closeDialog();
  },
});

onMounted(() => {
  loadTasks();
});

/**
 * 加载任务列表
 */
async function loadTasks() {
  loading.value = true;
  try {
    const response = await getTasks();
    // 后端返回格式: { code, message, data: [...] }
    const data = response.data || response;
    tasks.value = Array.isArray(data)
      ? data.map((task) => ({
          ...task,
          channel: normalizeTaskChannel(task?.channel),
          task_type: normalizeTaskType(task?.task_type),
        }))
      : [];
  } catch (e) {
    console.error("加载任务失败:", e);
    tasks.value = [];
    toast("加载任务列表失败: " + (e.message || "未知错误"), "error");
  }
  loading.value = false;
}

/**
 * 打开添加任务对话框
 */
function openAddDialog() {
  editingTask.value = null;
  resetForm();
  showAddDialog.value = true;
}

/**
 * 编辑任务
 */
function editTask(task) {
  editingTask.value = task;
  taskForm.name = task.name || "";
  taskForm.content = task.content || "";
  taskForm.description = task.description || "";
  taskForm.cron_expr = task.cron_expr || "";
  taskForm.channel = normalizeTaskChannel(task.channel);
  taskForm.session_id = task.session_id || "";
  taskForm.params = task.params || "";
  taskForm.enabled = task.enabled !== false;
  showAddDialog.value = true;
}

/**
 * 重置表单
 */
function resetForm() {
  taskForm.name = "";
  taskForm.content = "";
  taskForm.description = "";
  taskForm.cron_expr = "";
  taskForm.channel = "";
  taskForm.session_id = "";
  taskForm.params = "";
  taskForm.enabled = true;
}

/**
 * 切换任务启用状态
 */
async function toggleTask(task) {
  try {
    const ok = await confirm("确定要切换任务启用状态吗？", {
      title: "切换任务状态",
      confirmText: "切换",
      type: "warning",
    });
    if (!ok) return;
    await apiToggleTask(task.id);
    toast("任务状态已切换", "success");
    // 更新本地状态
    task.enabled = !task.enabled;
  } catch (e) {
    console.error("切换任务状态失败:", e);
    toast("切换任务状态失败: " + (e.message || "未知错误"), "error");
  }
}

/**
 * 立即执行任务
 */
async function executeTaskHandler(id) {
  const ok = await confirm("确定要立即执行这个任务吗？", {
    title: "立即执行",
    confirmText: "执行",
    type: "warning",
  });
  if (!ok) return;

  try {
    await apiExecuteTask(id);
    toast("执行指令已发送", "success");
  } catch (e) {
    console.error("立即执行任务失败:", e);
    toast("执行失败: " + (e.message || "未知错误"), "error");
  }
}

/**
 * 删除任务
 */
async function deleteTask(id) {
  const ok = await confirm("确定要删除这个任务吗？此操作不可恢复。", {
    title: "删除任务",
    confirmText: "删除",
    type: "danger",
  });
  if (!ok) return;

  try {
    await apiDeleteTask(id);
    // 从本地列表中移除
    tasks.value = tasks.value.filter((t) => t.id !== id);
    toast("任务已删除", "success");
  } catch (e) {
    console.error("删除任务失败:", e);
    toast("删除任务失败: " + (e.message || "未知错误"), "error");
  }
}

/**
 * 关闭对话框
 */
function closeDialog() {
  showAddDialog.value = false;
  editingTask.value = null;
  resetForm();
}

/**
 * 保存任务
 */
async function saveTask() {
  if (!taskForm.name || !taskForm.cron_expr || !taskForm.channel) {
    toast("请填写完整信息（任务名称、Cron表达式、通道为必填项）", "warning");
    return;
  }

  // 验证 JSON 参数格式
  if (taskForm.params) {
    try {
      JSON.parse(taskForm.params);
    } catch (e) {
      toast("参数格式错误，请输入有效的 JSON 格式", "warning");
      return;
    }
  }

  saving.value = true;

  try {
    const taskData = {
      name: taskForm.name,
      content: taskForm.content,
      description: taskForm.description,
      cron_expr: taskForm.cron_expr,
      channel: normalizeTaskChannel(taskForm.channel),
      session_id: taskForm.session_id,
      params: taskForm.params,
      enabled: taskForm.enabled,
    };

    if (editingTask.value) {
      // 更新现有任务
      taskData.id = editingTask.value.id;
      const response = await updateTask(taskData);
      const updatedTask = response.data || taskData;

      toast("任务已更新", "success");

      // 更新本地列表
      const index = tasks.value.findIndex((t) => t.id === editingTask.value.id);
      if (index !== -1) {
        tasks.value[index] = {
          ...tasks.value[index],
          ...updatedTask,
          channel: normalizeTaskChannel(updatedTask.channel ?? taskData.channel),
        };
      }
    } else {
      // 创建新任务
      const response = await createTask(taskData);
      const newTask = {
        ...(response.data || taskData),
        channel: normalizeTaskChannel(response.data?.channel ?? taskData.channel),
      };
      tasks.value.push(newTask);
      toast("任务已创建", "success");
    }

    closeDialog();
  } catch (e) {
    console.error("保存任务失败:", e);
    toast("保存任务失败: " + (e.message || "未知错误"), "error");
  }

  saving.value = false;
}
</script>

<style scoped>
.tasks-page {
  min-height: 0;
  overflow: hidden;
}

</style>

