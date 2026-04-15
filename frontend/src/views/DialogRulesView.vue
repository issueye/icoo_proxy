<template>
  <div class="dialog-rules-view app-page">
    <PageHeader
      title="对话规则"
      description="用双栏工作区管理分流规则：左侧查看列表和启用状态，右侧编辑当前规则的匹配条件与目标路由。"
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

    <div class="rules-workspace">
      <section class="rules-list-panel">
        <div class="panel-head">
          <div>
            <h3 class="section-title">规则列表</h3>
            <p class="panel-description">优先级越高越先匹配，未设置目标模型时沿用供应商原始映射。</p>
          </div>
          <span class="panel-chip">{{ routeRuleDrafts.length }} 条</span>
        </div>

        <div v-if="routeRuleDrafts.length === 0" class="empty-state">
          <div class="empty-title">暂未配置自定义规则</div>
          <p>新增规则后，可按模型名或消息内容把请求转发到指定供应商。</p>
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
          <p>从左侧选择已有规则，或新建一条规则开始编辑。</p>
        </div>

        <template v-else>
          <div class="panel-head">
            <div>
              <h3 class="section-title">规则详情</h3>
              <p class="panel-description">编辑匹配方式、目标供应商和目标模型。</p>
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
              <select v-model="selectedRule.matchType" class="field-input">
                <option value="model">模型名匹配</option>
                <option value="system_contains">System 包含</option>
                <option value="message_contains">任意消息包含</option>
                <option value="user_contains">用户消息包含</option>
                <option value="assistant_contains">助手消息包含</option>
              </select>
            </div>

            <div class="field field--wide">
              <label class="field-label">匹配内容</label>
              <input v-model="selectedRule.pattern" class="field-input" placeholder="如：gpt-* / 代码审查 / 翻译" />
            </div>

            <div class="field">
              <label class="field-label">目标供应商</label>
              <select v-model="selectedRule.providerId" class="field-input">
                <option value="">请选择</option>
                <option v-for="provider in providerStore.providers" :key="provider.id" :value="provider.id">
                  {{ provider.name }}
                </option>
              </select>
            </div>

            <div class="field">
              <label class="field-label">目标模型</label>
              <input v-model="selectedRule.targetModel" class="field-input" placeholder="可选，留空则沿用原模型映射" />
            </div>
          </div>

          <div class="rules-note settings-note settings-note--accent">
            <div class="field-label">编辑提示</div>
            <div class="settings-help">规则保存时会自动忽略没有匹配内容或未选择目标供应商的条目。</div>
          </div>
        </template>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue';
import { MessageSquareText, Plus, Save } from 'lucide-vue-next';
import PageHeader from '@/components/layout/PageHeader.vue';
import StatusBadge from '@/components/ui/StatusBadge.vue';
import { useGatewayStore } from '@/stores/gateway';
import { useProviderStore } from '@/stores/provider';

const gatewayStore = useGatewayStore();
const providerStore = useProviderStore();
const routeRuleDrafts = ref([]);
const selectedRuleIndex = ref(0);

const matchTypeLabelMap = {
  model: "模型名匹配",
  system_contains: "System 包含",
  message_contains: "任意消息包含",
  user_contains: "用户消息包含",
  assistant_contains: "助手消息包含",
};

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
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  padding: 12px;
  border: 1px solid var(--ui-border-subtle);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface-muted);
  text-align: left;
  transition: all 0.14s ease;
}

.rule-list-item:hover {
  background: var(--ui-bg-surface-hover);
  border-color: var(--ui-border-default);
}

.rule-list-item.is-active {
  background: var(--color-accent-soft);
  border-color: color-mix(in srgb, var(--color-accent) 24%, var(--ui-border-default));
}

.rule-list-item-head,
.rule-list-meta,
.editor-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.rule-list-title,
.empty-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.rule-list-meta,
.rule-list-pattern,
.empty-state p {
  font-size: 12px;
  line-height: 1.55;
  color: var(--color-text-muted);
}

.rules-editor-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rules-editor-empty {
  min-height: 280px;
}

.editor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.field--wide {
  grid-column: 1 / -1;
}

.rules-note {
  padding: 12px;
}

@media (max-width: 980px) {
  .rules-workspace {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .editor-grid {
    grid-template-columns: 1fr;
  }

  .editor-actions {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
