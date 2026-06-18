<template>
  <section class="content-panel">
    <div class="page-header">
      <div>
        <h2 class="page-title">模型别名管理</h2>
        <p class="page-description">为常用模型名称配置上游协议与目标模型，关联供应商后自动继承协议。</p>
      </div>
      <UButton variant="primary" @click="openCreate">
        <template #icon>
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>
        </template>
        新建模型别名
      </UButton>
    </div>

    <div class="section-grid grid-cols-2 lg:grid-cols-4">
      <StatCard icon="💎" label="别名总数" :value="store.items.length" />
      <StatCard icon="✓" label="已启用" :value="store.enabledCount" />
      <StatCard icon="🏢" label="关联供应商" :value="store.supplierCount" />
      <StatCard icon="📐" label="常用协议" value="Anthropic / OpenAI" />
    </div>

    <div class="section-grid">
      <PanelBlock title="模型别名列表">
        <UTable
          :columns="tableColumns"
          :rows="store.items"
          row-key="id"
          :loading="store.loading"
          loading-text="正在加载别名…"
          size="md"
          table-class="model-alias-table"
        >
          <template #cell-name="{ row }">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-strong">{{ row.name }}</span>
              <UTag :variant="row.enabled ? 'success' : 'error'" size="xs">
                {{ row.enabled ? "启用" : "停用" }}
              </UTag>
            </div>
            <p class="mt-0.5 text-[11px] text-muted">{{ formatTime(row.updated_at) }}</p>
          </template>

          <template #cell-supplier="{ row }">
            {{ row.supplier_name || "未关联" }}
          </template>

          <template #cell-protocol="{ row }">
            {{ row.upstream_protocol || "—" }}
          </template>

          <template #cell-model="{ row }">
            <span class="break-all text-sm text-secondary">{{ row.model }}</span>
          </template>

          <template #actions="{ row }">
            <div class="table-actions">
              <UIconButton icon="edit" label="编辑别名" @click="openEdit(row)" />
              <UIconButton
                icon="delete"
                label="删除别名"
                variant="error"
                :loading="store.deleting === row.id"
                :disabled="store.deleting === row.id"
                @click="removeAlias(row.id)"
              />
            </div>
          </template>
        </UTable>
      </PanelBlock>
    </div>

    <UModal
      v-model:open="modalOpen"
      :title="store.form.id ? '编辑模型别名' : '新建模型别名'"
      width="560px"
      @close="store.resetForm"
    >
      <form id="alias-form" class="space-y-3" @submit.prevent="submitAlias">
        <UInput v-model="store.form.name" label="别名名称" placeholder="例如：fast-model" />

        <USelect
          v-model="store.form.supplier_id"
          label="关联供应商"
          placeholder="请选择供应商"
          :options="store.supplierOptions"
        />

        <USelect
          v-model="store.form.upstream_protocol"
          label="上游协议"
          placeholder="选择上游协议（留空则继承供应商协议）"
          :options="store.upstreamProtocolOptions"
        />

        <div class="flex items-end gap-2">
          <USelect
            v-model="store.form.model"
            label="目标模型"
            placeholder="输入或选择模型"
            :options="store.modelOptions"
            :disabled="!store.form.supplier_id"
            searchable
            class="flex-1"
          />
          <UButton
            type="button"
            variant="secondary"
            size="sm"
            :loading="store.fetchingModels"
            :disabled="!store.form.supplier_id || store.fetchingModels"
            @click="fetchModelsForSupplier"
          >
            获取模型
          </UButton>
        </div>
        <p v-if="store.form.supplier_id && !store.modelOptions.length" class="text-[11px] text-muted">
          所选供应商暂无模型，可直接输入模型名或点击「获取模型」从上游拉取。
        </p>

        <USwitch v-model="store.form.enabled" label="启用该模型别名" />
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="secondary" @click="closeModal">取消</UButton>
          <UButton
            form="alias-form"
            variant="primary"
            native-type="submit"
            :loading="store.saving"
            :disabled="store.saving"
          >
            {{ store.saving ? "保存中..." : store.form.id ? "更新别名" : "创建别名" }}
          </UButton>
        </div>
      </template>
    </UModal>
  </section>
</template>

<script setup>
import { onMounted, ref } from "vue";
import { useModelAliasesStore } from "../stores/modelAliases";

import PanelBlock from "../components/PanelBlock.vue";
import StatCard from "../components/StatCard.vue";
import UButton from "../components/ued/UButton.vue";
import UIconButton from "../components/ued/UIconButton.vue";
import UInput from "../components/ued/UInput.vue";
import UModal from "../components/ued/UModal.vue";
import USelect from "../components/ued/USelect.vue";
import USwitch from "../components/ued/USwitch.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";

const store = useModelAliasesStore();
const modalOpen = ref(false);

const tableColumns = [
  { key: "name", title: "别名名称", width: "220px" },
  { key: "supplier", title: "关联供应商", width: "160px" },
  { key: "protocol", title: "上游协议", width: "140px" },
  { key: "model", title: "目标模型", minWidth: "200px" },
];

function formatTime(value) {
  if (!value) {
    return "—";
  }
  return new Date(value).toLocaleString();
}

function openCreate() {
  store.resetForm();
  modalOpen.value = true;
}

function openEdit(item) {
  store.select(item);
  modalOpen.value = true;
}

function closeModal() {
  modalOpen.value = false;
  store.resetForm();
}

async function submitAlias() {
  const isEdit = Boolean(store.form.id);
  await store.save();
  if (!store.error) {
    modalOpen.value = false;
    message.success(isEdit ? "模型别名已更新。" : "模型别名已新增。");
  }
}

async function removeAlias(id) {
  await store.remove(id);
  if (!store.error) {
    message.success("模型别名已删除。");
  }
}

async function fetchModelsForSupplier() {
  if (!store.form.supplier_id) return;
  const count = await store.fetchModels(store.form.supplier_id);
  if (count > 0) {
    message.success(`已从上游获取 ${count} 个模型。`);
  } else {
    message.info("该供应商暂无可用模型或获取失败。");
  }
}

onMounted(() => {
  store.load();
});
</script>
