<script setup>
import { computed, nextTick, ref, watch } from "vue";
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
  closeOnEsc: {
    type: Boolean,
    default: true,
  },
  persistent: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["close", "update:visible"]);
const surfaceRef = ref(null);

const drawerStyle = computed(() => ({
  "--ued-drawer-width": props.width,
}));

function handleClose() {
  emit("close");
  emit("update:visible", false);
}

function handleScrimClick() {
  if (!props.persistent && props.closeOnScrim) {
    handleClose();
  }
}

function handleKeydown(event) {
  if (!props.persistent && props.closeOnEsc && event.key === "Escape") {
    handleClose();
  }
}

watch(
  () => props.visible,
  async (visible) => {
    if (!visible) {
      return;
    }

    await nextTick();
    surfaceRef.value?.focus();
  },
);
</script>

<template>
  <Teleport to="body">
    <Transition name="ued-drawer-fade">
      <div v-if="visible" class="ued-drawer-layer" @keydown="handleKeydown">
        <div class="ued-drawer__scrim" @click="handleScrimClick"></div>

        <Transition name="ued-drawer-slide">
          <aside
            v-if="visible"
            ref="surfaceRef"
            class="ued-drawer"
            :style="drawerStyle"
            role="dialog"
            aria-modal="true"
            :aria-label="title || '抽屉'"
            tabindex="-1"
            @click.stop
          >
            <header class="ued-drawer__header">
              <div class="ued-drawer__heading">
                <div v-if="kicker" class="ued-kicker">{{ kicker }}</div>
                <h3 v-if="title" class="ued-drawer__title">{{ title }}</h3>
                <p v-if="description" class="ued-drawer__description">{{ description }}</p>
              </div>

              <button
                class="ued-icon-btn ued-drawer__close"
                type="button"
                aria-label="关闭抽屉"
                @click="handleClose"
              >
                <X :size="16" />
              </button>
            </header>

            <div v-if="$slots.summary" class="ued-drawer__summary">
              <slot name="summary" />
            </div>

            <div class="ued-drawer__body">
              <slot />
            </div>

            <footer v-if="$slots.footer" class="ued-drawer__footer">
              <slot name="footer" />
            </footer>
          </aside>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>