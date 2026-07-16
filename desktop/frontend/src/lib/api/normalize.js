export function valueOf(raw, snake, pascal, fallback = "") {
  return (
    raw?.[snake] ?? raw?.[pascal] ?? raw?.[snakeToPascal(snake)] ?? fallback
  );
}

export function snakeToPascal(value) {
  return String(value || "")
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");
}

export function boolOf(raw, snake, pascal, fallback = false) {
  return Boolean(valueOf(raw, snake, pascal, fallback));
}

export function normalizePage(raw, fallbackPage = 1, fallbackPageSize = 20) {
  if (Array.isArray(raw))
    return {
      items: raw,
      total: raw.length,
      page: fallbackPage,
      page_size: fallbackPageSize,
    };
  return {
    items: raw?.items || raw?.Items || [],
    total: Number(raw?.total ?? raw?.Total ?? 0),
    page: Number(raw?.page ?? raw?.Page ?? fallbackPage),
    page_size: Number(raw?.page_size ?? raw?.PageSize ?? fallbackPageSize),
  };
}

export function pageItems(items, page = 1, pageSize = 20) {
  const safePage = Math.max(1, Number(page || 1));
  const safePageSize = Math.max(1, Number(pageSize || 20));
  return {
    items: items.slice((safePage - 1) * safePageSize, safePage * safePageSize),
    total: items.length,
    page: safePage,
    page_size: safePageSize,
  };
}

export function matchesKeyword(item, keyword, fields) {
  const text = String(keyword || "")
    .trim()
    .toLowerCase();
  return (
    !text ||
    fields.some((field) =>
      String(item?.[field] || "")
        .toLowerCase()
        .includes(text),
    )
  );
}

export function maskSecret(secret) {
  const value = String(secret || "").trim();
  if (!value) return "";
  if (value.includes("...")) return value;
  if (value.length <= 8) return "****";
  return `${value.slice(0, 4)}...${value.slice(-4)}`;
}
