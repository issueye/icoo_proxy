<script setup>
import { cn } from "@/lib/utils"

defineProps({
  orientation: {
    type: String,
    default: "horizontal",
  },
  decorative: {
    type: Boolean,
    default: true,
  },
})
</script>

<template>
  <div
    :role="decorative ? 'none' : 'separator'"
    :aria-orientation="decorative ? undefined : orientation"
    :class="cn(
      'shrink-0 bg-border',
      orientation === 'horizontal' ? 'h-[1px] w-full' : 'h-full w-[1px]',
      $attrs.class
    )"
  />
</template>
