<template>
  <div class="providers-view app-page">
    <PageHeader
      title="供应商管理"
      description="使用表格管理供应商资产、接入模式、模型数量和运行状态，提升检索与编辑效率。"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openAddDialog">
          <Plus :size="14" />
          添加供应商
        </button>
      </template>
    </PageHeader>

    <section class="toolbar-surface providers-toolbar">
      <div class="toolbar-group providers-toolbar-main">
        <div class="toolbar-field">
          <label class="toolbar-label">搜索</label>
          <input v-model="keyword" class="form-input toolbar-input" placeholder="按名称或 API Base 搜索" />
        </div>
        <div class="toolbar-field">
          <label class="toolbar-label">类型</label>
          <Select v-model="typeFilter" :options="typeFilterOptions" class="toolbar-select" />
        </div>
        <div class="toolbar-field">
          <label class="toolbar-label">状态</label>
          <Select v-model="statusFilter" :options="statusFilterOptions" class="toolbar-select" />
        </div>
      </div>
      <div class="toolbar-summary">
        <span class="toolbar-chip">共 {{ filteredProviders.length }} 条</span>
        <span class="toolbar-chip">健康 {{ healthyProviders }} 条</span>
      </div>
    </section>

    <section v-if="providerStore.providers.length === 0" class="empty-state providers-empty">
      <Cpu :size="40" />
      <div class="providers-empty-title">暂未配置供应商</div>
      <p>先添加第一个供应商后，才能配置模型映射与统一路由策略。</p>
      <button class="btn btn-primary" @click="openAddDialog">
        <Plus :size="14" />
        添加第一个供应商
      </button>
    </section>

    <section v-else class="table-panel">
      <DataTable
        :columns="columns"
        :data="filteredProviders"
        :loading="providerStore.loading"
        empty-text="没有符合筛选条件的供应商"
        row-key="id"
      >
        <template #cell-name="{ row }">
          <div class="provider-cell-main">
            <span class="provider-name">{{ row.name }}</span>
            <span class="provider-base">{{ row.apiBase }}</span>
          </div>
        </template>

        <template #cell-type="{ value }">
          <span class="provider-type">{{ typeLabelMap[value] || value }}</span>
        </template>

        <template #cell-endpointMode="{ row }">
          <span class="provider-endpoint">{{ getEndpointModeLabel(row.endpointMode, row.type) }}</span>
        </template>

        <template #cell-modelCount="{ value }">
          <span class="provider-count">{{ value || 0 }} 个</span>
        </template>

        <template #cell-priority="{ value }">
          <span class="provider-count">{{ value }}</span>
        </template>

        <template #cell-status="{ row }">
          <StatusBadge
            :status="row.healthy ? 'success' : row.enabled ? 'warning' : 'error'"
            :label="row.healthy ? '正常' : row.enabled ? '异常' : '禁用'"
          />
        </template>

        <template #cell-actions="{ row }">
          <div class="row-actions">
            <button class="icon-btn" title="测试连接" @click.stop="handleTest(row)" :disabled="testing">
              <Zap :size="14" />
            </button>
            <button class="icon-btn" title="模型设置" @click.stop="openModelDialog(row)">
              <Database :size="14" />
            </button>
            <button class="icon-btn" title="编辑" @click.stop="openEditDialog(row)">
              <Pencil :size="14" />
            </button>
            <button class="icon-btn danger" title="删除" @click.stop="handleDelete(row)">
              <Trash2 :size="14" />
            </button>
          </div>
        </template>
      </DataTable>
    </section>

    <ModalDialog
      :visible="showDialog"
      :title="isEditing ? '编辑供应商' : '添加供应商'"
      size="lg"
      :showFooter="false"
      @close="showDialog = false"
    >
      <div class="form-grid">
        <div class="form-field">
          <label class="form-label">名称</label>
          <input v-model="form.name" class="form-input" placeholder="如: OpenAI" />
        </div>
        <div class="form-field">
          <label class="form-label">类型</label>
          <Select v-model="form.type" :options="providerTypeOptions" @update:modelValue="handleTypeChange" />
        </div>
        <div class="form-field">
          <label class="form-label">端点转发</label>
          <Select v-model="form.endpointMode" :options="endpointModeOptions" />
        </div>
        <div class="form-field">
          <label class="form-label">优先级</label>
          <input v-model.number="form.priority" class="form-input" type="number" min="0" />
        </div>
        <div class="form-field full">
          <label class="form-label">API Base URL</label>
          <input v-model="form.apiBase" class="form-input" placeholder="https://api.openai.com/v1" />
        </div>
        <div class="form-field full">
          <label class="form-label">API Key</label>
          <input v-model="form.apiKey" class="form-input" type="password" :placeholder="isEditing ? '留空则保留原有密钥' : 'sk-...'" />
        </div>
        <div class="form-field full">
          <label class="form-label">状态</label>
          <Select v-model="form.enabled" :options="enabledOptions" />
        </div>
      </div>
      <div class="form-actions">
        <button class="btn btn-secondary" @click="handleTestForm" :disabled="testing">
          <Zap :size="14" />
          {{ testing ? '测试中...' : '测试连接' }}
        </button>
        <div class="form-actions-right">
          <button class="btn btn-secondary" @click="showDialog = false">取消</button>
          <button class="btn btn-primary" @click="handleSave" :disabled="providerStore.loading">
            {{ providerStore.loading ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </ModalDialog>

    <ModalDialog
      :visible="showModelDialog"
      :title="`模型设置 - ${currentProvider?.name || ''}`"
      size="full"
      contentClass="model-dialog-content"
      :allowOverflow="true"
      @close="showModelDialog = false"
    >
      <div class="model-dialog-hero">
        <div class="model-dialog-hero-copy">
          <div class="settings-kicker">Mapping</div>
          <h3 class="model-dialog-title">模型映射</h3>
          <p class="panel-description">将统一路由名映射到当前供应商真实模型名，并指定默认模型。</p>
        </div>
        <div class="model-dialog-summary">
          <div class="summary-label">当前供应商</div>
          <div class="summary-value">{{ currentProvider?.name || '未选择' }}</div>
          <div class="summary-meta">
            <span class="summary-chip">{{ configuredModelCount }} 条映射</span>
            <span class="summary-chip">{{ modelForm.defaultModel || '未设置默认模型' }}</span>
          </div>
        </div>
      </div>

      <div class="model-editor-card">
        <div class="section-head">
          <div class="section-title">映射规则</div>
          <button class="btn btn-secondary add-inline-btn" @click="addModel">
            <Plus :size="14" />
            添加映射
          </button>
        </div>

        <div v-if="modelForm.llms.length === 0" class="empty-model-state">
          <div class="empty-model-state-title">还没有模型映射</div>
          <button class="btn btn-primary" @click="addModel">
            <Plus :size="14" />
            添加第一条映射
          </button>
        </div>

        <div v-else class="model-list">
          <div
            v-for="(model, index) in modelForm.llms"
            :key="index"
            class="mapping-row"
          >
            <div class="mapping-row-main">
              <div class="mapping-field">
                <label class="mapping-label">请求模型名称</label>
                <input
                  v-model="model.model"
                  class="model-input"
                  placeholder="如: gpt-4o"
                />
              </div>
              <div class="mapping-arrow">→</div>
              <div class="mapping-field">
                <label class="mapping-label">目标模型</label>
                <input
                  v-model="model.target"
                  class="model-input"
                  placeholder="如: claude-3-7-sonnet"
                />
              </div>
            </div>
            <button class="icon-btn danger mapping-remove-btn" title="删除映射" @click="removeModel(index)">
              <Trash2 :size="14" />
            </button>
          </div>
        </div>
      </div>

      <div class="default-model-section">
        <div class="section-head compact">
          <div class="section-title">默认模型</div>
        </div>
        <Select
          v-model="modelForm.defaultModel"
          :options="[
            { label: '无', value: '' },
            ...availableDefaultModels.map((m) => ({
              label: `${m.model} → ${m.target || m.model}`,
              value: m.model,
            })),
          ]"
        />
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showModelDialog = false">取消</button>
        <button class="btn btn-primary" @click="saveModels" :disabled="savingModels">
          {{ savingModels ? '保存中...' : '保存' }}
        </button>
      </template>
    </ModalDialog>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue';
import { useProviderStore } from '@/stores/provider';
import { Plus, Cpu, Zap, Pencil, Trash2, Database } from 'lucide-vue-next';
import StatusBadge from '@/components/ui/StatusBadge.vue';
import Select from '@/components/ui/Select.vue';
import PageHeader from '@/components/layout/PageHeader.vue';
import ModalDialog from '@/components/ModalDialog.vue';
import DataTable from '@/components/layout/DataTable.vue';
import { useConfirm } from '@/composables/useConfirm';
import { useToast } from '@/composables/useToast';

const providerStore = useProviderStore();
const { confirm } = useConfirm();
const { toast } = useToast();

const showDialog = ref(false);
const isEditing = ref(false);
const testing = ref(false);
const keyword = ref("");
const typeFilter = ref("all");
const statusFilter = ref("all");

const showModelDialog = ref(false);
const currentProvider = ref(null);
const savingModels = ref(false);

const defaultModelForm = () => ({
  llms: [],
  defaultModel: ''
});

const modelForm = ref(defaultModelForm());
const configuredModelCount = computed(() =>
  modelForm.value.llms.filter(m => (m.model || '').trim()).length
);
const availableDefaultModels = computed(() =>
  modelForm.value.llms.filter(m => (m.model || '').trim())
);

const endpointModeOptionsMap = {
  openai: [
    { value: 'chat_completions', label: 'Chat Completions (/chat/completions)' },
    { value: 'responses', label: 'Responses (/responses)' },
  ],
  anthropic: [
    { value: 'anthropic_messages', label: 'Anthropic Messages (/v1/messages)' },
  ],
  gemini: [
    { value: 'gemini_generate_content', label: 'Gemini GenerateContent' },
  ],
};

const typeLabelMap = {
  openai: "OpenAI 兼容",
  anthropic: "Anthropic",
  gemini: "Gemini",
};

const columns = [
  { key: 'name', title: '名称' },
  { key: 'type', title: '类型', class: 'w-[140px]' },
  { key: 'endpointMode', title: '端点模式' },
  { key: 'modelCount', title: '模型数', class: 'w-[88px]' },
  { key: 'priority', title: '优先级', class: 'w-[88px]' },
  { key: 'status', title: '状态', class: 'w-[120px]' },
  { key: 'actions', title: '操作', class: 'w-[168px]' },
];

const typeFilterOptions = [
  { label: '全部类型', value: 'all' },
  { label: 'OpenAI 兼容', value: 'openai' },
  { label: 'Anthropic', value: 'anthropic' },
  { label: 'Gemini', value: 'gemini' },
];

const statusFilterOptions = [
  { label: '全部状态', value: 'all' },
  { label: '健康', value: 'healthy' },
  { label: '异常', value: 'warning' },
  { label: '禁用', value: 'disabled' },
];

const providerTypeOptions = [
  { label: 'OpenAI 兼容', value: 'openai' },
  { label: 'Anthropic', value: 'anthropic' },
  { label: 'Google Gemini', value: 'gemini' },
];

const enabledOptions = [
  { label: '启用', value: true },
  { label: '禁用', value: false },
];

const defaultForm = () => ({
  id: '',
  name: '',
  type: 'openai',
  apiBase: '',
  apiKey: '',
  endpointMode: 'chat_completions',
  enabled: true,
  priority: 0,
});

const form = ref(defaultForm());
const endpointModeOptions = computed(() => endpointModeOptionsMap[form.value.type] || endpointModeOptionsMap.openai);
const healthyProviders = computed(() => providerStore.providers.filter((item) => item.healthy).length);

const filteredProviders = computed(() => {
  return providerStore.providers.filter((item) => {
    const matchesKeyword = !keyword.value
      || item.name.toLowerCase().includes(keyword.value.toLowerCase())
      || item.apiBase.toLowerCase().includes(keyword.value.toLowerCase());
    const matchesType = typeFilter.value === 'all' || item.type === typeFilter.value;
    const status = item.healthy ? 'healthy' : item.enabled ? 'warning' : 'disabled';
    const matchesStatus = statusFilter.value === 'all' || status === statusFilter.value;
    return matchesKeyword && matchesType && matchesStatus;
  });
});

function normalizeEndpointMode(type, endpointMode = '') {
  const options = endpointModeOptionsMap[type] || endpointModeOptionsMap.openai;
  const values = options.map((item) => item.value);
  if (values.includes(endpointMode)) return endpointMode;
  return options[0].value;
}

function getEndpointModeLabel(endpointMode, type) {
  const options = endpointModeOptionsMap[type] || [];
  return options.find((item) => item.value === normalizeEndpointMode(type, endpointMode))?.label || endpointMode || '-';
}

function handleTypeChange() {
  form.value.endpointMode = normalizeEndpointMode(form.value.type, form.value.endpointMode);
}

function openAddDialog() {
  isEditing.value = false;
  form.value = defaultForm();
  showDialog.value = true;
}

function openEditDialog(p) {
  isEditing.value = true;
  form.value = {
    id: p.id,
    name: p.name,
    type: p.type,
    apiBase: p.apiBase,
    apiKey: '',
    endpointMode: normalizeEndpointMode(p.type, p.endpointMode),
    enabled: p.enabled,
    priority: p.priority,
  };
  showDialog.value = true;
}

async function handleSave() {
  if (!form.value.name || !form.value.apiBase) {
    toast('请填写名称和 API Base URL', 'error');
    return;
  }
  try {
    if (isEditing.value) {
      await providerStore.updateProvider(form.value);
      toast('供应商已更新', 'success');
    } else {
      await providerStore.addProvider(form.value);
      toast('供应商已添加', 'success');
    }
    showDialog.value = false;
  } catch (e) {
    toast('操作失败: ' + e.message, 'error');
  }
}

async function handleDelete(p) {
  const ok = await confirm(`确定要删除供应商 "${p.name}" 吗？`);
  if (!ok) return;
  try {
    await providerStore.deleteProvider(p.id);
    toast('供应商已删除', 'success');
  } catch (e) {
    toast('删除失败: ' + e.message, 'error');
  }
}

async function handleTest(p) {
  testing.value = true;
  try {
    const result = await providerStore.testProvider({
      id: p.id, name: p.name, type: p.type, apiBase: p.apiBase, endpointMode: p.endpointMode,
    });
    if (result.success) {
      toast(`${p.name} 连接正常`, 'success');
    } else {
      toast(`${p.name} 连接失败: ${result.error}`, 'error');
    }
  } finally {
    testing.value = false;
  }
}

async function handleTestForm() {
  testing.value = true;
  try {
    const result = await providerStore.testProvider(form.value);
    if (result.success) {
      toast('连接测试成功', 'success');
    } else {
      toast('连接测试失败: ' + result.error, 'error');
    }
  } finally {
    testing.value = false;
  }
}

function openModelDialog(p) {
  currentProvider.value = p;
  modelForm.value = {
    llms: p.llms ? JSON.parse(JSON.stringify(p.llms)) : [],
    defaultModel: p.defaultModel || ''
  };
  modelForm.value.llms = modelForm.value.llms.map(m => ({
    model: m.model || '',
    target: m.target || ''
  }));
  showModelDialog.value = true;
}

function addModel() {
  modelForm.value.llms.push({ model: '', target: '' });
}

function removeModel(index) {
  modelForm.value.llms.splice(index, 1);
}

async function saveModels() {
  if (!currentProvider.value) return;

  const validLlms = modelForm.value.llms.filter(m => m.model.trim());

  savingModels.value = true;
  try {
    await providerStore.setProviderModels(
      currentProvider.value.id,
      validLlms,
      modelForm.value.defaultModel
    );
    toast('模型设置已保存', 'success');
    showModelDialog.value = false;
  } catch (e) {
    toast('保存失败: ' + e.message, 'error');
  } finally {
    savingModels.value = false;
  }
}

onMounted(() => {
  providerStore.fetchProviders();
});
</script>

<style scoped>
.providers-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.providers-toolbar {
  align-items: flex-end;
}

.providers-toolbar-main {
  flex: 1;
}

.toolbar-field {
  min-width: 160px;
}

.toolbar-label {
  display: block;
  margin-bottom: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.toolbar-input {
  min-width: 260px;
}

.providers-empty-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.providers-empty p {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
}

.table-panel {
  min-height: 0;
}

.provider-cell-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.provider-name {
  font-weight: 600;
  color: var(--color-text-primary);
}

.provider-base {
  max-width: 360px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--color-text-muted);
}

.provider-type,
.provider-endpoint,
.provider-count {
  font-size: 12px;
}

.provider-endpoint {
  color: var(--color-text-secondary);
}

.provider-type {
  color: var(--color-text-primary);
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.form-field.full {
  grid-column: 1 / -1;
}

.form-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--ui-border-subtle);
}

