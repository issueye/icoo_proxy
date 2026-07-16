<template>
  <div class="plugin-ext">
    <div v-if="!embedUrl" class="plugin-ext__empty">
      <p>未找到插件扩展页。</p>
      <p class="plugin-ext__hint">请确认插件已启动，并在 handshake 中声明 ui_pages。</p>
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
  height: calc(100vh - 180px);
  min-height: 480px;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid var(--border-color, #2a3544);
  background: #0f1419;
}
.plugin-ext__frame {
  width: 100%;
  height: 100%;
  border: 0;
  display: block;
  background: #0f1419;
}
.plugin-ext__empty {
  padding: 32px;
  color: #9aa7b8;
}
.plugin-ext__hint {
  font-size: 13px;
  margin-top: 8px;
}
</style>
