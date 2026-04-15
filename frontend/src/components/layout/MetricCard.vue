<script setup>
import { cn } from "@/lib/utils"

const props = defineProps({
  icon: {
    type: [Object, Function],
    default: null,
  },
  iconColor: {
    type: String,
    default: "text-primary",
  },
  iconBg: {
    type: String,
    default: "bg-primary/10",
  },
  value: {
    type: [String, Number],
    required: true,
  },
  label: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: "",
  },
})
</script>

<template>
  <div class="metric-card page-surface-soft">
    <div class="flex items-center gap-3">
      <!-- 图标 -->
      <div
        v-if="icon"
        :class="cn('w-8 h-8 rounded-md flex items-center justify-center flex-shrink-0', iconBg)"
      >
        <component :is="icon" :size="18" :class="iconColor" />
      </div>

      <!-- 数值和标签 -->
      <div class="min-w-0 flex-1">
        <p class="text-lg font-bold text-foreground truncate">{{ value }}</p>
        <p class="text-xs text-muted-foreground truncate">{{ label }}</p>
        <p v-if="description" class="text-[11px] text-muted-foreground mt-0.5 truncate">
          {{ description }}
        </p>
      </div>
    </div>
  </div>
</template>
