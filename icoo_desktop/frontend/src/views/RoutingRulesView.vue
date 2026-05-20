<template>
  <section class="page-section">
    <!-- 统计卡片部分 -->
    <div class="section-grid grid-cols-1 md:grid-cols-3">
      <StatCard icon="layers" label="协议总数" :value="String(store.routeDefinitions.length)" tone="info" />
      <StatCard icon="check" label="已启用策略" :value="String(store.enabledPolicyCount)" tone="success" />
      <StatCard icon="server" label="已配置上游" :value="String(store.configuredPolicyCount)" tone="info" />
    </div>

    <!-- 协议映射默认路由规则列表 -->
    <div class="section-grid">
      <UTable
        :columns="routeManagementColumns"
        :rows="store.routeManagementRows"
        row-key="key"
        action-width="74px"
        size="small"
        table-class="route-management-table"
      >
        <template #cell-protocol="{ row }">
          <div class="route-map__protocol-main">
            <p class="route-map__name">{{ row.label }}</p>
            <UTag code size="xs">{{ row.key }}</UTag>
          </div>
          <p v-if="row.description" class="route-map__desc">{{ row.description }}</p>
          <p class="route-map__helper">{{ row.helperText }}</p>
          <p v-if="row.warningText" class="route-map__warning">{{ row.warningText }}</p>
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
            <UIconButton
              icon="edit"
              :label="row.policy ? `编辑 ${row.label} 映射` : `配置 ${row.label} 映射`"
              @click="row.policy ? openPolicyEdit(row.policy) : openPolicyCreate(row.key)"
            />
          </div>
        </template>
      </UTable>
    </div>

    <!-- 路由策略编辑弹窗 -->
    <UModal
      v-model:open="policyModalOpen"
      :title="store.policyForm.id ? '编辑路由策略' : '新建路由策略'"
      width="560px"
      @close="store.resetPolicyForm"
    >
      <form id="policy-form" class="space-y-3" @submit.prevent="submitPolicy">
        <div class="grid gap-3 md:grid-cols-2">
          <USelect
            v-model="store.policyForm.downstream_protocol"
            label="下游协议"
            :options="store.policyOptions"
            disabled
          />
          <USelect
            v-model="store.policyForm.supplier_id"
            label="供应商"
            placeholder="请选择供应商"
            :options="supplierOptions"
            @change="handlePolicySupplierChange"
          />
        </div>

        <USelect
          v-model="store.policyForm.upstream_protocol"
          label="上游协议"
          placeholder="留空则继承供应商协议"
          :options="protocolOptions"
        />

        <label class="field-toggle">
          <input v-model="store.policyForm.enabled" type="checkbox" class="field-checkbox" />
          启用该路由策略
        </label>
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="closePolicyModal">取消</button>
          <button
            form="policy-form"
            class="btn btn-primary"
            :class="{ 'is-loading': store.saving }"
            :disabled="store.saving"
          >
            <span v-if="store.saving" class="btn__spinner" />
            {{ store.saving ? "保存中..." : "保存路由策略" }}
          </button>
        </div>
      </template>
    </UModal>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import { useSuppliersStore } from "../stores/suppliers";

import StatCard from "../components/StatCard.vue";
import UIconButton from "../components/ued/UIconButton.vue";
import UModal from "../components/ued/UModal.vue";
import USelect from "../components/ued/USelect.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";

const store = useSuppliersStore();
const policyModalOpen = ref(false);

const protocolOptions = [
  { label: "anthropic", value: "anthropic" },
  { label: "openai-chat", value: "openai-chat" },
  { label: "openai-responses", value: "openai-responses" },
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

function handlePolicySupplierChange(supplierID) {
  const supplier = store.allSuppliers.find((item) => item.id === supplierID);
  store.policyForm.upstream_protocol = supplier?.protocol || "";
}

async function submitPolicy() {
  const isEdit = Boolean(store.policyForm.id);
  await store.savePolicy();
  if (!store.error) {
    policyModalOpen.value = false;
    message.success(isEdit ? "路由策略已更新。" : "路由策略已新增。");
  }
}

onMounted(() => {
  store.load();
});
</script>

<style scoped>
/* 保持简洁与主样式库同步 */
</style>
