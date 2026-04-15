<template>
    <div class="page-shell settings-page">
        <div class="page-frame">
            <aside class="settings-sidebar surface-muted page-panel">
                <div class="settings-sidebar-top">
                    <div class="settings-sidebar-header">
                        <div>
                            <div class="settings-kicker">System</div>
                            <div class="section-title">设置中心</div>
                            <p class="settings-sidebar-description">管理外观、基础信息与桌面使用体验。</p>
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
                    <div class="settings-hero surface-muted">
                        <div>
                            <div class="settings-kicker">Workspace</div>
                            <div class="section-title">{{ currentMenuLabel }}</div>
                            <p class="settings-section-description">按照统一规范维护桌面工具的外观和基础信息。</p>
                        </div>
                        <div class="settings-hero-badges">
                            <span class="info-chip">
                                <PaletteIcon :size="12" />
                                视觉主题
                            </span>
                        </div>
                    </div>

                    <AppearanceSettings v-if="activeSection === 'appearance'" />
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
} from "lucide-vue-next";

import AppearanceSettings from "@/components/settings/AppearanceSettings.vue";
import AboutSettings from "@/components/settings/AboutSettings.vue";

const router = useRouter();

const menuItems = [
    { key: "appearance", label: "外观", icon: PaletteIcon },
    { key: "about", label: "关于", icon: InfoIcon },
];

const activeSection = ref("appearance");
const currentMenuLabel = computed(
    () => menuItems.find((item) => item.key === activeSection.value)?.label || "设置",
);
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
