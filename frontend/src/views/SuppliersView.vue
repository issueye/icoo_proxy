<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <button class="btn btn-primary" @click="openSupplierCreate">新建</button>
      </div>
    </Teleport>

    <div v-if="store.error" class="notice-error">
      {{ store.error }}
    </div>

    <div class="section-grid grid-cols-2 lg:grid-cols-4">
      <StatCard icon="server" label="供应商总数" :value="String(store.totalCount)" tone="info" />
      <StatCard icon="check" label="已启用" :value="String(store.enabledCount)" tone="success" />
      <StatCard icon="heart-pulse" label="已健康检查" :value="String(store.checkedCount)" />
      <StatCard icon="layers" label="已配置协议" :value="String(store.configuredPolicyCount)" tone="info" />
    </div>

    <div class="section-grid">
      <UTable :columns="routeManagementColumns" :rows="store.routeManagementRows" row-key="key" action-width="74px"
        size="small" table-class="route-management-table">
        <template #cell-protocol="{ row }">
          <div class="route-map__protocol-main">
            <p class="route-map__name">{{ row.label }}</p>
            <UTag code size="xs">{{ row.key }}</UTag>
          </div>
          <p class="route-map__desc">{{ row.description }}</p>
        </template>

        <template #cell-supplier="{ row }">
          <p class="route-map__value">{{ row.supplierName }}</p>
        </template>

        <template #cell-upstream="{ row }">
          <UTag code size="xs">{{ row.upstreamProtocol }}</UTag>
        </template>

        <template #cell-status="{ row }">
          <UTag :variant="row.statusVariant" size="xs" dot>{{ row.statusText }}</UTag>
        </template>

        <template #actions="{ row }">
          <div class="table-actions">
            <UIconButton icon="edit" :label="row.policy ? `编辑 ${row.label} 映射` : `配置 ${row.label} 映射`"
              @click="row.policy ? openPolicyEdit(row.policy) : openPolicyCreate(row.key)" />
          </div>
        </template>
      </UTable>
    </div>

    <UTable :columns="supplierTableColumns" :rows="store.items" action-width="148px" fixed fixed-field="freeze" min-width="1640px"
      table-class="supplier-table" pagination pagination-mode="server" :page="store.page" :page-size="store.pageSize"
      :total="store.total" :page-size-options="[8, 20, 50]" @page-change="store.changePage">
      <template #empty>
        当前尚未配置供应商。
      </template>
      <template #query>
        <div class="table-query-form">
          <UInput v-model="queryForm.keyword" label="关键词" hide-label placeholder="搜索名称、地址或说明"
            class="table-query-form__field" />
          <USelect v-model="queryForm.protocol" label="协议" hide-label :options="supplierFilterOptions"
            class="table-query-form__field table-query-form__field--compact" />
          <div class="table-query-form__actions">
            <button type="button" class="btn btn-secondary" @click="resetQuery">重置</button>
            <button type="button" class="btn btn-primary" @click="submitQuery">查询</button>
          </div>
        </div>
      </template>
      <template #cell-supplier="{ row }">
        <div class="flex items-center gap-2">
          <p class="font-medium text-[#262626]">{{ row.name }}</p>
        </div>
        <p class="mt-0.5 text-sm leading-5 text-[#595959] table-cell-wrap">
          {{ row.description || "暂无描述。" }}
        </p>
      </template>
      <template #cell-protocol="{ row }">
        <p class="font-medium text-[#262626]">{{ row.protocol }}</p>
        <div class="mt-1 flex flex-wrap gap-1.5">
          <UTag v-if="row.only_stream" variant="warning" size="xs">only_stream</UTag>
        </div>
      </template>
      <template #cell-address="{ row }">
        <p class="mt-0.5 break-all table-meta table-cell-wrap">{{ row.base_url }}</p>
      </template>
      <template #cell-user_agent="{ row }">
        <p v-if="row.user_agent" class="mt-0.5 table-meta table-cell-wrap">UA: {{ row.user_agent }}</p>
        <span v-else class="table-meta">使用默认 UA</span>
      </template>
      <template #cell-key="{ row }">
        <UTag code size="xs">{{ row.api_key_masked || "未保存 API Key" }}</UTag>
      </template>
      <template #cell-models="{ row }">
        <div class="flex flex-wrap gap-1.5">
          <UTag v-for="model in row.models || []" :key="`${model.name}-${model.max_tokens}`"
            :variant="row.default_model === model.name ? 'success' : 'info'" size="xs">
            {{ row.default_model === model.name ? `${formatModelTag(model)} · 默认` : formatModelTag(model) }}
          </UTag>
          <span v-if="!(row.models || []).length" class="table-meta">无模型</span>
        </div>
      </template>
      <template #cell-health="{ row }">
        <template v-if="store.healthFor(row.id)">
          <div class="flex flex-wrap items-center gap-1.5">
            <UTag :variant="healthTone(store.healthFor(row.id))" size="xs">
              {{ store.healthFor(row.id).status }}
            </UTag>
            <UTag variant="info" size="xs">{{ store.healthFor(row.id).duration_ms }} ms</UTag>
          </div>
          <p class="mt-0.5 table-meta">
            HTTP {{ store.healthFor(row.id).status_code || "无状态码" }}
          </p>
          <p class="mt-0.5 text-sm leading-5 text-[#595959] table-cell-wrap">
            {{ store.healthFor(row.id).message }}
          </p>
        </template>
        <span v-else class="table-meta">尚未检查</span>
      </template>
      <template #cell-status="{ row }">
        <UTag :variant="row.enabled ? 'success' : 'error'" size="xs">
          {{ row.enabled ? "启用" : "停用" }}
        </UTag>
      </template>
      <template #actions="{ row }">
        <div class="table-actions">
          <UIconButton icon="inspect" label="检查供应商" variant="info" :loading="store.checking === row.id"
            :disabled="store.checking === row.id" @click="checkSupplier(row.id)" />
          <UIconButton icon="edit" label="编辑供应商" @click="openSupplierEdit(row)" />
          <UIconButton icon="models" label="管理模型" @click="openModelEditor(row)" />
          <UIconButton icon="delete" label="删除供应商" variant="error" :loading="store.deleting === row.id"
            :disabled="store.deleting === row.id" @click="openDeleteConfirm(row)" />
        </div>
      </template>
    </UTable>

    <UModal v-model:open="supplierModalOpen" :title="store.form.id ? '编辑供应商' : '新建供应商'" width="640px"
      @close="store.resetForm">
      <form id="supplier-form" class="space-y-3" @submit.prevent="submitSupplier">
        <div class="grid gap-3 md:grid-cols-2">
          <FieldLabel label="名称">
            <input v-model="store.form.name" class="field-input" placeholder="例如：OpenAI 生产环境" />
          </FieldLabel>
          <USelect v-model="store.form.protocol" label="协议" :options="protocolOptions" />
        </div>

        <div class="grid gap-3 md:grid-cols-2">
          <USelect v-model="store.form.vendor" label="类型" :options="vendorOptions" />
        </div>

        <FieldLabel label="基础地址">
          <input v-model="store.form.base_url" class="field-input" placeholder="https://api.openai.com" />
        </FieldLabel>

        <FieldLabel label="API Key">
          <input v-model="store.form.api_key" class="field-input" placeholder="编辑时留空则保留已有密钥" />
        </FieldLabel>

        <FieldLabel label="User-Agent">
          <input v-model="store.form.user_agent" class="field-input" placeholder="留空则使用默认上游 UA" />
        </FieldLabel>

        <FieldLabel label="描述">
          <textarea v-model="store.form.description" class="field-input min-h-24" placeholder="填写该供应商配置的用途说明" />
        </FieldLabel>

        <div class="grid gap-3 md:grid-cols-2">
          <USelect v-model="store.form.default_model" label="默认模型" placeholder="请先在模型管理中配置候选模型"
            :options="supplierFormDefaultOptions" />
        </div>

        <div class="grid gap-3 md:grid-cols-2">
          <label class="field-toggle">
            <input v-model="store.form.enabled" type="checkbox" class="field-checkbox" />
            启用该供应商配置
          </label>
          <label class="field-toggle">
            <input v-model="store.form.only_stream" type="checkbox" class="field-checkbox" />
            仅流式上游
          </label>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="closeSupplierModal">取消</button>
          <button form="supplier-form" class="btn btn-primary" :class="{ 'is-loading': store.saving }"
            :disabled="store.saving">
            <span v-if="store.saving" class="btn__spinner" />
            {{ store.saving ? "保存中..." : store.form.id ? "更新供应商" : "创建供应商" }}
          </button>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="modelModalOpen" :title="store.modelForm.id ? `管理模型 · ${store.modelForm.name}` : '管理模型'"
      width="760px" @close="store.resetModelForm">
      <form id="model-form" class="space-y-3" @submit.prevent="submitModelEditor">
        <div class="flex items-center justify-between gap-3">
          <div>
            <p class="text-sm font-medium text-[#262626]">候选模型列表</p>
            <p class="mt-0.5 text-[11px] text-[#8c8c8c]">默认模型必须来自当前候选模型列表，未填写 max_tokens 时会回退到 32768。</p>
          </div>
          <button type="button" class="btn btn-secondary px-2 py-1 text-xs"
            @click="addModelRow(store.modelForm.models)">
            添加模型
          </button>
        </div>

        <div class="space-y-2">
          <div v-for="(model, index) in store.modelForm.models" :key="index" class="grid gap-2 md:grid-cols-[minmax(0,1fr)_180px_auto] md:items-end">
            <FieldLabel :label="`模型 ${index + 1}`">
              <input :value="model.name" class="field-input" :placeholder="index === 0 ? '例如：gpt-4.1-mini' : '继续添加模型'"
                @input="updateModelRow(store.modelForm.models, index, 'name', $event.target.value)" />
            </FieldLabel>
            <FieldLabel label="max_tokens">
              <input :value="model.max_tokens" type="number" min="1" step="1" class="field-input"
                placeholder="32768"
                @input="updateModelRow(store.modelForm.models, index, 'max_tokens', $event.target.value)" />
            </FieldLabel>
            <button type="button" class="btn btn-secondary shrink-0 px-2 py-2"
              :disabled="store.modelForm.models.length === 1" @click="removeModelRow(store.modelForm, index)">
              删除
            </button>
          </div>
        </div>

        <USelect v-model="store.modelForm.default_model" label="默认模型" placeholder="不设置默认模型"
          :options="modelEditorDefaultOptions" />
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="closeModelModal">取消</button>
          <button form="model-form" class="btn btn-primary" :class="{ 'is-loading': store.saving }"
            :disabled="store.saving">
            <span v-if="store.saving" class="btn__spinner" />
            {{ store.saving ? "保存中..." : "保存模型设置" }}
          </button>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="policyModalOpen" :title="store.policyForm.id ? '编辑路由策略' : '新建路由策略'" width="560px"
      @close="store.resetPolicyForm">
      <form id="policy-form" class="space-y-3" @submit.prevent="submitPolicy">
        <div class="grid gap-3 md:grid-cols-2">
          <USelect v-model="store.policyForm.downstream_protocol" label="下游协议" :options="store.policyOptions" />
          <USelect v-model="store.policyForm.supplier_id" label="供应商" placeholder="请选择供应商" :options="supplierOptions" />
        </div>

        <label class="field-toggle">
          <input v-model="store.policyForm.enabled" type="checkbox" class="field-checkbox" />
          启用该路由策略
        </label>
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="closePolicyModal">取消</button>
          <button form="policy-form" class="btn btn-primary" :class="{ 'is-loading': store.saving }"
            :disabled="store.saving">
            <span v-if="store.saving" class="btn__spinner" />
            {{ store.saving ? "保存中..." : "保存路由策略" }}
          </button>
        </div>
      </template>
    </UModal>

    <UConfirmDialog v-model:open="confirmState.open" title="确认删除供应商" :message="confirmState.message"
      description="删除后将同时移除该供应商对应的本地健康检查记录，且路由策略可能需要重新调整。" confirm-text="确认删除" cancel-text="取消"
      :loading="Boolean(store.deleting)" danger @confirm="confirmDelete" />
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { useSuppliersStore } from "../stores/suppliers";

