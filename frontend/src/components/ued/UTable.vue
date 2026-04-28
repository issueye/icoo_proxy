<template>
  <div class="table-shell" :class="shellClasses">
    <div v-if="$slots.query" class="table-query">
      <slot name="query" />
    </div>

    <div class="table-scroll" :style="scrollStyle">
      <table :class="tableClasses" :style="tableStyle">
        <colgroup v-if="tableColumns.length">
          <col
            v-for="column in tableColumns"
            :key="column.uid"
            :style="getColStyle(column)"
          />
        </colgroup>

        <thead v-if="showHeader">
          <tr>
            <th
              v-for="column in tableColumns"
              :key="column.uid"
              scope="col"
              :class="getHeaderClasses(column)"
              :style="getStickyStyle(column)"
            >
              <slot
                v-if="!column.isAction"
                :name="`header-${column.key}`"
                :column="column.raw"
              >
                {{ column.title }}
              </slot>
              <template v-else>{{ column.title }}</template>
            </th>
          </tr>
        </thead>

        <tbody v-if="hasRows">
          <tr
            v-for="(row, rowIndex) in visibleRows"
            :key="resolveRowKey(row, rowIndex)"
            :class="getRowClasses(row, rowIndex)"
          >
            <td
              v-for="column in tableColumns"
              :key="column.uid"
              :class="getCellClasses(column, row, rowIndex)"
              :style="getStickyStyle(column)"
            >
              <slot
                v-if="column.isAction"
                name="actions"
                :row="row"
                :index="rowIndex"
              />

              <template v-else-if="column.ellipsis && !hasCellSlot(column)">
                <UTooltip
                  :content="resolveTooltipContent(column, row, rowIndex)"
                  :disabled="!column.tooltip"
                >
                  <span class="table-cell-ellipsis">
                    {{ formatCellValue(resolveCellValue(column, row, rowIndex)) }}
                  </span>
                </UTooltip>
              </template>

              <slot
                v-else
                :name="`cell-${column.key}`"
                :row="row"
                :value="resolveCellValue(column, row, rowIndex)"
                :column="column.raw"
                :index="rowIndex"
              >
                {{ formatCellValue(resolveCellValue(column, row, rowIndex)) }}
              </slot>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="!hasRows" class="empty-state rounded-none border-0">
        <slot name="empty">{{ emptyText }}</slot>
      </div>
    </div>

    <div v-if="showPagination" class="table-pagination">
      <div v-if="paginationSummary" class="table-pagination__summary">
        {{ paginationSummary }}
      </div>

      <div class="table-pagination__controls">
        <div v-if="showSizeChanger" class="table-pagination__size">
          <span>每页</span>
          <USelect
            class="table-pagination__ued-select"
            label="每页条数"
            hide-label
            :model-value="currentPageSize"
            :options="pageSizeSelectOptions"
            @update:model-value="handlePageSizeChange"
          />
        </div>

        <div class="table-pagination__pages">
          <button
            type="button"
            class="btn btn-secondary table-pagination__button"
            :disabled="currentPage <= 1"
            @click="goToPage(currentPage - 1)"
          >
            上一页
          </button>

          <template v-for="item in pageItems" :key="item.key">
            <span v-if="item.type === 'ellipsis'" class="table-pagination__ellipsis">...</span>
            <button
              v-else
              type="button"
              class="btn btn-secondary table-pagination__button"
              :class="{ 'table-pagination__button--active': item.page === currentPage }"
              @click="goToPage(item.page)"
            >
              {{ item.page }}
            </button>
          </template>

          <button
            type="button"
            class="btn btn-secondary table-pagination__button"
            :disabled="currentPage >= pageCount"
            @click="goToPage(currentPage + 1)"
          >
            下一页
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, useSlots, watch } from "vue";
import USelect from "./USelect.vue";
import UTooltip from "./UTooltip.vue";

const emit = defineEmits(["update:page", "update:pageSize", "page-change"]);

