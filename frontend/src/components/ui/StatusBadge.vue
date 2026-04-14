<script setup>
import { cn } from "@/lib/utils"
import { cva } from "class-variance-authority"
import { Loader2 } from "lucide-vue-next"

const props = defineProps({
  status: {
    type: String,
    default: "neutral",
  },
  loading: {
    type: Boolean,
    default: false,
  },
  label: {
    type: String,
    default: "",
  },
  showLabel: {
    type: Boolean,
    default: true,
  },
})

const statusVariants = cva(
  "inline-flex items-center gap-1 h-5 px-1.5 rounded text-xs font-medium transition-colors duration-120",
  {
    variants: {
      status: {
        neutral: "text-muted-foreground hover:bg-secondary",
        success: "text-success hover:bg-secondary",
        warning: "text-warning hover:bg-secondary",
        error: "text-error hover:bg-secondary",
        info: "text-primary hover:bg-secondary",
      },
    },
    defaultVariants: {
      status: "neutral",
    },
  }
)
</script>

<template>
  <div :class="cn(statusVariants({ status }), $attrs.class)">
    <!-- 状态图标 -->
    <Loader2 v-if="loading" :size="10" class="animate-spin" />
    <div
      v-else
      class="w-1.5 h-1.5 rounded-full flex-shrink-0"
      :class="{
        'bg-muted-foreground': status === 'neutral',
        'bg-success': status === 'success',
        'bg-warning': status === 'warning',
        'bg-error': status === 'error',
        'bg-primary': status === 'info',
      }"
    />
    <!-- 标签文字 -->
    <span v-if="showLabel && label" class="truncate">{{ label }}</span>
    <slot />
  </div>
</template>
