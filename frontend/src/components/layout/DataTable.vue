<script setup>
import { cn } from "@/lib/utils"

const props = defineProps({
  columns: {
    type: Array,
    required: true,
  },
  data: {
    type: Array,
    required: true,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  emptyText: {
    type: String,
    default: "暂无数据",
  },
  rowKey: {
    type: String,
    default: "id",
  },
  hoverable: {
    type: Boolean,
    default: true,
  },
  bordered: {
    type: Boolean,
    default: false,
  },
  size: {
    type: String,
    default: "default",
  },
})

const emit = defineEmits(["row-click"])

function handleRowClick(row, index) {
  emit("row-click", row, index)
}
</script>

<template>
  <div class="data-table-shell overflow-x-auto">
    <table
      :class="cn(
        'data-table w-full text-sm',
        bordered ? 'border-collapse border border-border' : ''
      )"
    >
      <!-- 表头 -->
      <thead>
        <tr :class="cn('text-muted-foreground')">
          <th
            v-for="col in columns"
            :key="col.key || col.title"
            :class="cn(
              'px-3 py-2 text-left font-medium text-[11px]',
              bordered ? 'border border-border' : '',
              size === 'sm' ? 'px-2 py-1.5' : '',
              size === 'lg' ? 'px-4 py-2' : '',
              col.class || ''
            )"
          >
            {{ col.title }}
          </th>
        </tr>
      </thead>

      <!-- 表体 -->
      <tbody>
        <!-- 加载状态 -->
        <tr v-if="loading">
          <td :colspan="columns.length" class="py-7 text-center text-muted-foreground">
            <div class="flex items-center justify-center gap-2">
              <svg class="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              <span>加载中...</span>
            </div>
          </td>
        </tr>

        <!-- 空状态 -->
        <tr v-else-if="data.length === 0">
          <td :colspan="columns.length" class="py-10 text-center text-muted-foreground">
            {{ emptyText }}
          </td>
        </tr>

        <!-- 数据行 -->
        <tr
          v-else
          v-for="(row, index) in data"
          :key="row[rowKey] || index"
          :class="cn(
            'border-t border-border transition-colors',
            hoverable ? 'hover:bg-accent/35' : '',
            size === 'sm' ? 'text-xs' : ''
          )"
          @click="handleRowClick(row, index)"
        >
          <td
            v-for="col in columns"
            :key="col.key || col.title"
            :class="cn(
              'px-3 py-2.5',
              bordered ? 'border border-border' : '',
              size === 'sm' ? 'px-2 py-1.5' : '',
              size === 'lg' ? 'px-4 py-3' : '',
              col.class || '',
              col.align === 'center' ? 'text-center' : '',
              col.align === 'right' ? 'text-right' : ''
            )"
          >
            <!-- 自定义渲染 -->
            <slot :name="`cell-${col.key}`" :row="row" :column="col" :value="row[col.key]" :index="index">
              {{ row[col.key] ?? "-" }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.data-table-shell {
  border: 1px solid color-mix(in srgb, var(--color-border) 90%, transparent);
  border-radius: var(--radius-lg);
  background: color-mix(in srgb, var(--color-bg-secondary) 96%, white);
  box-shadow: var(--shadow-sm);
}

.data-table {
  border-collapse: separate;
  border-spacing: 0;
}

.data-table thead th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: color-mix(in srgb, var(--color-bg-tertiary) 94%, white);
  border-bottom: 1px solid color-mix(in srgb, var(--color-border) 92%, transparent);
}

.data-table tbody tr:first-child td {
  border-top: 0;
}
</style>
