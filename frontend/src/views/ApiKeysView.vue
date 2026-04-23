<template>
  <div class="app-page api-keys-view">
    <UEDPageHeader
      title="API 密钥管理"
      divided
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreate">新增密钥</button>
      </template>
    </UEDPageHeader>

    <section class="toolbar-surface page-toolbar">
      <div class="toolbar-group">
        <div class="toolbar-field toolbar-field--search">
          <label class="toolbar-label">搜索</label>
          <input
            v-model="keyword"
            class="form-input toolbar-input"
            placeholder="按名称、描述或密钥搜索"
          />
        </div>
        <div class="toolbar-field">
          <label class="toolbar-label">范围</label>
          <Select
            v-model="scopeFilter"
            :options="scopeFilterOptions"
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
    </section>

    <section class="table-panel panel-card">
      <DataTable
        :columns="columns"
        :data="filteredApiKeys"
        :loading="apiKeyStore.loading"
        empty-text="暂无 API 密钥"
        row-key="id"
      >
        <template #cell-name="{ row }">
          <div class="cell-main">{{ row.name || '-' }}</div>
          <div class="cell-sub">{{ row.description || '无描述' }}</div>
        </template>

        <template #cell-key="{ row }">
          <code class="secret-code">{{ maskKey(row.key) }}</code>
        </template>

        <template #cell-scopeMode="{ row }">
          <span class="table-tag">{{ row.scopeMode === 'restricted' ? '受限' : '全局' }}</span>
        </template>

        <template #cell-enabled="{ row }">
          <StatusBadge
            :status="row.enabled ? 'success' : 'neutral'"
            :label="row.enabled ? '启用' : '禁用'"
          />
        </template>

        <template #cell-binding="{ row }">
          <div v-if="row.scopeMode !== 'restricted'" class="cell-sub">全局访问</div>
          <template v-else>
            <div class="cell-sub">供应商 {{ row.providerIds.length }} 个</div>
            <div class="cell-sub">端点 {{ row.endpointIds.length }} 个</div>
          </template>
        </template>

        <template #cell-updatedAt="{ row }">
          {{ formatDate(row.updatedAt || row.createdAt) }}
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
      :title="editingId ? '编辑密钥' : '新增密钥'"
      description="配置网关访问密钥及其供应商/端点访问范围。"
      kicker="API KEY"
      width="560px"
      @close="handleDrawerClose"
    >
      <div class="drawer-form">
        <div class="form-grid two-columns">
          <div class="form-field">
            <label class="form-label">名称</label>
            <input
              v-model="form.name"
              class="form-input"
              placeholder="如：Default Gateway Key"
            />
          </div>

          <div class="form-field">
            <label class="form-label">范围模式</label>
            <Select v-model="form.scopeMode" :options="scopeModeOptions" />
          </div>

          <div class="form-field full">
            <label class="form-label">密钥</label>
            <div class="inline-actions inline-actions--stretch">
              <input
                v-model="form.key"
                class="form-input"
                :placeholder="editingId ? '留空则保留原有密钥' : '请输入 API Key'"
              />
              <button class="btn btn-secondary btn-sm" type="button" @click="generateKey">
                生成随机值
              </button>
            </div>
          </div>

          <div class="form-field full">
            <label class="form-label">描述</label>
            <input
              v-model="form.description"
              class="form-input"
              placeholder="说明此密钥的用途"
            />
          </div>

          <div class="form-field full checkbox-field">
            <label class="checkbox-row">
              <input v-model="form.enabled" type="checkbox" />
              <span>启用此密钥</span>
            </label>
          </div>

          <div class="form-field">
            <label class="form-label">关联供应商</label>
            <Select
              v-model="form.providerIds"
              :options="providerOptions"
              :disabled="form.scopeMode !== 'restricted'"
              multiple
              searchable
              placeholder="选择允许访问的供应商"
            />
          </div>

          <div class="form-field">
            <label class="form-label">关联端点</label>
            <Select
              v-model="form.endpointIds"
              :options="endpointOptions"
              :disabled="form.scopeMode !== 'restricted'"
              multiple
              searchable
              placeholder="选择允许访问的端点"
            />
            <p class="form-hint">
              {{ form.scopeMode === 'restricted' ? '仅允许命中的供应商和端点。' : '全局模式下不限制供应商和端点。' }}
            </p>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="drawer-footer">
          <button class="btn btn-secondary" @click="handleDrawerClose">取消</button>
          <button
            class="btn btn-primary"
            :disabled="apiKeyStore.loading || !form.name.trim()"
            @click="handleSubmit"
          >
            {{ apiKeyStore.loading ? '保存中...' : editingId ? '保存修改' : '创建密钥' }}
          </button>
        </div>
      </template>
    </FloatingDrawer>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { DataTable, UEDPageHeader } from '@/components/layout';
