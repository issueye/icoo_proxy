<script setup>
import { X } from "lucide-vue-next";

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  title: {
    type: String,
    default: "",
  },
  description: {
    type: String,
    default: "",
  },
  kicker: {
    type: String,
    default: "",
  },
  width: {
    type: String,
    default: "420px",
  },
  closeOnScrim: {
    type: Boolean,
    default: true,
  },
});

const emit = defineEmits(["close", "update:visible"]);

function handleClose() {
  emit("close");
  emit("update:visible", false);
}

function handleScrimClick() {
  if (props.closeOnScrim) {
    handleClose();
  }
}
</script>

<template>
  <Transition name="drawer-fade">
    <div v-if="visible" class="floating-drawer__scrim" @click="handleScrimClick"></div>
  </Transition>

  <Transition name="drawer-slide">
    <aside
      v-if="visible"
      class="floating-drawer"
      :style="{ '--drawer-width': width }"
    >
      <div class="floating-drawer__header">
        <div>
          <div v-if="kicker" class="settings-kicker">{{ kicker }}</div>
          <h3 class="floating-drawer__title">{{ title }}</h3>
          <p v-if="description" class="panel-description">{{ description }}</p>
        </div>
        <button class="icon-btn" title="关闭抽屉" @click="handleClose">
          <X :size="14" />
        </button>
      </div>

      <div v-if="$slots.summary" class="floating-drawer__summary">
        <slot name="summary" />
      </div>

      <div class="floating-drawer__body">
        <slot />
      </div>

      <div v-if="$slots.footer" class="floating-drawer__footer">
        <slot name="footer" />
      </div>
    </aside>
  </Transition>
</template>

<style scoped>
.floating-drawer__scrim {
  position: fixed;
  inset: 0;
  z-index: 30;
  background: rgba(15, 23, 42, 0.14);
}

.floating-drawer {
  position: fixed;
  top: 56px;
  right: 16px;
  bottom: 16px;
  z-index: 40;
  width: min(var(--drawer-width), calc(100vw - 32px));
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-dialog);
  overflow: hidden;
}

.floating-drawer__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.floating-drawer__title {
  margin: 0;
  font-size: 20px;
  line-height: 1.25;
  color: var(--color-text-primary);
}

.floating-drawer__summary,
.floating-drawer__body {
  padding: 14px;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-md);
  background: var(--ui-bg-surface-muted);
}

.floating-drawer__body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.floating-drawer__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.drawer-fade-enter-active,
.drawer-fade-leave-active {
  transition: opacity 0.16s ease;
}

.drawer-fade-enter-from,
.drawer-fade-leave-to {
  opacity: 0;
}

.drawer-slide-enter-active,
.drawer-slide-leave-active {
  transition: transform 0.18s ease, opacity 0.18s ease;
}

.drawer-slide-enter-from,
.drawer-slide-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

@media (max-width: 1180px) {
  .floating-drawer {
    top: 52px;
    right: 12px;
    bottom: 12px;
    width: min(var(--drawer-width), calc(100vw - 24px));
  }
}

@media (max-width: 720px) {
  .floating-drawer__footer {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
