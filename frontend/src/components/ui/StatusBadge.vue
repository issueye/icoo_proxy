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
  "status-badge inline-flex items-center gap-2 px-2 py-1 rounded-full text-xs font-semibold transition-colors duration-120",
  {
    variants: {
      status: {
        neutral: "status-badge--neutral",
        success: "status-badge--success",
        warning: "status-badge--warning",
        error: "status-badge--error",
        info: "status-badge--info",
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
      class="status-badge__dot"
      :class="{
        'status-badge__dot--neutral': status === 'neutral',
        'status-badge__dot--success': status === 'success',
        'status-badge__dot--warning': status === 'warning',
        'status-badge__dot--error': status === 'error',
        'status-badge__dot--info': status === 'info',
      }"
    />
    <!-- 标签文字 -->
    <span v-if="showLabel && label" class="truncate">{{ label }}</span>
    <slot />
  </div>
</template>

<style scoped>
.status-badge {
  border: 1px solid transparent;
  min-height: 22px;
}

.status-badge--neutral {
  background: var(--ui-bg-surface-muted);
  border-color: var(--ui-border-default);
  color: var(--color-text-secondary);
}

.status-badge--success {
  background: var(--ui-success-soft);
  border-color: color-mix(in srgb, var(--color-success) 22%, var(--ui-border-default));
  color: var(--color-success);
}

.status-badge--warning {
  background: var(--ui-warning-soft);
  border-color: color-mix(in srgb, var(--color-warning) 22%, var(--ui-border-default));
  color: var(--color-warning);
}

.status-badge--error {
  background: var(--ui-danger-soft);
  border-color: color-mix(in srgb, var(--color-error) 22%, var(--ui-border-default));
  color: var(--color-error);
}

.status-badge--info {
  background: var(--color-accent-soft);
  border-color: color-mix(in srgb, var(--color-accent) 20%, var(--ui-border-default));
  color: var(--color-accent);
}

.status-badge__dot {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 999px;
  flex-shrink: 0;
  box-shadow: none;
}

.status-badge__dot--neutral { background: var(--color-text-muted); }
.status-badge__dot--success { background: var(--color-success); }
.status-badge__dot--warning { background: var(--color-warning); }
.status-badge__dot--error { background: var(--color-error); }
.status-badge__dot--info { background: var(--color-accent); }
</style>
