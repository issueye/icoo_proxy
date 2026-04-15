<script setup>
import { cn } from "@/lib/utils"
import PageHeader from "./PageHeader.vue"
import QueryBar from "./QueryBar.vue"

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
  <section class="management-page">
    <div class="management-page__header">
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

      <div
        v-if="$slots.metrics"
        :class="cn('management-page__metrics', compact ? 'grid-cols-2' : 'grid-cols-4')"
      >
        <slot name="metrics" />
      </div>

      <div v-if="$slots.filters" class="management-page__filters">
        <QueryBar :compact="compact">
          <slot name="filters" />
        </QueryBar>
      </div>

      <div v-if="$slots.footer" class="management-page__footer">
        <slot name="footer" />
      </div>
    </div>

    <div class="management-page__body" :class="contentClass">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.management-page {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  gap: 14px;
}

.management-page__header {
  flex-shrink: 0;
}

.management-page__metrics {
  display: grid;
  gap: 10px;
}

.management-page__filters,
.management-page__footer {
  margin-top: 12px;
}

.management-page__body {
  flex: 1;
  min-height: 0;
}

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
