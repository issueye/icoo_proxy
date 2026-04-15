<script setup>
import { cn } from "@/lib/utils"
</script>

<template>
  <div :class="cn('card-content', $attrs.class)">
    <slot />
  </div>
</template>

<style scoped>
.card-content {
  padding: var(--panel-padding);
}
</style>
