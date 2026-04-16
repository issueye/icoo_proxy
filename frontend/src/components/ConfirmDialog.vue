<template>
  <UEDDialog
    :visible="visible"
    :title="state.title"
    :description="state.message"
    width="420px"
    max-width="min(420px, calc(100vw - 32px))"
    @close="handleCancel"
  >
    <div class="confirm-body">
      <div
        :class="[
          'confirm-icon',
          `confirm-icon--${state.type || 'default'}`,
        ]"
      >
        <AlertTriangle
          v-if="state.type === 'danger' || state.type === 'warning'"
          :size="22"
        />
        <Info v-else :size="22" />
      </div>
    </div>

    <template #footer>
      <button
        @click="handleCancel"
        class="btn btn-secondary flex-1"
      >
        {{ state.cancelText }}
      </button>
      <button
        @click="handleConfirm"
        :class="[
          'btn flex-1',
          btnClasses[state.type],
        ]"
      >
        {{ state.confirmText }}
      </button>
    </template>
  </UEDDialog>
</template>

<script setup>
import { AlertTriangle, Info } from "lucide-vue-next";
import { UEDDialog } from "@/components/layout";
import { useConfirm } from "@/composables/useConfirm.js";

const { visible, state, handleConfirm, handleCancel } = useConfirm();

const btnClasses = {
  default: "btn-primary",
  danger: "btn-danger",
  warning: "bg-amber-500 hover:bg-amber-600 text-white border border-amber-600",
};
</script>

<style scoped>
.confirm-body {
  display: flex;
  justify-content: center;
  padding: 4px 0;
}

.confirm-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid currentColor;
  border-color: color-mix(in srgb, currentColor 18%, transparent);
}

.confirm-icon--default {
  background: color-mix(in srgb, var(--ued-accent) 10%, transparent);
  color: var(--ued-accent);
}

.confirm-icon--danger {
  background: var(--ued-danger-soft);
  color: var(--ued-danger);
}

.confirm-icon--warning {
  background: var(--ued-warning-soft);
  color: var(--ued-warning);
}
</style>