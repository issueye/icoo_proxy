<template>
  <section class="page-section chat-page">
    <Teleport to="#app-topbar-actions">
      <div class="app-topbar-actions__group">
        <UButton variant="secondary" size="sm" :loading="loading" @click="loadSuppliers">
          {{ loading ? "刷新中..." : "刷新供应商" }}
        </UButton>
        <UButton
          variant="secondary"
          size="sm"
          :disabled="!messages.length"
          @click="clearChat"
        >
          清空对话
        </UButton>
      </div>
    </Teleport>

    <div class="chat-layout">
      <!-- ── Config sidebar ── -->
      <aside class="content-panel chat-config">
        <div class="panel-header chat-config__header">
          <div class="panel-header__heading">
            <h2 class="panel-title">会话配置</h2>
          </div>
        </div>

        <div class="panel-body chat-config__body">
          <USelect
            v-model="selectedSupplierId"
            label="供应商"
            :options="supplierOptions"
            searchable
            placeholder="选择供应商"
          />

          <USelect
            v-if="modelOptions.length"
            v-model="selectedModel"
            label="模型"
            :options="modelOptions"
            searchable
            placeholder="选择模型"
          />
          <UInput
            v-else
            v-model="selectedModel"
            label="模型"
            placeholder="手动输入模型名"
          />

          <div class="chat-adv">
            <button
              type="button"
              class="chat-adv__toggle"
              :aria-expanded="showAdvanced"
              @click="showAdvanced = !showAdvanced"
            >
              <span>高级参数</span>
              <span class="chat-adv__chevron" :class="{ 'is-open': showAdvanced }">▾</span>
            </button>
            <div v-show="showAdvanced" class="chat-adv__body">
              <UInput v-model="maxTokens" label="最大输出 Tokens" type="number" />
              <p class="chat-adv__hint">管理端聊天为非流式请求；仅流式供应商不可用。</p>
            </div>
          </div>

          <div v-if="selectedSupplier" class="chat-provider-card">
            <div class="chat-provider-card__title">
              <span class="chat-provider-card__name" :title="selectedSupplier.name">
                {{ selectedSupplier.name }}
              </span>
              <UTag
                size="xs"
                :variant="selectedSupplier.enabled ? 'success' : 'error'"
              >
                {{ selectedSupplier.enabled ? "启用" : "停用" }}
              </UTag>
            </div>
            <div class="chat-provider-card__tags">
              <UTag size="xs" variant="info">{{ selectedSupplier.protocol || "-" }}</UTag>
              <UTag v-if="isPluginSupplier" size="xs" variant="info">进程插件</UTag>
              <UTag v-if="selectedSupplier.only_stream" size="xs" variant="warning">仅流式</UTag>
            </div>
            <p class="chat-provider-card__url" :title="providerEndpointLabel">
              {{ providerEndpointLabel }}
            </p>
            <p v-if="isPluginSupplier" class="chat-provider-card__hint">
              经 bridge IPC 转发；凭据在插件扩展页管理。
            </p>
            <p
              v-else-if="selectedSupplier.only_stream"
              class="chat-provider-card__hint chat-provider-card__hint--warn"
            >
              该供应商仅支持流式，管理聊天可能失败。
            </p>
          </div>

          <div class="chat-session-stats">
            <div class="chat-session-stats__item">
              <span class="chat-session-stats__k">消息</span>
              <span class="chat-session-stats__v">{{ messageCount }}</span>
            </div>
            <div class="chat-session-stats__item">
              <span class="chat-session-stats__k">最近耗时</span>
              <span class="chat-session-stats__v">{{ lastLatency ? `${lastLatency} ms` : "—" }}</span>
            </div>
          </div>

          <UAlert
            v-if="!supplierOptions.length && !loading"
            type="warning"
            message="尚未配置供应商，请先到「供应商」页新增。"
          />
        </div>
      </aside>

      <!-- ── Chat main ── -->
      <main class="content-panel chat-console">
        <div class="panel-header chat-console__header">
          <div class="panel-header__heading">
            <h2 class="panel-title">{{ chatTitle }}</h2>
            <p class="panel-description">{{ chatSubtitle }}</p>
          </div>
          <div class="chat-console__meta">
            <UTag v-if="selectedModel" size="xs" variant="neutral">{{ selectedModel }}</UTag>
            <UTag v-if="lastLatency" size="xs" variant="info">{{ lastLatency }} ms</UTag>
          </div>
        </div>

        <div ref="messageListRef" class="chat-message-list" role="log" aria-live="polite">
          <div v-if="messages.length === 0 && !sending" class="chat-empty">
            <div class="chat-empty__icon" aria-hidden="true">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15a4 4 0 0 1-4 4H8l-5 3V7a4 4 0 0 1 4-4h10a4 4 0 0 1 4 4z" />
                <path d="M8 9h8" />
                <path d="M8 13h5" />
              </svg>
            </div>
            <p class="chat-empty__title">开始测试对话</p>
            <p class="chat-empty__desc">
              左侧选择供应商与模型，在下方输入消息。支持 HTTP 上游与进程插件。
            </p>
            <div class="chat-empty__chips">
              <button
                v-for="chip in quickPrompts"
                :key="chip"
                type="button"
                class="chat-chip"
                :disabled="!canCompose"
                @click="useQuickPrompt(chip)"
              >
                {{ chip }}
              </button>
            </div>
          </div>

          <article
            v-for="item in messages"
            :key="item.id"
            class="chat-message"
            :class="[`chat-message--${item.role}`]"
          >
            <div class="chat-message__avatar" aria-hidden="true">
              {{ avatarLetter(item.role) }}
            </div>
            <div class="chat-message__body">
              <div class="chat-message__meta">
                <span class="chat-message__role">{{ roleLabel(item.role) }}</span>
                <span v-if="item.model" class="chat-message__model">{{ item.model }}</span>
                <span v-if="item.durationMs" class="chat-message__latency">{{ item.durationMs }} ms</span>
                <span v-if="item.time" class="chat-message__time">{{ item.time }}</span>
                <button
                  v-if="item.role !== 'error' && item.content"
                  type="button"
                  class="chat-message__copy"
                  title="复制"
                  @click="copyMessage(item)"
                >
                  复制
                </button>
              </div>
              <div class="chat-message__content">{{ item.content }}</div>
              <div v-if="item.role === 'error' && item.retryable" class="chat-message__actions">
                <UButton size="xs" variant="secondary" :disabled="sending" @click="retryLast">
                  重试
                </UButton>
              </div>
            </div>
          </article>

          <div v-if="sending" class="chat-message chat-message--assistant chat-message--typing">
            <div class="chat-message__avatar" aria-hidden="true">助</div>
            <div class="chat-message__body">
              <div class="chat-message__meta">
                <span class="chat-message__role">助手</span>
                <span class="chat-message__model">{{ selectedModel }}</span>
              </div>
              <div class="chat-message__content chat-message__content--typing">
                <span class="chat-typing" aria-label="正在生成">
                  <i /><i /><i />
                </span>
                正在请求…
              </div>
            </div>
          </div>
        </div>

        <form class="chat-composer" @submit.prevent="sendMessage">
          <div class="chat-composer__box">
            <textarea
              ref="composerRef"
              v-model="draft"
              class="chat-composer__input"
              rows="1"
              :placeholder="composerPlaceholder"
              :disabled="sending || !canCompose"
              @keydown="onComposerKeydown"
              @input="autoResizeComposer"
            />
            <div class="chat-composer__bar">
              <span class="chat-composer__hint">
                <template v-if="!canCompose">请先选择供应商和模型</template>
                <template v-else>Enter 发送 · Shift+Enter 换行</template>
              </span>
              <div class="chat-composer__bar-right">
                <span v-if="draft.length" class="chat-composer__count">{{ draft.length }}</span>
                <UButton
                  variant="primary"
                  size="sm"
                  native-type="submit"
                  :loading="sending"
                  :disabled="sending || !canSend"
                >
                  {{ sending ? "发送中" : "发送" }}
                </UButton>
              </div>
            </div>
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

