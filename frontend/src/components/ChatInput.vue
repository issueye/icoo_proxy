<template>
    <div class="chat-input-shell">
        <div class="relative" ref="slashMenuRef">
            <div v-if="showSlashMenu" class="slash-menu rounded-md border bg-popover p-2 shadow-sm">
                <div class="slash-menu-header">
                    <div>
                        <div class="text-sm font-semibold text-foreground">
                            快捷指令
                        </div>
                        <div class="text-xs text-muted-foreground">
                            输入关键字快速填充常用提示词
                        </div>
                    </div>
                    <input
                        ref="slashSearchRef"
                        v-model="slashSearch"
                        type="text"
                        placeholder="筛选指令"
                        class="flex h-8 w-full rounded-md border border-input bg-background px-2 py-1 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring slash-search"
                    />
                </div>

                <div class="slash-menu-body">
                    <button
                        v-for="(cmd, index) in filteredSlashCommands"
                        :key="cmd.id"
                        class="slash-item"
                        :class="{ active: selectedSlashIndex === index }"
                        @click="selectSlashItem(cmd)"
                    >
                        <div class="slash-item-icon">
                            <component :is="cmd.icon" :size="16" />
                        </div>
                        <div class="min-w-0">
                            <div class="text-sm font-medium text-foreground">
                                {{ cmd.label }}
                            </div>
                            <div class="text-xs text-muted-foreground">
                                {{ cmd.description }}
                            </div>
                        </div>
                    </button>
                    <div v-if="filteredSlashCommands.length === 0" class="text-xs text-muted-foreground py-6 text-center">
                        没有匹配的快捷指令
                    </div>
                </div>
            </div>

            <div class="composer-card">
                <div class="composer-toolbar">
                    <div class="text-xs text-muted-foreground">
                        Enter 发送 · Shift+Enter 换行 · / 打开快捷指令
                    </div>
                </div>

                <div class="composer-main">
                    <textarea
                        ref="textareaRef"
                        v-model="inputText"
                        @keydown.enter.exact.prevent="handleSend"
                        @keydown.shift.enter="handleNewline"
                        @input="handleInput"
                        :disabled="disabled"
                        placeholder="输入消息，开始一轮新的任务或继续当前上下文..."
                        class="composer-textarea"
                        rows="1"
                    ></textarea>

                    <button
                        @click="handleSend"
                        :disabled="!canSend"
                        class="send-button"
                        :class="canSend ? 'ready' : 'disabled'"
                    >
                        <LoaderIcon v-if="disabled" :size="14" class="animate-spin" />
                        <SendIcon v-else :size="14" />
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, nextTick, watch, onMounted, onUnmounted } from "vue";
import { SendIcon, LoaderIcon, CodeIcon, SparklesIcon, CalculatorIcon, LanguagesIcon, FileTextIcon } from "lucide-vue-next";

const props = defineProps({
    disabled: {
        type: Boolean,
        default: false,
    },
});

const emit = defineEmits(["send"]);

const inputText = ref("");
const textareaRef = ref(null);
const slashMenuRef = ref(null);
const slashSearchRef = ref(null);
const showSlashMenu = ref(false);
const slashSearch = ref("");
const selectedSlashIndex = ref(0);

const slashCommands = [
    { id: "code", label: "/代码", description: "生成代码片段", icon: CodeIcon, action: () => insertText("请帮我写一段代码：") },
    { id: "translate", label: "/翻译", description: "翻译文本内容", icon: LanguagesIcon, action: () => insertText("请帮我翻译：") },
    { id: "summarize", label: "/总结", description: "总结长文本内容", icon: FileTextIcon, action: () => insertText("请帮我总结以下内容：") },
    { id: "analyze", label: "/分析", description: "分析数据或问题", icon: SparklesIcon, action: () => insertText("请帮我分析：") },
    { id: "math", label: "/计算", description: "数学计算", icon: CalculatorIcon, action: () => insertText("请帮我计算：") },
];

const filteredSlashCommands = computed(() => {
    if (!slashSearch.value) return slashCommands;
    const query = slashSearch.value.toLowerCase();
    return slashCommands.filter(cmd =>
        cmd.label.toLowerCase().includes(query) ||
        cmd.description.toLowerCase().includes(query)
    );
});

