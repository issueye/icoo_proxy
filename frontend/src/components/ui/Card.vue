<script setup>
import { cn } from "@/lib/utils"
</script>

<template>
  <div
    :class="cn('card-surface page-surface text-card-foreground', $attrs.class)"
  >
    <slot />
  </div>
</template>

<style scoped>
.card-surface {
  border-radius: var(--radius-md);
  overflow: hidden;
}
</style>
