<template>
    <section class="settings-section">
        <div class="settings-section-heading">
            <div>
                <div class="settings-kicker">About</div>
                <h2 class="settings-section-title">关于</h2>
                <p class="settings-section-description">
                    当前应用版本、定位与基础识别信息。
                </p>
            </div>
        </div>

        <div class="settings-card">
            <div class="flex items-start gap-3">
                <div class="about-mark">
                    <SparklesIcon :size="18" class="text-accent" />
                </div>
                <div class="min-w-0 flex-1">
                    <div class="flex items-center gap-2 flex-wrap">
                        <h3 class="text-sm font-semibold text-foreground">icooclaw</h3>
                        <span class="info-chip">v1.0.0</span>
                    </div>
                    <p class="settings-card-description">
                        AI 助手平台。当前关于页保持简洁，只展示必要信息，不额外堆叠装饰。
                    </p>
                </div>
            </div>
        </div>
    </section>
</template>

<script setup>
import { Sparkles as SparklesIcon } from "lucide-vue-next";
</script>

<style scoped>
.about-mark {
    width: 36px;
    height: 36px;
    border-radius: var(--radius-md);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: color-mix(in srgb, var(--color-accent) 10%, white);
    border: 1px solid color-mix(in srgb, var(--color-accent) 16%, transparent);
    flex-shrink: 0;
}
</style>
