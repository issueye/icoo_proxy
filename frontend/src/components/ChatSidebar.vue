<template>
    <div
        class="chat-sidebar bg-secondary page-panel sidebar-transition"
        :style="{ width: collapsed ? '0' : '280px' }"
    >
        <div
            class="session-content flex flex-col h-full overflow-hidden"
            :class="collapsed ? 'invisible' : 'visible'"
        >
            <div class="flex items-center justify-between px-2 py-1.5 border-b border-border">
                <div class="flex items-center gap-3">
                    <div
                        class="w-8 h-8 rounded-md bg-primary flex items-center justify-center flex-shrink-0"
                    >
                        <BotIcon :size="16" class="text-primary-foreground" />
                    </div>
                    <div class="min-w-0">
                        <div class="font-semibold text-sm text-foreground">
                            icooclaw
                        </div>
                    </div>
                </div>
                <button
                    @click="$emit('toggle')"
                    class="text-muted-foreground hover:text-foreground hover:bg-accent hover:text-accent-foreground rounded-md p-1.5 transition-colors"
                >
                    <PanelLeftCloseIcon :size="16" />
                </button>
            </div>

            <div class="px-2 pt-2.5 pb-2">
                <Button
                    @click="handleNewChat"
                    variant="brand"
                    size="default"
                    class="w-full"
                >
                    <PlusIcon :size="16" />
                    新建对话
                </Button>
            </div>

            <div class="px-2 pb-2">
                <div class="relative">
                    <SearchIcon
                        :size="14"
                        class="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none"
                    />
                    <input
                        v-model="searchQuery"
                        type="text"
                        placeholder="搜索会话..."
                        class="w-full pl-10 pr-8 py-2.5 text-sm bg-background border border-border rounded-md outline-none focus:border-ring transition-colors placeholder:text-muted-foreground text-foreground"
                    />
                    <button
                        v-if="searchQuery"
                        @click="clearSearch"
                        class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                    >
                        <XIcon :size="12" />
                    </button>
                </div>
            </div>

            <div class="flex-1 overflow-y-auto px-1.5 pb-1.5 space-y-0.5">
                <div
                    v-if="filteredSessions.length === 0"
                    class="text-center text-muted-foreground text-sm py-10"
                >
                    {{ searchQuery ? '未找到匹配的会话' : '暂无对话记录' }}
                </div>

                <template v-if="!searchQuery">
                    <!-- 今天 -->
                    <template v-if="groupedSessions.today.length > 0">
                        <div class="px-3 pt-2 pb-1.5 text-[11px] text-muted-foreground font-semibold uppercase tracking-[0.16em]">
                            今天
                        </div>
                        <div
                            v-for="session in groupedSessions.today"
                            :key="session.id"
                            class="session-row group"
                        >
                            <button
                                @click="$emit('select', session.id)"
                                class="session-btn"
                                :class="
                                    session.id === currentSessionId
                                        ? 'active'
                                        : ''
                                "
                            >
                                <MessageSquareIcon :size="14" class="flex-shrink-0 opacity-60" />
                                <span class="flex-1 truncate font-medium text-xs">{{ session.title || "新对话" }}</span>
                            </button>
                            <div class="session-actions">
                                <button
                                    @click.stop="copySessionId(session.id)"
                                    class="action-btn"
                                    title="复制会话ID"
                                >
                                    <CopyIcon :size="12" />
                                </button>
                                <button
                                    @click.stop="$emit('delete', session.id)"
                                    class="action-btn"
                                    title="删除会话"
                                >
                                    <Trash2Icon :size="12" />
                                </button>
                            </div>
                        </div>
                    </template>

                    <!-- 昨天 -->
                    <template v-if="groupedSessions.yesterday.length > 0">
                        <div class="px-3 pt-2 pb-1.5 text-[11px] text-muted-foreground font-semibold uppercase tracking-[0.16em]">
                            昨天
                        </div>
                        <div
                            v-for="session in groupedSessions.yesterday"
                            :key="session.id"
                            class="session-row group"
                        >
                            <button
                                @click="$emit('select', session.id)"
                                class="session-btn"
                                :class="
                                    session.id === currentSessionId
                                        ? 'active'
                                        : ''
                                "
                            >
                                <MessageSquareIcon :size="14" class="flex-shrink-0 opacity-60" />
                                <span class="flex-1 truncate font-medium text-xs">{{ session.title || "新对话" }}</span>
                            </button>
                            <div class="session-actions">
                                <button
                                    @click.stop="copySessionId(session.id)"
                                    class="action-btn"
                                    title="复制会话ID"
                                >
                                    <CopyIcon :size="12" />
                                </button>
                                <button
                                    @click.stop="$emit('delete', session.id)"
                                    class="action-btn"
                                    title="删除会话"
                                >
                                    <Trash2Icon :size="12" />
                                </button>
                            </div>
                        </div>
                    </template>

                    <!-- 更早 -->
                    <template v-if="groupedSessions.earlier.length > 0">
                        <div class="px-3 pt-2 pb-1.5 text-[11px] text-muted-foreground font-semibold uppercase tracking-[0.16em]">
                            更早
                        </div>
                        <div
                            v-for="session in groupedSessions.earlier"
                            :key="session.id"
                            class="session-row group"
                        >
                            <button
                                @click="$emit('select', session.id)"
                                class="session-btn"
                                :class="
                                    session.id === currentSessionId
                                        ? 'active'
                                        : ''
                                "
                            >
                                <MessageSquareIcon :size="14" class="flex-shrink-0 opacity-60" />
                                <span class="flex-1 truncate font-medium text-xs">{{ session.title || "新对话" }}</span>
                            </button>
                            <div class="session-actions">
                                <button
                                    @click.stop="copySessionId(session.id)"
                                    class="action-btn"
                                    title="复制会话ID"
                                >
                                    <CopyIcon :size="12" />
                                </button>
                                <button
                                    @click.stop="$emit('delete', session.id)"
                                    class="action-btn"
                                    title="删除会话"
                                >
                                    <Trash2Icon :size="12" />
                                </button>
                            </div>
                        </div>
                    </template>
                </template>

                <!-- 搜索结果 -->
                <template v-else>
                    <div
                        v-for="session in filteredSessions"
                        :key="session.id"
                        class="session-row group"
                    >
                        <button
                            @click="$emit('select', session.id)"
                            class="session-btn"
                            :class="
                                session.id === currentSessionId
                                    ? 'active'
                                    : ''
                            "
                        >
                            <MessageSquareIcon :size="14" class="flex-shrink-0 opacity-60" />
                            <span class="flex-1 truncate font-medium text-xs">{{ session.title || "新对话" }}</span>
                        </button>
                        <div class="session-actions">
                            <button
                                @click.stop="copySessionId(session.id)"
                                class="action-btn"
                                title="复制会话ID"
                            >
                                <CopyIcon :size="12" />
                            </button>
                            <button
                                @click.stop="$emit('delete', session.id)"
                                class="action-btn"
                                title="删除会话"
                            >
                                <Trash2Icon :size="12" />
                            </button>
                        </div>
                    </div>
                </template>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, computed } from "vue";
