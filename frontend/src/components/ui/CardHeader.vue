<script setup>
import { cn } from "@/lib/utils"
</script>

<template>
  <div :class="cn('card-header', $attrs.class)">
    <slot />
  </div>
</template>

<style scoped>
.card-header {
  display: grid;
  gap: var(--space-6);
  padding: var(--panel-padding);
  border-bottom: 1px solid var(--ui-border-subtle);
}
</style>
