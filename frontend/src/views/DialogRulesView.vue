<template>
  <div class="dialog-rules-view app-page">
    <UEDPageHeader
      title="对话规则"
      description="维护模型与消息匹配规则，决定请求转发到哪个供应商与目标模型。"
      :icon="MessageSquareText"
      divided
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
    </UEDPageHeader>

    <div class="rules-workspace">
      <section class="rules-list-panel">
        <div class="panel-head">
          <h3 class="section-title">规则列表</h3>
          <span class="panel-chip">{{ routeRuleDrafts.length }} 条</span>
        </div>

        <div v-if="routeRuleDrafts.length === 0" class="empty-state">
          <div class="empty-title">暂未配置自定义规则</div>
          <button class="btn btn-primary" type="button" @click="addRouteRule">
            <Plus :size="14" />
            添加第一条规则
          </button>
        </div>

        <div v-else class="rules-list">
          <button
            v-for="(rule, index) in routeRuleDrafts"
            :key="index"
            class="rule-list-item"
            :class="{ 'is-active': selectedRuleIndex === index }"
            @click="selectedRuleIndex = index"
          >
            <div class="rule-list-item-head">
              <span class="rule-list-title">{{ rule.name || `未命名规则 ${index + 1}` }}</span>
              <StatusBadge :status="rule.enabled ? 'success' : 'neutral'" :label="rule.enabled ? '启用' : '停用'" />
            </div>
            <div class="rule-list-meta">
              <span>{{ matchTypeLabelMap[rule.matchType] || rule.matchType }}</span>
              <span>优先级 {{ rule.priority || 100 }}</span>
            </div>
            <div class="rule-list-pattern">{{ rule.pattern || "等待填写匹配内容" }}</div>
          </button>
        </div>
      </section>

      <section class="rules-editor-panel">
        <div v-if="!selectedRule" class="empty-state rules-editor-empty">
          <div class="empty-title">请选择一条规则</div>
        </div>

        <template v-else>
          <div class="panel-head">
            <div>
              <h3 class="section-title">规则详情</h3>
            </div>
            <div class="editor-actions">
              <label class="toggle-chip">
                <input v-model="selectedRule.enabled" type="checkbox">
                <span>启用规则</span>
              </label>
              <button class="btn btn-ghost" type="button" @click="removeSelectedRule">删除</button>
            </div>
          </div>

          <div class="editor-grid">
            <div class="field">
              <label class="field-label">规则名</label>
              <input v-model="selectedRule.name" class="field-input" placeholder="例如：代码问题走 Claude" />
            </div>

            <div class="field">
              <label class="field-label">优先级</label>
              <input v-model.number="selectedRule.priority" type="number" class="field-input" placeholder="100" />
            </div>

            <div class="field">
              <label class="field-label">匹配方式</label>
              <Select v-model="selectedRule.matchType" :options="matchTypeOptions" />
            </div>

            <div class="field field--wide">
              <label class="field-label">匹配内容</label>
              <input v-model="selectedRule.pattern" class="field-input" placeholder="如：gpt-* / 代码审查 / 翻译" />
            </div>

            <div class="field">
              <label class="field-label">目标供应商</label>
              <Select v-model="selectedRule.providerId" :options="providerOptions" />
            </div>

            <div class="field">
              <label class="field-label">目标模型</label>
              <input v-model="selectedRule.targetModel" class="field-input" placeholder="可选，留空则沿用原模型映射" />
            </div>
          </div>

          <div class="rules-note settings-note settings-note--accent">
            <div class="field-label">编辑提示</div>
            <p>规则按优先级从高到低生效。目标模型留空时，会继续使用供应商上的模型映射结果。</p>
          </div>

          <section class="debug-panel">
            <div class="panel-head debug-panel-head">
              <div>
                <h3 class="section-title">规则调试</h3>
                <p class="panel-description">输入一组样例请求，查看当前会命中哪条规则以及最终路由结果。</p>
              </div>
              <button class="btn btn-secondary" type="button" @click="handleDebugRoute" :disabled="debugLoading">
                {{ debugLoading ? '调试中...' : '模拟路由' }}
              </button>
            </div>

            <div class="editor-grid">
              <div class="field">
                <label class="field-label">测试模型名</label>
                <input v-model="debugForm.model" class="field-input" placeholder="例如：gpt-4o" />
              </div>

              <div class="field field--wide">
                <label class="field-label">System Prompt</label>
                <textarea v-model="debugForm.systemPrompt" class="field-input field-textarea" placeholder="输入 system prompt，用于测试 system_contains 规则"></textarea>
              </div>

              <div class="field field--wide">
                <label class="field-label">用户消息</label>
                <textarea v-model="debugForm.userMessage" class="field-input field-textarea" placeholder="输入一条用户消息，用于测试 user_contains / message_contains 规则"></textarea>
              </div>
            </div>

            <div v-if="debugResult" class="debug-result">
              <div class="debug-result-head">
                <StatusBadge
                  :status="debugResult.matched ? 'success' : 'warning'"
                  :label="debugResult.matched ? '命中规则' : '使用回退策略'"
                />
                <span class="panel-chip">{{ debugResult.providerName || '未解析供应商' }}</span>
              </div>

              <div class="debug-result-grid">
                <div class="detail-item">
                  <span class="detail-label">命中规则</span>
                  <span class="detail-value">{{ debugResult.matchedRule || '--' }}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">目标供应商</span>
                  <span class="detail-value">{{ debugResult.providerName || '--' }}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">目标模型</span>
                  <span class="detail-value">{{ debugResult.targetModel || debugForm.model || '--' }}</span>
                </div>
                <div class="detail-item detail-item--wide">
                  <span class="detail-label">原因说明</span>
                  <span class="detail-value">{{ debugResult.matchedReason || debugResult.fallbackReason || '--' }}</span>
                </div>
              </div>
            </div>
          </section>
        </template>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { MessageSquareText, Plus, Save } from 'lucide-vue-next';
