/** Pure helpers for UTable (no Vue deps). */

export function normalizeCssSize(value) {
  if (value === 0) return "0px";
  if (!value) return "";
  return typeof value === "number" ? `${value}px` : String(value);
}

export function normalizePositiveInteger(value, fallback) {
  const numeric = Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) {
    return fallback;
  }
  return Math.floor(numeric);
}

export function clampPage(page, maxPage) {
  return Math.min(Math.max(page, 1), Math.max(maxPage, 1));
}

export function normalizeAlign(value) {
  if (value === "center" || value === "right") return value;
  return "left";
}

export function normalizeFixed(value) {
  if (value === "left" || value === true) return "left";
  if (value === "right") return "right";
  return "";
}

export function normalizeSize(value) {
  const aliases = { small: "sm", middle: "md", large: "lg" };
  const resolved = aliases[value] || value;
  return ["xs", "sm", "md", "lg"].includes(resolved) ? resolved : "md";
}

export function parsePixelSize(value) {
  if (!value) return 0;
  const match = String(value).match(/^(\d+(?:\.\d+)?)px$/);
  return match ? Number(match[1]) : 0;
}

export function appendCssSize(base, size) {
  if (!size) return base;
  if (base === "0px") return size;
  return `calc(${base} + ${size})`;
}

export function formatCellValue(value) {
  if (value === null || value === undefined || value === "") {
    return "-";
  }
  return String(value);
}

export function resolveColumnOption(column, customField, fallbackField) {
  if (customField && Object.prototype.hasOwnProperty.call(column, customField)) {
    return column[customField];
  }
  return column?.[fallbackField];
}

export function buildPageItems(current, totalPages) {
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

export function estimateColumnWidth(column) {
  const minWidth = parsePixelSize(column.minWidth);
  if (minWidth) return minWidth;
  const width = parsePixelSize(column.width);
  if (width) return width;
  if (column.isAction) return 180;
  if (column.isSelection) return 44;
  if (column.ellipsis) return 220;
  return 160;
}

export function withStickyOffsets(columns) {
  const next = columns.map((column) => ({ ...column, stickyStyle: {} }));
  let leftOffset = "0px";
  let lastLeftIndex = -1;
  let firstRightIndex = -1;

  next.forEach((column, index) => {
    if (column.fixed === "left") lastLeftIndex = index;
    if (column.fixed === "right" && firstRightIndex === -1) firstRightIndex = index;
  });

  for (const column of next) {
    if (column.fixed !== "left") continue;
    column.stickyStyle.left = leftOffset;
    leftOffset = appendCssSize(leftOffset, column.width || column.minWidth || "0px");
  }

  let rightOffset = "0px";
  for (let index = next.length - 1; index >= 0; index -= 1) {
    const column = next[index];
    if (column.fixed !== "right") continue;
    column.stickyStyle.right = rightOffset;
    rightOffset = appendCssSize(rightOffset, column.width || column.minWidth || "0px");
  }

  if (lastLeftIndex >= 0) next[lastLeftIndex].isStickyLeftLast = true;
  if (firstRightIndex >= 0) next[firstRightIndex].isStickyRightFirst = true;
  return next;
}
