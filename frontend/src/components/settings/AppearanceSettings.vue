<template>
    <section class="settings-section">
        <div class="settings-section-heading">
            <h2 class="settings-section-title">外观设置</h2>
            <div class="info-chip">当前主题 · {{ themeLabel }}</div>
        </div>

        <div class="settings-card">
            <div class="settings-card-head">
                <div class="settings-card-title">主题模式</div>
            </div>

            <div class="settings-segment">
                <button
                    @click="themeStore.setTheme('light')"
                    :class="['settings-segment-btn', { 'is-active': themeStore.theme === 'light' }]"
                >
                    <SunIcon :size="13" />
                    浅色
                </button>
                <button
                    @click="themeStore.setTheme('dark')"
                    :class="['settings-segment-btn', { 'is-active': themeStore.theme === 'dark' }]"
                >
                    <MoonIcon :size="13" />
                    深色
                </button>
            </div>
        </div>

        <div class="settings-card">
            <div class="settings-card-head">
                <div class="settings-card-title">强调色</div>
            </div>

            <div class="settings-swatch-grid">
                <button
                    v-for="color in colorList"
                    :key="color.key"
                    @click="themeStore.setColorTheme(color.key)"
                    :class="['settings-swatch-card', { 'is-active': themeStore.colorTheme === color.key }]"
                >
                    <div
                        class="color-theme-btn"
                        :class="{ active: themeStore.colorTheme === color.key }"
                        :style="{ backgroundColor: color.color }"
                    />
                    <span class="text-[11px] text-muted-foreground">{{ color.name }}</span>
                </button>
            </div>
        </div>

        <div class="settings-card settings-card--soft">
            <div class="settings-card-head">
                <div class="settings-card-title">控件预览</div>
            </div>

            <div class="settings-preview-grid">
                <div class="settings-preview-card">
                    <strong>按钮</strong>
                    <div class="flex flex-wrap gap-2">
                        <button class="btn btn-primary btn-sm">主要按钮</button>
                        <button class="btn btn-secondary btn-sm">次要按钮</button>
                    </div>
                </div>

                <div class="settings-preview-card">
                    <strong>标签</strong>
                    <div class="flex flex-wrap gap-2">
                        <span class="settings-preview-pill bg-accent/10 text-accent">
                            <CheckIcon :size="11" />
                            当前主题
                        </span>
                        <span class="settings-preview-pill bg-secondary text-muted-foreground">
                            默认标签
                        </span>
                    </div>
                </div>

                <div class="settings-preview-card">
                    <strong>进度</strong>
                    <div class="settings-preview-bar">
                        <span style="width: 62%"></span>
                    </div>
                    <div class="settings-help mt-2">视觉重心落在内容而不是装饰。</div>
                </div>
            </div>
        </div>
    </section>
</template>

<script setup>
import { computed } from "vue";
import { Moon as MoonIcon, Sun as SunIcon, Check as CheckIcon } from "lucide-vue-next";
import { useThemeStore } from "@/stores/theme";

const themeStore = useThemeStore();

const colorList = computed(() => themeStore.getColorThemeList());
const themeLabel = computed(() => (themeStore.theme === "light" ? "浅色" : "深色"));
</script>
