<script setup>
import { cn } from "@/lib/utils"

const props = defineProps({
  compact: {
    type: Boolean,
    default: false,
  },
})
</script>

<template>
  <div
    :class="cn(
      'query-bar toolbar-surface',
      compact ? 'query-bar--compact' : ''
    )"
  >
    <slot />
  </div>
</template>

<style scoped>
.query-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  align-content: flex-start;
  gap: var(--control-gap);
  padding: var(--space-10) var(--panel-padding);
}

.query-bar--compact {
  padding: var(--space-8) var(--space-10);
}

.query-bar :deep(.query-bar-search) {
  flex: 0 0 auto;
  width: min(100%, 16rem);
  min-width: 0;
}

.query-bar :deep(.query-bar-filter) {
  flex: 0 0 auto;
  width: min(100%, 12rem);
  min-width: 0;
}

.query-bar :deep(.query-bar-search > *),
.query-bar :deep(.query-bar-filter > *) {
  min-width: 0;
}

.query-bar :deep(.toolbar-field),
.query-bar :deep(.field) {
  gap: var(--space-6);
}

@media (max-width: 640px) {
  .query-bar :deep(.query-bar-search),
  .query-bar :deep(.query-bar-filter) {
    width: 100%;
  }
}
</style>