const props = defineProps({
  columns: {
    type: Array,
    default: () => [],
  },
  rows: {
    type: Array,
    default: () => [],
  },
  rowKey: {
    type: [String, Function],
    default: "id",
  },
  actionTitle: {
    type: String,
    default: "操作",
  },
  actionWidth: {
    type: [String, Number],
    default: "96px",
  },
  actionAlign: {
    type: String,
    default: "center",
  },
  emptyText: {
    type: String,
    default: "暂无数据。",
  },
  fixed: {
    type: Boolean,
    default: false,
  },
  fixedField: {
    type: String,
    default: "fixed",
  },
  tableClass: {
    type: [String, Array, Object],
    default: "",
  },
  stripe: {
    type: Boolean,
    default: false,
  },
  stickyHeader: {
    type: Boolean,
    default: true,
  },
  showHeader: {
    type: Boolean,
    default: true,
  },
  maxHeight: {
    type: [String, Number],
    default: "",
  },
  minWidth: {
    type: [String, Number],
    default: "",
  },
  size: {
    type: String,
    default: "middle",
  },
  rowClassName: {
    type: [String, Function],
    default: "",
  },
  pagination: {
    type: Boolean,
    default: false,
  },
  page: {
    type: Number,
    default: 1,
  },
  pageSize: {
    type: Number,
    default: 10,
  },
  total: {
    type: Number,
    default: 0,
  },
  pageSizeOptions: {
    type: Array,
    default: () => [10, 20, 50, 100],
  },
  showSizeChanger: {
    type: Boolean,
    default: true,
  },
  showTotal: {
    type: Boolean,
    default: true,
  },
  paginationMode: {
    type: String,
    default: "client",
  },
});

const slots = useSlots();
const currentPage = ref(normalizePositiveInteger(props.page, 1));
const currentPageSize = ref(normalizePositiveInteger(props.pageSize, 10));

const normalizedColumns = computed(() =>
  props.columns
    .filter((column) => !column.hidden)
    .map((column, index) => normalizeColumn(column, index)),
);

const actionColumn = computed(() => {
  if (!slots.actions) return null;
  const actionWidth = normalizeCssSize(props.actionWidth) || "180px";
  return {
    uid: "__actions__",
    key: "actions",
    title: props.actionTitle,
    width: actionWidth,
    minWidth: actionWidth,
    align: props.actionAlign,
    fixed: "right",
    ellipsis: false,
    tooltip: false,
    isAction: true,
    raw: {
      key: "actions",
      title: props.actionTitle,
    },
  };
});

const tableColumns = computed(() => {
  const columns = [...normalizedColumns.value];
  if (actionColumn.value) {
    columns.push(actionColumn.value);
  }
  return withStickyOffsets(columns);
});

const totalRows = computed(() => {
  if (!props.pagination) {
    return props.rows.length;
  }
  if (props.paginationMode === "server") {
    return normalizePositiveInteger(props.total, props.rows.length);
  }
  return props.rows.length;
});

const pageCount = computed(() => Math.max(1, Math.ceil(totalRows.value / currentPageSize.value)));

const visibleRows = computed(() => {
  if (!props.pagination || props.paginationMode === "server") {
    return props.rows;
  }

  const start = (currentPage.value - 1) * currentPageSize.value;
  return props.rows.slice(start, start + currentPageSize.value);
});

const hasRows = computed(() => visibleRows.value.length > 0);

const showPagination = computed(() => props.pagination);

const hasColumnSizing = computed(() =>
  tableColumns.value.some((column) => Boolean(column.width || column.minWidth)),
);

const shellClasses = computed(() => ({
  "table-shell--empty": !hasRows.value,
  "table-shell--fixed": props.fixed,
  "table-shell--sticky-header": props.stickyHeader,
  "table-shell--with-pagination": showPagination.value,
  "table-shell--with-query": Boolean(slots.query),
}));

const tableClasses = computed(() => [
  "admin-table",
  props.fixed ? "admin-table-fixed" : "",
  props.tableClass,
  props.stripe ? "admin-table--stripe" : "",
  props.stickyHeader ? "admin-table--sticky-header" : "",
  `admin-table--${normalizeSize(props.size)}`,
]);

const scrollStyle = computed(() => {
  const style = {};
  const maxHeight = normalizeCssSize(props.maxHeight);
  if (maxHeight) {
    style.maxHeight = maxHeight;
    style.overflowY = "auto";
  }
  return style;
});