import FieldLabel from "../components/FieldLabel.vue";
import StatCard from "../components/StatCard.vue";
import UConfirmDialog from "../components/ued/UConfirmDialog.vue";
import UIconButton from "../components/ued/UIconButton.vue";
import UInput from "../components/ued/UInput.vue";
import UModal from "../components/ued/UModal.vue";
import USelect from "../components/ued/USelect.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";

const DEFAULT_MODEL_MAX_TOKENS = 32768;

const store = useSuppliersStore();
const supplierModalOpen = ref(false);
const modelModalOpen = ref(false);
const policyModalOpen = ref(false);
const queryForm = reactive({
  keyword: "",
  protocol: "all",
});
const confirmState = reactive({
  open: false,
  id: "",
  message: "",
});

const protocolOptions = [
  { label: "anthropic", value: "anthropic" },
  { label: "openai-chat", value: "openai-chat" },
  { label: "openai-responses", value: "openai-responses" },
];

const supplierFilterOptions = [
  { label: "全部协议", value: "all" },
  ...protocolOptions,
];

const vendorOptions = [
  { label: "openai", value: "openai" },
  { label: "deepseek", value: "deepseek" },
  { label: "anthropic", value: "anthropic" },
];

const supplierOptions = computed(() =>
  store.allSuppliers.map((supplier) => ({
    label: `${supplier.name} (${supplier.protocol})`,
    value: supplier.id,
  })),
);

