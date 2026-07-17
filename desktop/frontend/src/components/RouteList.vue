<template>
  <div v-if="items.length === 0" class="empty-state">
    {{ emptyText }}
  </div>
  <div v-else class="space-y-1">
    <article v-for="item in items" :key="item.name" class="table-row">
      <div class="min-w-0">
        <p class="text-xs font-medium text-strong truncate">{{ item.name }}</p>
        <p class="mt-0.5 text-xs text-muted truncate">{{ item.upstream }}</p>
      </div>
      <UTag code size="xs">{{ item.model }}</UTag>
    </article>
  </div>
</template>

<script setup>
import UTag from "./ued/UTag.vue";

defineProps({
  items: {
    type: Array,
    default: () => [],
  },
  emptyText: {
    type: String,
    default: "暂无数据。",
  },
});
</script>