const tableStyle = computed(() => {
  const style = {};
  const minWidth = resolveTableMinWidth();
  if (minWidth) {
    style.minWidth = minWidth;
  }
  return style;
});

const normalizedPageSizeOptions = computed(() => {
  const values = props.pageSizeOptions
    .map((value) => normalizePositiveInteger(value, 0))
    .filter((value) => value > 0);

  if (!values.includes(currentPageSize.value)) {
    values.push(currentPageSize.value);
  }

  return Array.from(new Set(values)).sort((a, b) => a - b);
});

const pageSizeSelectOptions = computed(() =>
  normalizedPageSizeOptions.value.map((value) => ({
    label: `${value} 条`,
    value,
  })),
);

const pageItems = computed(() => buildPageItems(currentPage.value, pageCount.value));

const paginationSummary = computed(() => {
  if (!props.showTotal) {
    return "";
  }

  if (totalRows.value === 0) {
    return "共 0 条";
  }

  const start = props.paginationMode === "server"
    ? (currentPage.value - 1) * currentPageSize.value + 1
    : Math.min((currentPage.value - 1) * currentPageSize.value + 1, totalRows.value);
  const end = Math.min(currentPage.value * currentPageSize.value, totalRows.value);

  return `第 ${start}-${end} 条，共 ${totalRows.value} 条`;
});

watch(
  () => props.page,
  (value) => {
    currentPage.value = clampPage(normalizePositiveInteger(value, 1), pageCount.value);
  },
);

watch(
  () => props.pageSize,
  (value) => {
    currentPageSize.value = normalizePositiveInteger(value, 10);
    currentPage.value = clampPage(currentPage.value, pageCount.value);
  },
);

watch(
  [totalRows, currentPageSize],
  () => {
    currentPage.value = clampPage(currentPage.value, pageCount.value);
  },
);

function normalizeColumn(column, index) {
  const key = String(column.key ?? column.dataIndex ?? `column-${index}`);
  const fixedValue = resolveColumnOption(column, props.fixedField, "fixed");
  return {
    uid: `${key}-${index}`,
    key,
    title: column.title ?? "",
    dataIndex: column.dataIndex ?? column.key,
    width: normalizeCssSize(column.width),
    minWidth: normalizeCssSize(column.minWidth),
    align: normalizeAlign(column.align),
    fixed: normalizeFixed(fixedValue),
    ellipsis: Boolean(column.ellipsis),
    tooltip: column.tooltip,
    className: column.className,
    headerClass: column.headerClass,
    cellClass: column.cellClass,
    isAction: false,
    raw: column,
  };
}

function withStickyOffsets(columns) {
  const next = columns.map((column) => ({ ...column, stickyStyle: {} }));
  let leftOffset = "0px";
  let lastLeftIndex = -1;
  let firstRightIndex = -1;

  next.forEach((column, index) => {
    if (column.fixed === "left") {
      lastLeftIndex = index;
    }
    if (column.fixed === "right" && firstRightIndex === -1) {
      firstRightIndex = index;
    }
  });

  for (const column of next) {
    if (column.fixed !== "left") continue;
    column.stickyStyle.left = leftOffset;
    leftOffset = appendCssSize(leftOffset, column.width);
  }

  let rightOffset = "0px";
  for (let index = next.length - 1; index >= 0; index -= 1) {
    const column = next[index];
    if (column.fixed !== "right") continue;
    column.stickyStyle.right = rightOffset;
    rightOffset = appendCssSize(rightOffset, column.width);
  }

  if (lastLeftIndex >= 0) {
    next[lastLeftIndex].isStickyLeftLast = true;
  }
  if (firstRightIndex >= 0) {
    next[firstRightIndex].isStickyRightFirst = true;
  }

  return next;
}

function appendCssSize(base, size) {
  if (!size) return base;
  if (base === "0px") return size;
  return `calc(${base} + ${size})`;
}

function normalizeCssSize(value) {
  if (value === 0) return "0px";
  if (!value) return "";
  return typeof value === "number" ? `${value}px` : String(value);
}

function normalizePositiveInteger(value, fallback) {
  const numeric = Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) {
    return fallback;
  }
  return Math.floor(numeric);
}

