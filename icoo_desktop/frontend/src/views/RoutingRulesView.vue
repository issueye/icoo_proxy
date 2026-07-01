<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton
          variant="secondary"
          :loading="store.loading"
          :disabled="store.loading"
          @click="store.load"
        >
          {{ store.loading ? "鍒锋柊涓?.." : "鍒锋柊" }}
        </UButton>
      </div>
    </Teleport>

    <!-- 缁熻鍗＄墖閮ㄥ垎 -->
    <div class="section-grid grid-cols-1 md:grid-cols-3">
      <StatCard icon="layers" label="鍗忚鎬绘暟" :value="String(store.routeDefinitions.length)" tone="info" />
      <StatCard icon="check" label="宸插惎鐢ㄧ瓥鐣? :value="String(store.enabledPolicyCount)" tone="success" />
      <StatCard icon="server" label="宸查厤缃笂娓? :value="String(store.configuredPolicyCount)" tone="info" />
    </div>

    <!-- 鍗忚鏄犲皠榛樿璺敱瑙勫垯鍒楄〃 -->
    <div class="section-grid">
      <UTable
        :columns="routeManagementColumns"
        :rows="store.routeManagementRows"
        row-key="key"
        action-width="74px"
        size="sm"
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
              :label="row.policy ? `缂栬緫 ${row.label} 鏄犲皠` : `閰嶇疆 ${row.label} 鏄犲皠`"
              @click="row.policy ? openPolicyEdit(row.policy) : openPolicyCreate(row.key)"
            />
          </div>
        </template>
      </UTable>
    </div>

    <!-- 璺敱绛栫暐缂栬緫寮圭獥 -->
    <UModal
      v-model:open="policyModalOpen"
      :title="store.policyForm.id ? '缂栬緫璺敱绛栫暐' : '鏂板缓璺敱绛栫暐'"
      width="560px"
      @close="store.resetPolicyForm"
    >
      <form id="policy-form" class="space-y-3" @submit.prevent="submitPolicy">
        <div class="grid gap-3 md:grid-cols-2">
          <USelect
            v-model="store.policyForm.downstream_protocol"
            label="涓嬫父鍗忚"
            :options="store.policyOptions"
            disabled
          />
          <USelect
            v-model="store.policyForm.supplier_id"
            label="渚涘簲鍟?
            placeholder="璇烽€夋嫨渚涘簲鍟?
            :options="supplierOptions"
          />
        </div>

        <USelect
          v-model="store.policyForm.upstream_protocol"
          label="涓婃父鍗忚"
          placeholder="鐣欑┖鍒欑户鎵夸緵搴斿晢鍗忚"
          :options="protocolOptions"
        />

        <USwitch v-model="store.policyForm.enabled" label="鍚敤璇ヨ矾鐢辩瓥鐣? />
      </form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <UButton variant="secondary" @click="closePolicyModal">鍙栨秷</UButton>
          <UButton
            form="policy-form"
            variant="primary"
            native-type="submit"
            :loading="store.saving"
            :disabled="store.saving"
          >
            {{ store.saving ? "淇濆瓨涓?.." : "淇濆瓨璺敱绛栫暐" }}
          </UButton>
        </div>
      </template>
    </UModal>
  </section>
</template>

<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useSuppliersStore } from "../stores/suppliers";
import { useStoreError } from "../composables/useStoreError";

import StatCard from "../components/StatCard.vue";
import UButton from "../components/ued/UButton.vue";
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
  { key: "protocol", title: "涓嬫父鍗忚", width: "40%" },
  { key: "supplier", title: "渚涘簲鍟?, width: "20%" },
  { key: "upstream", title: "涓婃父鍗忚", width: "20%" },
  { key: "status", title: "鐘舵€?, width: "12%" },
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
    // 浠呭湪鏈墜鍔ㄦ寚瀹氫笂娓稿崗璁紙绌哄€硷級鎴栧綋鍓嶅€肩户鎵胯嚜鏃т緵搴斿晢鏃讹紝鑷姩鍚屾鏂颁緵搴斿晢鍗忚
    if (!currentUpstream || (oldSupplier && currentUpstream === oldSupplier.protocol)) {
      store.policyForm.upstream_protocol = newSupplier.protocol;
    }
  }
);

async function submitPolicy() {
  const isEdit = Boolean(store.policyForm.id);
  await store.savePolicy();
  if (!store.error) {
    policyModalOpen.value = false;
    message.success(isEdit ? "璺敱绛栫暐宸叉洿鏂般€? : "璺敱绛栫暐宸叉柊澧炪€?);
  }
}

onMounted(() => {
  store.load();
});
</script>

</style>