import { FloatingDrawer, Select, StatusBadge } from '@/components/ui';
import { useApiKeyStore } from '@/stores/api-key';
import { useEndpointStore } from '@/stores/endpoint';
import { useProviderStore } from '@/stores/provider';

const apiKeyStore = useApiKeyStore();
const endpointStore = useEndpointStore();
const providerStore = useProviderStore();

const columns = [
  { key: 'name', title: '名称' },
  { key: 'key', title: '密钥', class: 'w-[160px]' },
  { key: 'scopeMode', title: '范围', class: 'w-[96px]' },
  { key: 'enabled', title: '状态', class: 'w-[96px]' },
  { key: 'binding', title: '供应商 / 端点', class: 'w-[160px]' },
  { key: 'updatedAt', title: '更新时间', class: 'w-[180px]' },
  { key: 'actions', title: '操作', class: 'w-[160px]' },
];

const scopeFilterOptions = [
  { label: '全部', value: 'all' },
  { label: '受限', value: 'restricted' },
];

const statusFilterOptions = [
  { label: '全部', value: 'all' },
  { label: '启用', value: 'enabled' },
  { label: '禁用', value: 'disabled' },
];

const scopeModeOptions = [
  { label: '全局', value: 'all' },
  { label: '受限', value: 'restricted' },
];

const keyword = ref('');
const scopeFilter = ref('all');
const statusFilter = ref('all');
const drawerVisible = ref(false);
const editingId = ref('');
const form = ref(createEmptyForm());

function createEmptyForm() {
  return {
    id: '',
    name: '',
    key: '',
    description: '',
    enabled: true,
    scopeMode: 'all',
    providerIds: [],
    endpointIds: [],
  };
}

const providerOptions = computed(() =>
  providerStore.providers.map((item) => ({
    label: item.name || item.id,
    value: item.id,
  })),
);

const endpointOptions = computed(() => {
  const providerIds = Array.isArray(form.value.providerIds) ? form.value.providerIds : [];
  return endpointStore.endpoints
    .filter((item) => providerIds.length === 0 || providerIds.includes(item.providerId))
    .map((item) => ({
      label: item.name || item.id,
      value: item.id,
    }));
});

const filteredApiKeys = computed(() => {
  const q = keyword.value.trim().toLowerCase();
  return apiKeyStore.apiKeys.filter((item) => {
    if (scopeFilter.value !== 'all' && item.scopeMode !== scopeFilter.value) return false;
    if (statusFilter.value === 'enabled' && !item.enabled) return false;
    if (statusFilter.value === 'disabled' && item.enabled) return false;
    if (!q) return true;
    return [item.name, item.description, item.key].some((value) =>
      (value || '').toLowerCase().includes(q),
    );
  });
});

watch(
  () => form.value.scopeMode,
  (scopeMode) => {
    if (scopeMode === 'restricted') return;
    form.value.providerIds = [];
    form.value.endpointIds = [];
  },
);

watch(
  () => [...form.value.providerIds],
  (providerIds) => {
    if (!providerIds.length) return;
    const allowedEndpointIds = new Set(
      endpointStore.endpoints
        .filter((item) => providerIds.includes(item.providerId))
        .map((item) => item.id),
    );
    form.value.endpointIds = form.value.endpointIds.filter((id) => allowedEndpointIds.has(id));
  },
);

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
    key: '',
    description: item.description,
    enabled: item.enabled,
    scopeMode: item.scopeMode || 'all',
    providerIds: [...(item.providerIds || [])],
    endpointIds: [...(item.endpointIds || [])],
  };
  drawerVisible.value = true;
}

async function handleSubmit() {
  const payload = {
    ...form.value,
    providerIds: Array.isArray(form.value.providerIds) ? form.value.providerIds : [],
    endpointIds: Array.isArray(form.value.endpointIds) ? form.value.endpointIds : [],
  };

  if (editingId.value) {
    await apiKeyStore.updateAPIKey(payload);
  } else {
    await apiKeyStore.addAPIKey(payload);
  }

  handleDrawerClose();
}

async function handleDelete(item) {
  if (!window.confirm(`确认删除 API 密钥“${item.name || item.id}”吗？`)) return;
  await apiKeyStore.deleteAPIKey(item.id);
  if (editingId.value === item.id) {
    handleDrawerClose();
  }
}

onMounted(async () => {
  await Promise.all([
    providerStore.fetchProviders(),
    endpointStore.fetchEndpoints(),
    apiKeyStore.fetchAPIKeys(),
  ]);
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
  min-width: 160px;
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
.drawer-footer {
  display: flex;
  gap: 8px;
  align-items: center;
}

.inline-actions--stretch .form-input {
  flex: 1;
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

.checkbox-field {
  display: flex;
  align-items: center;
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