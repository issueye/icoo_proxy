<template>
  <div
    class="chat-header flex items-center justify-between px-2 py-2.5 border-b border-border flex-shrink-0 bg-muted"
  >
    <div class="flex items-center gap-3">
      <button
        v-if="sidebarCollapsed"
        @click="$emit('toggle-sidebar')"
        class="text-muted-foreground hover:text-foreground hover:bg-accent hover:text-accent-foreground rounded-md p-1.5 transition-colors"
      >
        <PanelLeftOpenIcon :size="16" />
      </button>

      <div v-if="sidebarCollapsed" class="flex items-center gap-2">
        <div
          class="w-7 h-7 rounded-sm bg-primary flex items-center justify-center flex-shrink-0"
        >
          <BotIcon :size="14" class="text-primary-foreground" />
        </div>
      </div>

      <div class="min-w-0">
        <h1 class="text-base font-semibold text-foreground truncate max-w-[320px]">
          {{ title || "新对话" }}
        </h1>
      </div>
    </div>

    <div class="flex items-center gap-2">
      <slot name="actions"></slot>
    </div>
  </div>
</template>

<script setup>
import { computed } from "vue";
import {
  BotIcon,
  CopyIcon,
  PanelLeftOpenIcon,
} from "lucide-vue-next";
import { useToast } from "@/composables/useToast";

const props = defineProps({
  title: { type: String, default: "" },
  sessionId: { type: String, default: "" },
  sidebarCollapsed: { type: Boolean, default: false },
});

defineEmits(["toggle-sidebar"]);

const { toast } = useToast();

const shortSessionId = computed(() => {
  if (!props.sessionId) return "";
  return props.sessionId.length > 12
    ? `${props.sessionId.slice(0, 6)}...${props.sessionId.slice(-4)}`
    : props.sessionId;
});

async function copySessionId() {
  if (!props.sessionId) return;

  try {
    await navigator.clipboard.writeText(props.sessionId);
    toast("会话ID已复制", "success");
  } catch {
    toast("复制会话ID失败", "error");
  }
}
</script>

<style scoped>
.chat-header {
  background: hsl(var(--muted));
}

.session-id-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
  max-width: 180px;
  padding: 0.2rem 0.45rem;
  border-radius: var(--radius);
  border: 1px solid hsl(var(--border));
  background: hsl(var(--background));
  color: hsl(var(--muted-foreground));
  font-size: 0.7rem;
  line-height: 1;
  transition: all 0.18s ease;
}

.session-id-chip:hover {
  color: hsl(var(--foreground));
  background: hsl(var(--accent));
  border-color: hsl(var(--primary) / 0.3);
}

@media (max-width: 900px) {
  .chat-header {
    padding: 1rem;
  }
}
</style>
