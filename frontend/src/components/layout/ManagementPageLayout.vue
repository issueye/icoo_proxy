<script setup>
import { cn } from "@/lib/utils"
import PageHeader from "./PageHeader.vue"
import QueryBar from "./QueryBar.vue"
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
  contentClass: {
    type: String,
    default: "",
  },
  compact: {
    type: Boolean,
    default: false,
  },
})
</script>

<template>
  <section class="management-page h-full flex flex-col min-h-0 p-2">
    <!-- 固定头部区域 -->
    <div class="flex-shrink-0">
      <!-- 标题栏 -->
      <PageHeader
        :title="title"
        :description="description"
        :icon="icon"
        :compact="compact"
      >
        <template #actions>
          <slot name="actions" />
        </template>
      </PageHeader>

      <Separator v-if="$slots.metrics || $slots.filters" />

      <!-- 指标卡片 -->
      <div
        v-if="$slots.metrics"
        :class="cn('grid gap-3 py-3', compact ? 'grid-cols-2' : 'grid-cols-4')"
      >
        <slot name="metrics" />
      </div>

      <Separator v-if="$slots.metrics && $slots.filters" />

      <!-- 筛选栏 -->
      <div v-if="$slots.filters" class="py-3">
        <QueryBar :compact="compact">
          <slot name="filters" />
        </QueryBar>
      </div>

      <Separator v-if="$slots.footer" />

      <!-- 底部操作区 -->
      <div v-if="$slots.footer" class="py-3">
        <slot name="footer" />
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="flex-1 min-h-0" :class="contentClass">
      <slot />
    </div>
  </section>
</template>

<style scoped>
@media (max-width: 960px) {
  .grid-cols-4 {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .grid-cols-4,
  .grid-cols-2 {
    grid-template-columns: 1fr;
  }
}
</style>
