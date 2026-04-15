<template>
  <div class="providers-view app-page">
    <PageHeader
      title="供应商管理"
      description="统一管理供应商接入、端点转发模式、模型映射和健康状态。"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openAddDialog">
          <Plus :size="14" /> 添加供应商
        </button>
      </template>
    </PageHeader>

    <div class="provider-grid">
      <div v-if="providerStore.providers.length === 0" class="empty-state">
        <Cpu :size="48" />
        <p>暂未配置供应商</p>
        <button class="btn btn-primary" @click="openAddDialog">
          <Plus :size="14" /> 添加第一个供应商
        </button>
      </div>

      <div v-for="p in providerStore.providers" :key="p.id" class="provider-card">
        <div class="card-header">
          <div class="card-title-row">
            <StatusBadge
              :status="p.healthy ? 'success' : p.enabled ? 'warning' : 'error'"
              :label="p.healthy ? '正常' : p.enabled ? '异常' : '禁用'"
            />
            <span class="card-title">{{ p.name }}</span>
          </div>
          <div class="card-actions">
            <button class="icon-btn" title="测试连接" @click="handleTest(p)" :disabled="testing">
              <Zap :size="14" />
            </button>
            <button class="icon-btn" title="模型设置" @click="openModelDialog(p)">
              <Database :size="14" />
            </button>
            <button class="icon-btn" title="编辑" @click="openEditDialog(p)">
              <Pencil :size="14" />
            </button>
            <button class="icon-btn danger" title="删除" @click="handleDelete(p)">
              <Trash2 :size="14" />
            </button>
          </div>
        </div>
        <div class="card-body">
          <div class="card-meta">
            <span class="meta-type">{{ p.type }}</span>
            <span class="meta-endpoint">{{ getEndpointModeLabel(p.endpointMode, p.type) }}</span>
            <span class="meta-base">{{ p.apiBase }}</span>
          </div>
          <div class="card-footer">
            <span class="meta-models">{{ p.modelCount }} 个模型</span>
            <span class="meta-priority">优先级: {{ p.priority }}</span>
          </div>
        </div>
      </div>
    </div>

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
          <select v-model="form.type" class="form-input" @change="handleTypeChange">
            <option value="openai">OpenAI 兼容</option>
            <option value="anthropic">Anthropic</option>
            <option value="gemini">Google Gemini</option>
          </select>
        </div>
        <div class="form-field">
          <label class="form-label">端点转发</label>
          <select v-model="form.endpointMode" class="form-input">
            <option
              v-for="option in endpointModeOptions"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </option>
          </select>
        </div>
        <div class="form-field full">
          <label class="form-label">API Base URL</label>
          <input v-model="form.apiBase" class="form-input" placeholder="https://api.openai.com/v1" />
        </div>
        <div class="form-field full">
          <label class="form-label">API Key</label>
          <input v-model="form.apiKey" class="form-input" type="password" :placeholder="isEditing ? '留空则保留原有密钥' : 'sk-...'" />
        </div>
        <div class="form-field">
          <label class="form-label">优先级</label>
          <input v-model.number="form.priority" class="form-input" type="number" min="0" />
        </div>
        <div class="form-field">
          <label class="form-label">状态</label>
          <select v-model="form.enabled" class="form-input">
            <option :value="true">启用</option>
            <option :value="false">禁用</option>
          </select>
        </div>
      </div>
      <div class="form-actions">
        <button class="btn btn-secondary" @click="handleTestForm" :disabled="testing">
          <Zap :size="14" /> {{ testing ? '测试中...' : '测试连接' }}
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
      :showFooter="false"
      @close="showModelDialog = false"
    >
      <div class="model-dialog-hero">
        <div class="model-dialog-hero-copy">
          <h3 class="model-dialog-title">模型映射</h3>
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
            <Plus :size="14" /> 添加映射
          </button>
        </div>

        <div v-if="modelForm.llms.length === 0" class="empty-model-state">
          <div class="empty-model-state-title">还没有模型映射</div>
          <button class="btn btn-primary" @click="addModel">
            <Plus :size="14" /> 添加第一条映射
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
        <select v-model="modelForm.defaultModel" class="form-input">
          <option value="">无</option>
          <option
            v-for="m in availableDefaultModels"
            :key="m.model"
            :value="m.model"
          >
            {{ m.model }} → {{ m.target || m.model }}
          </option>
        </select>
      </div>

      <div class="model-dialog-actions">
        <div class="model-dialog-actions-buttons">
          <button class="btn btn-secondary" @click="showModelDialog = false">取消</button>
          <button class="btn btn-primary" @click="saveModels" :disabled="savingModels">
            {{ savingModels ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </ModalDialog>
  </div>
</template>

<script setup>
import { computed, ref, onMounted } from 'vue';
import { useProviderStore } from '@/stores/provider';
import { Plus, Cpu, Zap, Pencil, Trash2, Database } from 'lucide-vue-next';
import StatusBadge from '@/components/ui/StatusBadge.vue';
import PageHeader from '@/components/layout/PageHeader.vue';
import ModalDialog from '@/components/ModalDialog.vue';
import { useConfirm } from '@/composables/useConfirm';
import { useToast } from '@/composables/useToast';

const providerStore = useProviderStore();
const { confirm } = useConfirm();
const { toast } = useToast();

const showDialog = ref(false);
const isEditing = ref(false);
const testing = ref(false);

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
  // 确保所有模型条目都有 target 字段
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
  
  // 过滤掉空的模型条目
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
.provider-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
}

.empty-state {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 24px;
  color: var(--color-text-muted);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-lg);
}

