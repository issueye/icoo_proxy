<script setup>
import { cn } from "@/lib/utils"
</script>

<template>
  <h3 :class="cn('card-title', $attrs.class)">
    <slot />
  </h3>
</template>

<style scoped>
.card-title {
  margin: 0;
  font-size: 15px;
  line-height: 1.4;
  font-weight: 600;
  color: var(--color-text-primary);
}
</style>