watch(slashSearch, () => {
    selectedSlashIndex.value = 0;
});

function insertText(text) {
    inputText.value = text;
    closeSlashMenu();
    textareaRef.value?.focus();
}

function selectSlashItem(cmd) {
    if (cmd) {
        cmd.action();
    }
    closeSlashMenu();
}

function closeSlashMenu() {
    showSlashMenu.value = false;
    slashSearch.value = "";
    selectedSlashIndex.value = 0;
}

function handleInput(e) {
    autoResize();
    const text = inputText.value;
    if (text === "/") {
        showSlashMenu.value = true;
        nextTick(() => {
            slashSearchRef.value?.focus();
        });
    } else if (text.startsWith("/")) {
        showSlashMenu.value = true;
    } else {
        showSlashMenu.value = false;
    }
}

function handleClickOutside(e) {
    if (slashMenuRef.value && !slashMenuRef.value.contains(e.target) && !textareaRef.value?.contains(e.target)) {
        closeSlashMenu();
    }
}

onMounted(() => {
    document.addEventListener("click", handleClickOutside);
});

onUnmounted(() => {
    document.removeEventListener("click", handleClickOutside);
});

const canSend = computed(() => !props.disabled && inputText.value.trim().length > 0);

function handleSend() {
    if (!canSend.value) return;
    const text = inputText.value.trim();
    emit("send", text);
    inputText.value = "";
    closeSlashMenu();
    nextTick(() => {
        if (textareaRef.value) {
            textareaRef.value.style.height = "auto";
        }
    });
}

function handleNewline() {
    // shift+enter 自然换行，不需要额外处理
}

function autoResize() {
    const el = textareaRef.value;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 200) + "px";
}

// 外部可调用聚焦
function focus() {
    textareaRef.value?.focus();
}

defineExpose({ focus });
</script>

<style scoped>
.chat-input-shell {
    padding: 0;
}

.slash-menu {
    position: absolute;
    left: 0;
    right: 0;
    bottom: calc(100% + 6px);
    z-index: 20;
}

.slash-menu-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 10px;
    padding-bottom: 6px;
    margin-bottom: 6px;
    border-bottom: 1px solid hsl(var(--border));
}

.slash-search {
    width: 180px;
}

.slash-menu-body {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 6px;
}

.slash-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px;
    border-radius: var(--radius);
    border: 1px solid hsl(var(--border));
    background: hsl(var(--background));
    transition: all 0.18s ease;
    text-align: left;
}

.slash-item:hover,
.slash-item.active {
    border-color: hsl(var(--primary) / 0.3);
    background: hsl(var(--secondary));
}

.slash-item-icon {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius);
    color: hsl(var(--primary));
    background: hsl(var(--background));
}

.composer-card {
    border-radius: var(--radius);
    border: 1px solid hsl(var(--border));
    background: hsl(var(--background));
}

.composer-toolbar {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 8px;
    padding: 6px 8px;
    border-bottom: 1px solid hsl(var(--border));
}

.composer-main {
    position: relative;
}

.composer-textarea {
    width: 100%;
    min-height: 64px;
    max-height: 240px;
    padding: 10px 46px 10px 10px;
    background: transparent;
    color: hsl(var(--foreground));
    resize: none;
    outline: none;
    border: none;
    line-height: 1.7;
    font-size: 0.9rem;
}

.composer-textarea::placeholder {
    color: hsl(var(--muted-foreground));
}

.send-button {
    position: absolute;
    right: 8px;
    bottom: 8px;
    width: 30px;
    height: 30px;
    border-radius: var(--radius);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.18s ease;
}

.send-button.ready {
    background: hsl(var(--primary));
    color: hsl(var(--primary-foreground));
}

.send-button.ready:hover {
    background: hsl(var(--primary) / 0.9);
}

.send-button.disabled {
    background: hsl(var(--secondary));
    color: hsl(var(--muted-foreground));
    cursor: not-allowed;
}

@media (max-width: 900px) {
    .slash-menu-header,
    .composer-toolbar {
        flex-direction: column;
        align-items: stretch;
    }

    .slash-search {
        width: 100%;
    }

    .slash-menu-body {
        grid-template-columns: 1fr;
    }
}
</style>
