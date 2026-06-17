import { readonly, ref } from "vue";
import { normalizeVariant, MESSAGE_TYPES } from "./variant";

const items = ref([]);
let seed = 0;

function normalizeOptions(type, content, options = {}) {
  if (typeof content === "object" && content !== null) {
    return {
      type,
      content: content.content || "",
      duration: content.duration,
      key: content.key,
      closable: content.closable,
    };
  }

  return {
    type,
    content,
    duration: options.duration,
    key: options.key,
    closable: options.closable,
  };
}

export const messageItems = readonly(items);

export function closeMessage(idOrKey) {
  items.value = items.value.filter((item) => item.id !== idOrKey && item.key !== idOrKey);
}

export function openMessage(options = {}) {
  const type = normalizeType(options.type);
  const key = options.key;
  const id = key || `ued-message-${Date.now()}-${seed++}`;
  const duration = options.duration ?? (type === "loading" ? 0 : 3000);
  const item = {
    id,
    key,
    type,
    content: options.content || "",
    closable: Boolean(options.closable),
  };

  const currentIndex = key ? items.value.findIndex((message) => message.key === key) : -1;
  if (currentIndex >= 0) {
    items.value.splice(currentIndex, 1, item);
  } else {
    items.value.push(item);
  }

  if (duration > 0) {
    window.setTimeout(() => closeMessage(id), duration);
  }

  return () => closeMessage(id);
}

export const message = {
  open: openMessage,
  success(content, options) {
    return openMessage(normalizeOptions("success", content, options));
  },
  info(content, options) {
    return openMessage(normalizeOptions("info", content, options));
  },
  warning(content, options) {
    return openMessage(normalizeOptions("warning", content, options));
  },
  error(content, options) {
    return openMessage(normalizeOptions("error", content, options));
  },
  loading(content, options) {
    return openMessage(normalizeOptions("loading", content, options));
  },
  /**
   * Tie a message lifecycle to a promise.
   * Shows `loading` while pending, then `success`/`error` on settle.
   * Returns the original promise so it can be awaited inline.
   *
   * @example
   *   await message.promise(saveSupplier(), {
   *     loading: "保存中…",
   *     success: "Provider 已更新。",
   *     error: (err) => `保存失败：${err.message}`,
   *   });
   */
  promise(promiseLike, { key, loading = "处理中…", success = "操作成功。", error = "操作失败。", duration } = {}) {
    const messageKey = key || `ued-message-promise-${Date.now()}-${seed++}`;
    openMessage({ key: messageKey, type: "loading", content: loading, duration: 0 });
    return Promise.resolve(promiseLike).then(
      (result) => {
        const text = typeof success === "function" ? success(result) : success;
        openMessage({ key: messageKey, type: "success", content: text, duration });
        return result;
      },
      (err) => {
        const text = typeof error === "function" ? error(err) : error;
        openMessage({ key: messageKey, type: "error", content: text, duration });
        throw err;
      },
    );
  },
  destroy(key) {
    if (key) {
      closeMessage(key);
      return;
    }
    items.value = [];
  },
};

export function useMessage() {
  return message;
}

function normalizeType(type) {
  return normalizeVariant(type, "info", MESSAGE_TYPES);
}
