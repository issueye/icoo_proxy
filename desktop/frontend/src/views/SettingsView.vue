<template>
  <section class="page-section page-section--scroll">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton
          size="sm"
          variant="secondary"
          :loading="store.loading"
          :disabled="store.loading || store.saving"
          @click="store.load"
        >
          {{ store.loading ? "刷新中..." : "重新读取" }}
        </UButton>
        <UButton
          size="sm"
          variant="primary"
          :loading="store.saving"
          :disabled="store.loading || store.saving"
          @click="submit"
        >
          {{ store.saving ? "保存中..." : "保存并重载" }}
        </UButton>
      </div>
    </Teleport>

    <div class="settings-layout">
      <aside class="settings-nav" aria-label="设置分类">
        <button
          v-for="item in settingSections"
          :key="item.key"
          type="button"
          class="settings-nav-item"
          :class="{ 'settings-nav-item--active': activeSection === item.key }"
          @click="activeSection = item.key"
        >
          <span class="settings-nav-item__title">{{ item.title }}</span>
          <span class="settings-nav-item__desc">{{ item.description }}</span>
        </button>
      </aside>

      <div class="settings-detail">
        <PanelBlock
          v-if="activeSection === 'appearance'"
          title="外观设置"
          description="主题、界面密度与控件尺寸"
        >
          <div class="settings-field-group">
            <div class="settings-field-copy">
              <p class="settings-field-title">主题颜色</p>
              <p class="settings-field-desc">切换后标题栏和主操作颜色同步更新。</p>
            </div>
            <div class="settings-control-grid settings-control-grid--swatches">
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

          <div class="settings-field-group">
            <div class="settings-field-copy">
              <p class="settings-field-title">界面密度</p>
              <p class="settings-field-desc">
                宽松加大页面留白与表格行高；紧缩适合宽表与运维台。
              </p>
            </div>
            <div class="settings-control-stack">
              <div class="density-mode-grid">
                <button
                  v-for="opt in uiPrefs.densityOptions"
                  :key="opt.value"
                  type="button"
                  class="density-mode-card"
                  :class="{ 'density-mode-card--active': uiPrefs.density === opt.value }"
                  @click="uiPrefs.setDensity(opt.value)"
                >
                  <span class="density-mode-card__preview" :data-mode="opt.value" aria-hidden="true">
                    <i /><i /><i />
                  </span>
                  <span class="density-mode-card__title">{{ opt.label }}</span>
                  <span class="density-mode-card__desc">{{ opt.description }}</span>
                </button>
              </div>
              <p class="settings-field-desc">
                当前：{{ uiPrefs.density === "comfortable" ? "宽松" : "紧缩" }}
                · 页面 {{ densityPreview.page }} · 表行 {{ densityPreview.row }}
              </p>
            </div>
          </div>

          <div class="settings-field-group">
            <div class="settings-field-copy">
              <p class="settings-field-title">按钮尺寸</p>
              <p class="settings-field-desc">在密度模式之上微调按钮、输入框高度。</p>
            </div>
            <div class="settings-control-stack">
              <div class="settings-control-grid">
                <UButton
                  v-for="opt in uiPrefs.buttonSizeOptions"
                  :key="opt.value"
                  size="sm"
                  :variant="uiPrefs.buttonSize === opt.value ? 'primary' : 'secondary'"
                  @click="uiPrefs.setButtonSize(opt.value)"
                >
                  {{ opt.label }}
                </UButton>
              </div>
              <div class="settings-preview-row">
                <UButton size="xs">XS</UButton>
                <UButton size="sm">SM</UButton>
                <UButton size="md">MD</UButton>
                <UButton size="lg">LG</UButton>
              </div>
            </div>
          </div>
        </PanelBlock>

        <div v-else-if="store.loading" class="empty-state">
          正在加载项目设置...
        </div>

        <form v-else class="section-grid" @submit.prevent="submit">
          <PanelBlock
            v-if="activeSection === 'runtime'"
            title="核心运行"
            description="监听地址、超时和默认 Token"
          >
            <div class="settings-form-grid">
              <UInput v-model="store.form.proxy_host" label="代理主机" hint="PROXY_HOST" placeholder="127.0.0.1" />
              <UInput v-model="store.form.proxy_port" label="代理端口" hint="PROXY_PORT" type="number" />
              <UInput v-model="store.form.proxy_read_timeout_seconds" label="读取超时（秒）" hint="PROXY_READ_TIMEOUT_SECONDS" type="number" />
              <UInput v-model="store.form.proxy_write_timeout_seconds" label="写入超时（秒）" hint="PROXY_WRITE_TIMEOUT_SECONDS" type="number" />
              <UInput v-model="store.form.proxy_shutdown_timeout_seconds" label="关闭等待时间（秒）" hint="PROXY_SHUTDOWN_TIMEOUT_SECONDS" type="number" />
              <UInput v-model="store.form.default_max_tokens" label="默认最大 Token 数" hint="PROXY_DEFAULT_MAX_TOKENS" type="number" />
            </div>
          </PanelBlock>

          <PanelBlock
            v-if="activeSection === 'logs'"
            title="日志参数"
            description="链路日志位置和内容记录策略"
          >
            <div class="settings-form-grid">
              <UInput v-model="store.form.proxy_chain_log_path" label="日志路径" hint="PROXY_CHAIN_LOG_PATH" placeholder=".data/bridge-chain.log" />
              <UInput v-model="store.form.proxy_chain_log_max_body_bytes" label="Body 最大记录字节数" hint="PROXY_CHAIN_LOG_MAX_BODY_BYTES" type="number" />
            </div>
            <div class="settings-switch-row">
              <USwitch v-model="store.form.proxy_chain_log_bodies" label="记录请求与响应体" hint="可能包含敏感内容，并增加日志体积。" />
            </div>
          </PanelBlock>
        </form>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from "vue";
import PanelBlock from "../components/PanelBlock.vue";
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
const activeSection = ref("appearance");

const densityPreview = computed(() => {
  if (uiPrefs.density === "comfortable") {
    return { page: "12px", row: "36px" };
  }
  return { page: "8px", row: "30px" };
});

const settingSections = [
  {
    key: "appearance",
    title: "外观",
    description: "主题、密度、控件尺寸",
  },
  {
    key: "runtime",
    title: "核心运行",
    description: "监听、超时、Token",
  },
  {
    key: "logs",
    title: "日志参数",
    description: "链路日志和记录范围",
  },
];

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