const quickPrompts = ["ping", "用一句话介绍你自己", "1+1等于几？"];

const loading = ref(false);
const sending = ref(false);
const showAdvanced = ref(false);
const suppliers = ref([]);
const selectedSupplierId = ref("");
const selectedModel = ref("");
const maxTokens = ref(1024);
const draft = ref("");
const messages = ref([]);
const lastLatency = ref(0);
const lastFailedPayload = ref(null);
const messageListRef = ref(null);
const composerRef = ref(null);
let messageSeed = 0;

const supplierOptions = computed(() =>
  suppliers.value.map((supplier) => ({
    label: supplier.enabled ? supplier.name : `${supplier.name}（停用）`,
    value: supplier.id,
  })),
);

const selectedSupplier = computed(
  () => suppliers.value.find((s) => s.id === selectedSupplierId.value) || null,
);

const isPluginSupplier = computed(
  () => String(selectedSupplier.value?.vendor || "").toLowerCase() === "plugin",
);

const modelOptions = computed(() =>
  (selectedSupplier.value?.models || [])
    .filter((model) => model.enabled !== false && model.name)
    .map((model) => ({
      label: model.name,
      value: model.name,
    })),
);

const providerEndpointLabel = computed(() => {
  const s = selectedSupplier.value;
  if (!s) return "";
  if (isPluginSupplier.value) {
    return s.base_url || `plugin://${s.plugin_id || "?"}`;
  }
  return s.base_url || "未配置基础地址";
});

