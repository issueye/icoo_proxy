<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton
          size="sm"
          variant="secondary"
          :loading="store.loading"
          :disabled="store.loading"
          @click="store.load"
        >
          {{ store.loading ? "刷新中..." : "刷新" }}
        </UButton>
      </div>
    </Teleport>

    <div class="stat-grid stat-grid--3">
      <StatCard icon="layers" label="协议总数" :value="String(store.routeDefinitions.length)" tone="info" />
      <StatCard icon="check" label="已启用策略" :value="String(store.enabledPolicyCount)" tone="success" />
      <StatCard icon="server" label="已配置上游" :value="String(store.configuredPolicyCount)" tone="info" />
    </div>

    <UTable
        :columns="routeManagementColumns"
        :rows="store.routeManagementRows"
        row-key="key"
        action-width="74px"
        size="sm"
        fixed
        class="grow"
        table-class="route-management-table"
      >
        <template #empty>
          <div class="empty-action">
            <p class="empty-action__title">当前没有可配置的路由协议</p>
            <p class="empty-action__desc">请先确认网关服务已正常启动，然后刷新路由规则。</p>
            <div class="empty-action__actions">
              <UButton size="sm" variant="primary" :loading="store.loading" @click="store.load">刷新路由规则</UButton>
            </div>
          </div>
        </template>

        <template #cell-protocol="{ row }">
          <div class="route-map__protocol-main" :title="routeProtocolTitle(row)">
            <UTag code size="xs">{{ row.key }}</UTag>
          </div>
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

    <UModal
      v-model:open="policyModalOpen"
      :title="store.policyForm.id ? '编辑路由策略' : '新建路由策略'"
      width="560px"
      @close="store.resetPolicyForm"
    >
      <form id="policy-form" class="space-y-2" @submit.prevent="submitPolicy">
        <div class="grid gap-2 md:grid-cols-2">
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
          />
        </div>

        <USelect
          v-model="store.policyForm.upstream_protocol"
          label="上游协议"
          placeholder="留空则继承供应商协议"
          :options="protocolOptions"
        />

        <USwitch v-model="store.policyForm.enabled" label="启用该路由策略" />
        <USwitch
          v-if="store.policyForm.id"
          v-model="forcePolicyUpdate"
          label="允许强制修改"
          hint="当前规则有活跃请求时仍允许保存。只影响后续新请求，已开始的请求继续使用原路由。"
        />
      </form>
      <template #footer>
        <div class="flex justify-end gap-1.5">
          <UButton size="sm" variant="secondary" @click="closePolicyModal">取消</UButton>
          <UButton
            size="sm"
            form="policy-form"
            variant="primary"
            native-type="submit"
            :loading="store.saving"
            :disabled="store.saving"
          >
            {{ store.saving ? "保存中..." : forcePolicyUpdate ? "强制保存" : "保存路由策略" }}
          </UButton>
        </div>
      </template>
    </UModal>

    <UConfirmDialog
      v-model:open="forceSwitchConfirm.open"
      title="强制修改路由策略"
      message="当前路由策略正在处理请求，是否仍然强制保存？"
      description="强制修改只影响后续新请求，已经开始的请求会继续使用原路由。"
      confirm-text="强制保存"
      cancel-text="取消"
      :loading="store.saving"
      danger
      @confirm="confirmForceSwitch"
    />
  </section>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useSuppliersStore } from "../stores/suppliers";
import { useStoreError } from "../composables/useStoreError";

import StatCard from "../components/StatCard.vue";
import UButton from "../components/ued/UButton.vue";
import UConfirmDialog from "../components/ued/UConfirmDialog.vue";
import UIconButton from "../components/ued/UIconButton.vue";
import UModal from "../components/ued/UModal.vue";
import USelect from "../components/ued/USelect.vue";
import USwitch from "../components/ued/USwitch.vue";
import UTable from "../components/ued/UTable.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";

const store = useSuppliersStore();
useStoreError(store);
const policyModalOpen = ref(false);
const forcePolicyUpdate = ref(false);
const forceSwitchConfirm = reactive({
  open: false,
  isEdit: false,
});

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
  forcePolicyUpdate.value = false;
  store.policyForm.downstream_protocol = protocol;
  policyModalOpen.value = true;
}

function openPolicyEdit(policy) {
  store.selectPolicy(policy);
  forcePolicyUpdate.value = false;
  policyModalOpen.value = true;
}

function closePolicyModal() {
  policyModalOpen.value = false;
  forcePolicyUpdate.value = false;
  store.resetPolicyForm();
}

function routeProtocolTitle(row) {
  return [row.label, row.key, row.description, row.helperText, row.warningText]
    .filter(Boolean)
    .join(" · ");
}

watch(
  () => store.policyForm.supplier_id,
  (newSupplierID, oldSupplierID) => {
    if (!newSupplierID) {
      return;
    }
    const newSupplier = store.allSuppliers.find((item) => item.id === newSupplierID);
    if (!newSupplier) {
      return;
    }
    const oldSupplier = store.allSuppliers.find((item) => item.id === oldSupplierID);
    const currentUpstream = store.policyForm.upstream_protocol;
    // Keep manually selected upstream protocols. Auto-sync only inherited values.
    if (!currentUpstream || (oldSupplier && currentUpstream === oldSupplier.protocol)) {
      store.policyForm.upstream_protocol = newSupplier.protocol;
    }
  }
);

async function submitPolicy() {
  const isEdit = Boolean(store.policyForm.id);
  const result = await store.savePolicy({
    force: forcePolicyUpdate.value,
    silentActiveRuleError: true,
  });
  if (isActiveRuleError(result.error)) {
    forceSwitchConfirm.isEdit = isEdit;
    forceSwitchConfirm.open = true;
    return;
  }
  if (result.ok) {
    policyModalOpen.value = false;
    forcePolicyUpdate.value = false;
    message.success(isEdit ? "路由策略已更新。" : "路由策略已新增。");
  }
}

function isActiveRuleError(error) {
  return String(error || "").includes("active requests");
}

async function confirmForceSwitch() {
  const result = await store.savePolicy({ force: true });
  if (result.ok) {
    forceSwitchConfirm.open = false;
    policyModalOpen.value = false;
    forcePolicyUpdate.value = false;
    message.success(forceSwitchConfirm.isEdit ? "路由策略已更新。" : "路由策略已新增。");
  }
}

onMounted(() => {
  store.load();
});
</script>
