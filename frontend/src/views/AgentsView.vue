<template>
  <div class="page-shell">
    <div class="page-frame">
      <section class="surface-panel page-panel agents-page flex flex-col w-full min-w-0">
          <ManagePage
            title="智能体管理"
            description="维护 master 与 subagent 两类智能体，默认智能体仅允许使用 master。"
            :icon="BotIcon"
            :columns="columns"
            :data="filteredAgents"
            :loading="loading"
            :metrics="metrics"
            :filters="filterConfig"
            :primary-action="{ key: 'add', label: '新建智能体' }"
            @search="handleSearch"
            @filter-change="handleFilterChange"
            @action="handleAction"
            @refresh="loadData"
          >
            <template #cell-name="{ row: agent }">
              <div class="flex items-start gap-3 min-w-0">
                <div
                  :class="[
                    'w-10 h-10 rounded-md flex items-center justify-center transition-all flex-shrink-0',
                    agent.enabled
                      ? 'bg-primary/20 text-primary'
                      : 'bg-slate-500/10 text-slate-500',
                  ]"
                >
                  <BotIcon :size="20" />
                </div>
                <div class="min-w-0">
                  <div class="flex items-center gap-2 flex-wrap">
                    <span class="font-semibold text-foreground">{{ agent.name }}</span>
                  </div>
                  <p v-if="agent.description" class="text-sm text-muted-foreground mt-1 line-clamp-2">
                    {{ agent.description }}
                  </p>
                  <p v-else class="text-sm text-muted-foreground mt-1">暂无描述</p>
                </div>
              </div>
            </template>

            <template #cell-type="{ row: agent }">
              <Badge variant="secondary">
                {{ formatAgentType(agent.type) }}
              </Badge>
            </template>

            <template #cell-status="{ row: agent }">
              <Badge :variant="agent.enabled ? 'default' : 'outline'">
                {{ agent.enabled ? "已启用" : "未启用" }}
              </Badge>
            </template>

            <template #cell-default="{ row: agent }">
              <Badge
                v-if="defaultAgent?.agent_id === agent.id"
                class="bg-amber-500/10 text-amber-500 border-amber-500/20 hover:bg-amber-500/10"
              >
                默认
              </Badge>
            </template>

            <template #cell-prompt="{ row: agent }">
              <div class="space-y-1 text-xs text-muted-foreground">
                <div class="flex items-start gap-1.5">
                  <MessageSquareTextIcon :size="10" class="mt-0.5 flex-shrink-0" />
                  <span class="line-clamp-2">{{ agent.system_prompt || "未设置额外系统提示词" }}</span>
                </div>
                <div class="flex items-start gap-1.5">
                  <BracesIcon :size="10" class="mt-0.5 flex-shrink-0" />
                  <span class="font-mono break-all">
                    {{ formatMetadata(agent.metadata) }}
                  </span>
                </div>
              </div>
            </template>

            <template #cell-actions="{ row: agent }">
              <div class="flex items-center gap-1 justify-end">
                <IconButton
                  v-if="defaultAgent?.agent_id !== agent.id && agent.type === 'master'"
                  @click="handleSetDefault(agent)"
                  variant="ghost"
                  size="sm"
                  title="设为默认"
                >
                  <StarIcon :size="14" />
                </IconButton>
                <IconButton
                  @click="editAgent(agent)"
                  variant="ghost"
                  size="sm"
                  title="编辑"
                >
                  <EditIcon :size="14" />
                </IconButton>
                <IconButton
                  @click="removeAgent(agent)"
                  variant="destructive"
                  size="sm"
                  title="删除"
                >
                  <TrashIcon :size="14" />
                </IconButton>
              </div>
            </template>
          </ManagePage>
      </section>
    </div>

    <ModalDialog
      v-model:visible="dialogVisible"
      :title="editingAgent ? '编辑智能体' : '新建智能体'"
      size="lg"
      :loading="saving"
      :confirm-disabled="!agentForm.name"
      confirm-text="保存"
      loading-text="保存中..."
      @confirm="saveAgent"
    >
      <div class="dialog-body">
        <section class="dialog-section">
          <h4 class="dialog-section-title">基本信息</h4>
          <Input
            v-model="agentForm.name"
            label="智能体名称"
            placeholder="例如：customer-support"
          />

          <div>
            <label class="block text-sm text-muted-foreground mb-2">智能体类型</label>
            <div class="grid grid-cols-2 gap-2">
              <Button
                v-for="typeOption in agentTypes"
                :key="typeOption.value"
                type="button"
                @click="agentForm.type = typeOption.value"
                :variant="agentForm.type === typeOption.value ? 'default' : 'outline'"
                size="default"
                class="text-left justify-start h-auto p-3 flex-col items-start"
              >
                <span class="font-medium">{{ typeOption.label }}</span>
                <span class="text-xs opacity-80 font-normal">{{ typeOption.description }}</span>
              </Button>
            </div>
          </div>

          <Input
            v-model="agentForm.description"
            label="描述"
            placeholder="简要说明这个智能体负责什么"
          />

          <p class="dialog-note">
            `master` 用于主对话与默认智能体；`subagent` 用于被主智能体委派执行子任务。
          </p>
        </section>

        <section class="dialog-section">
          <h4 class="dialog-section-title">行为配置</h4>
          <Textarea
            v-model="agentForm.system_prompt"
            label="系统提示词"
            placeholder="请输入额外系统提示词"
            :rows="7"
          />

          <div>
            <Textarea
              v-model="agentForm.metadata"
              label="元数据 (JSON)"
              placeholder='{"team":"ops","scene":"support"}'
              :rows="4"
              class="font-mono"
            />
            <p class="dialog-helper mt-2">可选，用于保存标签、场景和扩展属性。</p>
          </div>
        </section>

        <section class="dialog-section">
          <h4 class="dialog-section-title">状态选项</h4>
          <label class="flex items-center gap-3 rounded-md border border-border bg-background/70 p-3">
            <input
              v-model="agentForm.enabled"
              type="checkbox"
              class="w-4 h-4 rounded border-border bg-background text-primary focus:ring-primary"
            />
            <span class="text-sm text-muted-foreground">启用该智能体</span>
          </label>
        </section>
      </div>
    </ModalDialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import {
  Bot as BotIcon,
  Braces as BracesIcon,
  CheckCircle as CheckCircleIcon,
  Edit as EditIcon,
  GitBranch as GitBranchIcon,
  MessageSquareText as MessageSquareTextIcon,
  Star as StarIcon,
  Trash as TrashIcon,
} from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";
import { ManagePage } from "@/components/layout";
import { Button, IconButton, Badge, Input, Textarea } from "@/components/ui";
import {
  createAgent,
  deleteAgent,
  getAgents,
  getDefaultAgent,
  setDefaultAgent,
  updateAgent,
} from "@/services/api.js";
import { useConfirm } from "@/composables/useConfirm.js";
import { useToast } from "@/composables/useToast.js";