function clampPage(page, maxPage) {
  return Math.min(Math.max(page, 1), Math.max(maxPage, 1));
}

function buildPageItems(current, totalPages) {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, index) => ({
      type: "page",
      page: index + 1,
      key: `page-${index + 1}`,
    }));
  }

  const pages = new Set([1, totalPages, current, current - 1, current + 1]);
  if (current <= 3) {
    pages.add(2);
    pages.add(3);
    pages.add(4);
  }
  if (current >= totalPages - 2) {
    pages.add(totalPages - 1);
    pages.add(totalPages - 2);
    pages.add(totalPages - 3);
  }

  const sortedPages = Array.from(pages)
    .filter((page) => page >= 1 && page <= totalPages)
    .sort((a, b) => a - b);

  const items = [];
  sortedPages.forEach((page, index) => {
    items.push({ type: "page", page, key: `page-${page}` });
    const nextPage = sortedPages[index + 1];
    if (nextPage && nextPage - page > 1) {
      items.push({ type: "ellipsis", key: `ellipsis-${page}-${nextPage}` });
    }
  });

  return items;
}

function resolveTableMinWidth() {
  const explicitMinWidth = normalizeCssSize(props.minWidth);
  if (explicitMinWidth) {
    return explicitMinWidth;
  }
  if (!props.fixed) {
    return "";
  }

  const total = tableColumns.value.reduce((sum, column) => {
    return sum + estimateColumnWidth(column);
  }, 0);
  return total > 0 ? `${Math.max(total, 960)}px` : "960px";
}

function estimateColumnWidth(column) {
  const minWidth = parsePixelSize(column.minWidth);
  if (minWidth) return minWidth;

  const width = parsePixelSize(column.width);
  if (width) return width;

  if (column.isAction) return 180;
  if (column.ellipsis) return 220;
  return 180;
}

function parsePixelSize(value) {
  if (!value) return 0;
  const match = String(value).match(/^(\d+(?:\.\d+)?)px$/);
  return match ? Number(match[1]) : 0;
}

function normalizeAlign(value) {
  if (value === "center" || value === "right") return value;
  return "left";
}

function normalizeFixed(value) {
  if (value === "left" || value === true) return "left";
  if (value === "right") return "right";
  return "";
}

function resolveColumnOption(column, customField, fallbackField) {
  if (customField && Object.prototype.hasOwnProperty.call(column, customField)) {
    return column[customField];
  }
  return column?.[fallbackField];
}

function normalizeSize(value) {
  return ["small", "middle", "large"].includes(value) ? value : "middle";
}

function resolveRowKey(row, rowIndex) {
  if (typeof props.rowKey === "function") {
    return props.rowKey(row, rowIndex);
  }
  return row?.[props.rowKey] ?? rowIndex;
}

function resolveCellValue(column, row, rowIndex) {
  if (typeof column.raw.render === "function") {
    return column.raw.render(row, rowIndex);
  }

  const dataIndex = column.dataIndex;
  if (Array.isArray(dataIndex)) {
    return dataIndex.reduce((value, key) => value?.[key], row);
  }

  return row?.[dataIndex];
}

function resolveTooltipContent(column, row, rowIndex) {
  if (typeof column.tooltip === "function") {
    return column.tooltip(row, rowIndex);
  }
  return formatCellValue(resolveCellValue(column, row, rowIndex));
}

function formatCellValue(value) {
  if (value === null || value === undefined || value === "") {
    return "-";
  }
  return String(value);
}

function hasCellSlot(column) {
  return Boolean(slots[`cell-${column.key}`]);
}

function isStickyColumn(column) {
  return column.fixed === "left" || column.fixed === "right";
}

function getColStyle(column) {
  if (!hasColumnSizing.value) return undefined;
  const style = {};
  if (column.width) {
    style.width = column.width;
  }
  if (column.minWidth) {
    style.minWidth = column.minWidth;
  }
  return Object.keys(style).length ? style : undefined;
}

function getStickyStyle(column) {
  if (!isStickyColumn(column)) return undefined;
  return column.stickyStyle;
}