import { UEDPageHeader } from '@/components/layout';
import StatusBadge from '@/components/ui/StatusBadge.vue';
import Select from '@/components/ui/Select.vue';
import { useGatewayStore } from '@/stores/gateway';
import { useProviderStore } from '@/stores/provider';
import { useToast } from '@/composables/useToast';

const gatewayStore = useGatewayStore();
const providerStore = useProviderStore();
const { toast } = useToast();
const routeRuleDrafts = ref([]);
const selectedRuleIndex = ref(0);
const debugLoading = ref(false);
const debugResult = ref(null);
const debugForm = ref({
  model: 'gpt-4o',
  systemPrompt: '',
  userMessage: '',
});

const matchTypeLabelMap = {
  model: "模型名匹配",
  system_contains: "System 包含",
  message_contains: "任意消息包含",
  user_contains: "用户消息包含",
  assistant_contains: "助手消息包含",
};

const matchTypeOptions = [
  { label: "模型名匹配", value: "model" },
  { label: "System 包含", value: "system_contains" },
  { label: "任意消息包含", value: "message_contains" },
  { label: "用户消息包含", value: "user_contains" },
  { label: "助手消息包含", value: "assistant_contains" },
];

const providerOptions = computed(() => [
  { label: "请选择", value: "" },
  ...providerStore.providers.map((provider) => ({
    label: provider.name,
    value: provider.id,
  })),
]);

const selectedRule = computed(() => routeRuleDrafts.value[selectedRuleIndex.value] || null);

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
  selectedRuleIndex.value = routeRuleDrafts.value.length - 1;
}

function removeRouteRule(index) {
  routeRuleDrafts.value.splice(index, 1);
  if (routeRuleDrafts.value.length === 0) {
    selectedRuleIndex.value = 0;
    return;
  }
  selectedRuleIndex.value = Math.min(selectedRuleIndex.value, routeRuleDrafts.value.length - 1);
}

function removeSelectedRule() {
  if (selectedRule.value) {
    removeRouteRule(selectedRuleIndex.value);
  }
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
  selectedRuleIndex.value = 0;
  toast('规则已保存', 'success');
}

async function handleDebugRoute() {
  debugLoading.value = true;
  try {
    debugResult.value = await gatewayStore.debugRoute(debugForm.value);
  } catch (e) {
    toast(e.message || '规则调试失败', 'error');
  } finally {
    debugLoading.value = false;
  }
}

watch(
  () => routeRuleDrafts.value.length,
  (length) => {
    if (length > 0 && selectedRuleIndex.value > length - 1) {
      selectedRuleIndex.value = length - 1;
    }
  }
);

onMounted(async () => {
  await Promise.all([
    gatewayStore.fetchRouteRules(),
    providerStore.fetchProviders(),
  ]);
  routeRuleDrafts.value = gatewayStore.routeRules.map((item) => ({ ...item }));
});
</script>

<style scoped>
.dialog-rules-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rules-workspace {
  display: grid;
  grid-template-columns: 340px minmax(0, 1fr);
  gap: 16px;
  min-height: 0;
  flex: 1;
}

.rules-list-panel,
.rules-editor-panel {
  padding: 16px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-rest);
}

.rules-list-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.rules-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  overflow-y: auto;
}

.rule-list-item {
  padding: 12px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
  text-align: left;
  transition: all 0.14s ease;
}

.rule-list-item:hover,
.rule-list-item.is-active {
  border-color: color-mix(in srgb, var(--color-accent) 26%, var(--ui-border-default));
  background: var(--color-accent-soft);
}

.rule-list-item-head,
.rule-list-meta,
.panel-head,
.debug-result-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.rule-list-item-head {
  margin-bottom: 6px;
}

.rule-list-title {
  font-size: 13px;
  font-weight: 700;
}

.rule-list-meta {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 8px;
}

.rule-list-pattern {
  font-size: 12px;
  color: var(--color-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rules-editor-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.editor-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.editor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.field--wide,
.detail-item--wide {
  grid-column: 1 / -1;
}

.field-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.field-input {
  width: 100%;
  min-height: 38px;
  padding: 9px 12px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
  color: var(--color-text-primary);
}

.field-textarea {
  min-height: 92px;
  resize: vertical;
}

.rules-note,
.debug-panel {
  padding: 14px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
}

.rules-note p,
.panel-description {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-text-muted);
}

.debug-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.debug-panel-head {
  align-items: flex-start;
}

.debug-result {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
  border: 1px solid color-mix(in srgb, var(--color-accent) 24%, var(--ui-border-default));
  border-radius: var(--radius-sm);
  background: var(--color-accent-soft);
}

.debug-result-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.detail-label {
  font-size: 12px;
  color: var(--color-text-muted);
}

.detail-value {
  font-size: 13px;
  color: var(--color-text-primary);
  line-height: 1.6;
  word-break: break-word;
}

@media (max-width: 960px) {
  .rules-workspace {
    grid-template-columns: 1fr;
  }

  .editor-grid,
  .debug-result-grid {
    grid-template-columns: 1fr;
  }
}
</style>