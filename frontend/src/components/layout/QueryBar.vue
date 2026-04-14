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
      'query-bar flex flex-wrap items-center content-start gap-3 p-3',
      'rounded-md border border-border bg-secondary',
      compact ? 'gap-2 p-2' : 'gap-3 p-3'
    )"
  >
    <slot />
  </div>
</template>

<style scoped>
.query-bar :deep(.query-bar-search) {
  flex: 0 0 auto;
  width: min(100%, 15rem);
  min-width: 0;
}

.query-bar :deep(.query-bar-filter) {
  flex: 0 0 auto;
  width: min(100%, 11rem);
  min-width: 0;
}

.query-bar :deep(.query-bar-search > *),
.query-bar :deep(.query-bar-filter > *) {
  min-width: 0;
}

@media (max-width: 640px) {
  .query-bar :deep(.query-bar-search),
  .query-bar :deep(.query-bar-filter) {
    width: 100%;
  }
}
</style>
