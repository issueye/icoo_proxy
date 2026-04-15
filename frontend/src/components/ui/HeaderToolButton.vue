<script setup>
import { cn } from "@/lib/utils"

const props = defineProps({
  disabled: {
    type: Boolean,
    default: false,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  active: {
    type: Boolean,
    default: false,
  },
  title: {
    type: String,
    default: "",
  },
  iconSize: {
    type: Number,
    default: 14,
  },
})

const emit = defineEmits(["click"])

function handleClick(event) {
  if (!props.disabled && !props.loading) {
    emit("click", event)
  }
}
</script>

<template>
  <button
    :class="cn(
      'header-tool-btn icon-btn inline-flex items-center justify-center h-7 w-8 p-0',
      'bg-transparent border-none',
      'cursor-pointer transition-all duration-120',
      { 'opacity-50 cursor-not-allowed': disabled || loading },
      $attrs.class
    )"
    :disabled="disabled || loading"
    :title="title"
    @click="handleClick"
  >
    <!-- 加载状态 -->
    <svg
      v-if="loading"
      class="animate-spin"
      :style="{ width: `${iconSize}px`, height: `${iconSize}px` }"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        class="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="4"
      />
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      />
    </svg>
    <!-- 默认插槽（图标） -->
    <slot v-else />
  </button>
</template>
