<template>
  <div class="table-shell" :class="shellClasses">
    <div v-if="$slots.query" class="table-query">
      <slot name="query" />
    </div>

    <!-- Outer frame (padding when query+pagination). Scroll lives on the INNER pane. -->
    <div
      class="table-body"
      :class="{ 'table-body--padded': hasQueryAndPagination }"
    >
      <div
        class="table-scroll"
        :class="{ 'table-scroll--empty': !hasRows && !loading }"
        :style="scrollStyle"
      >
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
                :title="column.title || undefined"
              >
                <slot
                  v-if="column.isSelection"
                  name="header-selection"
                  :selected="allVisibleSelected"
                  :indeterminate="someVisibleSelected"
                  :select="onSelectAllVisible"
                >
                  <input
                    type="checkbox"
                    class="table-selection__checkbox"
                    :checked="allVisibleSelected"
                    :indeterminate.prop="someVisibleSelected"
                    :aria-label="allVisibleSelected ? '取消全选' : '全选当前页'"
                    @click.prevent="onSelectAllVisible(!allVisibleSelected)"
                  />
                </slot>
                <slot
                  v-else-if="!column.isAction"
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
              :tabindex="rowClickable ? 0 : undefined"
              @click="handleRowClick(row, rowIndex, $event)"
              @keydown.enter.prevent="handleRowClick(row, rowIndex, $event)"
            >
              <td
                v-for="column in tableColumns"
                :key="column.uid"
                :class="getCellClasses(column, row, rowIndex)"
                :style="getStickyStyle(column)"
                :title="resolveCellTitle(column, row, rowIndex)"
                @click.stop="column.isSelection ? toggleRowSelection(row, rowIndex) : undefined"
              >
                <slot
                  v-if="column.isSelection"
                  name="cell-selection"
                  :row="row"
                  :selected="isRowSelectedByIndex(rowIndex)"
                >
                  <input
                    type="checkbox"
                    class="table-selection__checkbox"
                    :checked="isRowSelectedByIndex(rowIndex)"
                    :aria-label="`选择行 ${resolveRowKey(row, rowIndex)}`"
                    readonly
                    tabindex="-1"
                  />
                </slot>

                <div v-else-if="column.isAction" class="table-actions">
                  <slot name="actions" :row="row" :index="rowIndex" />
                </div>

                <div v-else class="table-cell-content">
                  <template v-if="!cellSlotMap[column.key]">
                    <UTooltip
                      v-if="column.tooltip"
                      :content="resolveTooltipContent(column, row, rowIndex)"
                    >
                      <span class="table-cell-ellipsis">
                        {{ formatCellValue(resolveCellValue(column, row, rowIndex)) }}
                      </span>
                    </UTooltip>
                    <span v-else class="table-cell-ellipsis">
                      {{ formatCellValue(resolveCellValue(column, row, rowIndex)) }}
                    </span>
                  </template>
                  <slot
                    v-else
                    :name="`cell-${column.key}`"
                    :row="row"
                    :value="resolveCellValue(column, row, rowIndex)"
                    :column="column.raw"
                    :index="rowIndex"
                  />
                </div>
              </td>
            </tr>
          </tbody>
        </table>

        <div
          v-if="!hasRows && !loading"
          class="table-empty-state empty-state rounded-none border-0"
        >
          <slot name="empty">{{ emptyText }}</slot>
        </div>
      </div>
    </div>

    <div v-if="loading" class="table-loading" aria-live="polite" role="status">
      <span class="table-loading__spinner" aria-hidden="true" />
      <span class="table-loading__text">{{ loadingText }}</span>
    </div>

    <div v-if="showPagination" class="table-pagination" role="navigation" aria-label="表格分页">
      <div class="table-pagination__left">
        <slot name="pagination-left" :selected-count="selectedCount">
          <span v-if="selectable && selectedCount > 0" class="table-pagination__selected">
            已选 {{ selectedCount }} 项
          </span>
        </slot>
      </div>
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
          <UButton
            variant="secondary"
            size="sm"
            class="table-pagination__button"
            :disabled="currentPage <= 1"
            aria-label="首页"
            @click="goToPage(1)"
          >
            «
          </UButton>
          <UButton
            variant="secondary"
            size="sm"
            class="table-pagination__button"
            :disabled="currentPage <= 1"
            aria-label="上一页"
            @click="goToPage(currentPage - 1)"
          >
            上一页
          </UButton>

          <template v-for="item in pageItems" :key="item.key">
            <span v-if="item.type === 'ellipsis'" class="table-pagination__ellipsis" aria-hidden="true">…</span>
            <UButton
              v-else
              variant="secondary"
              size="sm"
              class="table-pagination__button"
              :class="{ 'table-pagination__button--active': item.page === currentPage }"
              :aria-current="item.page === currentPage ? 'page' : undefined"
              :aria-label="`第 ${item.page} 页`"
              @click="goToPage(item.page)"
            >
              {{ item.page }}
            </UButton>
          </template>

          <UButton
            variant="secondary"
            size="sm"
            class="table-pagination__button"
            :disabled="currentPage >= pageCount"
            aria-label="下一页"
            @click="goToPage(currentPage + 1)"
          >
            下一页
          </UButton>
          <UButton
            variant="secondary"
            size="sm"
            class="table-pagination__button"
            :disabled="currentPage >= pageCount"
            aria-label="末页"
            @click="goToPage(pageCount)"
          >
            »
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, useSlots, watch } from "vue";
import UButton from "./UButton.vue";
import USelect from "./USelect.vue";
import UTooltip from "./UTooltip.vue";
import {
  buildPageItems,
  clampPage,
  estimateColumnWidth,
  formatCellValue,
  normalizeAlign,
  normalizeCssSize,
  normalizeFixed,
  normalizePositiveInteger,
  normalizeSize,
  resolveColumnOption,
  withStickyOffsets,
} from "./tableUtils";

