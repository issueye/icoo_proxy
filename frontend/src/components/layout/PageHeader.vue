<script setup>
import { cn } from "@/lib/utils"
import Separator from "@/components/ui/Separator.vue"

const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: "",
  },
  icon: {
    type: [Object, Function],
    default: null,
  },
  compact: {
    type: Boolean,
    default: false,
  },
})
</script>

<template>
  <div :class="cn('flex items-start justify-between gap-4', compact ? 'py-2' : 'py-3')">
    <!-- 左侧：标题和描述 -->
    <div class="flex-1 min-w-0">
      <h2 :class="cn('flex items-center gap-2 font-semibold text-foreground', compact ? 'text-sm' : 'text-base')">
        <component :is="icon" v-if="icon" :size="compact ? 16 : 18" class="text-primary flex-shrink-0" />
        {{ title }}
      </h2>
      <p v-if="description" :class="cn('text-muted-foreground mt-0.5', compact ? 'text-xs' : 'text-xs')">
        {{ description }}
      </p>
    </div>

    <!-- 右侧：操作按钮 -->
    <div v-if="$slots.actions" class="flex items-center gap-2 flex-shrink-0">
      <slot name="actions" />
    </div>
  </div>
</template>