const routeManagementColumns = [
  { key: "protocol", title: "下游协议", width: "40%" },
  { key: "supplier", title: "供应商", width: "20%" },
  { key: "upstream", title: "上游协议", width: "20%" },
  { key: "status", title: "状态", width: "12%" },
];

const supplierTableColumns = [
  { key: "supplier", title: "供应商", width: "12%", freeze: "left" },
  { key: "protocol", title: "协议 / 地址", width: "12%" },
  { key: "address", title: "地址", width: "18%" },
  { key: "user_agent", title: "User-Agent", width: "14%" },
  { key: "key", title: "API Key", width: "12%" },
  { key: "models", title: "模型 / Max Tokens", width: "18%" },
  { key: "health", title: "健康状态", width: "14%" },
  { key: "status", title: "状态", width: "5%", freeze: "right" },
];

const supplierFormDefaultOptions = computed(() => [
  { label: "不设置默认模型", value: "" },
  ...store.form.models
    .map((item) => getModelName(item))
    .filter(Boolean)
    .map((item) => ({ label: item, value: item })),
]);

const modelEditorDefaultOptions = computed(() => [
  { label: "不设置默认模型", value: "" },
  ...store.modelForm.models
    .map((item) => getModelName(item))
    .filter(Boolean)
    .map((item) => ({ label: item, value: item })),
]);