function getHeaderClasses(column) {
  return [
    column.headerClass,
    column.className,
    getAlignClass(column.align),
    {
      "is-sticky": isStickyColumn(column),
      "is-sticky-left": column.fixed === "left",
      "is-sticky-left-last": column.isStickyLeftLast,
      "is-sticky-right": column.fixed === "right",
      "is-sticky-right-first": column.isStickyRightFirst,
      "actions-header": column.isAction,
    },
  ];
}

function getCellClasses(column, row, rowIndex) {
  return [
    column.cellClass,
    column.className,
    getAlignClass(column.align),
    {
      "is-sticky": isStickyColumn(column),
      "is-sticky-left": column.fixed === "left",
      "is-sticky-left-last": column.isStickyLeftLast,
      "is-sticky-right": column.fixed === "right",
      "is-sticky-right-first": column.isStickyRightFirst,
      "is-ellipsis": column.ellipsis,
      "actions-header": column.isAction,
    },
    typeof column.raw.cellClassName === "function"
      ? column.raw.cellClassName(row, rowIndex)
      : "",
  ];
}

function getRowClasses(row, rowIndex) {
  return [
    {
      "is-striped": props.stripe,
    },
    typeof props.rowClassName === "function"
      ? props.rowClassName(row, rowIndex)
      : props.rowClassName,
  ];
}

function getAlignClass(align) {
  if (align === "center") return "is-align-center";
  if (align === "right") return "is-align-right";
  return "is-align-left";
}

function goToPage(page) {
  if (!props.pagination) {
    return;
  }

  const nextPage = clampPage(normalizePositiveInteger(page, 1), pageCount.value);
  if (nextPage === currentPage.value) {
    return;
  }

  currentPage.value = nextPage;
  emit("update:page", nextPage);
  emit("page-change", { page: nextPage, pageSize: currentPageSize.value });
}

function handlePageSizeChange(value) {
  const nextPageSize = normalizePositiveInteger(value, currentPageSize.value);
  if (nextPageSize === currentPageSize.value) {
    return;
  }

  currentPageSize.value = nextPageSize;
  currentPage.value = clampPage(1, Math.ceil(totalRows.value / nextPageSize));
  emit("update:pageSize", nextPageSize);
  emit("update:page", currentPage.value);
  emit("page-change", { page: currentPage.value, pageSize: nextPageSize });
}
</script>

<style scoped>
.table-query {
  padding: 14px;
  border-bottom: 1px solid var(--ued-color-divider, #e6ebf2);
  background: #fbfcfe;
}

.table-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-top: 1px solid var(--ued-color-divider, #e6ebf2);
  background: #fbfcfe;
}

.table-pagination__summary {
  min-width: 0;
  color: #6b7a90;
  font-size: 12px;
  line-height: 18px;
}

.table-pagination__controls {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  flex-wrap: wrap;
  margin-left: auto;
}

.table-pagination__size {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #6b7a90;
  font-size: 12px;
  line-height: 18px;
}

.table-pagination__ued-select {
  min-width: 0;
}

.table-pagination__ued-select :deep(.ued-field) {
  display: inline-flex;
  align-items: center;
}

.table-pagination__ued-select :deep(.ued-select__control) {
  min-width: 92px;
  min-height: 28px;
  padding-top: 0;
  padding-bottom: 0;
  font-size: 12px;
}

.table-pagination__ued-select :deep(.ued-select__menu) {
  min-width: 92px;
}

.table-pagination__pages {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.table-pagination__button {
  min-width: 32px;
  min-height: 28px;
  padding: 0 10px;
  box-shadow: none;
}

.table-pagination__button--active {
  border-color: var(--ued-color-primary, #3157d5);
  background: var(--ued-color-primary, #3157d5);
  color: #ffffff;
}

.table-pagination__button--active:hover:not(:disabled) {
  border-color: var(--ued-color-primary-hover, #2448bd);
  background: var(--ued-color-primary-hover, #2448bd);
  color: #ffffff;
}

.table-pagination__ellipsis {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  color: #8b98ab;
  font-size: 12px;
  line-height: 18px;
}

@media (max-width: 760px) {
  .table-pagination {
    align-items: flex-start;
    flex-direction: column;
  }

  .table-pagination__controls {
    width: 100%;
    margin-left: 0;
    justify-content: space-between;
  }
}
</style>