const emit = defineEmits([
  "update:page",
  "update:pageSize",
  "update:selectedKeys",
  "page-change",
  "row-click",
  "selection-change",
]);

const props = defineProps({
  columns: { type: Array, default: () => [] },
  rows: { type: Array, default: () => [] },
  rowKey: { type: [String, Function], default: "id" },
  actionTitle: { type: String, default: "操作" },
  actionWidth: { type: [String, Number], default: "96px" },
  actionAlign: { type: String, default: "center" },
  emptyText: { type: String, default: "暂无数据。" },
  /** Fixed layout improves ellipsis stability across columns. */
  fixed: { type: Boolean, default: true },
  fixedField: { type: String, default: "fixed" },
  tableClass: { type: [String, Array, Object], default: "" },
  stripe: { type: Boolean, default: false },
  stickyHeader: { type: Boolean, default: true },
  showHeader: { type: Boolean, default: true },
  maxHeight: { type: [String, Number], default: "" },
  minWidth: { type: [String, Number], default: "" },
  /** Compact density by default (sm) for console UIs. */
  size: { type: String, default: "sm" },
  rowClassName: { type: [String, Function], default: "" },
  loading: { type: Boolean, default: false },
  loadingText: { type: String, default: "加载中…" },
  selectable: { type: Boolean, default: false },
  selectionKey: { type: [String, Function], default: "" },
  selectedKeys: { type: Array, default: () => [] },
  rowClickable: { type: Boolean, default: false },
  pagination: { type: Boolean, default: false },
  page: { type: Number, default: 1 },
  pageSize: { type: Number, default: 10 },
  total: { type: Number, default: 0 },
  pageSizeOptions: { type: Array, default: () => [10, 20, 50, 100] },
  showSizeChanger: { type: Boolean, default: true },
  showTotal: { type: Boolean, default: true },
  paginationMode: { type: String, default: "client" },
});

const slots = useSlots();
const currentPage = ref(normalizePositiveInteger(props.page, 1));
const currentPageSize = ref(normalizePositiveInteger(props.pageSize, 10));

/** Precompute which cell slots exist — avoid repeated slots lookups per cell. */
const cellSlotMap = computed(() => {
  const map = Object.create(null);
  for (const name of Object.keys(slots)) {
    if (name.startsWith("cell-")) {
      map[name.slice(5)] = true;
    }
  }
  return map;
});

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
    isSelection: false,
    raw: { key: "actions", title: props.actionTitle },
  };
});

const selectionColumn = computed(() => {
  if (!props.selectable) return null;
  return {
    uid: "__selection__",
    key: "__selection__",
    title: "",
    width: "40px",
    minWidth: "40px",
    align: "center",
    fixed: "left",
    ellipsis: false,
    tooltip: false,
    isAction: false,
    isSelection: true,
    raw: { key: "__selection__", title: "" },
  };
});

