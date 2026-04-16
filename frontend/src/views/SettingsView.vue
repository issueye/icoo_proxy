<template>
    <div class="page-shell settings-page">
        <div class="page-frame">
            <aside class="settings-sidebar surface-muted page-panel">
                <div class="settings-sidebar-top">
                    <div class="settings-sidebar-header">
                        <div>
                            <div class="section-title">设置中心</div>
                        </div>
                        <button @click="router.back()" class="btn btn-secondary btn-icon">
                            <ArrowLeftIcon :size="18" />
                        </button>
                    </div>
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
                        <span class="settings-nav-label">{{ item.label }}</span>
                    </button>
                </nav>

                <div class="settings-sidebar-bottom">
                    <div class="info-chip settings-sidebar-chip">
                        当前分区 · {{ currentMenuLabel }}
                    </div>
                </div>
            </aside>

            <main class="settings-main surface-panel page-panel">
                <div class="settings-main-inner">
                    <GatewaySettings v-if="activeSection === 'gateway'" />
                    <AppearanceSettings v-else-if="activeSection === 'appearance'" />
                    <AboutSettings v-else-if="activeSection === 'about'" />
                </div>
            </main>
        </div>
    </div>
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

const router = useRouter();

const menuItems = [
    { key: "gateway", label: "网关", icon: NetworkIcon, badge: "监听地址" },
    { key: "appearance", label: "外观", icon: PaletteIcon, badge: "视觉主题" },
    { key: "about", label: "关于", icon: InfoIcon, badge: "版本信息" },
];

const activeSection = ref("gateway");
const currentMenu = computed(
    () => menuItems.find((item) => item.key === activeSection.value) || menuItems[0],
);
const currentMenuLabel = computed(() => currentMenu.value.label || "设置");
const currentMenuIcon = computed(() => currentMenu.value.icon || NetworkIcon);
const currentMenuBadge = computed(() => currentMenu.value.badge || "设置项");
</script>

<style scoped>
.settings-sidebar {
    width: 264px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    border: 1px solid var(--ui-border-default);
    border-radius: var(--radius-lg);
    background: var(--ui-bg-surface);
    box-shadow: var(--shadow-rest);
}

.settings-sidebar-top,
.settings-sidebar-bottom {
    padding: 16px;
}

.settings-sidebar-top {
    border-bottom: 1px solid var(--ui-border-subtle);
}

.settings-sidebar-bottom {
    border-top: 1px solid var(--ui-border-subtle);
}

.settings-sidebar-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
}

.settings-sidebar-description {
    margin: 4px 0 0;
    font-size: 12px;
    line-height: 1.5;
    color: var(--color-text-muted);
}

.settings-nav {
    flex: 1;
    overflow-y: auto;
    padding: 12px 10px;
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.settings-nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    min-height: 36px;
    padding: 0 10px;
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    color: var(--color-text-secondary);
    text-align: left;
    transition: all 0.14s ease;
}

.settings-nav-item:hover {
    background: var(--ui-bg-surface-hover);
    border-color: var(--ui-border-subtle);
    color: var(--color-text-primary);
}

.settings-nav-item.is-active {
    background: var(--color-accent-soft);
    border-color: color-mix(in srgb, var(--color-accent) 24%, var(--ui-border-default));
    color: var(--color-accent);
}

.settings-nav-icon {
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--ui-border-default);
    border-radius: var(--radius-sm);
    background: var(--ui-bg-surface-muted);
}

.settings-nav-label {
    font-size: 13px;
    font-weight: 600;
}

.settings-sidebar-chip {
    width: 100%;
    justify-content: center;
}

.settings-main {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    border: 1px solid var(--ui-border-default);
    border-radius: var(--radius-lg);
    background: var(--ui-bg-surface);
    box-shadow: var(--shadow-rest);
}

.settings-main-inner {
    padding: 16px;
    height: 100%;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
}

.settings-hero {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 16px;
    padding: 14px 16px;
    border: 1px solid var(--ui-border-default);
    border-radius: var(--radius-md);
    background: var(--ui-bg-surface-muted);
}

.settings-page :deep(.page-panel) {
    border-radius: var(--radius-lg);
}

.settings-hero-badges {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 8px;
}

@media (max-width: 1024px) {
    .settings-sidebar {
        width: 236px;
    }
}

@media (max-width: 860px) {
    .page-frame {
        flex-direction: column;
    }

    .settings-sidebar {
        width: 100%;
    }

    .settings-hero {
        flex-direction: column;
    }

    .settings-hero-badges {
        justify-content: flex-start;
    }
}
</style>

