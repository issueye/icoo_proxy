<template>
  <Teleport to="body">
    <div
      class="toast-stack fixed top-3 right-3 z-[200] flex flex-col gap-2 pointer-events-none"
    >
      <TransitionGroup name="toast-slide">
        <div
          v-for="item in toasts"
          :key="item.id"
          :class="[
            'toast-card pointer-events-auto flex items-center gap-3 min-w-[260px] max-w-[380px]',
            typeClasses[item.type],
          ]"
        >
          <div :class="['toast-icon', iconColor[item.type]]">
            <CheckCircle v-if="item.type === 'success'" :size="18" />
            <XCircle v-else-if="item.type === 'error'" :size="18" />
            <AlertTriangle v-else-if="item.type === 'warning'" :size="18" />
            <Info v-else :size="18" />
          </div>

          <span class="toast-message">{{
            item.message
          }}</span>

          <button
            @click="removeToast(item.id)"
            class="toast-close"
          >
            <X :size="14" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup>
import { CheckCircle, XCircle, AlertTriangle, Info, X } from "lucide-vue-next";
import { useToast } from "@/composables/useToast.js";

const { toasts, removeToast } = useToast();

const typeClasses = {
  success: "bg-green-500/8 border-green-500/20",
  error: "bg-red-500/8 border-red-500/20",
  warning: "bg-amber-500/8 border-amber-500/20",
  info: "bg-accent/8 border-accent/20",
};

const iconColor = {
  success: "text-green-500",
  error: "text-red-500",
  warning: "text-amber-500",
  info: "text-accent",
};
</script>

<style scoped>
.toast-card {
  padding: 10px 12px;
  border-radius: var(--radius-md);
  border-width: 1px;
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(10px);
}

.toast-icon {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
}

.toast-message {
  flex: 1;
  font-size: 0.8125rem;
  line-height: 1.45;
  color: hsl(var(--foreground));
}

.toast-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  padding: 2px;
  border: 0;
  background: transparent;
  color: hsl(var(--muted-foreground));
  transition: color 0.16s ease;
}

.toast-close:hover {
  color: hsl(var(--foreground));
}

.toast-slide-enter-active {
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.toast-slide-leave-active {
  transition: all 0.2s ease-in;
}
.toast-slide-enter-from {
  opacity: 0;
  transform: translateX(100%);
}
.toast-slide-leave-to {
  opacity: 0;
  transform: translateX(100%);
}
</style>

