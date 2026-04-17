<template>
  <div class="app-page endpoints-view">
    <UEDPageHeader
      title="端点管理"
      description="将上游能力抽象为可路由资源，为不同协议之间的转换和精细化暴露打基础。"
      divided
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreate">新增端点</button>
      </template>
    </UEDPageHeader>

    <section class="toolbar-surface page-toolbar">
      <div class="toolbar-group">
        <div class="toolbar-field toolbar-field--search">
          <label class="toolbar-label">搜索</label>
          <input
            v-model="keyword"
            class="form-input toolbar-input"
            placeholder="按名称、路径或协议搜索"
          />
        </div>
        <div class="toolbar-field">
          <label class="toolbar-label">供应商</label>
          <Select
            v-model="providerFilter"
            :options="providerFilterOptions"
            class="toolbar-select"
          />
        </div>
        <div class="toolbar-field">
          <label class="toolbar-label">状态</label>
          <Select
            v-model="statusFilter"
            :options="statusFilterOptions"
            class="toolbar-select"
          />
        </div>
      </div>
      <div class="toolbar-summary">
        <span class="toolbar-chip">共 {{ endpointStore.endpointCount }} 个</span>
        <span class="toolbar-chip">启用 {{ endpointStore.enabledEndpoints.length }} 个</span>
        <span class="toolbar-chip">当前显示 {{ filteredEndpoints.length }} 个</span>
      </div>
    </section>

    <section class="table-panel panel-card">
      <DataTable
        :columns="columns"
        :data="filteredEndpoints"
        :loading="endpointStore.loading"
        empty-text="暂无端点"
        row-key="id"
      >
        <template #cell-name="{ row }">
          <div class="cell-main">{{ row.name || '-' }}</div>
          <div class="cell-sub">{{ row.capability || '未设置 capability' }}</div>
        </template>

        <template #cell-providerId="{ row }">
          {{ providerName(row.providerId) }}
        </template>

        <template #cell-path="{ row }">
          <div class="cell-main"><code>{{ row.path }}</code></div>
          <div class="cell-sub">{{ row.method || 'POST' }}</div>
        </template>

        <template #cell-protocol="{ row }">
          <div class="cell-sub">{{ row.requestProtocol || '-' }}</div>
          <div class="cell-sub">→ {{ row.responseProtocol || '-' }}</div>
        </template>

        <template #cell-priority="{ row }">
          {{ row.priority }}
        </template>

        <template #cell-enabled="{ row }">
          <StatusBadge
            :status="row.enabled ? 'success' : 'neutral'"
            :label="row.enabled ? '启用' : '禁用'"
          />
        </template>

        <template #cell-actions="{ row }">
          <div class="row-actions" @click.stop>
            <button class="btn btn-secondary btn-sm" @click="openEdit(row)">编辑</button>
            <button class="btn btn-danger btn-sm" @click="handleDelete(row)">删除</button>
          </div>
        </template>
      </DataTable>
    </section>

    <FloatingDrawer
      v-model:visible="drawerVisible"
      :title="editingId ? '编辑端点' : '新增端点'"
      description="配置网关入口端点及其关联供应商、协议和优先级。"
      kicker="ENDPOINT"
      width="620px"
      @close="handleDrawerClose"
    >
      <div class="drawer-form">
        <div class="form-grid two-columns">
          <div class="form-field">
            <label class="form-label">名称</label>
            <input
              v-model="form.name"
              class="form-input"
              placeholder="如：OpenAI Responses"
            />
          </div>

          <div class="form-field">
            <label class="form-label">供应商</label>
            <Select v-model="form.providerId" :options="providerOptions" />
          </div>

          <div class="form-field">
            <label class="form-label">路径</label>
            <input
              v-model="form.path"
              class="form-input"
              placeholder="/v1/chat/completions"
            />
          </div>

          <div class="form-field">
            <label class="form-label">方法</label>
            <Select v-model="form.method" :options="methodOptions" />
          </div>

          <div class="form-field">
            <label class="form-label">能力标识</label>
            <input
              v-model="form.capability"
              class="form-input"
              placeholder="chat / responses"
            />
          </div>

          <div class="form-field">
            <label class="form-label">优先级</label>
            <input v-model.number="form.priority" type="number" class="form-input" min="0" />
          </div>

          <div class="form-field">
            <label class="form-label">请求协议</label>
            <Select
              v-model="form.requestProtocol"
              :options="protocolOptions"
              searchable
              clearable
              placeholder="选择或清空请求协议"
            />
            <p class="form-hint">留空时使用当前适配器默认请求协议。</p>
          </div>

          <div class="form-field">
            <label class="form-label">响应协议</label>
            <Select
              v-model="form.responseProtocol"
              :options="protocolOptions"
              searchable
              clearable
              placeholder="选择或清空响应协议"
            />
            <p class="form-hint">留空时使用当前适配器默认响应协议。</p>
          </div>

          <div class="form-field full">
            <label class="form-label">备注</label>
            <input
              v-model="form.remark"
              class="form-input"
              placeholder="说明此端点的用途或迁移备注"
            />
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
      </div>

      <template #footer>
        <div class="drawer-footer">
          <button class="btn btn-secondary" @click="handleDrawerClose">取消</button>
          <button
            class="btn btn-primary"
            :disabled="endpointStore.loading || !form.name.trim() || !form.providerId.trim() || !form.path.trim()"
            @click="handleSubmit"
          >
            {{ endpointStore.loading ? '保存中...' : editingId ? '保存修改' : '创建端点' }}
          </button>
        </div>
      </template>
    </FloatingDrawer>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue';