.provider-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-secondary);
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border);
}
.card-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.card-actions {
  display: flex;
  gap: 4px;
}
.card-body {
  padding: 12px 16px;
}
.card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.meta-type {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--color-bg-tertiary);
  color: var(--color-text-muted);
  text-transform: uppercase;
  font-weight: 600;
}
.meta-endpoint {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-accent) 10%, var(--color-bg-tertiary));
  color: var(--color-accent);
  font-weight: 700;
}
.meta-base {
  font-size: 12px;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.card-footer {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: var(--color-text-muted);
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.form-field.full {
  grid-column: 1 / -1;
}
.form-label {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-muted);
  margin-bottom: 4px;
}

.form-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--color-border);
}

.form-actions-right {
  display: flex;
  gap: 8px;
}

/* 模型对话框样式 */
.model-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.model-dialog-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.7fr) minmax(260px, 0.9fr);
  gap: 16px;
  align-items: stretch;
}

.model-dialog-hero-copy,
.model-dialog-summary,
.model-editor-card,
.default-model-section {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-secondary);
}

.model-dialog-hero-copy {
  padding: 18px 20px;
  background:
    linear-gradient(135deg, rgba(37, 99, 235, 0.08), rgba(37, 99, 235, 0.02)),
    var(--color-bg-secondary);
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
  font-weight: 700;
  color: var(--color-text-primary);
}

.summary-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}

.summary-chip {
  display: inline-flex;
  align-items: center;
  padding: 6px 10px;
  border-radius: 999px;
  background: var(--color-bg-tertiary);
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.model-editor-card {
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

.section-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.model-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 360px;
  overflow-y: auto;
}

.model-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 13px;
  box-sizing: border-box;
}

.model-input:focus {
  outline: none;
  border-color: var(--color-accent);
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
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-lg);
  background:
    linear-gradient(180deg, rgba(37, 99, 235, 0.04), transparent),
    var(--color-bg-primary);
}

.empty-model-state-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.mapping-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  align-items: start;
  padding: 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-primary);
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

.mapping-label {
  display: block;
  margin-bottom: 6px;
  font-size: 12px;
  font-weight: 700;
  color: var(--color-text-secondary);
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

.default-model-section {
  padding: 18px 20px 20px;
}

.model-dialog-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16px;
  padding: 16px 20px 0;
  border-top: 1px solid var(--color-border);
}

.model-dialog-actions-buttons {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

@media (max-width: 960px) {
  .model-dialog-hero {
    grid-template-columns: 1fr;
  }

  .mapping-row-main {
    grid-template-columns: 1fr;
  }

  .mapping-arrow {
    display: none;
  }
}

@media (max-width: 720px) {
  .section-head,
  .model-dialog-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .add-inline-btn,
  .model-dialog-actions-buttons .btn {
    width: 100%;
    justify-content: center;
  }

  .model-dialog-actions-buttons {
    width: 100%;
  }

  .mapping-row {
    grid-template-columns: 1fr;
  }
}
</style>