const chatTitle = computed(() => {
  if (!selectedSupplier.value) return "聊天";
  return selectedSupplier.value.name;
});

const chatSubtitle = computed(() => {
  if (!selectedSupplier.value) return "选择供应商开始测试";
  if (!selectedModel.value) return "请选择或输入模型";
  return isPluginSupplier.value
    ? `进程插件 · ${selectedModel.value}`
    : `${selectedSupplier.value.protocol || "上游"} · ${selectedModel.value}`;
});

const messageCount = computed(
  () => messages.value.filter((m) => m.role === "user" || m.role === "assistant").length,
);

const canCompose = computed(() =>
  Boolean(selectedSupplierId.value && selectedModel.value && selectedSupplier.value?.enabled !== false),
);

const canSend = computed(() => Boolean(canCompose.value && draft.value.trim()));

const composerPlaceholder = computed(() => {
  if (!selectedSupplierId.value) return "请先选择供应商…";
  if (!selectedModel.value) return "请先选择模型…";
  if (selectedSupplier.value && !selectedSupplier.value.enabled) return "供应商已停用";
  return "输入消息，Enter 发送…";
});

watch(selectedSupplierId, () => {
  selectedModel.value = modelOptions.value[0]?.value || "";
});

watch(
  messages,
  () => {
    nextTick(scrollToBottom);
  },
  { deep: true },
);

watch(sending, (v) => {
  if (v) nextTick(scrollToBottom);
});

async function loadSuppliers() {
  loading.value = true;
  try {
    suppliers.value = await ListSuppliers();
    if (!selectedSupplierId.value && suppliers.value.length) {
      const preferred =
        suppliers.value.find((s) => s.enabled) || suppliers.value[0];
      selectedSupplierId.value = preferred.id;
    } else if (selectedSupplierId.value) {
      // Keep selection; refresh models list for current supplier.
      const still = suppliers.value.find((s) => s.id === selectedSupplierId.value);
      if (!still) {
        selectedSupplierId.value = suppliers.value[0]?.id || "";
      } else if (!selectedModel.value && modelOptions.value.length) {
        selectedModel.value = modelOptions.value[0].value;
      }
    }
  } catch (error) {
    message.error(error?.message || String(error));
  } finally {
    loading.value = false;
  }
}

