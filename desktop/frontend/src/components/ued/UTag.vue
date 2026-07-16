<template>
  <component
    :is="as"
    class="ued-tag"
    :class="[`ued-tag--${normalizedVariant}`, `ued-tag--${normalizedSize}`, { 'ued-tag--code': code, 'ued-tag--dot': dot }]"
  >
    <span v-if="dot" class="ued-tag__dot" aria-hidden="true"></span>
    <span class="ued-tag__content">
      <slot />
    </span>
  </component>
</template>

<script setup>
import { computed } from "vue";
import { normalizeVariant, normalizeSize, TAG_VARIANTS, CONTROL_SIZES } from "./variant";

const props = defineProps({
  variant: {
    type: String,
    default: "neutral",
  },
  size: {
    type: String,
    default: "sm",
  },
  code: {
    type: Boolean,
    default: false,
  },
  dot: {
    type: Boolean,
    default: false,
  },
  as: {
    type: String,
    default: "span",
  },
});

const normalizedVariant = computed(() =>
  normalizeVariant(props.variant, "neutral", TAG_VARIANTS),
);

const normalizedSize = computed(() =>
  normalizeSize(props.size, "sm", CONTROL_SIZES),
);
</script>