.form-actions-right {
  display: flex;
  gap: 8px;
}

.model-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.model-dialog-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.7fr) minmax(260px, 0.9fr);
  gap: 16px;
}

.model-dialog-hero-copy,
.model-dialog-summary,
.model-editor-card,
.default-model-section {
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface);
}

.model-dialog-hero-copy {
  padding: 18px 20px;
  background: var(--ui-bg-surface-muted);
}

.model-dialog-title {
  margin: 0;
  font-size: 20px;
  line-height: 1.25;
  color: var(--color-text-primary);
}

.model-dialog-summary {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 18px 20px;
}

.summary-label {
  font-size: 12px;
  color: var(--color-text-muted);
}

.summary-value {
  margin-top: 6px;
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.summary-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}

.model-editor-card,
.default-model-section {
  padding: 18px 20px 20px;
}

.section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.section-head.compact {
  margin-bottom: 12px;
}

.model-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 360px;
  overflow-y: auto;
}

.add-inline-btn {
  flex-shrink: 0;
}

.empty-model-state {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
  padding: 28px;
  border: 1px dashed var(--ui-border-strong);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface-muted);
}

.empty-model-state-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.mapping-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  align-items: start;
  padding: 16px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface-muted);
}

.mapping-row-main {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 28px minmax(0, 1fr);
  gap: 12px;
  align-items: center;
}

.mapping-field {
  min-width: 0;
}

.mapping-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  font-size: 18px;
  font-weight: 700;
}

.mapping-remove-btn {
  width: 32px;
  height: 32px;
}

@media (max-width: 960px) {
  .model-dialog-hero,
  .mapping-row-main {
    grid-template-columns: 1fr;
  }

  .mapping-arrow {
    display: none;
  }
}

@media (max-width: 820px) {
  .providers-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar-input {
    min-width: 0;
  }
}

@media (max-width: 720px) {
  .form-grid {
    grid-template-columns: 1fr;
  }

  .form-actions,
  .section-head {
    flex-direction: column;
    align-items: stretch;
  }

  .row-actions {
    justify-content: flex-start;
  }

  .mapping-row {
    grid-template-columns: 1fr;
  }

  :deep(.modal-shell__footer) {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