const tableColumns = computed(() => {
  const columns = [];
  if (selectionColumn.value) columns.push(selectionColumn.value);
  columns.push(...normalizedColumns.value);
  if (actionColumn.value) columns.push(actionColumn.value);
  return withStickyOffsets(columns);
});

const totalRows = computed(() => {
  if (!props.pagination) return props.rows.length;
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

// --- Selection (O(1) lookups via Set; avoid indexOf) --------------------------
const selectedKeySet = computed(() => new Set(props.selectedKeys));

function resolveSelectionKey(row, rowIndex) {
  const keySource = props.selectionKey || props.rowKey;
  if (typeof keySource === "function") return keySource(row, rowIndex);
  return row?.[keySource] ?? rowIndex;
}

function isRowSelectedByIndex(rowIndex) {
  const row = visibleRows.value[rowIndex];
  if (!row) return false;
  return selectedKeySet.value.has(resolveSelectionKey(row, rowIndex));
}

const visibleRowKeys = computed(() =>
  visibleRows.value.map((row, index) => resolveSelectionKey(row, index)),
);

const allVisibleSelected = computed(
  () =>
    visibleRowKeys.value.length > 0 &&
    visibleRowKeys.value.every((key) => selectedKeySet.value.has(key)),
);

const someVisibleSelected = computed(
  () =>
    !allVisibleSelected.value &&
    visibleRowKeys.value.some((key) => selectedKeySet.value.has(key)),
);

const selectedCount = computed(() => selectedKeySet.value.size);

function emitSelection(nextKeys) {
  emit("update:selectedKeys", nextKeys);
  emit("selection-change", nextKeys);
}

function toggleRowSelection(row, rowIndex) {
  const key = resolveSelectionKey(row, rowIndex);
  const next = new Set(props.selectedKeys);
  if (next.has(key)) next.delete(key);
  else next.add(key);
  emitSelection(Array.from(next));
}

function onSelectAllVisible(select) {
  const next = new Set(props.selectedKeys);
  visibleRowKeys.value.forEach((key) => {
    if (select) next.add(key);
    else next.delete(key);
  });
  emitSelection(Array.from(next));
}

function handleRowClick(row, rowIndex, event) {
  if (!props.rowClickable) return;
  // Ignore clicks originating from interactive controls inside the row.
  const target = event?.target;
  if (target?.closest?.("button, a, input, select, textarea, label, .table-actions")) {
    return;
  }
  emit("row-click", { row, index: rowIndex, event });
}

const showPagination = computed(() => props.pagination);
const hasQuery = computed(() => Boolean(slots.query));
/** Query + pagination together: inset the table body for clearer framing. */
const hasQueryAndPagination = computed(() => hasQuery.value && showPagination.value);

const hasColumnSizing = computed(() =>
  tableColumns.value.some((column) => Boolean(column.width || column.minWidth)),
);

const shellClasses = computed(() => ({
  "table-shell--empty": !hasRows.value,
  "table-shell--fixed": props.fixed,
  "table-shell--sticky-header": props.stickyHeader,
  "table-shell--with-pagination": showPagination.value,
  "table-shell--with-query": hasQuery.value,
  "table-shell--with-query-pagination": hasQueryAndPagination.value,
  "table-shell--loading": props.loading,
  "table-shell--selectable": props.selectable,
  [`table-shell--size-${normalizeSize(props.size)}`]: true,
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
  if (minWidth) style.minWidth = minWidth;
  return style;
});

const normalizedPageSizeOptions = computed(() => {
  const values = props.pageSizeOptions
    .map((value) => normalizePositiveInteger(value, 0))
    .filter((value) => value > 0);
  if (!values.includes(currentPageSize.value)) values.push(currentPageSize.value);
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
  if (!props.showTotal) return "";
  if (totalRows.value === 0) return "共 0 条";
  const start =
    props.paginationMode === "server"
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

watch([totalRows, currentPageSize], () => {
  currentPage.value = clampPage(currentPage.value, pageCount.value);
});

function normalizeColumn(column, index) {
  const key = String(column.key ?? column.dataIndex ?? `column-${index}`);
  const fixedValue = resolveColumnOption(column, props.fixedField, "fixed");
  // Ellipsis is on by default; set column.ellipsis = false to allow wrap/grow.
  const ellipsis = column.ellipsis !== false;
  return {
    uid: `${key}-${index}`,
    key,
    title: column.title ?? "",
    dataIndex: column.dataIndex ?? column.key,
    width: normalizeCssSize(column.width),
    minWidth: normalizeCssSize(column.minWidth),
    align: normalizeAlign(column.align),
    fixed: normalizeFixed(fixedValue),
    ellipsis,
    tooltip: column.tooltip,
    className: column.className,
    headerClass: column.headerClass,
    cellClass: column.cellClass,
    isAction: false,
    isSelection: false,
    raw: column,
  };
}

function resolveTableMinWidth() {
  const explicitMinWidth = normalizeCssSize(props.minWidth);
  if (explicitMinWidth) return explicitMinWidth;
  if (!props.fixed) return "";
  const total = tableColumns.value.reduce((sum, column) => sum + estimateColumnWidth(column), 0);
  return total > 0 ? `${Math.max(total, 720)}px` : "720px";
}

function resolveRowKey(row, rowIndex) {
  if (typeof props.rowKey === "function") return props.rowKey(row, rowIndex);
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
  if (typeof column.tooltip === "function") return column.tooltip(row, rowIndex);
  if (typeof column.raw?.titleText === "function") {
    return column.raw.titleText(row, rowIndex);
  }
  return formatCellValue(resolveCellValue(column, row, rowIndex));
}

/** Native title on <td> for overflow; empty for action/selection cells. */
function resolveCellTitle(column, row, rowIndex) {
  if (column.isAction || column.isSelection || column.ellipsis === false) {
    return undefined;
  }
  if (column.raw?.title === false || column.raw?.nativeTitle === false) {
    return undefined;
  }
  if (typeof column.raw?.titleText === "function") {
    const text = column.raw.titleText(row, rowIndex);
    return text ? String(text) : undefined;
  }
  if (typeof column.tooltip === "function") {
    const text = column.tooltip(row, rowIndex);
    return text ? String(text) : undefined;
  }
  // Custom slots: prefer explicit titleText; else data field string.
  const value = resolveCellValue(column, row, rowIndex);
  if (value === null || value === undefined || value === "") {
    return undefined;
  }
  if (typeof value === "object") {
    return undefined;
  }
  const text = formatCellValue(value);
  return text === "-" ? undefined : text;
}

function isStickyColumn(column) {
  return column.fixed === "left" || column.fixed === "right";
}

function getColStyle(column) {
  if (!hasColumnSizing.value) return undefined;
  const style = {};
  if (column.width) style.width = column.width;
  if (column.minWidth) style.minWidth = column.minWidth;
  return Object.keys(style).length ? style : undefined;
}

function getStickyStyle(column) {
  if (!isStickyColumn(column)) return undefined;
  const style = { ...column.stickyStyle };
  if (column.width) {
    style.width = column.width;
    style.minWidth = column.width;
    style.maxWidth = column.width;
  }
  return style;
}

function getAlignClass(align) {
  if (align === "center") return "is-align-center";
  if (align === "right") return "is-align-right";
  return "is-align-left";
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
      "selection-header": column.isSelection,
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
      "actions-cell": column.isAction,
      "selection-cell": column.isSelection,
    },
    typeof column.raw.cellClassName === "function"
      ? column.raw.cellClassName(row, rowIndex)
      : "",
  ];
}

function getRowClasses(row, rowIndex) {
  return [
    {
      "is-striped": props.stripe && rowIndex % 2 === 1,
      "is-selected": props.selectable && isRowSelectedByIndex(rowIndex),
      "is-clickable": props.rowClickable,
    },
    typeof props.rowClassName === "function"
      ? props.rowClassName(row, rowIndex)
      : props.rowClassName,
  ];
}

function goToPage(page) {
  if (!props.pagination) return;
  const nextPage = clampPage(normalizePositiveInteger(page, 1), pageCount.value);
  if (nextPage === currentPage.value) return;
  currentPage.value = nextPage;
  emit("update:page", nextPage);
  emit("page-change", { page: nextPage, pageSize: currentPageSize.value });
}

function handlePageSizeChange(value) {
  const nextPageSize = normalizePositiveInteger(value, currentPageSize.value);
  if (nextPageSize === currentPageSize.value) return;
  currentPageSize.value = nextPageSize;
  currentPage.value = clampPage(1, Math.ceil(totalRows.value / nextPageSize) || 1);
  emit("update:pageSize", nextPageSize);
  emit("update:page", currentPage.value);
  emit("page-change", { page: currentPage.value, pageSize: nextPageSize });
}
</script>

<style scoped>
/* Shell fills remaining page height when parent uses .grow / flex. */
.table-shell {
  display: flex;
  flex-direction: column;
  min-height: 0;
  flex: 1 1 auto;
  overflow: hidden;
  position: relative;
}

.table-query {
  flex: 0 0 auto;
  padding: var(--ued-space-panel-sm, 6px) var(--ued-space-panel, 8px);
  border-bottom: 1px solid var(--ued-table-border, var(--ued-color-divider));
  background: color-mix(in srgb, var(--ued-color-primary) 3%, var(--ued-color-bg-card));
}

/* Outer body frame: owns padding; does NOT scroll. */
.table-body {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--ued-table-body-bg, var(--ued-table-row-bg));
}

.table-body--padded {
  padding: var(--ued-space-4, 6px) var(--ued-space-5, 8px);
  box-sizing: border-box;
}

/* Inner pane: all scrollbars live here (inside the table frame). */
.table-scroll {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: auto;
  overscroll-behavior: contain;
  background: var(--ued-table-row-bg, var(--ued-color-bg-card));
  scrollbar-width: thin;
  scrollbar-color: color-mix(in srgb, var(--ued-color-primary) 22%, var(--ued-color-border)) transparent;
}

.table-body--padded .table-scroll {
  border: 1px solid var(--ued-table-border, var(--ued-color-border));
  border-radius: var(--ued-radius-md, 4px);
}

.table-scroll :deep(.admin-table) {
  width: 100%;
  flex: 0 0 auto;
}

.table-body--padded .table-scroll :deep(.admin-table thead th) {
  background: var(--ued-table-header-bg);
}

.table-scroll::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.table-scroll::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--ued-color-primary) 22%, var(--ued-color-border));
  border-radius: 999px;
  border: 2px solid transparent;
  background-clip: content-box;
}