const { confirm } = useConfirm();
const { toast } = useToast();

const loading = ref(true);
const saving = ref(false);
const showDialog = ref(false);
const searchQuery = ref("");
const filterStatus = ref("");
const filterType = ref("");
const agents = ref([]);
const defaultAgent = ref(null);
const editingAgent = ref(null);

const agentTypes = [
  { label: "Master", value: "master", description: "主对话入口，可设为默认智能体。" },
  { label: "Subagent", value: "subagent", description: "用于任务委派，不参与默认智能体选择。" },
];

const columns = [
  { key: "name", title: "名称", width: "280px" },
  { key: "type", title: "类型", width: "100px" },
  { key: "status", title: "状态", width: "80px" },
  { key: "default", title: "默认", width: "60px" },
  { key: "prompt", title: "提示词与元数据" },
  { key: "actions", title: "操作", align: "right", width: "120px" },
];

const metrics = computed(() => [
  {
    icon: BotIcon,
    iconColor: "text-primary",
    iconBg: "bg-primary/10",
    value: agents.value.length,
    label: "智能体总数",
  },
  {
    icon: CheckCircleIcon,
    iconColor: "text-green-500",
    iconBg: "bg-green-500/10",
    value: masterCount.value,
    label: "Master 智能体",
  },
  {
    icon: GitBranchIcon,
    iconColor: "text-slate-500",
    iconBg: "bg-slate-500/10",
    value: subAgentCount.value,
    label: "Subagent 智能体",
  },
  {
    icon: StarIcon,
    iconColor: "text-amber-500",
    iconBg: "bg-amber-500/10",
    value: defaultAgent.value?.name || "未设置",
    label: "默认智能体",
  },
]);

