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
  "status-badge inline-flex items-center gap-2 px-2.5 py-1 rounded-full text-xs font-semibold transition-colors duration-120",
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
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.18);
}

.status-badge--neutral {
  background: color-mix(in srgb, var(--color-bg-tertiary) 92%, white);
  border-color: color-mix(in srgb, var(--color-border) 92%, transparent);
  color: var(--color-text-secondary);
}

.status-badge--success {
  background: color-mix(in srgb, var(--color-success) 12%, white);
  border-color: color-mix(in srgb, var(--color-success) 22%, transparent);
  color: var(--color-success);
}

.status-badge--warning {
  background: color-mix(in srgb, var(--color-warning) 12%, white);
  border-color: color-mix(in srgb, var(--color-warning) 22%, transparent);
  color: #b45309;
}

.status-badge--error {
  background: color-mix(in srgb, var(--color-error) 12%, white);
  border-color: color-mix(in srgb, var(--color-error) 22%, transparent);
  color: #b91c1c;
}

.status-badge--info {
  background: color-mix(in srgb, var(--color-accent) 12%, white);
  border-color: color-mix(in srgb, var(--color-accent) 20%, transparent);
  color: var(--color-accent);
}

.status-badge__dot {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 999px;
  flex-shrink: 0;
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.4);
}

.status-badge__dot--neutral { background: var(--color-text-muted); }
.status-badge__dot--success { background: var(--color-success); }
.status-badge__dot--warning { background: var(--color-warning); }
.status-badge__dot--error { background: var(--color-error); }
.status-badge__dot--info { background: var(--color-accent); }
</style>
