<template>
  <div
    ref="triggerRef"
    class="ued-tooltip"
    @mouseenter="handleMouseEnter"
    @mouseleave="handleMouseLeave"
    @focus="handleMouseEnter"
    @blur="handleMouseLeave"
  >
    <slot />
    <Teleport to="body">
      <Transition name="ued-tooltip">
        <div
          v-if="visible"
          ref="tooltipRef"
          class="ued-tooltip__popup"
          :class="[`ued-tooltip__popup--${actualPlacement}`]"
          :style="popupStyle"
        >
          <div class="ued-tooltip__arrow"></div>
          <div class="ued-tooltip__inner">
            <slot name="content">{{ content }}</slot>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";

const props = defineProps({
  content: { type: String, default: "" },
  placement: { type: String, default: "top" },
  disabled: { type: Boolean, default: false },
  delay: { type: Number, default: 200 },
});

const SPACING = 8;

const triggerRef = ref(null);
const tooltipRef = ref(null);
const visible = ref(false);
const actualPlacement = ref(props.placement);
const popupOffset = ref({ top: 0, left: 0 });
let enterTimer = null;

const popupStyle = computed(() => ({
  top: popupOffset.value.top + "px",
  left: popupOffset.value.left + "px",
}));

function handleMouseEnter() {
  if (props.disabled) return;
  clearTimeout(enterTimer);
  enterTimer = setTimeout(() => {
    visible.value = true;
  }, props.delay);
}

function handleMouseLeave() {
  clearTimeout(enterTimer);
  visible.value = false;
}

watch(visible, async (val) => {
  if (val) {
    window.addEventListener("scroll", updatePosition, true);
    window.addEventListener("resize", updatePosition);
    await nextTick();
    updatePosition();
  } else {
    window.removeEventListener("scroll", updatePosition, true);
    window.removeEventListener("resize", updatePosition);
  }
});

onBeforeUnmount(() => {
  clearTimeout(enterTimer);
  window.removeEventListener("scroll", updatePosition, true);
  window.removeEventListener("resize", updatePosition);
});

function updatePosition() {
  const trigger = triggerRef.value;
  const popup = tooltipRef.value;
  if (!trigger || !popup) return;

  // Use viewport coordinates + position:fixed so the popup tracks the trigger
  // correctly even when the page or an ancestor scrolls.
  const triggerRect = trigger.getBoundingClientRect();
  const popupRect = popup.getBoundingClientRect();

  const fitsAbove = triggerRect.top >= popupRect.height + SPACING;
  const fitsBelow = window.innerHeight - triggerRect.bottom >= popupRect.height + SPACING;

  let placement = props.placement;
  if (placement === "top" && !fitsAbove && fitsBelow) placement = "bottom";
  if (placement === "bottom" && !fitsBelow && fitsAbove) placement = "top";
  actualPlacement.value = placement;

  let top = 0;
  let left = 0;

  switch (placement) {
    case "top":
      top = triggerRect.top - popupRect.height - SPACING;
      left = triggerRect.left + (triggerRect.width - popupRect.width) / 2;
      break;
    case "bottom":
      top = triggerRect.bottom + SPACING;
      left = triggerRect.left + (triggerRect.width - popupRect.width) / 2;
      break;
    case "left":
      top = triggerRect.top + (triggerRect.height - popupRect.height) / 2;
      left = triggerRect.left - popupRect.width - SPACING;
      break;
    case "right":
      top = triggerRect.top + (triggerRect.height - popupRect.height) / 2;
      left = triggerRect.right + SPACING;
      break;
  }

  // Clamp into viewport so the popup never overflows the window edges.
  left = Math.max(SPACING, Math.min(left, window.innerWidth - popupRect.width - SPACING));
  top = Math.max(SPACING, Math.min(top, window.innerHeight - popupRect.height - SPACING));

  popupOffset.value = { top, left };
}
</script>

<style scoped>
.ued-tooltip {
  position: relative;
  display: inline-flex;
  max-width: 100%;
}

/* Fixed positioning pairs with viewport coords from getBoundingClientRect,
   keeping the popup aligned to the trigger through scroll/resize. */
.ued-tooltip__popup {
  position: fixed;
  z-index: 100;
  pointer-events: none;
}

.ued-tooltip__inner {
  max-width: 320px;
  padding: 6px 10px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: var(--ued-radius-md);
  background: var(--ued-color-tooltip-bg);
  color: var(--ued-color-tooltip-fg);
  font-size: var(--ued-font-size-sm);
  line-height: 1.5;
  word-break: break-word;
  box-shadow: var(--ued-shadow-popup);
}

.ued-tooltip__arrow {
  position: absolute;
  width: 6px;
  height: 6px;
  background: var(--ued-color-tooltip-bg);
  transform: rotate(45deg);
}

.ued-tooltip__popup--top .ued-tooltip__arrow {
  bottom: -3px;
  left: 50%;
  margin-left: -3px;
}

.ued-tooltip__popup--bottom .ued-tooltip__arrow {
  top: -3px;
  left: 50%;
  margin-left: -3px;
}

.ued-tooltip__popup--left .ued-tooltip__arrow {
  right: -3px;
  top: 50%;
  margin-top: -3px;
}

.ued-tooltip__popup--right .ued-tooltip__arrow {
  left: -3px;
  top: 50%;
  margin-top: -3px;
}

.ued-tooltip-enter-active,
.ued-tooltip-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.ued-tooltip-enter-from,
.ued-tooltip-leave-to {
  opacity: 0;
  transform: scale(0.96);
}
</style>
