<template>
  <section class="page-section">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton
          variant="secondary"
          :loading="store.loading"
          :disabled="store.loading || store.saving"
          @click="store.load"
        >
          {{ store.loading ? "刷新中..." : "重新读取" }}
        </UButton>
        <UButton
          variant="primary"
          :loading="store.saving"
          :disabled="store.loading || store.saving"
          @click="submit"
        >
          {{ store.saving ? "保存中..." : "保存并重载" }}
        </UButton>
      </div>
    </Teleport>

    <PanelBlock title="外观设置">
      <div class="divide-y divide-[var(--ued-color-divider)]">
        <div class="py-3">
          <p class="text-sm font-medium text-strong">主题颜色</p>
          <p class="mt-0.5 text-[11px] text-muted">选择界面主色调，即时生效。</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <button
              v-for="opt in uiPrefs.themeOptions"
              :key="opt.value"
              class="theme-swatch"
              :class="{ 'theme-swatch--active': uiPrefs.theme === opt.value }"
              :title="opt.label"
              @click="uiPrefs.setTheme(opt.value)"
            >
              <span class="theme-swatch__dot" :style="{ background: opt.color }" />
              <span class="theme-swatch__label">{{ opt.label }}</span>
            </button>
          </div>
        </div>

        <div class="py-3">
          <p class="text-sm font-medium text-strong">按钮尺寸</p>
          <p class="mt-0.5 text-[11px] text-muted">调整全局按钮和输入控件的大小。</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <UButton
              v-for="opt in uiPrefs.buttonSizeOptions"
              :key="opt.value"
              :variant="uiPrefs.buttonSize === opt.value ? 'primary' : 'secondary'"
              @click="uiPrefs.setButtonSize(opt.value)"
            >
              {{ opt.label }}
            </UButton>
          </div>
          <div class="mt-3">
            <UButton size="sm">示例按钮 SM</UButton>
            <span class="mx-1" />
            <UButton size="md">示例按钮 MD</UButton>
            <span class="mx-1" />
            <UButton size="lg">示例按钮 LG</UButton>
          </div>
        </div>
      </div>
    </PanelBlock>

    <div v-if="store.loading" class="empty-state">
      正在加载项目设置...
    </div>

    <template v-else>
      <form class="section-grid" @submit.prevent="submit">
        <PanelBlock title="核心运行">
          <div class="grid gap-3 md:grid-cols-2">
            <UInput v-model="store.form.proxy_host" label="PROXY_HOST" placeholder="127.0.0.1" />
            <UInput v-model="store.form.proxy_port" label="PROXY_PORT" type="number" />
            <UInput v-model="store.form.proxy_read_timeout_seconds" label="PROXY_READ_TIMEOUT_SECONDS" type="number" />
            <UInput v-model="store.form.proxy_write_timeout_seconds" label="PROXY_WRITE_TIMEOUT_SECONDS" type="number" />
            <UInput v-model="store.form.proxy_shutdown_timeout_seconds" label="PROXY_SHUTDOWN_TIMEOUT_SECONDS" type="number" />
            <UInput v-model="store.form.default_max_tokens" label="PROXY_DEFAULT_MAX_TOKENS" type="number" />
          </div>
        </PanelBlock>

        <PanelBlock title="日志参数">
          <div class="grid gap-3 md:grid-cols-2">
            <UInput v-model="store.form.proxy_chain_log_path" label="PROXY_CHAIN_LOG_PATH" placeholder=".data/bridge-chain.log" />
            <UInput v-model="store.form.proxy_chain_log_max_body_bytes" label="PROXY_CHAIN_LOG_MAX_BODY_BYTES" type="number" />
          </div>
          <div class="mt-3">
            <USwitch v-model="store.form.proxy_chain_log_bodies" label="记录请求与响应体" />
          </div>
        </PanelBlock>
      </form>
    </template>
  </section>
</template>

<script setup>
import { onMounted } from "vue";
import PanelBlock from "../components/PanelBlock.vue";
import StatCard from "../components/StatCard.vue";
import UButton from "../components/ued/UButton.vue";
import UInput from "../components/ued/UInput.vue";
import USwitch from "../components/ued/USwitch.vue";
import { message } from "../components/ued/message";
import { useSettingsStore } from "../stores/settings";
import { useUiPrefsStore } from "../stores/uiPrefs";
import { useStoreError } from "../composables/useStoreError";

const store = useSettingsStore();
useStoreError(store);
const uiPrefs = useUiPrefsStore();

async function submit() {
  await store.save();
  if (!store.error) {
    message.success("项目设置已保存并重载。");
  }
}

onMounted(() => {
  store.load();
});
</script>
