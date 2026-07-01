<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton variant="primary" @click="openSupplierCreate">新建</UButton>
      </div>
    </Teleport>

    <div class="section-grid grid-cols-1 md:grid-cols-3">
      <StatCard icon="server" label="供应商总数" :value="String(store.totalCount)" tone="info" />
      <StatCard icon="check" label="已启用" :value="String(store.enabledCount)" tone="success" />
      <StatCard icon="heart-pulse" label="已健康检查" :value="String(store.checkedCount)" />
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
            <UButton type="button" variant="secondary" @click="resetQuery">重置</UButton>
            <UButton type="button" variant="primary" @click="submitQuery">查询</UButton>
          </div>
        </div>
      </template>
      <template #cell-supplier="{ row }">
        <p class="table-cell-wrap font-medium text-strong" :title="supplierTitle(row)">
          {{ row.name }}<span v-if="row.description" class="text-secondary"> · {{ row.description }}</span>
        </p>
      </template>
      <template #cell-protocol="{ row }">
        <div class="table-cell-inline">
          <span class="table-cell-inline__text font-medium text-strong">{{ row.protocol }}</span>
          <UTag v-if="row.only_stream" variant="warning" size="xs">only_stream</UTag>
        </div>
      </template>
      <template #cell-address="{ row }">
        <p class="table-meta table-cell-wrap" :title="row.base_url">{{ row.base_url }}</p>
      </template>
      <template #cell-user_agent="{ row }">
        <p v-if="row.user_agent" class="table-meta table-cell-wrap" :title="row.user_agent">UA: {{ row.user_agent }}</p>
        <span v-else class="table-meta">使用默认 UA</span>
      </template>
      <template #cell-key="{ row }">
        <UTag code size="xs">{{ row.api_key_masked || "未保存 API Key" }}</UTag>
      </template>
      <template #cell-models="{ row }">
        <div v-if="(row.models || []).length" class="table-cell-inline" :title="formatModelList(row.models)">
          <UTag variant="info" size="xs">{{ formatModelTag(row.models[0]) }}</UTag>
          <span v-if="row.models.length > 1" class="table-meta">+{{ row.models.length - 1 }}</span>
        </div>
        <div v-else class="table-cell-inline">
          <span v-if="!(row.models || []).length" class="table-meta">无模型</span>
        </div>
      </template>
      <template #cell-health="{ row }">
        <template v-if="store.healthFor(row.id)">
          <div class="table-cell-inline" :title="store.healthFor(row.id).message">
            <UTag :variant="healthTone(store.healthFor(row.id))" size="xs">
              {{ store.healthFor(row.id).status }}
            </UTag>
            <UTag variant="info" size="xs">{{ store.healthFor(row.id).duration_ms }} ms</UTag>
            <span class="table-meta table-cell-inline__text">HTTP {{ store.healthFor(row.id).status_code || "无状态码" }}</span>
          </div>
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
        <div class="form-grid">
          <UInput v-model="store.form.name" label="名称" placeholder="例如：OpenAI 生产环境" />
          <USelect v-model="store.form.protocol" label="协议" :options="protocolOptions" />
        </div>

        <div class="form-grid">
          <USelect v-model="store.form.vendor" label="类型" :options="vendorOptions" />
        </div>

        <UInput v-model="store.form.base_url" label="基础地址" placeholder="https://api.openai.com" />

        <UInput v-model="store.form.models_url" label="模型列表地址" placeholder="留空则用基础地址 + /v1/models" />

        <UInput v-model="store.form.api_key" label="API Key" placeholder="编辑时留空则保留已有密钥" />

        <UInput v-model="store.form.user_agent" label="User-Agent" placeholder="留空则使用默认上游 UA" />

        <UInput v-model="store.form.description" label="描述" textarea placeholder="填写该供应商配置的用途说明" />

        <UAlert type="info" message="模型已拆分为独立资源。保存供应商后，请在列表中点击“管理模型”添加候选模型；填写“模型列表地址”可自定义从上游获取模型时使用的接口。" />

        <div class="grid gap-3 md:grid-cols-2">
          <USwitch v-model="store.form.enabled" label="启用该供应商" />
          <USwitch v-model="store.form.only_stream" label="仅流式上游" />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton type="button" variant="secondary" @click="closeSupplierModal">取消</UButton>
          <UButton form="supplier-form" variant="primary" native-type="submit" :loading="store.saving" :disabled="store.saving">
            {{ store.saving ? "保存中..." : store.form.id ? "更新供应商" : "创建供应商" }}
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="modelModalOpen" :title="store.modelForm.id ? `管理模型 - ${store.modelForm.name}` : '管理模型'"
      width="760px" @close="store.resetModelForm">
      <form id="model-form" class="space-y-3" @submit.prevent="submitModelEditor">
        <div class="flex items-center justify-between gap-3">
          <div>
            <p class="text-sm font-medium text-strong">候选模型列表</p>
            <p class="mt-0.5 text-[11px] text-muted">未填写 max_tokens 时会回退到 32768。</p>
          </div>
          <div class="flex items-center gap-2">
            <UButton type="button" variant="secondary" size="xs" :loading="store.fetchingModels" :disabled="store.fetchingModels" @click="fetchModelsForSupplier">
              从上游获取模型
            </UButton>
            <UButton type="button" variant="secondary" size="xs" @click="addModelRow(store.modelForm.models)">
              添加模型
            </UButton>
          </div>
        </div>

        <div class="space-y-2">
          <div v-for="(model, index) in store.modelForm.models" :key="index" class="grid gap-2 md:grid-cols-[minmax(0,1fr)_180px_auto] md:items-end">
            <UInput
              :model-value="model.name"
              :label="`模型 ${index + 1}`"
              :placeholder="index === 0 ? '例如：gpt-4.1-mini' : '继续添加模型'"
              @update:modelValue="updateModelRow(store.modelForm.models, index, 'name', $event)" />
            <UInput
              :model-value="model.max_tokens"
              label="max_tokens"
              type="number"
              placeholder="32768"
              @update:modelValue="updateModelRow(store.modelForm.models, index, 'max_tokens', $event)" />
            <UButton type="button" variant="secondary" size="sm" :disabled="store.modelForm.models.length === 1" @click="removeModelRow(store.modelForm, index)">
              删除
            </UButton>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton type="button" variant="secondary" @click="closeModelModal">取消</UButton>
          <UButton form="model-form" variant="primary" native-type="submit" :loading="store.saving" :disabled="store.saving">
            {{ store.saving ? "保存中..." : "保存模型设置" }}
          </UButton>
        </div>
      </template>
    </UModal>

    <UConfirmDialog v-model:open="confirmState.open" title="确认删除供应商" :message="confirmState.message"
      description="删除后相关模型和路由策略可能需要重新调整。" confirm-text="确认删除" cancel-text="取消"
      :loading="Boolean(store.deleting)" danger @confirm="confirmDelete" />
  </section>
