<template>
  <UEDPageShell class="settings-page" sidebar-width="280px" gap="16px">
    <template #sidebar>
      <div class="settings-sidebar-shell">
        <div class="settings-sidebar-top">
          <UEDPageHeader
            title="设置中心"
            compact
          >
            <template #actions>
              <button @click="router.back()" class="btn btn-secondary btn-icon" title="返回">
                <ArrowLeftIcon :size="16" />
              </button>
            </template>
          </UEDPageHeader>
        </div>

        <nav class="settings-nav">
          <button
            v-for="item in menuItems"
            :key="item.key"
            @click="activeSection = item.key"
            class="settings-nav-item"
            :class="{ 'is-active': activeSection === item.key }"
          >
            <div class="settings-nav-icon">
              <component :is="item.icon" :size="16" />
            </div>
            <div class="settings-nav-copy">
              <span class="settings-nav-label">{{ item.label }}</span>
              <span class="settings-nav-meta">{{ item.badge }}</span>
            </div>
          </button>
        </nav>

        <div class="settings-sidebar-bottom">
          <div class="info-chip settings-sidebar-chip">
            当前分区 · {{ currentMenuLabel }}
          </div>
        </div>
      </div>
    </template>

    <div class="app-page settings-main-page">
      <UEDPageHeader
        :title="currentMenu.label"
        :description="currentMenu.badge"
        :icon="currentMenu.icon"
        divided
      />

      <section class="settings-main-panel ued-panel ued-panel--raised">
        <div class="settings-main-inner">
          <GatewaySettings v-if="activeSection === 'gateway'" />
          <AppearanceSettings v-else-if="activeSection === 'appearance'" />
          <AboutSettings v-else-if="activeSection === 'about'" />
        </div>
      </section>
    </div>
  </UEDPageShell>
</template>

<script setup>
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import {
  ArrowLeft as ArrowLeftIcon,
  Palette as PaletteIcon,
  Info as InfoIcon,
  Network as NetworkIcon,
} from "lucide-vue-next";

import GatewaySettings from "@/components/settings/GatewaySettings.vue";
import AppearanceSettings from "@/components/settings/AppearanceSettings.vue";
import AboutSettings from "@/components/settings/AboutSettings.vue";
import { UEDPageHeader, UEDPageShell } from "@/components/layout";

const router = useRouter();

const menuItems = [
  { key: "gateway", label: "网关", icon: NetworkIcon },
  { key: "appearance", label: "外观", icon: PaletteIcon },
  { key: "about", label: "关于", icon: InfoIcon },
];

const activeSection = ref("gateway");
const currentMenu = computed(
  () => menuItems.find((item) => item.key === activeSection.value) || menuItems[0],
);
const currentMenuLabel = computed(() => currentMenu.value.label || "设置");
</script>

<style scoped>
.settings-page {
  height: 100%;
}

.settings-sidebar-shell {
  display: flex;
  flex-direction: column;
  min-height: 100%;
}

.settings-sidebar-top,
.settings-sidebar-bottom {
  padding: 8px;
}

.settings-sidebar-top {
  border-bottom: 1px solid var(--ued-border-subtle);
}

.settings-sidebar-bottom {
  margin-top: auto;
  border-top: 1px solid var(--ued-border-subtle);
}

.settings-nav {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 8px;
  min-height: 0;
  overflow-y: auto;
}

.settings-nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 52px;
  padding: 10px;
  border: 1px solid transparent;
  border-radius: var(--ued-radius-md);
  color: var(--ued-text-secondary);
  text-align: left;
  transition: all 0.14s ease;
}

.settings-nav-item:hover {
  background: var(--ued-bg-panel-hover);
  border-color: var(--ued-border-subtle);
  color: var(--ued-text-primary);
}

.settings-nav-item.is-active {
  background: linear-gradient(180deg, color-mix(in srgb, var(--ued-accent-soft) 88%, white), var(--ued-accent-soft));
  border-color: color-mix(in srgb, var(--ued-accent) 22%, var(--ued-border-default));
  color: var(--ued-accent);
  box-shadow: var(--ued-shadow-rest);
}

.settings-nav-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ued-border-default);
  border-radius: var(--ued-radius-sm);
  background: var(--ued-bg-panel-muted);
  flex-shrink: 0;
}

.settings-nav-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.settings-nav-label {
  font-size: 13px;
  font-weight: 600;
  color: currentColor;
}

.settings-nav-meta {
  font-size: 11px;
  line-height: 1.5;
  color: var(--ued-text-muted);
}

.settings-sidebar-chip {
  width: 100%;
  justify-content: center;
}

.settings-main-page {
  padding: 0;
  gap: 16px;
  background: transparent;
}

.settings-main-panel {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.settings-main-inner {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  padding: 16px;
}

@media (max-width: 860px) {
  .settings-page :deep(.ued-page-shell__sidebar) {
    min-height: auto;
  }

  .settings-nav-meta {
    display: none;
  }

  .settings-main-page {
    min-height: auto;
  }
}
</style>