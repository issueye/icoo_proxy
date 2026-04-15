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
  "icon-btn border-none transition-all duration-120 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        default: "text-muted-foreground hover:text-foreground",
        primary: "text-primary hover:bg-primary/10 hover:text-primary",
        destructive: "text-destructive hover:bg-destructive/10 hover:text-destructive",
        ghost: "text-muted-foreground hover:text-foreground",
        status: "text-muted-foreground hover:text-foreground",
        "status-success": "text-success hover:bg-green-500/10 hover:text-success",
        "status-warning": "text-warning hover:bg-amber-500/10 hover:text-warning",
        "status-error": "text-error hover:bg-red-500/10 hover:text-error",
      },
      size: {
        sm: "h-7 w-7 p-1",
        default: "h-8 w-8 p-1.5",
        md: "h-9 w-9 p-1.5",
        lg: "h-10 w-10 p-2",
        header: "h-7 w-8 rounded-[6px] p-0",
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