function useQuickPrompt(text) {
  if (!canCompose.value) return;
  draft.value = text;
  nextTick(() => {
    autoResizeComposer();
    composerRef.value?.focus();
  });
}

function onComposerKeydown(event) {
  // Enter sends; Shift+Enter inserts newline. IME composition skip.
  if (event.key !== "Enter" || event.shiftKey || event.isComposing) return;
  event.preventDefault();
  sendMessage();
}

function autoResizeComposer() {
  const el = composerRef.value;
  if (!el) return;
  el.style.height = "auto";
  const max = 160;
  el.style.height = `${Math.min(el.scrollHeight, max)}px`;
}

async function sendMessage() {
  const content = draft.value.trim();
  if (!canSend.value || sending.value) return;

  const history = messages.value
    .filter((item) => item.role === "user" || item.role === "assistant")
    .map((item) => ({ role: item.role, content: item.content }));

  const payload = {
    model: selectedModel.value,
    messages: [...history, { role: "user", content }],
    max_tokens: Number(maxTokens.value || 1024),
  };

  messages.value.push(createMessage("user", content));
  draft.value = "";
  nextTick(autoResizeComposer);
  sending.value = true;
  lastFailedPayload.value = null;

  try {
    const result = await ChatWithSupplier(selectedSupplierId.value, payload);
    lastLatency.value = Number(result.duration_ms || 0);
    messages.value.push(
      createMessage("assistant", result.message?.content || "", result.model || selectedModel.value, {
        durationMs: lastLatency.value,
      }),
    );
  } catch (error) {
    const text = error?.message || String(error);
    lastFailedPayload.value = payload;
    messages.value.push(createMessage("error", text, "", { retryable: true }));
    message.error(text);
  } finally {
    sending.value = false;
    nextTick(() => composerRef.value?.focus());
  }
}

async function retryLast() {
  if (!lastFailedPayload.value || sending.value) return;
  // Remove last error bubble before retry.
  const last = messages.value[messages.value.length - 1];
  if (last?.role === "error") {
    messages.value = messages.value.slice(0, -1);
  }
  const payload = lastFailedPayload.value;
  sending.value = true;
  try {
    const result = await ChatWithSupplier(selectedSupplierId.value, payload);
    lastLatency.value = Number(result.duration_ms || 0);
    lastFailedPayload.value = null;
    messages.value.push(
      createMessage("assistant", result.message?.content || "", result.model || selectedModel.value, {
        durationMs: lastLatency.value,
      }),
    );
  } catch (error) {
    const text = error?.message || String(error);
    messages.value.push(createMessage("error", text, "", { retryable: true }));
    message.error(text);
  } finally {
    sending.value = false;
  }
}

function clearChat() {
  messages.value = [];
  lastLatency.value = 0;
  lastFailedPayload.value = null;
  nextTick(() => composerRef.value?.focus());
}

function createMessage(role, content, model = "", extra = {}) {
  const now = new Date();
  const time = `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}`;
  return {
    id: `chat-message-${Date.now()}-${messageSeed++}`,
    role,
    content,
    model,
    time,
    durationMs: extra.durationMs || 0,
    retryable: Boolean(extra.retryable),
  };
}

function roleLabel(role) {
  if (role === "user") return "用户";
  if (role === "error") return "错误";
  return "助手";
}

function avatarLetter(role) {
  if (role === "user") return "我";
  if (role === "error") return "!";
  return "助";
}

async function copyMessage(item) {
  const text = String(item?.content || "");
  if (!text) return;
  try {
    await navigator.clipboard.writeText(text);
    message.success("已复制到剪贴板");
  } catch {
    message.error("复制失败");
  }
}

function scrollToBottom() {
  const el = messageListRef.value;
  if (el) {
    el.scrollTop = el.scrollHeight;
  }
}

onMounted(async () => {
  await loadSuppliers();
  nextTick(() => {
    autoResizeComposer();
    composerRef.value?.focus();
  });
});
</script>
