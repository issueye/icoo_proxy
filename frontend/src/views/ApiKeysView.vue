<template>
  <div class="app-page api-keys-view">
    <UEDPageHeader title="API 密钥管理" description="统一管理网关访问凭证，并为后续按供应商/端点限制访问范围做准备。" divided>
      <template #actions>
        <button class="btn btn-primary" @click="openCreate">
          新增密钥
        </button>
      </template>
    </UEDPageHeader>

    <section class="toolbar-surface page-toolbar">
      <div class="toolbar-group">
        <div class="toolbar-field">
          <label class="toolbar-label">搜索</label>
          <input v-model="keyword" class="form-input toolbar-input" placeholder="按名称、描述或密钥搜索" />
        </div>
        <div class="toolbar-field">
          <label class="toolbar-label">范围</label>
          <select v-model="scopeFilter" class="form-input toolbar-input">
            <option value="all">全部</option>
            <option value="restricted">受限</option>
          </select>
        </div>
        <div class="toolbar-field">
          <label class="toolbar-label">状态</label>
          <select v-model="statusFilter" class="form-input toolbar-input">
            <option value="all">全部</option>
            <option value="enabled">启用</option>
            <option value="disabled">禁用</option>
          </select>
        </div>
      </div>
      <div class="toolbar-summary">
        <span class="toolbar-chip">共 {{ apiKeyStore.apiKeyCount }} 个</span>
        <span class="toolbar-chip">启用 {{ apiKeyStore.enabledApiKeys.length }} 个</span>
        <span class="toolbar-chip">当前显示 {{ filteredApiKeys.length }} 个</span>
      </div>
    </section>

    <section class="table-panel panel-card">
      <div v-if="filteredApiKeys.length === 0" class="empty-state compact-empty">
        <div class="empty-title">暂无 API 密钥</div>
        <p>可先创建全局密钥，后续再逐步切换到按供应商或端点受限的访问控制。</p>
      </div>

      <table v-else class="simple-table">
        <thead>
          <tr>
            <th>名称</th>
            <th>密钥</th>
            <th>范围</th>
            <th>状态</th>
            <th>供应商 / 端点</th>
            <th>更新时间</th>
            <th class="actions-col">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in filteredApiKeys" :key="item.id">
            <td>
              <div class="cell-main">{{ item.name || '-' }}</div>
              <div class="cell-sub">{{ item.description || '无描述' }}</div>
            </td>
            <td>
              <code class="secret-code">{{ maskKey(item.key) }}</code>
            </td>
            <td>
              <span class="table-tag">{{ item.scopeMode === 'restricted' ? '受限' : '全局' }}</span>
            </td>
            <td>
              <StatusBadge :status="item.enabled ? 'success' : 'neutral'" :label="item.enabled ? '启用' : '禁用'" />
            </td>
            <td>
              <div class="cell-sub">供应商 {{ item.providerIds.length }} 个</div>
              <div class="cell-sub">端点 {{ item.endpointIds.length }} 个</div>
            </td>
            <td>{{ formatDate(item.updatedAt || item.createdAt) }}</td>
            <td class="actions-col">
              <div class="row-actions">
                <button class="btn btn-secondary btn-sm" @click="openEdit(item)">编辑</button>
                <button class="btn btn-danger btn-sm" @click="handleDelete(item)">删除</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="panel-card editor-card">
      <div class="section-head compact">
        <div class="section-title">{{ editingId ? '编辑密钥' : '新增密钥' }}</div>
        <button v-if="editingId" class="btn btn-secondary btn-sm" @click="resetForm">取消编辑</button>
      </div>

      <div class="form-grid two-columns">
        <div class="form-field">
          <label class="form-label">名称</label>
          <input v-model="form.name" class="form-input" placeholder="如：Default Gateway Key" />
        </div>
        <div class="form-field">
          <label class="form-label">范围模式</label>
          <select v-model="form.scopeMode" class="form-input">
            <option value="all">全局</option>
            <option value="restricted">受限</option>
          </select>
        </div>
        <div class="form-field full">
          <label class="form-label">密钥</label>
          <div class="inline-actions">
            <input v-model="form.key" class="form-input" :placeholder="editingId ? '留空则保留原有密钥' : '请输入 API Key'" />
            <button class="btn btn-secondary btn-sm" type="button" @click="generateKey">生成随机值</button>
          </div>
        </div>
        <div class="form-field full">
          <label class="form-label">描述</label>
          <input v-model="form.description" class="form-input" placeholder="说明此密钥的用途" />
        </div>
        <div class="form-field full checkbox-field">
          <label class="checkbox-row">
            <input v-model="form.enabled" type="checkbox" />
            <span>启用此密钥</span>
          </label>
        </div>
        <div class="form-field">
          <label class="form-label">关联供应商 ID（每行一个）</label>
          <textarea v-model="providerIdsInput" class="form-input form-textarea" rows="5" placeholder="provider-1&#10;provider-2"></textarea>
        </div>
        <div class="form-field">
          <label class="form-label">关联端点 ID（每行一个）</label>
          <textarea v-model="endpointIdsInput" class="form-input form-textarea" rows="5" placeholder="endpoint-1&#10;endpoint-2"></textarea>
        </div>
      </div>

      <div class="editor-actions">
        <button class="btn btn-secondary" @click="resetForm">重置</button>
        <button class="btn btn-primary" :disabled="apiKeyStore.loading || !form.name.trim()" @click="handleSubmit">
          {{ apiKeyStore.loading ? '保存中...' : editingId ? '保存修改' : '创建密钥' }}
        </button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue';
