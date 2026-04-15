<template>
  <div class="dialog-rules-view app-page">
    <PageHeader
      title="自定义对话规则"
      description="用统一的规则编排模型分流逻辑，让不同请求按语义落到正确供应商。"
      :icon="MessageSquareText"
    >
      <template #actions>
        <button class="btn btn-secondary" type="button" @click="addRouteRule">
          <Plus :size="14" />
          新增规则
        </button>
        <button class="btn btn-primary" type="button" @click="handleSaveRouteRules" :disabled="gatewayStore.loading">
          <Save :size="14" />
          保存规则
        </button>
      </template>
    </PageHeader>

    <section class="surface-card">
      <div class="panel-head">
        <div>
          <div class="panel-title">规则列表</div>
          <p class="panel-description">
            优先级越高越先匹配。若目标模型留空，则继续沿用供应商原有模型映射。
          </p>
        </div>
        <div class="panel-chip">
          {{ routeRuleDrafts.length }} 条规则
        </div>
      </div>

      <div v-if="routeRuleDrafts.length === 0" class="empty-hint">
        暂未配置自定义对话规则
      </div>

      <div v-else class="route-rule-list">
        <div v-for="(rule, index) in routeRuleDrafts" :key="index" class="route-rule-card">
          <div class="route-rule-grid">
            <div class="field">
              <label class="field-label">规则名</label>
              <input v-model="rule.name" class="field-input" placeholder="例如：代码问题走 Claude">
            </div>

            <div class="field">
              <label class="field-label">匹配方式</label>
              <select v-model="rule.matchType" class="field-input">
                <option value="model">模型名匹配</option>
                <option value="system_contains">System 包含</option>
                <option value="message_contains">任意消息包含</option>
                <option value="user_contains">用户消息包含</option>
                <option value="assistant_contains">助手消息包含</option>
              </select>
            </div>

            <div class="field">
              <label class="field-label">匹配内容</label>
              <input v-model="rule.pattern" class="field-input" placeholder="如：gpt-* / 代码审查 / 翻译">
            </div>

            <div class="field">
              <label class="field-label">目标供应商</label>
              <select v-model="rule.providerId" class="field-input">
                <option value="">请选择</option>
                <option v-for="provider in providerStore.providers" :key="provider.id" :value="provider.id">
                  {{ provider.name }}
                </option>
              </select>
            </div>

            <div class="field">
              <label class="field-label">目标模型</label>
              <input v-model="rule.targetModel" class="field-input" placeholder="可选，留空则沿用原模型映射">
            </div>

            <div class="field">
              <label class="field-label">优先级</label>
              <input v-model.number="rule.priority" type="number" class="field-input" placeholder="100">
            </div>
          </div>

          <div class="route-rule-actions">
            <label class="toggle-chip">
              <input v-model="rule.enabled" type="checkbox">
              <span>启用规则</span>
            </label>

            <button class="btn btn-ghost" type="button" @click="removeRouteRule(index)">
              删除
            </button>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue';
import { MessageSquareText, Plus, Save } from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';
import { useGatewayStore } from '@/stores/gateway';
import { useProviderStore } from '@/stores/provider';

const gatewayStore = useGatewayStore();
const providerStore = useProviderStore();
const routeRuleDrafts = ref([]);

function createEmptyRouteRule() {
  return {
    name: "",
    matchType: "model",
    pattern: "",
    providerId: "",
    targetModel: "",
    priority: 100,
    enabled: true,
  };
}

function addRouteRule() {
  routeRuleDrafts.value.push(createEmptyRouteRule());
}

function removeRouteRule(index) {
  routeRuleDrafts.value.splice(index, 1);
}

async function handleSaveRouteRules() {
  const sanitized = routeRuleDrafts.value
    .map((rule) => ({
      name: rule.name?.trim() || "",
      matchType: rule.matchType || "model",
      pattern: rule.pattern?.trim() || "",
      providerId: rule.providerId || "",
      targetModel: rule.targetModel?.trim() || "",
      priority: Number(rule.priority) || 100,
      enabled: !!rule.enabled,
    }))
    .filter((rule) => rule.pattern && rule.providerId);

  await gatewayStore.saveRouteRules(sanitized);
  routeRuleDrafts.value = sanitized.length > 0 ? sanitized.map((item) => ({ ...item })) : [];
}

onMounted(async () => {
  await Promise.all([
    gatewayStore.fetchRouteRules(),
    providerStore.fetchProviders(),
  ]);
  routeRuleDrafts.value = gatewayStore.routeRules.map((item) => ({ ...item }));
});
</script>

<style scoped>
.surface-card {
  margin-top: 8px;
  padding: 20px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--color-accent) 6%, transparent), transparent 160px),
    var(--color-bg-secondary);
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.panel-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.panel-description {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-text-muted);
}

.panel-chip {
  display: inline-flex;
  align-items: center;
  padding: 6px 10px;
  border-radius: 999px;
  background: var(--color-bg-tertiary, var(--color-bg-primary));
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.route-rule-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.route-rule-card {
  padding: 14px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  background: var(--color-bg-tertiary, var(--color-bg-primary));
}

.route-rule-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.route-rule-actions {
  margin-top: 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.empty-hint {
  font-size: 13px;
  color: var(--color-text-muted);
  padding: 24px;
  text-align: center;
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-lg);
}

@media (max-width: 1100px) {
  .route-rule-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .dialog-rules-view {
    padding: 16px;
  }

  .panel-head,
  .route-rule-actions {
    flex-direction: column;
    align-items: flex-start;
  }

  .route-rule-grid {
    grid-template-columns: 1fr;
  }
}
</style>
