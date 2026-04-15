<template>
  <Transition name="confirm-fade">
    <div
      v-if="visible"
      class="fixed inset-0 z-[100] flex items-center justify-center"
    >
      <!-- 遮罩 -->
      <div
        class="absolute inset-0 bg-black/60 backdrop-blur-sm"
        @click="handleCancel"
      />

      <!-- 弹窗 -->
      <Transition name="confirm-scale">
        <div
          v-if="visible"
          class="confirm-shell relative w-full max-w-sm mx-4 overflow-hidden"
        >
          <div class="confirm-body">
            <div
              :class="[
                'confirm-icon',
                iconClasses[state.type],
              ]"
            >
              <AlertTriangle
                v-if="state.type === 'danger' || state.type === 'warning'"
                :size="24"
              />
              <Info v-else :size="24" />
            </div>

            <h3 class="confirm-title">
              {{ state.title }}
            </h3>
            <p class="confirm-message">
              {{ state.message }}
            </p>
          </div>

          <div class="confirm-actions">
            <button
              @click="handleCancel"
              class="btn btn-secondary flex-1"
            >
              {{ state.cancelText }}
            </button>
            <button
              @click="handleConfirm"
              :class="[
                'btn flex-1',
                btnClasses[state.type],
              ]"
            >
              {{ state.confirmText }}
            </button>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<script setup>
import { AlertTriangle, Info } from "lucide-vue-next";
import { useConfirm } from "@/composables/useConfirm.js";

const { visible, state, handleConfirm, handleCancel } = useConfirm();

const iconClasses = {
  default: "bg-accent/10 text-accent",
  danger: "bg-red-500/10 text-red-500",
  warning: "bg-amber-500/10 text-amber-500",
};

const btnClasses = {
  default: "btn-primary",
  danger: "btn-danger",
  warning: "bg-amber-500 hover:bg-amber-600 text-white border border-amber-600",
};
</script>

<style scoped>
.confirm-shell {
  border: 1px solid color-mix(in srgb, var(--color-border) 90%, transparent);
  border-radius: var(--radius-lg);
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--color-accent) 3%, transparent), transparent 72px),
    color-mix(in srgb, var(--color-bg-secondary) 97%, white);
  box-shadow: var(--shadow-lg);
}

.confirm-body {
  padding: 18px 18px 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.confirm-icon {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 12px;
  border: 1px solid currentColor;
  border-color: color-mix(in srgb, currentColor 22%, transparent);
}

.confirm-title {
  margin: 0 0 6px;
  font-size: 0.95rem;
  font-weight: 700;
  color: hsl(var(--foreground));
}

.confirm-message {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: hsl(var(--muted-foreground));
}

.confirm-actions {
  display: flex;
  gap: 10px;
  padding: 14px 18px 18px;
}

.confirm-fade-enter-active,
.confirm-fade-leave-active {
  transition: opacity 0.2s ease;
}
.confirm-fade-enter-from,
.confirm-fade-leave-to {
  opacity: 0;
}
.confirm-scale-enter-active {
  transition: all 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.confirm-scale-leave-active {
  transition: all 0.15s ease-in;
}
.confirm-scale-enter-from {
  opacity: 0;
  transform: scale(0.9);
}
.confirm-scale-leave-to {
  opacity: 0;
  transform: scale(0.95);
}
</style>