const filterConfig = [
  {
    key: "type",
    placeholder: "全部类型",
    options: [
      { label: "全部类型", value: "" },
      { label: "Master", value: "master" },
      { label: "Subagent", value: "subagent" },
    ],
  },
  {
    key: "status",
    placeholder: "全部状态",
    options: [
      { label: "全部状态", value: "" },
      { label: "已启用", value: "enabled" },
      { label: "未启用", value: "disabled" },
    ],
  },
];

const agentForm = reactive({
  name: "",
  type: "master",
  description: "",
  system_prompt: "",
  metadata: "{}",
  enabled: true,
});

const masterCount = computed(() => agents.value.filter((item) => item.type !== "subagent").length);
const subAgentCount = computed(() => agents.value.filter((item) => item.type === "subagent").length);

const filteredAgents = computed(() => {
  let result = agents.value;

  if (searchQuery.value) {
    const keyword = searchQuery.value.toLowerCase();
    result = result.filter(
      (item) =>
        item.name?.toLowerCase().includes(keyword) ||
        item.description?.toLowerCase().includes(keyword),
    );
  }

  if (filterStatus.value === "enabled") {
    result = result.filter((item) => item.enabled);
  } else if (filterStatus.value === "disabled") {
    result = result.filter((item) => !item.enabled);
  }
  if (filterType.value) {
    result = result.filter((item) => (item.type || "master") === filterType.value);
  }

  return result;
});

function handleSearch(value) {
  searchQuery.value = value;
}

function handleFilterChange({ key, value }) {
  if (key === "status") {
    filterStatus.value = value;
  } else if (key === "type") {
    filterType.value = value;
  }
}

function handleAction({ action, row }) {
  if (action === "add") {
    openAddDialog();
  }
}

const dialogVisible = computed({
  get: () => showDialog.value || !!editingAgent.value,
  set: (value) => {
    if (!value) closeDialog();
  },
});

onMounted(() => {
  loadData();
});

async function loadData() {
  loading.value = true;
  try {
    const [agentResponse, defaultResponse] = await Promise.all([
      getAgents(),
      getDefaultAgent().catch(() => ({ data: null })),
    ]);
    agents.value = normalizeAgents(agentResponse.data);
    defaultAgent.value = defaultResponse.data || null;
  } catch (error) {
    console.error("加载智能体失败:", error);
    agents.value = [];
    defaultAgent.value = null;
    toast("加载智能体列表失败: " + (error.message || "未知错误"), "error");
  }
  loading.value = false;
}

function formatMetadata(metadata) {
  if (!metadata || (typeof metadata === "object" && Object.keys(metadata).length === 0)) {
    return "无元数据";
  }

  try {
    return JSON.stringify(metadata);
  } catch {
    return "元数据格式异常";
  }
}

function formatAgentType(type) {
  return (type || "master") === "subagent" ? "Subagent" : "Master";
}

function normalizeAgent(agent) {
  return {
    ...agent,
    type: agent?.type === "subagent" ? "subagent" : "master",
  };
}

function normalizeAgents(list) {
  return Array.isArray(list) ? list.map((item) => normalizeAgent(item)) : [];
}

function openAddDialog() {
  editingAgent.value = null;
  resetForm();
  showDialog.value = true;
}

function editAgent(agent) {
  editingAgent.value = agent;
  agentForm.name = agent.name || "";
  agentForm.type = agent.type === "subagent" ? "subagent" : "master";
  agentForm.description = agent.description || "";
  agentForm.system_prompt = agent.system_prompt || "";
  agentForm.metadata = formatMetadataForEdit(agent.metadata);
  agentForm.enabled = agent.enabled !== false;
  showDialog.value = true;
}

