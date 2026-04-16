<template>
    <section class="settings-section gateway-settings">
        <div class="settings-section-heading">
            <h2 class="settings-section-title">网关设置</h2>
            <div class="info-chip">当前监听 · {{ gatewayAddress }}</div>
        </div>

        <div class="settings-card">
            <div class="settings-card-head">
                <div class="settings-card-title">监听地址</div>
            </div>

            <div class="gateway-settings-grid">
                <label class="settings-field">
                    <span class="settings-field-label">监听 IP / Host</span>
                    <input
                        v-model.trim="form.listenHost"
                        class="input"
                        type="text"
                        placeholder="127.0.0.1 / 0.0.0.0 / localhost"
                    >
                    <span class="settings-help">支持 `127.0.0.1`、`0.0.0.0`、`localhost` 或局域网 IP。</span>
                </label>

                <label class="settings-field">
                    <span class="settings-field-label">监听端口</span>
                    <input
                        v-model.trim="form.listenPort"
                        class="input"
                        type="number"
                        min="1"
                        max="65535"
                        placeholder="16790"
                    >
                    <span class="settings-help">有效范围 1 - 65535。</span>
                </label>
            </div>

            <div class="settings-actions">
                <button class="btn btn-secondary" type="button" @click="resetForm" :disabled="gatewayStore.loading">
                    重置
                </button>
                <button class="btn btn-primary" type="button" @click="handleSave" :disabled="gatewayStore.loading">
                    保存设置
                </button>
            </div>
        </div>

        <div class="settings-card settings-card--soft">
            <div class="settings-card-head">
                <div class="settings-card-title">生效说明</div>
            </div>

            <div class="gateway-settings-notes">
                <div class="gateway-settings-note">
                    <strong>当前地址</strong>
                    <code>{{ gatewayAddress }}</code>
                </div>
                <div class="gateway-settings-note">
                    <strong>访问入口</strong>
                    <code>http://{{ gatewayAddress }}/v1</code>
                </div>
                <div class="settings-help">
                    保存后会写入网关配置；如果网关已在运行，建议停止后重新启动，以确保监听地址切换到新配置。
                </div>
            </div>
        </div>
    </section>
</template>

<script setup>
import { computed, onMounted, reactive } from "vue";
import { useGatewayStore } from "@/stores/gateway";
import { useToast } from "@/composables/useToast";

const gatewayStore = useGatewayStore();
const { toast } = useToast();

const form = reactive({
    listenHost: "127.0.0.1",
    listenPort: "16790",
});

const gatewayAddress = computed(() => `${form.listenHost || "127.0.0.1"}:${form.listenPort || "16790"}`);

function syncForm() {
    form.listenHost = gatewayStore.gatewayConfig.listenHost || gatewayStore.host || "127.0.0.1";
    form.listenPort = String(gatewayStore.gatewayConfig.listenPort || gatewayStore.port || 16790);
}

function resetForm() {
    syncForm();
}

function isValidHost(value) {
    const host = value.trim();
    if (!host) return false;
    if (host === "localhost") return true;
    if (/^(\d{1,3}\.){3}\d{1,3}$/.test(host)) {
        return host.split(".").every((part) => {
            const n = Number(part);
            return Number.isInteger(n) && n >= 0 && n <= 255;
        });
    }
    return false;
}

async function handleSave() {
    const listenHost = form.listenHost.trim();
    const listenPort = Number(form.listenPort);

    if (!isValidHost(listenHost)) {
        toast("请输入有效的监听 IP 或 localhost", "error");
        return;
    }

    if (!Number.isInteger(listenPort) || listenPort < 1 || listenPort > 65535) {
        toast("端口号必须在 1 到 65535 之间", "error");
        return;
    }

    try {
        await gatewayStore.saveConfig({
            listenHost,
            listenPort,
        });
        syncForm();
        toast("网关设置已保存", "success");
    } catch {
        toast("保存失败，请稍后重试", "error");
    }
}

onMounted(async () => {
    await gatewayStore.fetchConfig();
    await gatewayStore.fetchStatus();
    syncForm();
});
</script>

<style scoped>
.gateway-settings {
    min-height: 0;
    overflow: auto;
}

.gateway-settings-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
}

.settings-field {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.settings-field-label {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-primary);
}

.settings-help {
    font-size: 12px;
    line-height: 1.5;
    color: var(--color-text-muted);
}

.settings-actions {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
    gap: 12px;
}

.gateway-settings-notes {
    display: grid;
    gap: 12px;
}

.gateway-settings-note {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.gateway-settings-note code {
    width: fit-content;
    max-width: 100%;
    padding: 6px 10px;
    border-radius: var(--radius-sm);
    border: 1px solid var(--ui-border-default);
    background: var(--ui-bg-surface);
    color: var(--color-text-primary);
    word-break: break-all;
}

@media (max-width: 860px) {
    .gateway-settings-grid {
        grid-template-columns: 1fr;
    }

    .settings-actions {
        justify-content: stretch;
        flex-direction: column;
    }
}
</style>