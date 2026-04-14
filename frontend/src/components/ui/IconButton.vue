<script setup>
import { cn } from "@/lib/utils"
import { cva } from "class-variance-authority"

const props = defineProps({
  variant: {
    type: String,
    default: "default",
  },
  size: {
    type: String,
    default: "default",
  },
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
})

const emit = defineEmits(["click"])

const iconButtonVariants = cva(
  "inline-flex items-center justify-center border-none cursor-pointer transition-all duration-120 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
  {
    variants: {
      variant: {
        default: "text-muted-foreground hover:text-foreground hover:bg-accent/10",
        primary: "text-primary hover:bg-primary/10",
        destructive: "text-destructive hover:bg-destructive/10",
        ghost: "text-muted-foreground hover:text-foreground hover:bg-secondary",
        status: "text-muted-foreground hover:bg-secondary",
        "status-success": "text-success hover:bg-secondary",
        "status-warning": "text-warning hover:bg-secondary",
        "status-error": "text-error hover:bg-secondary",
      },
      size: {
        sm: "h-5 w-5 p-0.5",
        default: "h-6 w-6 p-1",
        md: "h-8 w-8 p-1.5",
        lg: "h-10 w-10 p-2",
        header: "h-6 w-7 p-0",
      },
    },
    compoundVariants: [
      {
        variant: "default",
        active: true,
        class: "text-foreground bg-accent/10",
      },
      {
        variant: "primary",
        active: true,
        class: "text-primary bg-primary/15",
      },
    ],
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function handleClick(event) {
  if (!props.disabled && !props.loading) {
    emit("click", event)
  }
}
</script>

<template>
  <button
    :class="cn(iconButtonVariants({ variant, size, active }), $attrs.class)"
    :disabled="disabled || loading"
    :title="title"
    @click="handleClick"
  >
    <!-- 加载状态 -->
    <svg
      v-if="loading"
      class="animate-spin"
      :class="cn(size === 'sm' ? 'h-3 w-3' : 'h-4 w-4')"
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
