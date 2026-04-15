<script setup>
import { cn } from "@/lib/utils"
</script>

<template>
  <div
    :class="cn('page-surface rounded-md text-card-foreground', $attrs.class)"
  >
    <slot />
  </div>
</template>