import { DataTable, UEDPageHeader } from '@/components/layout';
import { FloatingDrawer, Select, StatusBadge } from '@/components/ui';
import { useEndpointStore } from '@/stores/endpoint';
import { useProviderStore } from '@/stores/provider';

const endpointStore = useEndpointStore();
const providerStore = useProviderStore();

const columns = [
  { key: 'name', title: '名称' },
  { key: 'providerId', title: '供应商', class: 'w-[140px]' },
  { key: 'path', title: '路径 / 方法' },
  { key: 'protocol', title: '协议转换', class: 'w-[160px]' },
  { key: 'priority', title: '优先级', class: 'w-[88px]' },
  { key: 'enabled', title: '状态', class: 'w-[96px]' },
  { key: 'actions', title: '操作', class: 'w-[160px]' },
];

const statusFilterOptions = [
  { label: '全部', value: 'all' },
  { label: '启用', value: 'enabled' },
  { label: '禁用', value: 'disabled' },
];

const methodOptions = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'PATCH', value: 'PATCH' },
  { label: 'DELETE', value: 'DELETE' },
];

const protocolOptions = [
  { label: 'OpenAI Chat', value: 'openai_chat' },
  { label: 'OpenAI Responses', value: 'openai_responses' },
  { label: 'Anthropic Messages', value: 'anthropic_messages' },
  { label: 'Gemini Generate Content', value: 'gemini_generate_content' },
];

const keyword = ref('');
const providerFilter = ref('all');
const statusFilter = ref('all');
const drawerVisible = ref(false);
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

const providerOptions = computed(() => [
  { label: '请选择供应商', value: '' },
  ...providerStore.providers.map((item) => ({
    label: item.name || item.id,
    value: item.id,
  })),
]);

const providerFilterOptions = computed(() => [
  { label: '全部', value: 'all' },
  ...providerStore.providers.map((item) => ({
    label: item.name || item.id,
    value: item.id,
  })),
]);

const filteredEndpoints = computed(() => {
  const q = keyword.value.trim().toLowerCase();
  return endpointStore.endpoints.filter((item) => {
    if (providerFilter.value !== 'all' && item.providerId !== providerFilter.value) return false;
    if (statusFilter.value === 'enabled' && !item.enabled) return false;
    if (statusFilter.value === 'disabled' && item.enabled) return false;
    if (!q) return true;
    return [item.name, item.path, item.requestProtocol, item.responseProtocol].some((value) =>
      (value || '').toLowerCase().includes(q),
    );
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

function handleDrawerClose() {
  drawerVisible.value = false;
  resetForm();
}

function openCreate() {
  resetForm();
  drawerVisible.value = true;
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
  drawerVisible.value = true;
}

async function handleSubmit() {
  if (editingId.value) {
    await endpointStore.updateEndpoint(form.value);
  } else {
    await endpointStore.addEndpoint(form.value);
  }
  handleDrawerClose();
}

async function handleDelete(item) {
  if (!window.confirm(`确认删除端点“${item.name || item.id}”吗？`)) return;
  await endpointStore.deleteEndpoint(item.id);
  if (editingId.value === item.id) {
    handleDrawerClose();
  }
}

onMounted(async () => {
  await Promise.all([providerStore.fetchProviders(), endpointStore.fetchEndpoints()]);
});
</script>

<style scoped>
.page-toolbar {
  margin-top: 16px;
}

.toolbar-field--search {
  min-width: min(320px, 100%);
  flex: 1 1 320px;
}

.toolbar-select {
  min-width: 180px;
}

.table-panel {
  margin-top: 16px;
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
.drawer-footer,
.checkbox-group {
  display: flex;
  gap: 8px;
  align-items: center;
}

.drawer-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
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

.form-hint {
  margin-top: 6px;
  color: var(--text-secondary);
  font-size: 12px;
}

.drawer-footer {
  justify-content: flex-end;
}

@media (max-width: 960px) {
  .form-grid.two-columns {
    grid-template-columns: 1fr;
  }
}
</style>