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
          description="主题与控件尺寸"
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
              <p class="settings-field-title">按钮尺寸</p>
              <p class="settings-field-desc">控制按钮、输入框和选择器的整体高度。</p>
            </div>
            <div class="settings-control-stack">
              <div class="settings-control-grid">
                <UButton
                  v-for="opt in uiPrefs.buttonSizeOptions"
                  :key="opt.value"
                  :variant="uiPrefs.buttonSize === opt.value ? 'primary' : 'secondary'"
                  @click="uiPrefs.setButtonSize(opt.value)"
                >
                  {{ opt.label }}
                </UButton>
              </div>
              <div class="settings-preview-row">
                <UButton size="sm">示例按钮 SM</UButton>
                <UButton size="md">示例按钮 MD</UButton>
                <UButton size="lg">示例按钮 LG</UButton>
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
import { onMounted, ref } from "vue";
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

const settingSections = [
  {
    key: "appearance",
    title: "外观",
    description: "主题颜色、控件尺寸",
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
