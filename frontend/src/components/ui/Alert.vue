<script setup>
import { cva } from "class-variance-authority"
import { cn } from "@/lib/utils"

const props = defineProps({
  variant: {
    type: String,
    default: "default",
  },
})

const alertVariants = cva(
  "alert-box",
  {
    variants: {
      variant: {
        default: "alert-box--default",
        destructive: "alert-box--destructive",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)
</script>

<template>
  <div
    :class="cn(alertVariants({ variant }), $attrs.class)"
    role="alert"
  >
    <slot />
  </div>
</template>

<style scoped>
.alert-box {
  position: relative;
  width: 100%;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  padding: var(--space-12) var(--space-14);
  background: var(--ui-bg-surface-muted);
  color: var(--color-text-primary);
}

.alert-box--default {
  border-color: var(--ui-border-default);
  background: var(--ui-bg-surface-muted);
}

.alert-box--destructive {
  border-color: color-mix(in srgb, var(--color-error) 32%, white);
  background: color-mix(in srgb, var(--color-error) 8%, white);
  color: var(--color-error);
}

.alert-box :deep(svg) {
  position: absolute;
  top: 14px;
  left: 14px;
}

.alert-box :deep(svg ~ *) {
  padding-left: 28px;
}
</style>
