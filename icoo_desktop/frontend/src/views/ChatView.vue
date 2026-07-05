<template>
  <section class="page-section chat-page">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton variant="secondary" @click="loadSuppliers">刷新供应商</UButton>
        <UButton variant="secondary" @click="clearChat">清空对话</UButton>
      </div>
    </Teleport>

    <div class="chat-layout">
      <aside class="content-panel chat-config">
        <div class="panel-header">
          <div class="panel-header__heading">
            <h2 class="panel-title">聊天配置</h2>
            <p class="panel-description">选择要直连的供应商和模型</p>
          </div>
        </div>
        <div class="panel-body chat-config__body">
          <USelect
            v-model="selectedSupplierId"
            label="供应商"
            :options="supplierOptions"
            searchable
            placeholder="选择供应商" />

          <USelect
            v-if="modelOptions.length"
            v-model="selectedModel"
            label="模型"
            :options="modelOptions"
            searchable
            placeholder="选择模型" />
          <UInput
            v-else
            v-model="selectedModel"
            label="模型"
            placeholder="该供应商未配置模型，请手动输入" />

          <UInput v-model="maxTokens" label="最大输出 Tokens" type="number" />

          <div v-if="selectedSupplier" class="chat-provider-summary">
            <div class="chat-provider-summary__row">
              <span>协议</span>
              <UTag size="xs" variant="info">{{ selectedSupplier.protocol }}</UTag>
            </div>
            <div class="chat-provider-summary__row">
              <span>状态</span>
              <UTag size="xs" :variant="selectedSupplier.enabled ? 'success' : 'error'">
                {{ selectedSupplier.enabled ? "启用" : "停用" }}
              </UTag>
            </div>
            <div class="chat-provider-summary__url" :title="selectedSupplier.base_url">
              {{ selectedSupplier.base_url || "未配置基础地址" }}
            </div>
          </div>

          <UAlert
            v-if="!supplierOptions.length && !loading"
            type="warning"
            message="当前尚未配置供应商，请先到供应商页面新增配置。" />
        </div>
      </aside>

      <main class="content-panel chat-console">
        <div class="panel-header">
          <div class="panel-header__heading">
            <h2 class="panel-title">聊天</h2>
            <p class="panel-description">{{ chatSubtitle }}</p>
          </div>
          <UTag v-if="lastLatency" size="xs" variant="neutral">{{ lastLatency }} ms</UTag>
        </div>

        <div ref="messageListRef" class="chat-message-list">
          <div v-if="messages.length === 0" class="chat-empty">
            <p class="chat-empty__title">开始一轮供应商直连聊天</p>
            <p class="chat-empty__desc">左侧选择供应商和模型后，在下方输入消息发送。</p>
          </div>

          <article
            v-for="item in messages"
            :key="item.id"
            class="chat-message"
            :class="[`chat-message--${item.role}`]"
          >
            <div class="chat-message__meta">
              <UTag size="xs" :variant="item.role === 'user' ? 'primary' : item.role === 'error' ? 'error' : 'success'">
                {{ roleLabel(item.role) }}
              </UTag>
              <span v-if="item.model" class="chat-message__model">{{ item.model }}</span>
            </div>
            <div class="chat-message__content">{{ item.content }}</div>
          </article>
        </div>

        <form class="chat-composer" @submit.prevent="sendMessage">
          <textarea
            v-model="draft"
            class="ued-input chat-composer__input"
            rows="3"
            placeholder="输入消息..."
            :disabled="sending"
            @keydown.enter.ctrl.prevent="sendMessage"
          ></textarea>
          <div class="chat-composer__actions">
            <span class="chat-composer__hint">Ctrl + Enter 发送</span>
            <UButton
              variant="primary"
              native-type="submit"
              :loading="sending"
              :disabled="sending || !canSend"
            >
              {{ sending ? "发送中..." : "发送" }}
            </UButton>
          </div>
        </form>
      </main>
    </div>
  </section>
</template>

<script setup>
import { computed, nextTick, onMounted, ref, watch } from "vue";
import { ChatWithSupplier, ListSuppliers } from "../lib/apiClient";
import UAlert from "../components/ued/UAlert.vue";
import UButton from "../components/ued/UButton.vue";
import UInput from "../components/ued/UInput.vue";
import USelect from "../components/ued/USelect.vue";
import UTag from "../components/ued/UTag.vue";
import { message } from "../components/ued/message";

const loading = ref(false);
const sending = ref(false);
const suppliers = ref([]);
const selectedSupplierId = ref("");
const selectedModel = ref("");
const maxTokens = ref(1024);
const draft = ref("");
const messages = ref([]);
const lastLatency = ref(0);
const messageListRef = ref(null);
let messageSeed = 0;

const supplierOptions = computed(() =>
  suppliers.value.map((supplier) => ({
    label: supplier.enabled ? supplier.name : `${supplier.name}（停用）`,
    value: supplier.id,
  })),
);

const selectedSupplier = computed(() =>
  suppliers.value.find((supplier) => supplier.id === selectedSupplierId.value) || null,
);

const modelOptions = computed(() =>
  (selectedSupplier.value?.models || [])
    .filter((model) => model.enabled !== false && model.name)
    .map((model) => ({
      label: model.name,
      value: model.name,
    })),
);

const chatSubtitle = computed(() => {
  if (!selectedSupplier.value) {
    return "请选择供应商";
  }
  return `${selectedSupplier.value.name} / ${selectedModel.value || "未选择模型"}`;
});

const canSend = computed(() =>
  Boolean(selectedSupplierId.value && selectedModel.value && draft.value.trim()),
);

watch(selectedSupplierId, () => {
  selectedModel.value = modelOptions.value[0]?.value || "";
});

watch(messages, () => {
  nextTick(scrollToBottom);
}, { deep: true });

async function loadSuppliers() {
  loading.value = true;
  try {
    suppliers.value = await ListSuppliers();
    if (!selectedSupplierId.value && suppliers.value.length) {
      const preferred = suppliers.value.find((supplier) => supplier.enabled) || suppliers.value[0];
      selectedSupplierId.value = preferred.id;
    }
  } catch (error) {
    message.error(error?.message || String(error));
  } finally {
    loading.value = false;
  }
}

async function sendMessage() {
  const content = draft.value.trim();
  if (!canSend.value || sending.value) {
    return;
  }

  const userMessage = createMessage("user", content);
  messages.value.push(userMessage);
  draft.value = "";
  sending.value = true;

  try {
    const result = await ChatWithSupplier(selectedSupplierId.value, {
      model: selectedModel.value,
      messages: messages.value
        .filter((item) => item.role === "user" || item.role === "assistant")
        .map((item) => ({ role: item.role, content: item.content })),
      max_tokens: Number(maxTokens.value || 1024),
    });
    lastLatency.value = result.duration_ms;
    messages.value.push(createMessage("assistant", result.message.content, result.model));
  } catch (error) {
    const text = error?.message || String(error);
    messages.value.push(createMessage("error", text));
    message.error(text);
  } finally {
    sending.value = false;
  }
}

function clearChat() {
  messages.value = [];
  lastLatency.value = 0;
}

function createMessage(role, content, model = "") {
  return {
    id: `chat-message-${Date.now()}-${messageSeed++}`,
    role,
    content,
    model,
  };
}

function roleLabel(role) {
  if (role === "user") {
    return "用户";
  }
  if (role === "error") {
    return "错误";
  }
  return "助手";
}

function scrollToBottom() {
  const el = messageListRef.value;
  if (el) {
    el.scrollTop = el.scrollHeight;
  }
}

onMounted(loadSuppliers);
</script>