function getModelName(model) {
  return String(model?.name || "").trim();
}

function getModelMaxTokens(model) {
  const parsed = Number.parseInt(model?.max_tokens, 10);
  return parsed > 0 ? parsed : DEFAULT_MODEL_MAX_TOKENS;
}

function formatModelTag(model) {
  const name = getModelName(model);
  if (!name) {
    return `未命名模型 · ${getModelMaxTokens(model)}`;
  }
  return `${name} · ${getModelMaxTokens(model)}`;
}

function healthTone(record) {
  if (!record) {
    return "neutral";
  }
  if (record.status === "reachable") {
    return "success";
  }
  if (record.status === "warning") {
    return "warning";
  }
  return "error";
}

function openDeleteConfirm(item) {
  confirmState.open = true;
  confirmState.id = item.id;
  confirmState.message = `确定要删除供应商"${item.name}"吗？`;
}

async function submitQuery() {
  await store.applyFilters(queryForm);
}

async function resetQuery() {
  queryForm.keyword = "";
  queryForm.protocol = "all";
  await store.resetFilters();
}

function openSupplierCreate() {
  store.resetForm();
  supplierModalOpen.value = true;
}

function openSupplierEdit(item) {
  store.select(item);
  supplierModalOpen.value = true;
}

function closeSupplierModal() {
  supplierModalOpen.value = false;
  store.resetForm();
}