function formatMetadataForEdit(metadata) {
  if (!metadata || (typeof metadata === "object" && Object.keys(metadata).length === 0)) {
    return "{}";
  }

  try {
    return JSON.stringify(metadata, null, 2);
  } catch {
    return "{}";
  }
}

function resetForm() {
  agentForm.name = "";
  agentForm.type = "master";
  agentForm.description = "";
  agentForm.system_prompt = "";
  agentForm.metadata = "{}";
  agentForm.enabled = true;
}

function closeDialog() {
  showDialog.value = false;
  editingAgent.value = null;
  resetForm();
}

async function saveAgent() {
  if (!agentForm.name.trim()) {
    toast("请输入智能体名称", "warning");
    return;
  }

  let metadata = {};
  const metadataText = agentForm.metadata.trim();
  if (metadataText) {
    try {
      metadata = JSON.parse(metadataText);
    } catch {
      toast("元数据格式错误，请输入有效的 JSON", "warning");
      return;
    }
  }

  saving.value = true;

  const payload = {
    name: agentForm.name.trim(),
    type: agentForm.type,
    description: agentForm.description.trim(),
    system_prompt: agentForm.system_prompt,
    metadata,
    enabled: agentForm.enabled,
  };

  try {
    if (editingAgent.value) {
      const response = await updateAgent({
        id: editingAgent.value.id,
        ...payload,
      });
      const updated = normalizeAgent(response.data || { id: editingAgent.value.id, ...payload });
      const index = agents.value.findIndex((item) => item.id === editingAgent.value.id);
      if (index !== -1) {
        agents.value[index] = { ...agents.value[index], ...updated };
      }
      if (defaultAgent.value?.agent_id === editingAgent.value.id) {
        if (updated.type !== "master") {
          defaultAgent.value = null;
        } else {
          defaultAgent.value = {
            ...defaultAgent.value,
            name: updated.name,
            type: updated.type,
            description: updated.description,
            system_prompt: updated.system_prompt,
          };
        }
      }
      toast("智能体已更新", "success");
    } else {
      const response = await createAgent(payload);
      const created = normalizeAgent(response.data || payload);
      agents.value.push(created);
      agents.value.sort((a, b) => (a.name || "").localeCompare(b.name || "", "zh-CN"));
      toast("智能体已创建", "success");
    }

    closeDialog();
  } catch (error) {
    console.error("保存智能体失败:", error);
    toast("保存智能体失败: " + (error.message || "未知错误"), "error");
  }

  saving.value = false;
}

async function handleSetDefault(agent) {
  if (agent.type !== "master") {
    toast("仅 master 类型可以设为默认智能体", "warning");
    return;
  }
  try {
    await setDefaultAgent(agent.id);
    defaultAgent.value = {
      agent_id: agent.id,
      name: agent.name,
      type: agent.type,
      description: agent.description,
      system_prompt: agent.system_prompt,
    };
    toast(`已将 ${agent.name} 设为默认智能体`, "success");
  } catch (error) {
    console.error("设置默认智能体失败:", error);
    toast("设置默认智能体失败: " + (error.message || "未知错误"), "error");
  }
}

async function removeAgent(agent) {
  const isDefault = defaultAgent.value?.agent_id === agent.id;
  const ok = await confirm(
    isDefault
      ? `当前默认智能体是“${agent.name}”，删除后默认配置将失效。确定继续吗？`
      : `确定要删除智能体“${agent.name}”吗？此操作不可恢复。`,
    {
      title: "删除智能体",
      confirmText: "删除",
      type: "danger",
    },
  );

  if (!ok) return;

  try {
    await deleteAgent(agent.id);
    agents.value = agents.value.filter((item) => item.id !== agent.id);
    if (isDefault) {
      defaultAgent.value = null;
    }
    toast("智能体已删除", "success");
  } catch (error) {
    console.error("删除智能体失败:", error);
    toast("删除智能体失败: " + (error.message || "未知错误"), "error");
  }
}
</script>

<style scoped>
.agents-page {
  min-height: 0;
  overflow: hidden;
}

</style>

