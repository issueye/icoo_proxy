<template>
  <button
    :type="nativeType"
    class="ued-button"
    :class="[
      `ued-button--${normalizedVariant}`,
      `ued-button--${normalizedSize}`,
      {
        'ued-button--block': block,
        'is-loading': loading,
      },
    ]"
    :disabled="disabled || loading"
    @click="$emit('click', $event)"
  >
    <span v-if="loading" class="ued-button__spinner" />
    <span v-else-if="$slots.icon" class="ued-button__icon" aria-hidden="true">
      <slot name="icon" />
    </span>
    <span><slot /></span>
    <span v-if="$slots.suffix" class="ued-button__icon" aria-hidden="true">
      <slot name="suffix" />
    </span>
  </button>
</template>

<script setup>
import { computed } from "vue";
import { normalizeVariant, normalizeSize, BUTTON_VARIANTS, CONTROL_SIZES } from "./variant";

defineEmits(["click"]);

const props = defineProps({
  variant: {
    type: String,
    default: "primary",
  },
  size: {
    type: String,
    default: "md",
  },
  block: {
    type: Boolean,
    default: false,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  nativeType: {
    type: String,
    default: "button",
  },
});

const normalizedVariant = computed(() =>
  normalizeVariant(props.variant, "primary", BUTTON_VARIANTS),
);

const normalizedSize = computed(() =>
  normalizeSize(props.size, "md", CONTROL_SIZES),
);
</script>
