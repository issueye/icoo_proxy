<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton variant="primary" @click="openCreate">新建模型</UButton>
      </div>
    </Teleport>

    <div class="model-catalog-toolbar">
      <div>
        <p class="model-catalog-toolbar__title">模型目录</p>
        <p class="model-catalog-toolbar__desc">统一维护可分配给供应商的模型名称、图标和默认上下文参数。</p>
      </div>
      <div class="model-catalog-toolbar__meta">
        <UTag variant="info" size="xs">共 {{ store.items.length }} 个</UTag>
        <UTag variant="neutral" size="xs">自定义 {{ store.customCount }} 个</UTag>
      </div>
    </div>

    <div v-if="store.loading" class="model-catalog-empty">正在加载模型…</div>
    <div v-else-if="!store.items.length" class="model-catalog-empty">
      <p class="empty-action__title">当前尚未配置模型</p>
      <UButton size="sm" variant="primary" @click="openCreate">新建模型</UButton>
    </div>
    <div v-else class="model-card-grid">
      <article v-for="item in store.items" :key="item.id" class="model-card">
        <div class="model-card__header">
          <ModelBrandIcon :icon="item.icon" />
          <div class="model-card__identity">
            <div class="model-card__title-row">
              <h2>{{ item.name }}</h2>
              <UTag v-if="item.built_in" variant="neutral" size="xs">内置</UTag>
            </div>
            <p>{{ item.family || "自定义模型" }}</p>
          </div>
          <div v-if="!item.built_in" class="model-card__actions">
            <UIconButton icon="edit" label="编辑模型" @click="openEdit(item)" />
            <UIconButton icon="delete" label="删除模型" variant="error" :loading="store.deleting === item.id"
              :disabled="store.deleting === item.id" @click="openDelete(item)" />
          </div>
        </div>
        <p class="model-card__description">{{ item.description || "暂无说明" }}</p>
        <div class="model-card__footer">
          <span>默认 max_tokens</span>
          <strong>{{ Number(item.max_tokens || 32768).toLocaleString() }}</strong>
        </div>
      </article>
    </div>

    <UModal v-model:open="modalOpen" :title="store.form.id ? '编辑模型' : '新建模型'" width="560px" @close="store.resetForm">
      <form id="catalog-model-form" class="space-y-3" @submit.prevent="submit">
        <div class="form-grid">
          <UInput v-model="store.form.name" label="模型名称" placeholder="例如：gpt-4.1-mini" required />
          <UInput v-model="store.form.family" label="模型家族" placeholder="例如：OpenAI" />
        </div>
        <div class="form-grid">
          <USelect v-model="store.form.icon" label="图标" :options="iconOptions" />
          <UInput v-model="store.form.max_tokens" label="默认 max_tokens" type="number" placeholder="32768" />
        </div>
        <div class="model-icon-preview">
          <ModelBrandIcon :icon="store.form.icon" />
          <span>{{ selectedIconLabel }}</span>
        </div>
        <UInput v-model="store.form.description" label="说明" textarea placeholder="填写模型用途或版本说明" />
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="secondary" @click="closeModal">取消</UButton>
          <UButton form="catalog-model-form" variant="primary" native-type="submit" :loading="store.saving" :disabled="store.saving">
            {{ store.saving ? "保存中..." : "保存模型" }}
          </UButton>
        </div>
      </template>
    </UModal>

    <UConfirmDialog v-model:open="confirmOpen" title="确认删除模型" :message="confirmMessage"
      description="已分配到供应商的模型不会被自动移除。" confirm-text="确认删除" cancel-text="取消"
      :loading="Boolean(store.deleting)" danger @confirm="confirmDelete" />
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import ModelBrandIcon from "../components/ModelBrandIcon.vue";
import UButton from "../components/ued/UButton.vue";
import UConfirmDialog from "../components/ued/UConfirmDialog.vue";
import UIconButton from "../components/ued/UIconButton.vue";
import UInput from "../components/ued/UInput.vue";
import UModal from "../components/ued/UModal.vue";
import USelect from "../components/ued/USelect.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";
import { useStoreError } from "../composables/useStoreError";
import { useModelCatalogStore } from "../stores/modelCatalog";

const store = useModelCatalogStore();
useStoreError(store);
const modalOpen = ref(false);
const confirmOpen = ref(false);
const deleteTarget = ref(null);

const iconOptions = [
  { label: "GPT", value: "gpt" },
  { label: "DeepSeek", value: "deepseek" },
  { label: "GLM", value: "glm" },
  { label: "Claude", value: "claude" },
  { label: "通用", value: "custom" },
];
const selectedIconLabel = computed(() => iconOptions.find((item) => item.value === store.form.icon)?.label || "通用");
const confirmMessage = computed(() => deleteTarget.value ? `确定要删除模型“${deleteTarget.value.name}”吗？` : "");

function openCreate() { store.resetForm(); modalOpen.value = true; }
function openEdit(item) { store.select(item); modalOpen.value = true; }
function closeModal() { modalOpen.value = false; store.resetForm(); }
function openDelete(item) { deleteTarget.value = item; confirmOpen.value = true; }

async function submit() {
  const isEdit = Boolean(store.form.id);
  await store.save();
  if (!store.error) {
    modalOpen.value = false;
    message.success(isEdit ? "模型已更新。" : "模型已新增。");
  }
}

async function confirmDelete() {
  if (!deleteTarget.value) return;
  await store.remove(deleteTarget.value.id);
  if (!store.error) {
    confirmOpen.value = false;
    deleteTarget.value = null;
    message.success("模型已删除。");
  }
}

onMounted(() => store.load());
</script>