import {
    BotIcon,
    PlusIcon,
    PanelLeftCloseIcon,
    MessageSquareIcon,
    Trash2Icon,
    SearchIcon,
    XIcon,
    Copy as CopyIcon,
} from "lucide-vue-next";
import Button from "@/components/ui/Button.vue";
import { useToast } from "@/composables/useToast";

const { toast } = useToast();

const props = defineProps({
    sessions: { type: Array, default: () => [] },
    currentSessionId: { type: String, default: null },
    collapsed: { type: Boolean, default: false },
});

const emit = defineEmits(["new", "select", "delete", "toggle"]);

const searchQuery = ref("");

async function copySessionId(sessionId) {
    if (!sessionId) return;
    try {
        await navigator.clipboard.writeText(sessionId);
        toast("会话ID已复制", "success");
    } catch {
        toast("复制会话ID失败", "error");
    }
}

const filteredSessions = computed(() => {
    if (!searchQuery.value.trim()) {
        return props.sessions;
    }
    const query = searchQuery.value.toLowerCase();
    return props.sessions.filter(
        (session) =>
            session.title?.toLowerCase().includes(query) ||
            session.id?.toLowerCase().includes(query)
    );
});

const groupedSessions = computed(() => {
    const groups = {
        today: [],
        yesterday: [],
        earlier: [],
    };
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);

    filteredSessions.value.forEach((session) => {
        const sessionDate = new Date(session.created_at || session.updated_at || now);
        if (sessionDate >= today) {
            groups.today.push(session);
        } else if (sessionDate >= yesterday) {
            groups.yesterday.push(session);
        } else {
            groups.earlier.push(session);
        }
    });

    return groups;
});

function clearSearch() {
    searchQuery.value = "";
}

function handleNewChat() {
    emit("new");
    searchQuery.value = "";
}
</script>

<style scoped>
.chat-sidebar {
    flex-shrink: 0;
    overflow: hidden;
}

.session-row {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 0 8px;
}

.session-content {
    border-top: 1px solid hsl(var(--border));
    border-left: 1px solid hsl(var(--border));
    border-bottom: 1px solid hsl(var(--border));
}

.session-btn {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    border-radius: var(--radius);
    border: none;
    background: transparent;
    color: var(--color-text-muted);
    cursor: pointer;
    transition: all 0.12s;
    text-align: left;
    min-width: 0;
}

.session-btn:hover {
    background: var(--color-bg-hover);
    color: var(--color-text-primary);
}

.session-btn.active {
    background: var(--color-primary/0.1);
    color: var(--color-primary);
}

.session-actions {
    display: flex;
    gap: 2px;
    opacity: 0;
    transition: opacity 0.12s;
}

.group:hover .session-actions {
    opacity: 1;
}

.action-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    padding: 0;
    border: none;
    border-radius: var(--radius);
    background: transparent;
    color: var(--color-text-muted);
    cursor: pointer;
    transition: all 0.12s;
}

.action-btn:hover {
    background: var(--color-bg-hover);
    color: var(--color-text-primary);
}

.action-btn:last-child:hover {
    color: var(--color-destructive);
}
</style>