.table-scroll--empty {
  overflow-x: auto;
}

.table-empty-state {
  display: grid;
  flex: 1 1 auto;
  width: 100%;
  min-height: 120px;
  place-items: center;
  place-content: center;
  padding: var(--ued-space-8, 12px) var(--ued-space-5, 8px);
  background: var(--ued-table-body-bg, var(--ued-table-row-bg));
  color: var(--ued-color-text-muted);
  font-size: var(--ued-font-size-sm);
  box-sizing: border-box;
}

.table-loading {
  position: absolute;
  inset: 0;
  z-index: 5;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--ued-space-3, 4px);
  background: color-mix(in srgb, var(--ued-color-bg-card) 72%, transparent);
  backdrop-filter: blur(1px);
  color: var(--ued-color-primary);
  font-size: var(--ued-font-size-sm);
}

.table-loading__spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 999px;
  animation: table-loading-spin 0.75s linear infinite;
}

@keyframes table-loading-spin {
  to {
    transform: rotate(360deg);
  }
}

.table-pagination {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: var(--ued-space-control, 6px);
  padding: var(--ued-space-panel-sm, 6px) var(--ued-space-panel, 8px);
  border-top: 1px solid var(--ued-table-border, var(--ued-color-divider));
  background: color-mix(in srgb, var(--ued-color-primary) 3%, var(--ued-color-bg-card));
}

