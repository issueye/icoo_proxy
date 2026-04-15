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
        <div class="floating-drawer__heading">
          <div v-if="kicker" class="settings-kicker">{{ kicker }}</div>
          <h3 class="floating-drawer__title">{{ title }}</h3>
          <p v-if="description" class="panel-description">{{ description }}</p>
        </div>
        <button class="icon-btn" title="关闭抽屉" @click="handleClose">
          <X :size="14" />
        </button>
      </div>

      <div class="floating-drawer__body">
        <div v-if="$slots.summary" class="floating-drawer__summary">
          <slot name="summary" />
        </div>
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
  top: 4px;
  right: 4px;
  bottom: 0;
  z-index: 40;
  height: calc(100% - 8px);
  width: min(var(--drawer-width), 100vw);
  display: flex;
  flex-direction: column;
  gap: 0;
  border: 1px solid var(--ui-border-default);
  border-right: none;
  border-bottom: none;
  border-radius: var(--radius-drawer);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-dialog);
  overflow: hidden;
}

.floating-drawer__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-12);
  padding: var(--drawer-header-padding);
  border-bottom: 1px solid var(--ui-border-default);
  background: var(--ui-bg-surface);
  flex-shrink: 0;
}

.floating-drawer__heading {
  display: grid;
  gap: var(--space-4);
  min-width: 0;
}

.floating-drawer__title {
  margin: 0;
  font-size: 16px;
  line-height: 1.4;
  font-weight: 600;
  color: var(--color-text-primary);
}

.floating-drawer__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--section-gap);
  padding: var(--drawer-body-padding);
  background: var(--ui-bg-surface);
}

.floating-drawer__summary {
  display: grid;
  gap: var(--space-10);
  padding-bottom: var(--space-12);
  border-bottom: 1px solid var(--ui-border-subtle);
}

.floating-drawer__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--control-gap);
  padding: var(--drawer-footer-padding);
  border-top: 1px solid var(--ui-border-default);
  background: var(--ui-bg-surface);
  flex-shrink: 0;
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
    width: min(var(--drawer-width), 100vw);
  }
}

@media (max-width: 720px) {
  .floating-drawer {
    top: 48px;
    left: 0;
    width: 100vw;
  }

  .floating-drawer__footer {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
