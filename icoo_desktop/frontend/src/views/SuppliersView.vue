<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton variant="primary" @click="openSupplierCreate">鏂板缓</UButton>
      </div>
    </Teleport>

    <div class="section-grid grid-cols-1 md:grid-cols-3">
      <StatCard icon="server" label="渚涘簲鍟嗘€绘暟" :value="String(store.totalCount)" tone="info" />
      <StatCard icon="check" label="宸插惎鐢? :value="String(store.enabledCount)" tone="success" />
      <StatCard icon="heart-pulse" label="宸插仴搴锋鏌? :value="String(store.checkedCount)" />
    </div>

    <UTable :columns="supplierTableColumns" :rows="store.items" action-width="148px" fixed fixed-field="freeze" min-width="1640px"
      table-class="supplier-table" pagination pagination-mode="server" :page="store.page" :page-size="store.pageSize"
      :total="store.total" :page-size-options="[8, 20, 50]" @page-change="store.changePage">
      <template #empty>
        褰撳墠灏氭湭閰嶇疆渚涘簲鍟嗐€?
      </template>
      <template #query>
        <div class="table-query-form">
          <UInput v-model="queryForm.keyword" label="鍏抽敭璇? hide-label placeholder="鎼滅储鍚嶇О銆佸湴鍧€鎴栬鏄?
            class="table-query-form__field" />
          <USelect v-model="queryForm.protocol" label="鍗忚" hide-label :options="supplierFilterOptions"
            class="table-query-form__field table-query-form__field--compact" />
          <div class="table-query-form__actions">
            <UButton type="button" variant="secondary" @click="resetQuery">閲嶇疆</UButton>
            <UButton type="button" variant="primary" @click="submitQuery">鏌ヨ</UButton>
          </div>
        </div>
      </template>
      <template #cell-supplier="{ row }">
        <div class="flex items-center gap-2">
          <p class="font-medium text-strong">{{ row.name }}</p>
        </div>
        <p class="mt-0.5 text-sm leading-5 text-secondary table-cell-wrap">
          {{ row.description || "鏆傛棤鎻忚堪銆? }}
        </p>
      </template>
      <template #cell-protocol="{ row }">
        <p class="font-medium text-strong">{{ row.protocol }}</p>
        <div class="mt-1 flex flex-wrap gap-1.5">
          <UTag v-if="row.only_stream" variant="warning" size="xs">only_stream</UTag>
        </div>
      </template>
      <template #cell-address="{ row }">
        <p class="mt-0.5 break-all table-meta table-cell-wrap">{{ row.base_url }}</p>
      </template>
      <template #cell-user_agent="{ row }">
        <p v-if="row.user_agent" class="mt-0.5 table-meta table-cell-wrap">UA: {{ row.user_agent }}</p>
        <span v-else class="table-meta">浣跨敤榛樿 UA</span>
      </template>
      <template #cell-key="{ row }">
        <UTag code size="xs">{{ row.api_key_masked || "鏈繚瀛?API Key" }}</UTag>
      </template>
      <template #cell-models="{ row }">
        <div class="flex flex-wrap gap-1.5">
          <UTag v-for="model in row.models || []" :key="`${model.name}-${model.max_tokens}`"
            variant="info" size="xs">
            {{ formatModelTag(model) }}
          </UTag>
          <span v-if="!(row.models || []).length" class="table-meta">鏃犳ā鍨?/span>
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
            HTTP {{ store.healthFor(row.id).status_code || "鏃犵姸鎬佺爜" }}
          </p>
          <p class="mt-0.5 text-sm leading-5 text-secondary table-cell-wrap">
            {{ store.healthFor(row.id).message }}
          </p>
        </template>
        <span v-else class="table-meta">灏氭湭妫€鏌?/span>
      </template>
      <template #cell-status="{ row }">
        <UTag :variant="row.enabled ? 'success' : 'error'" size="xs">
          {{ row.enabled ? "鍚敤" : "鍋滅敤" }}
        </UTag>
      </template>
      <template #actions="{ row }">
        <div class="table-actions">
          <UIconButton icon="inspect" label="妫€鏌ヤ緵搴斿晢" variant="info" :loading="store.checking === row.id"
            :disabled="store.checking === row.id" @click="checkSupplier(row.id)" />
          <UIconButton icon="edit" label="缂栬緫渚涘簲鍟? @click="openSupplierEdit(row)" />
          <UIconButton icon="models" label="绠＄悊妯″瀷" @click="openModelEditor(row)" />
          <UIconButton icon="delete" label="鍒犻櫎渚涘簲鍟? variant="error" :loading="store.deleting === row.id"
            :disabled="store.deleting === row.id" @click="openDeleteConfirm(row)" />
        </div>
      </template>
    </UTable>

    <UModal v-model:open="supplierModalOpen" :title="store.form.id ? '缂栬緫渚涘簲鍟? : '鏂板缓渚涘簲鍟?" width="640px"
      @close="store.resetForm">
      <form id="supplier-form" class="space-y-3" @submit.prevent="submitSupplier">
        <div class="form-grid">
          <UInput v-model="store.form.name" label="鍚嶇О" placeholder="渚嬪锛歄penAI 鐢熶骇鐜" />
          <USelect v-model="store.form.protocol" label="鍗忚" :options="protocolOptions" />
        </div>

        <div class="form-grid">
          <USelect v-model="store.form.vendor" label="绫诲瀷" :options="vendorOptions" />
        </div>

        <UInput v-model="store.form.base_url" label="鍩虹鍦板潃" placeholder="https://api.openai.com" />

        <UInput v-model="store.form.models_url" label="妯″瀷鍒楄〃鍦板潃" placeholder="鐣欑┖鍒欑敤 鍩虹鍦板潃 + /v1/models" />

        <UInput v-model="store.form.api_key" label="API Key" placeholder="缂栬緫鏃剁暀绌哄垯淇濈暀宸叉湁瀵嗛挜" />

        <UInput v-model="store.form.user_agent" label="User-Agent" placeholder="鐣欑┖鍒欎娇鐢ㄩ粯璁や笂娓?UA" />

        <UInput v-model="store.form.description" label="鎻忚堪" textarea placeholder="濉啓璇ヤ緵搴斿晢閰嶇疆鐨勭敤閫旇鏄? />

        <UAlert type="info" message="妯″瀷宸叉媶鍒嗕负鐙珛璧勬簮銆備繚瀛樹緵搴斿晢鍚庯紝璇峰湪鍒楄〃涓偣鍑烩€滅鐞嗘ā鍨嬧€濇坊鍔犲€欓€夋ā鍨嬶紱濉啓鈥滄ā鍨嬪垪琛ㄥ湴鍧€鈥濆彲鑷畾涔変粠涓婃父鑾峰彇妯″瀷鏃朵娇鐢ㄧ殑鎺ュ彛銆? />

        <div class="grid gap-3 md:grid-cols-2">
          <USwitch v-model="store.form.enabled" label="鍚敤璇ヤ緵搴斿晢" />
          <USwitch v-model="store.form.only_stream" label="浠呮祦寮忎笂娓? />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton type="button" variant="secondary" @click="closeSupplierModal">鍙栨秷</UButton>
          <UButton form="supplier-form" variant="primary" native-type="submit" :loading="store.saving" :disabled="store.saving">
            {{ store.saving ? "淇濆瓨涓?.." : store.form.id ? "鏇存柊渚涘簲鍟? : "鍒涘缓渚涘簲鍟? }}
          </UButton>
        </div>
      </template>
    </UModal>

    <UModal v-model:open="modelModalOpen" :title="store.modelForm.id ? `绠＄悊妯″瀷 路 ${store.modelForm.name}` : '绠＄悊妯″瀷'"
      width="760px" @close="store.resetModelForm">
      <form id="model-form" class="space-y-3" @submit.prevent="submitModelEditor">
        <div class="flex items-center justify-between gap-3">
          <div>
            <p class="text-sm font-medium text-strong">鍊欓€夋ā鍨嬪垪琛?/p>
            <p class="mt-0.5 text-[11px] text-muted">鏈～鍐?max_tokens 鏃朵細鍥為€€鍒?32768銆?/p>
          </div>
          <div class="flex items-center gap-2">
            <UButton type="button" variant="secondary" size="xs" :loading="store.fetchingModels" :disabled="store.fetchingModels" @click="fetchModelsForSupplier">
              浠庝笂娓歌幏鍙栨ā鍨?
            </UButton>
            <UButton type="button" variant="secondary" size="xs" @click="addModelRow(store.modelForm.models)">
              娣诲姞妯″瀷
            </UButton>
          </div>
        </div>

        <div class="space-y-2">
          <div v-for="(model, index) in store.modelForm.models" :key="index" class="grid gap-2 md:grid-cols-[minmax(0,1fr)_180px_auto] md:items-end">
            <UInput
              :model-value="model.name"
              :label="`妯″瀷 ${index + 1}`"
              :placeholder="index === 0 ? '渚嬪锛歡pt-4.1-mini' : '缁х画娣诲姞妯″瀷'"
              @update:modelValue="updateModelRow(store.modelForm.models, index, 'name', $event)" />
            <UInput
              :model-value="model.max_tokens"
              label="max_tokens"
              type="number"
              placeholder="32768"
              @update:modelValue="updateModelRow(store.modelForm.models, index, 'max_tokens', $event)" />
            <UButton type="button" variant="secondary" size="sm" :disabled="store.modelForm.models.length === 1" @click="removeModelRow(store.modelForm, index)">
              鍒犻櫎
            </UButton>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton type="button" variant="secondary" @click="closeModelModal">鍙栨秷</UButton>
          <UButton form="model-form" variant="primary" native-type="submit" :loading="store.saving" :disabled="store.saving">
            {{ store.saving ? "淇濆瓨涓?.." : "淇濆瓨妯″瀷璁剧疆" }}
          </UButton>
        </div>
      </template>
    </UModal>

    <UConfirmDialog v-model:open="confirmState.open" title="纭鍒犻櫎渚涘簲鍟? :message="confirmState.message"
      description="鍒犻櫎鍚庣浉鍏虫ā鍨嬪拰璺敱绛栫暐鍙兘闇€瑕侀噸鏂拌皟鏁淬€? confirm-text="纭鍒犻櫎" cancel-text="鍙栨秷"
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
  { label: "鍏ㄩ儴鍗忚", value: "all" },
  ...protocolOptions,
];

const vendorOptions = [
  { label: "openai", value: "openai" },
  { label: "deepseek", value: "deepseek" },
  { label: "anthropic", value: "anthropic" },
];



const supplierTableColumns = [
  { key: "supplier", title: "渚涘簲鍟?, width: "12%", freeze: "left" },
  { key: "protocol", title: "鍗忚 / 鍦板潃", width: "12%" },
  { key: "address", title: "鍦板潃", width: "18%" },
  { key: "user_agent", title: "User-Agent", width: "14%" },
  { key: "key", title: "API Key", width: "12%" },
  { key: "models", title: "妯″瀷 / Max Tokens", width: "18%" },
  { key: "health", title: "鍋ュ悍鐘舵€?, width: "14%" },
  { key: "status", title: "鐘舵€?, width: "5%", freeze: "right" },
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
    return `鏈懡鍚嶆ā鍨?路 ${getModelMaxTokens(model)}`;
  }
  return `${name} 路 ${getModelMaxTokens(model)}`;
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
  confirmState.message = `纭畾瑕佸垹闄や緵搴斿晢 "${item.name}" 鍚楋紵`;
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
    message.success(isEdit ? "渚涘簲鍟嗗凡鏇存柊銆? : "渚涘簲鍟嗗凡鏂板銆傝缁х画閰嶇疆妯″瀷銆?);
  }
}

async function submitModelEditor() {
  await store.saveModelEditor();
  if (!store.error) {
    modelModalOpen.value = false;
    message.success("妯″瀷璁剧疆宸蹭繚瀛樸€?);
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
    message.success("渚涘簲鍟嗗凡鍒犻櫎銆?);
  }
}

async function checkSupplier(id) {
  await store.check(id);
  if (!store.error) {
    message.success("渚涘簲鍟嗗仴搴锋鏌ュ畬鎴愩€?);
  }
}

async function fetchModelsForSupplier() {
  if (!store.modelForm.id) {
    return;
  }
  const count = await store.fetchModels(store.modelForm.id);
  if (count > 0) {
    message.success(`宸蹭粠涓婃父鑾峰彇 ${count} 涓柊妯″瀷銆俙);
  } else {
    message.info("鏆傛棤鏂版ā鍨嬫垨璇ヤ緵搴斿晢涓嶆敮鎸佽嚜鍔ㄨ幏鍙栥€?);
  }
}

onMounted(() => {
  queryForm.keyword = store.keyword;
  queryForm.protocol = store.protocol;
  store.load();
});
</script>


