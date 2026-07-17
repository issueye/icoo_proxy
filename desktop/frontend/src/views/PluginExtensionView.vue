<template>
  <div class="plugin-ext">
    <div v-if="!embedUrl" class="plugin-ext__empty">
      <p class="plugin-ext__title">未找到插件扩展页</p>
      <p class="plugin-ext__hint">
        请确认插件已启动，并在 handshake 中声明 <code>ui_pages</code>。可在「插件」页检查状态后重试。
      </p>
    </div>
    <iframe
      v-else
      class="plugin-ext__frame"
      :src="embedUrl"
      :title="title"
      referrerpolicy="no-referrer"
    />
  </div>
</template>

<script setup>
import { computed } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();

const title = computed(() => route.meta?.title || "插件页面");

const embedUrl = computed(() => {
  const base =
    (typeof window !== "undefined" && window.__ICOOSERVER_URL) ||
    "http://127.0.0.1:18181";
  const path = route.meta?.embedPath || route.query?.embed;
  if (!path) return "";
  // embedPath is like /api/v1/plugins/grokbuild/ui/
  const url = new URL(String(path), base.endsWith("/") ? base : base + "/");
  return url.toString();
});
</script>

<style scoped>
.plugin-ext {
  /* Fill the main content area under topbar; light shell matches desktop UED. */
  height: calc(100vh - 132px);
  min-height: 420px;
  border-radius: var(--ued-radius-lg, 6px);
  overflow: hidden;
  border: 1px solid var(--ued-color-border, #d4d7dc);
  background: var(--ued-color-bg-page, #f3f4f6);
  box-shadow: none;
}
.plugin-ext__frame {
  width: 100%;
  height: 100%;
  border: 0;
  display: block;
  background: var(--ued-color-bg-page, #f3f4f6);
}
.plugin-ext__empty {
  padding: var(--ued-space-12, 16px) var(--ued-space-8, 12px);
  color: var(--ued-color-text-muted, #6b7280);
}
.plugin-ext__title {
  margin: 0;
  font-size: var(--ued-font-size-base, 13px);
  font-weight: 600;
  color: var(--ued-color-text, #1f2329);
}
.plugin-ext__hint {
  font-size: var(--ued-font-size-sm, 12px);
  margin-top: var(--ued-space-3, 4px);
  line-height: 1.45;
  max-width: 36rem;
}
.plugin-ext__hint code {
  font-size: var(--ued-font-size-xs, 11px);
  padding: 1px 4px;
  border-radius: var(--ued-radius-sm, 3px);
  background: var(--ued-color-secondary, #eef0f3);
  border: 1px solid var(--ued-color-border-light, #e4e6ea);
}
</style>
