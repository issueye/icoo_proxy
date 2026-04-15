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
                    <SparklesIcon :size="18" class="about-mark-icon" />
                </div>
                <div class="min-w-0 flex-1">
                    <div class="flex items-center gap-2 flex-wrap">
                        <h3 class="text-sm font-semibold text-foreground">icoo_proxy</h3>
                        <span class="info-chip">v1.0.0</span>
                    </div>
                    <p class="settings-card-description">
                        本地 AI 网关管理工具。关于页只保留必要识别信息，延续桌面工具的克制表达。
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
    border-radius: var(--radius-sm);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: var(--color-accent-soft);
    border: 1px solid color-mix(in srgb, var(--color-accent) 16%, var(--ui-border-default));
    flex-shrink: 0;
}

.about-mark-icon {
    color: var(--color-accent);
}
</style>