import { UEDPageHeader } from '@/components/layout';
import StatusBadge from '@/components/ui/StatusBadge.vue';
import { useApiKeyStore } from '@/stores/api-key';

const apiKeyStore = useApiKeyStore();

const keyword = ref('');
const scopeFilter = ref('all');
const statusFilter = ref('all');
const editingId = ref('');
const providerIdsInput = ref('');
const endpointIdsInput = ref('');
const form = ref(createEmptyForm());

function createEmptyForm() {
  return {
    id: '',
    name: '',
    key: '',
    description: '',
    enabled: true,
    scopeMode: 'all',
  };
}

const filteredApiKeys = computed(() => {
  const q = keyword.value.trim().toLowerCase();
  return apiKeyStore.apiKeys.filter((item) => {
    if (scopeFilter.value !== 'all' && item.scopeMode !== scopeFilter.value) return false;
    if (statusFilter.value === 'enabled' && !item.enabled) return false;
    if (statusFilter.value === 'disabled' && item.enabled) return false;
    if (!q) return true;
    return [item.name, item.description, item.key].some((value) => (value || '').toLowerCase().includes(q));
  });
});

function splitLines(value) {
  return value
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function generateKey() {
  form.value.key = `ik_${Math.random().toString(36).slice(2)}${Math.random().toString(36).slice(2)}`;
}

function maskKey(value) {
  if (!value) return '-';
  if (value.length <= 8) return value;
  return `${value.slice(0, 4)}...${value.slice(-4)}`;
}

function formatDate(value) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleString();
}

function resetForm() {
  editingId.value = '';
  providerIdsInput.value = '';
  endpointIdsInput.value = '';
  form.value = createEmptyForm();
}

function openCreate() {
  resetForm();
}

function openEdit(item) {
  editingId.value = item.id;
  form.value = {
    id: item.id,
    name: item.name,
    key: '',
    description: item.description,
    enabled: item.enabled,
    scopeMode: item.scopeMode || 'all',
  };
  providerIdsInput.value = (item.providerIds || []).join('\n');
  endpointIdsInput.value = (item.endpointIds || []).join('\n');
}

async function handleSubmit() {
  const payload = {
    ...form.value,
    providerIds: splitLines(providerIdsInput.value),
    endpointIds: splitLines(endpointIdsInput.value),
  };

  if (editingId.value) {
    await apiKeyStore.updateAPIKey(payload);
  } else {
    await apiKeyStore.addAPIKey(payload);
  }

  resetForm();
}

async function handleDelete(item) {
  if (!window.confirm(`确认删除 API 密钥“${item.name || item.id}”吗？`)) return;
  await apiKeyStore.deleteAPIKey(item.id);
  if (editingId.value === item.id) {
    resetForm();
  }
}

onMounted(() => {
  apiKeyStore.fetchAPIKeys();
});
</script>

<style scoped>
.page-toolbar,
.editor-card {
  margin-top: 16px;
}

.compact-empty {
  padding: 32px 20px;
}

.empty-title {
  font-size: 18px;
  font-weight: 600;
}

.simple-table {
  width: 100%;
  border-collapse: collapse;
}

.simple-table th,
.simple-table td {
  padding: 14px 12px;
  border-bottom: 1px solid var(--ui-border-soft);
  text-align: left;
  vertical-align: middle;
}

.actions-col {
  width: 160px;
}

.cell-main {
  font-weight: 600;
}

.cell-sub {
  margin-top: 4px;
  color: var(--text-secondary);
  font-size: 12px;
}

.secret-code {
  font-size: 12px;
}

.table-tag {
  display: inline-flex;
  padding: 4px 8px;
  border-radius: 999px;
  background: var(--ui-bg-elevated);
  color: var(--text-secondary);
  font-size: 12px;
}

.row-actions,
.inline-actions,
.editor-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.form-grid.two-columns {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.form-field.full {
  grid-column: 1 / -1;
}

.form-textarea {
  min-height: 120px;
  resize: vertical;
}

.checkbox-field {
  display: flex;
  align-items: center;
}

.checkbox-row {
  display: inline-flex;
  gap: 8px;
  align-items: center;
}

@media (max-width: 960px) {
  .form-grid.two-columns {
    grid-template-columns: 1fr;
  }
}
</style>