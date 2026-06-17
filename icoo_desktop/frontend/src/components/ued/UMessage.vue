<template>
  <Teleport to="body">
    <TransitionGroup name="ued-message" tag="div" class="ued-message" role="status" aria-live="polite">
      <div
        v-for="item in messageItems"
        :key="item.id"
        class="ued-message__notice"
        :class="`ued-message__notice--${item.type}`"
      >
        <span class="ued-message__icon" aria-hidden="true">
          <span v-if="item.type === 'loading'" class="ued-message__spinner" />
          <template v-else>{{ iconMap[item.type] }}</template>
        </span>
        <span class="ued-message__content">{{ item.content }}</span>
        <button
          v-if="item.closable"
          type="button"
          class="ued-message__close"
          aria-label="关闭消息"
          @click="closeMessage(item.id)"
        >
          ×
        </button>
      </div>
    </TransitionGroup>
  </Teleport>
</template>

<script setup>
import { closeMessage, messageItems } from "./message";

const iconMap = {
  success: "✓",
  info: "i",
  warning: "!",
  error: "×",
};
</script>

<style scoped>
.ued-message__spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: var(--ued-radius-pill);
  animation: ued-message-spin 0.8s linear infinite;
}

@keyframes ued-message-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
