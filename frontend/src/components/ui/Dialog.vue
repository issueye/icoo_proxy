<script setup>
import {
  DialogRoot,
  DialogTrigger,
  DialogPortal,
  DialogOverlay,
  DialogContent as DialogContentPrimitive,
  DialogClose,
} from "radix-vue"
import { cn } from "@/lib/utils"
import { X } from "lucide-vue-next"

const props = defineProps({
  open: {
    type: Boolean,
    default: undefined,
  },
  defaultOpen: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(["update:open"])

const onUpdateOpen = (value) => {
  emit("update:open", value)
}
</script>

<template>
  <DialogRoot
    :open="open"
    :default-open="defaultOpen"
    @update:open="onUpdateOpen"
  >
    <DialogTrigger as-child>
      <slot name="trigger" />
    </DialogTrigger>
    <DialogPortal>
      <DialogOverlay
        class="dialog-overlay"
      />
      <DialogContentPrimitive
        class="dialog-content"
      >
        <slot />
        <DialogClose
          class="dialog-close"
        >
          <X class="h-4 w-4" />
          <span class="sr-only">Close</span>
        </DialogClose>
      </DialogContentPrimitive>
    </DialogPortal>
  </DialogRoot>
</template>

<style scoped>
.dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  background: rgba(15, 23, 42, 0.2);
  backdrop-filter: blur(2px);
}

.dialog-content {
  position: fixed;
  left: 50%;
  top: 50%;
  z-index: 50;
  width: min(640px, calc(100vw - 32px));
  max-height: 80vh;
  transform: translate(-50%, -50%);
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-dialog);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-dialog);
  overflow: hidden;
}

.dialog-close {
  position: absolute;
  top: 12px;
  right: 12px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;
}

.dialog-close:hover {
  background: var(--ui-bg-surface-muted);
  border-color: var(--ui-border-default);
  color: var(--color-text-primary);
}

.dialog-close:focus-visible {
  outline: none;
  box-shadow: var(--focus-ring);
}
</style>
