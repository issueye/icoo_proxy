<script setup>
import { cn } from "@/lib/utils"

defineProps({
  src: {
    type: String,
    default: "",
  },
  alt: {
    type: String,
    default: "",
  },
  fallback: {
    type: String,
    default: "",
  },
})
</script>

<template>
  <div
    :class="cn(
      'relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full',
      $attrs.class
    )"
  >
    <img
      v-if="src"
      :src="src"
      :alt="alt"
      class="aspect-square h-full w-full"
    />
    <div
      v-else
      class="flex h-full w-full items-center justify-center rounded-full bg-muted"
    >
      <span class="text-sm font-medium text-muted-foreground">{{ fallback }}</span>
    </div>
  </div>
</template>