function openModelEditor(item) {
  store.selectModelEditor(item);
  modelModalOpen.value = true;
}

function closeModelModal() {
  modelModalOpen.value = false;
  store.resetModelForm();
}

function addModelRow(target) {
  target.push({
    name: "",
    max_tokens: DEFAULT_MODEL_MAX_TOKENS,
  });
}

function updateModelRow(target, index, field, value) {
  if (!target[index]) {
    return;
  }
  target[index][field] = value;
}

function removeModelRow(form, index) {
  if (form.models.length === 1) {
    return;
  }
  const removed = form.models[index];
  form.models.splice(index, 1);
  if (removed?.name && form.default_model === removed.name) {
    form.default_model = "";
  }
}

async function submitSupplier() {
  const isEdit = Boolean(store.form.id);
  await store.save();
  if (!store.error) {
    supplierModalOpen.value = false;
    message.success(isEdit ? "供应商已更新。" : "供应商已新增。");
  }
}

async function submitModelEditor() {
  await store.saveModelEditor();
  if (!store.error) {
    modelModalOpen.value = false;
    message.success("模型设置已保存。");
  }
}

function openPolicyCreate(protocol = "anthropic") {
  store.resetPolicyForm();
  store.policyForm.downstream_protocol = protocol;
  policyModalOpen.value = true;
}

function openPolicyEdit(policy) {
  store.selectPolicy(policy);
  policyModalOpen.value = true;
}

function closePolicyModal() {
  policyModalOpen.value = false;
  store.resetPolicyForm();
}

async function submitPolicy() {
  const isEdit = Boolean(store.policyForm.id);
  await store.savePolicy();
  if (!store.error) {
    policyModalOpen.value = false;
    message.success(isEdit ? "路由策略已更新。" : "路由策略已新增。");
  }
}

async function confirmDelete() {
  if (!confirmState.id) {
    return;
  }
  await store.remove(confirmState.id);
  if (!store.error) {
    confirmState.open = false;
    confirmState.id = "";
    confirmState.message = "";
    message.success("供应商已删除。");
  }
}

async function checkSupplier(id) {
  await store.check(id);
  if (!store.error) {
    message.success("供应商健康检查完成。");
  }
}

onMounted(() => {
  queryForm.keyword = store.keyword;
  queryForm.protocol = store.protocol;
  store.load();
});
</script>

<style scoped>
.table-query-form {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  flex-wrap: wrap;
}

.table-query-form__field {
  width: 240px;
  min-width: 0;
}

.table-query-form__field--compact {
  width: 180px;
}

.table-query-form__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

@media (max-width: 760px) {

  .table-query-form__field,
  .table-query-form__field--compact,
  .table-query-form__actions {
    width: 100%;
    margin-left: 0;
  }
}
</style>