.table-pagination__summary {
  min-width: 0;
  color: var(--ued-color-text-muted);
  font-size: var(--ued-font-size-sm);
  line-height: 16px;
  white-space: nowrap;
}

.table-pagination__controls {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--ued-space-3, 4px);
  flex-wrap: wrap;
  margin-left: auto;
}

.table-pagination__size {
  display: inline-flex;
  align-items: center;
  gap: var(--ued-space-2, 3px);
  color: var(--ued-color-text-muted);
  font-size: var(--ued-font-size-sm);
}

.table-pagination__size > span {
  width: 35px;
}

.table-pagination__ued-select :deep(.ued-select__control) {
  min-width: 70px;
  min-height: 22px;
  padding-top: 0;
  padding-bottom: 0;
  font-size: var(--ued-font-size-xs);
}

.table-pagination__pages {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-wrap: wrap;
}

.table-pagination__button {
  min-width: 24px;
  min-height: 22px;
  padding: 0 5px;
  box-shadow: none;
}

.table-pagination__button--active {
  border-color: var(--ued-color-primary);
  background: var(--ued-color-primary);
  color: var(--ued-color-primary-foreground, #fff);
}

.table-pagination__button--active:hover:not(:disabled) {
  border-color: var(--ued-color-primary-hover);
  background: var(--ued-color-primary-hover);
  color: var(--ued-color-primary-foreground, #fff);
}

.table-pagination__ellipsis {
  min-width: 14px;
  color: var(--ued-color-text-muted);
  font-size: var(--ued-font-size-sm);
  text-align: center;
}

.table-pagination__left {
  display: flex;
  align-items: center;
  gap: var(--ued-space-3, 4px);
  min-width: 0;
}

.table-pagination__selected {
  display: inline-flex;
  align-items: center;
  height: 18px;
  padding: 0 6px;
  border: 1px solid color-mix(in srgb, var(--ued-color-primary) 30%, transparent);
  border-radius: var(--ued-radius-pill, 999px);
  background: var(--ued-color-primary-soft);
  color: var(--ued-color-primary);
  font-size: var(--ued-font-size-xs);
  font-weight: 600;
  white-space: nowrap;
}

.table-selection__checkbox {
  width: 14px;
  height: 14px;
  margin: 0;
  cursor: pointer;
  accent-color: var(--ued-color-primary);
  vertical-align: middle;
}

.table-actions {
  display: inline-flex;
  flex-wrap: nowrap;
  align-items: center;
  justify-content: center;
  gap: 4px;
  max-width: 100%;
  min-width: 0;
}

/* Local table surface — colors come from theme tokens in main.css. */
:deep(.admin-table) {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  background: var(--ued-table-body-bg, var(--ued-table-row-bg, var(--ued-color-bg-card)));
  color: var(--ued-color-text-secondary);
}

:deep(.admin-table tbody) {
  background: var(--ued-table-body-bg, var(--ued-table-row-bg));
}

:deep(.admin-table th),
:deep(.admin-table td) {
  vertical-align: middle;
}

:deep(.admin-table th) {
  position: relative;
  font-weight: 600;
  color: var(--ued-table-header-fg, var(--ued-color-text-muted));
  background: var(--ued-table-header-bg, var(--ued-color-muted));
  border-bottom: 1px solid var(--ued-table-border, var(--ued-color-border));
  user-select: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

:deep(.admin-table td) {
  background: var(--ued-table-row-bg, var(--ued-color-bg-card));
  border-bottom: 1px solid var(--ued-table-border-row, var(--ued-color-divider));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Action / selection cells must not force max-width:0 ellipsis collapse. */
:deep(.admin-table td.actions-cell),
:deep(.admin-table td.selection-cell),
:deep(.admin-table th.actions-header),
:deep(.admin-table th.selection-header) {
  max-width: none;
  overflow: visible;
  text-overflow: clip;
  white-space: nowrap;
}

:deep(.admin-table td.actions-cell) {
  text-align: center;
}

/* Keep bottom hairline on last row so the table body is visually closed. */
:deep(.admin-table tbody tr:last-child td) {
  border-bottom: 1px solid var(--ued-table-border-row, var(--ued-color-divider));
}

:deep(.admin-table tbody tr:hover td) {
  background: var(--ued-table-row-hover, var(--ued-color-primary-soft));
}

:deep(.admin-table tbody tr.is-selected td) {
  background: var(--ued-table-row-selected, var(--ued-color-primary-soft));
}

:deep(.admin-table tbody tr.is-selected:hover td) {
  background: var(--ued-table-row-selected-hover);
}

:deep(.admin-table--stripe tbody tr:nth-child(even) td) {
  background: var(--ued-table-row-stripe);
}

:deep(.admin-table--stripe tbody tr:nth-child(even):hover td) {
  background: var(--ued-table-row-hover);
}

:deep(.admin-table--stripe tbody tr.is-selected td),
:deep(.admin-table--stripe tbody tr.is-selected:nth-child(even) td) {
  background: var(--ued-table-row-selected);
}

:deep(.admin-table--stripe tbody tr.is-selected:hover td) {
  background: var(--ued-table-row-selected-hover);
}

:deep(.admin-table tbody tr.is-clickable) {
  cursor: pointer;
}

:deep(.admin-table tbody tr.is-clickable:focus-visible) {
  outline: 2px solid var(--ued-color-focus-ring);
  outline-offset: -2px;
}

:deep(.admin-table--sticky-header thead) {
  position: sticky;
  top: 0;
  z-index: 20;
}

:deep(.admin-table th.is-sticky),
:deep(.admin-table td.is-sticky) {
  position: sticky;
  z-index: 10;
  background-clip: padding-box;
}

:deep(.admin-table thead th.is-sticky) {
  z-index: 30;
  background: var(--ued-table-header-bg);
}

:deep(.admin-table tbody td.is-sticky) {
  background: var(--ued-table-row-bg);
}

:deep(.admin-table tbody tr:hover td.is-sticky) {
  background: var(--ued-table-row-hover);
}

:deep(.admin-table tbody tr.is-selected td.is-sticky) {
  background: var(--ued-table-row-selected);
}

:deep(.admin-table--stripe tbody tr:nth-child(even) td.is-sticky) {
  background: var(--ued-table-row-stripe);
}

:deep(.admin-table .is-sticky-left-last) {
  box-shadow: 8px 0 10px -10px var(--ued-table-sticky-shadow);
}

:deep(.admin-table .is-sticky-right-first) {
  box-shadow: -8px 0 10px -10px var(--ued-table-sticky-shadow);
}

:deep(.admin-table .is-align-center) {
  text-align: center;
}

:deep(.admin-table .is-align-right) {
  text-align: right;
}

:deep(.admin-table .is-align-left) {
  text-align: left;
}

:deep(.table-cell-content) {
  display: block;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.table-cell-ellipsis) {
  display: block;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Keep custom slot content single-line so multi-block markup cannot grow rows. */
:deep(.admin-table td:not(.actions-cell):not(.selection-cell) .table-cell-content > *) {
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.admin-table td:not(.actions-cell):not(.selection-cell) .table-cell-content p),
:deep(
    .admin-table
      td:not(.actions-cell):not(.selection-cell)
      .table-cell-content
      > div:not(.table-cell-inline):not(.table-actions)
  ) {
  display: block;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Inline flex rows (tags + text) stay one line and clip. */
:deep(.admin-table td:not(.actions-cell) .table-cell-inline) {
  display: inline-flex !important;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  overflow: hidden;
  white-space: nowrap;
  vertical-align: middle;
}

/* Density — prefer CSS tokens so global density mode stays consistent */
:deep(.admin-table--xs th) {
  height: calc(var(--ued-table-header-height, 28px) - 2px);
  padding: 0 var(--ued-space-3, 4px);
  font-size: var(--ued-font-size-xs);
}
:deep(.admin-table--xs td) {
  height: calc(var(--ued-table-row-height, 30px) - 2px);
  padding: 0 var(--ued-space-3, 4px);
  font-size: var(--ued-font-size-sm);
}
:deep(.admin-table--sm th) {
  height: var(--ued-table-header-height, 28px);
  padding: 0 var(--ued-space-table-cell-x, 8px);
  font-size: var(--ued-font-size-xs);
}
:deep(.admin-table--sm td) {
  height: var(--ued-table-row-height, 30px);
  padding: 0 var(--ued-space-table-cell-x, 8px);
  font-size: var(--ued-font-size-sm);
}
:deep(.admin-table--md th) {
  height: calc(var(--ued-table-header-height, 28px) + 2px);
  padding: 0 var(--ued-space-table-cell-x, 8px);
  font-size: var(--ued-font-size-sm);
}
:deep(.admin-table--md td) {
  height: calc(var(--ued-table-row-height, 30px) + 2px);
  padding: 0 var(--ued-space-table-cell-x, 8px);
  font-size: var(--ued-font-size-sm);
}
:deep(.admin-table--lg th) {
  height: calc(var(--ued-table-header-height, 28px) + 4px);
  padding: 0 var(--ued-space-5, 8px);
  font-size: var(--ued-font-size-sm);
}
:deep(.admin-table--lg td) {
  height: calc(var(--ued-table-row-height, 30px) + 6px);
  padding: 0 var(--ued-space-5, 8px);
  font-size: var(--ued-font-size-base);
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

  .table-pagination__summary {
    white-space: normal;
  }
}
</style>