</template>

<script setup>
import { onMounted, reactive, ref } from "vue";
import { useSuppliersStore } from "../stores/suppliers";
import { useStoreError } from "../composables/useStoreError";

import StatCard from "../components/StatCard.vue";
import UAlert from "../components/ued/UAlert.vue";
import UButton from "../components/ued/UButton.vue";
import UConfirmDialog from "../components/ued/UConfirmDialog.vue";
import UIconButton from "../components/ued/UIconButton.vue";
import UInput from "../components/ued/UInput.vue";
import UModal from "../components/ued/UModal.vue";
import USelect from "../components/ued/USelect.vue";
import USwitch from "../components/ued/USwitch.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";

const DEFAULT_MODEL_MAX_TOKENS = 32768;

const store = useSuppliersStore();
useStoreError(store);
const supplierModalOpen = ref(false);
const modelModalOpen = ref(false);
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
    return `未命名模型 - ${getModelMaxTokens(model)}`;
  }
  return `${name} - ${getModelMaxTokens(model)}`;
}

function formatModelList(models) {
  return (models || []).map(formatModelTag).join("，");
}

function supplierTitle(row) {
  return [row.name, row.description].filter(Boolean).join(" · ");
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
  confirmState.message = `确定要删除供应商 "${item.name}" 吗？`;
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
  form.models.splice(index, 1);
}

async function submitSupplier() {
  const isEdit = Boolean(store.form.id);
  await store.save();
  if (!store.error) {
    supplierModalOpen.value = false;
    message.success(isEdit ? "供应商已更新。" : "供应商已新增。请继续配置模型。");
  }
}

async function submitModelEditor() {
  await store.saveModelEditor();
  if (!store.error) {
    modelModalOpen.value = false;
    message.success("模型设置已保存。");
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

async function fetchModelsForSupplier() {
  if (!store.modelForm.id) {
    return;
  }
  const count = await store.fetchModels(store.modelForm.id);
  if (count > 0) {
    message.success(`已从上游获取 ${count} 个新模型。`);
  } else {
    message.info("暂无新模型或该供应商不支持自动获取。");
  }
}

onMounted(() => {
  queryForm.keyword = store.keyword;
  queryForm.protocol = store.protocol;
  store.load();
});
</script>


