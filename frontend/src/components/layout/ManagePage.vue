<script setup>
import {
  ManagementPageLayout,
  MetricCard,
  DataTable,
} from "@/components/layout"
import {
  Button,
  IconButton,
  SearchInput,
  Select,
} from "@/components/ui"
import {
  Plus,
  Edit,
  Trash2,
  RefreshCw,
} from "lucide-vue-next"

const props = defineProps({
  // 页面配置
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

  // 数据配置
  columns: {
    type: Array,
    required: true,
  },
  data: {
    type: Array,
    default: () => [],
  },
  loading: {
    type: Boolean,
    default: false,
  },
  emptyText: {
    type: String,
    default: "暂无数据",
  },
  renderTable: {
    type: Boolean,
    default: true,
  },

  // 指标卡片配置
  metrics: {
    type: Array,
    default: () => [],
  },

  // 筛选配置
  searchable: {
    type: Boolean,
    default: true,
  },
  searchPlaceholder: {
    type: String,
    default: "搜索...",
  },
  searchValue: {
    type: String,
    default: "",
  },
  filters: {
    type: Array,
    default: () => [],
  },

  // 操作按钮配置
  primaryAction: {
    type: Object,
    default: null,
  },

  // 行操作配置
  rowActions: {
    type: Array,
    default: () => [
      { key: "edit", label: "编辑", icon: Edit, variant: "ghost" },
      { key: "delete", label: "删除", icon: Trash2, variant: "destructive" },
    ],
  },

  // 表单配置（用于新增/编辑弹窗）
  formFields: {
    type: Array,
    default: () => [],
  },
  formTitle: {
    type: String,
    default: "",
  },
  formSize: {
    type: String,
    default: "md",
  },
})

const emit = defineEmits([
  "search",
  "filter-change",
  "row-click",
  "action",
  "refresh",
  "save",
  "delete",
])

// 内部状态
const internalSearchQuery = ref(props.searchValue)
const internalFilters = ref(
  props.filters.reduce((acc, f) => {
    acc[f.key] = f.defaultValue ?? ""
    return acc
  }, {})
)

// 计算属性
const searchQuery = computed({
  get: () => internalSearchQuery.value,
  set: (val) => {
    internalSearchQuery.value = val
    emit("search", val)
  },
})

function handleFilterChange(key, value) {
  internalFilters.value[key] = value
  emit("filter-change", { key, value, filters: { ...internalFilters.value } })
}

function handleRowClick(row, index) {
  emit("row-click", row, index)
}

function handleAction(action, row) {
  emit("action", { action, row })
}

function handleRefresh() {
  emit("refresh")
}
</script>

<template>
  <ManagementPageLayout
    :title="title"
    :description="description"
    :icon="icon"
    :compact="compact"
  >
    <!-- 操作按钮 -->
    <template #actions>
      <IconButton
        v-if="!loading"
        variant="ghost"
        size="sm"
        @click="handleRefresh"
        title="刷新"
      >
        <RefreshCw :size="14" />
      </IconButton>
      <Button
        v-if="primaryAction"
        :variant="primaryAction.variant || 'brand'"
        :size="primaryAction.size || 'sm'"
        @click="handleAction(primaryAction.key || 'add', null)"
      >
        <Plus :size="14" />
        {{ primaryAction.label || "新建" }}
      </Button>
      <slot name="actions" />
    </template>

    <!-- 指标卡片 -->
    <template #metrics>
      <MetricCard
        v-for="metric in metrics"
        :key="metric.key || metric.label"
        :icon="metric.icon"
        :icon-color="metric.iconColor || 'text-primary'"
        :icon-bg="metric.iconBg || 'bg-primary/10'"
        :value="metric.value"
        :label="metric.label"
        :description="metric.description"
      />
      <slot name="metrics" />
    </template>

    <!-- 筛选栏 -->
    <template #filters>
      <SearchInput
        v-if="searchable"
        v-model="searchQuery"
        :placeholder="searchPlaceholder"
        class="query-bar-search"
      />
      <Select
        v-for="filter in filters"
        :key="filter.key"
        :model-value="internalFilters[filter.key]"
        :options="filter.options"
        :placeholder="filter.placeholder || '全部'"
        class="query-bar-filter"
        @update:model-value="handleFilterChange(filter.key, $event)"
      />
      <slot name="filters" />
    </template>

    <!-- 内容区域 -->
    <DataTable
      v-if="renderTable"
      :columns="columns"
      :data="data"
      :loading="loading"
      :empty-text="emptyText"
      @row-click="handleRowClick"
    >
      <!-- 透传单元格插槽 -->
      <template v-for="col in columns" :key="col.key" #[`cell-${col.key}`]="slotProps">
        <slot :name="`cell-${col.key}`" v-bind="slotProps" />
      </template>

      <!-- 默认操作列 -->
      <template v-if="rowActions && rowActions.length > 0" #cell-actions="slotProps">
        <div class="flex items-center gap-1">
          <IconButton
            v-for="action in rowActions"
            :key="action.key"
            :variant="action.variant || 'ghost'"
            size="sm"
            :title="action.label"
            @click="handleAction(action.key, slotProps.row)"
          >
            <component :is="action.icon" :size="14" />
          </IconButton>
          <slot name="row-actions" v-bind="slotProps" />
        </div>
      </template>
    </DataTable>
    <slot v-else />

    <!-- 底部插槽 -->
    <template #footer>
      <slot name="footer" />
    </template>
  </ManagementPageLayout>
  <slot name="overlay" />
</template>
