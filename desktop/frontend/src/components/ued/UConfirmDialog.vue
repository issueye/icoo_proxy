<template>
  <UModal :open="open" :title="title" width="420px" @update:open="handleOpen">
    <div class="space-y-1">
      <p class="text-xs leading-5 text-secondary">{{ message }}</p>
      <p v-if="description" class="text-xs leading-4 text-muted">{{ description }}</p>
    </div>
    <template #footer>
      <div class="flex justify-end gap-1.5">
        <UButton size="sm" variant="secondary" @click="handleCancel">{{ cancelText }}</UButton>
        <UButton size="sm" :variant="danger ? 'error' : 'primary'" :loading="loading" @click="$emit('confirm')">
          {{ confirmText }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup>
import UButton from "./UButton.vue";
import UModal from "./UModal.vue";

const emit = defineEmits(["update:open", "confirm", "cancel"]);

defineProps({
  open: {
    type: Boolean,
    default: false,
  },
  title: {
    type: String,
    default: "确认操作",
  },
  message: {
    type: String,
    default: "",
  },
  description: {
    type: String,
    default: "",
  },
  confirmText: {
    type: String,
    default: "确认",
  },
  cancelText: {
    type: String,
    default: "取消",
  },
  loading: {
    type: Boolean,
    default: false,
  },
  danger: {
    type: Boolean,
    default: false,
  },
});

// Mask/ESC dismiss only toggles open — `cancel` is reserved for an explicit
// cancel-button click, so consumers can distinguish "abandoned" from "closed".
function handleOpen(value) {
  emit("update:open", value);
}

function handleCancel() {
  emit("update:open", false);
  emit("cancel");
}
</script>
