<template>
  <div class="app-page endpoints-view">
    <UEDPageHeader title="端点管理" description="将上游能力抽象为可路由资源，为不同协议之间的转换和精细化暴露打基础。" divided>
      <template #actions>
        <button class="btn btn-primary" @click="openCreate">新增端点</button>
      </template>
    </UEDPageHeader>

    <section class="toolbar-surface page-toolbar">
      <div class="toolbar-group">
        <div class="toolbar-field">
          <label class="toolbar-label">搜索</label>
          <input v-model="keyword" class="form-input toolbar-input" placeholder="按名称、路径或协议搜索" />
        </div>
        <div class="toolbar-field">
          <label class="toolbar-label">供应商</label>
          <select v-model="providerFilter" class="form-input toolbar-input">
            <option value="all">全部</option>
            <option v-for="item in providerStore.providers" :key="item.id" :value="item.id">
              {{ item.name || item.id }}
            </option>
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
        <span class="toolbar-chip">共 {{ endpointStore.endpointCount }} 个</span>
        <span class="toolbar-chip">启用 {{ endpointStore.enabledEndpoints.length }} 个</span>
        <span class="toolbar-chip">当前显示 {{ filteredEndpoints.length }} 个</span>
      </div>
    </section>

    <section class="table-panel panel-card">
      <div v-if="filteredEndpoints.length === 0" class="empty-state compact-empty">
        <div class="empty-title">暂无端点</div>
        <p>可先从默认 chat / responses 能力开始建模，后续再扩展为协议转换入口。</p>
      </div>

      <table v-else class="simple-table">
        <thead>
          <tr>
            <th>名称</th>
            <th>供应商</th>
            <th>路径 / 方法</th>
            <th>协议转换</th>
            <th>优先级</th>
            <th>状态</th>
            <th class="actions-col">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in filteredEndpoints" :key="item.id">
            <td>
              <div class="cell-main">{{ item.name || '-' }}</div>
              <div class="cell-sub">{{ item.capability || '未设置 capability' }}</div>
            </td>
            <td>{{ providerName(item.providerId) }}</td>
            <td>
              <div class="cell-main"><code>{{ item.path }}</code></div>
              <div class="cell-sub">{{ item.method || 'POST' }}</div>
            </td>
            <td>
              <div class="cell-sub">{{ item.requestProtocol || '-' }}</div>
              <div class="cell-sub">→ {{ item.responseProtocol || '-' }}</div>
            </td>
            <td>{{ item.priority }}</td>
            <td>
              <StatusBadge :status="item.enabled ? 'success' : 'neutral'" :label="item.enabled ? '启用' : '禁用'" />
            </td>
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
        <div class="section-title">{{ editingId ? '编辑端点' : '新增端点' }}</div>
        <button v-if="editingId" class="btn btn-secondary btn-sm" @click="resetForm">取消编辑</button>
      </div>

      <div class="form-grid two-columns">
        <div class="form-field">
          <label class="form-label">名称</label>
          <input v-model="form.name" class="form-input" placeholder="如：OpenAI Responses" />
        </div>
        <div class="form-field">
          <label class="form-label">供应商</label>
          <select v-model="form.providerId" class="form-input">
            <option value="">请选择供应商</option>
            <option v-for="item in providerStore.providers" :key="item.id" :value="item.id">
              {{ item.name || item.id }}
            </option>
          </select>
        </div>
        <div class="form-field">
          <label class="form-label">路径</label>
          <input v-model="form.path" class="form-input" placeholder="/v1/chat/completions" />
        </div>
        <div class="form-field">
          <label class="form-label">方法</label>
          <input v-model="form.method" class="form-input" placeholder="POST" />
        </div>
        <div class="form-field">
          <label class="form-label">能力标识</label>
          <input v-model="form.capability" class="form-input" placeholder="chat / responses" />
        </div>
        <div class="form-field">
          <label class="form-label">优先级</label>
          <input v-model.number="form.priority" type="number" class="form-input" min="0" />
        </div>
        <div class="form-field">
          <label class="form-label">请求协议</label>
          <input v-model="form.requestProtocol" class="form-input" placeholder="openai_chat" />
        </div>
        <div class="form-field">
          <label class="form-label">响应协议</label>
          <input v-model="form.responseProtocol" class="form-input" placeholder="openai_chat" />
        </div>
        <div class="form-field full">
          <label class="form-label">备注</label>
          <input v-model="form.remark" class="form-input" placeholder="说明此端点的用途或迁移备注" />
        </div>
        <div class="form-field full checkbox-group">
          <label class="checkbox-row">
            <input v-model="form.enabled" type="checkbox" />
            <span>启用此端点</span>
          </label>
          <label class="checkbox-row">
            <input v-model="form.isDefault" type="checkbox" />
            <span>设为默认端点</span>
          </label>
        </div>
      </div>

      <div class="editor-actions">
        <button class="btn btn-secondary" @click="resetForm">重置</button>
        <button class="btn btn-primary" :disabled="endpointStore.loading || !form.name.trim() || !form.providerId.trim() || !form.path.trim()" @click="handleSubmit">
          {{ endpointStore.loading ? '保存中...' : editingId ? '保存修改' : '创建端点' }}
        </button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue';
import { UEDPageHeader } from '@/components/layout';
import StatusBadge from '@/components/ui/StatusBadge.vue';
import { useEndpointStore } from '@/stores/endpoint';
import { useProviderStore } from '@/stores/provider';

const endpointStore = useEndpointStore();
const providerStore = useProviderStore();

const keyword = ref('');
const providerFilter = ref('all');
const statusFilter = ref('all');
const editingId = ref('');
const form = ref(createEmptyForm());

function createEmptyForm() {
  return {
    id: '',
    name: '',
    providerId: '',
    path: '',
    method: 'POST',
    capability: '',
    requestProtocol: '',
    responseProtocol: '',
    enabled: true,
    priority: 0,
    isDefault: false,
    remark: '',
  };
}

const filteredEndpoints = computed(() => {
  const q = keyword.value.trim().toLowerCase();
  return endpointStore.endpoints.filter((item) => {
    if (providerFilter.value !== 'all' && item.providerId !== providerFilter.value) return false;
    if (statusFilter.value === 'enabled' && !item.enabled) return false;
    if (statusFilter.value === 'disabled' && item.enabled) return false;
    if (!q) return true;
    return [item.name, item.path, item.requestProtocol, item.responseProtocol].some((value) => (value || '').toLowerCase().includes(q));
  });
});

function providerName(id) {
  const provider = providerStore.providers.find((item) => item.id === id);
  return provider?.name || id || '-';
}

function resetForm() {
  editingId.value = '';
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
    providerId: item.providerId,
    path: item.path,
    method: item.method || 'POST',
    capability: item.capability,
    requestProtocol: item.requestProtocol,
    responseProtocol: item.responseProtocol,
    enabled: item.enabled,
    priority: item.priority || 0,
    isDefault: item.isDefault === true,
    remark: item.remark || '',
  };
}

async function handleSubmit() {
  if (editingId.value) {
    await endpointStore.updateEndpoint(form.value);
  } else {
    await endpointStore.addEndpoint(form.value);
  }
  resetForm();
}

async function handleDelete(item) {
  if (!window.confirm(`确认删除端点“${item.name || item.id}”吗？`)) return;
  await endpointStore.deleteEndpoint(item.id);
  if (editingId.value === item.id) {
    resetForm();
  }
}

onMounted(async () => {
  await providerStore.fetchProviders();
  await endpointStore.fetchEndpoints();
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

.row-actions,
.editor-actions,
.checkbox-group {
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