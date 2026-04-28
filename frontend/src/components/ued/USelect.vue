<template>
  <UFormField :label="label" :hint="hint" :error="error" :required="required" :hide-label="hideLabel">
    <div ref="rootRef" class="ued-select" :class="{ 'is-open': open, 'is-disabled': disabled, 'is-error': error }">
      <button
        type="button"
        class="ued-select__control"
        :disabled="disabled"
        :aria-expanded="open"
        :aria-controls="listboxId"
        aria-haspopup="listbox"
        @click="toggle"
        @keydown="handleControlKeydown"
      >
        <span class="ued-select__value" :class="{ 'is-placeholder': !selectedOption }">
          {{ selectedOption?.label || placeholder }}
        </span>
        <span class="ued-select__arrow" aria-hidden="true">
          <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6" /></svg>
        </span>
      </button>
    </div>
    <Teleport to="body">
      <div
        v-if="open"
        :id="listboxId"
        ref="menuRef"
        class="ued-select__menu"
        :class="[`is-${placement}`]"
        :style="menuStyle"
        role="listbox"
        tabindex="-1"
      >
        <button
          v-for="(option, index) in normalizedOptions"
          :key="option.value"
          type="button"
          class="ued-select__option"
          :class="{ 'is-selected': option.value === modelValue, 'is-active': index === activeIndex }"
          role="option"
          :aria-selected="option.value === modelValue"
          @mouseenter="activeIndex = index"
          @click="choose(option)"
        >
          <span>{{ option.label }}</span>
          <span v-if="option.value === modelValue" class="ued-select__check" aria-hidden="true">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg>
          </span>
        </button>
        <div v-if="normalizedOptions.length === 0" class="ued-select__empty">
          暂无选项
        </div>
      </div>
    </Teleport>
  </UFormField>
</template>

<script setup>
import { Teleport, computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import UFormField from "./UFormField.vue";

const emit = defineEmits(["update:modelValue", "change"]);

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: "",
  },
  label: {
    type: String,
    default: "",
  },
  placeholder: {
    type: String,
    default: "请选择",
  },
  hint: {
    type: String,
    default: "",
  },
  error: {
    type: String,
    default: "",
  },
  required: {
    type: Boolean,
    default: false,
  },
  hideLabel: {
    type: Boolean,
    default: false,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  options: {
    type: Array,
    default: () => [],
  },
});

const rootRef = ref(null);
const menuRef = ref(null);
const open = ref(false);
const activeIndex = ref(-1);
const placement = ref("bottom");
const menuPosition = ref({
  top: 0,
  left: 0,
  width: 0,
  maxHeight: 224,
});
const listboxId = `ued-select-${Math.random().toString(36).slice(2, 9)}`;
const VIEWPORT_MARGIN = 12;
const MENU_OFFSET = 6;
const DEFAULT_MENU_HEIGHT = 224;

const normalizedOptions = computed(() =>
  props.options.map((option) => {
    if (typeof option === "string" || typeof option === "number") {
      return { label: String(option), value: option };
    }
    return {
      label: option.label ?? String(option.value ?? ""),
      value: option.value ?? "",
    };
  }),
);

const selectedOption = computed(() => normalizedOptions.value.find((option) => option.value === props.modelValue));

const menuStyle = computed(() => ({
  top: `${menuPosition.value.top}px`,
  left: `${menuPosition.value.left}px`,
  width: `${menuPosition.value.width}px`,
  maxHeight: `${menuPosition.value.maxHeight}px`,
}));

watch(
  open,
  async (value) => {
    if (!value) {
      return;
    }

    const selectedIndex = normalizedOptions.value.findIndex((option) => option.value === props.modelValue);
    activeIndex.value = selectedIndex >= 0 ? selectedIndex : 0;

    await nextTick();
    updateMenuPosition();
    scrollActiveOptionIntoView();
  },
  { flush: "post" },
);

watch(
  () => props.options,
  async () => {
    if (!open.value) {
      return;
    }
    await nextTick();
    updateMenuPosition();
    scrollActiveOptionIntoView();
  },
  { deep: true },
);

watch(
  () => props.modelValue,
  async () => {
    if (!open.value) {
      return;
    }
    await nextTick();
    updateMenuPosition();
    scrollActiveOptionIntoView();
  },
);

function toggle() {
  if (props.disabled) {
    return;
  }
  open.value = !open.value;
}

function close() {
  open.value = false;
}

function choose(option) {
  emit("update:modelValue", option.value);
  emit("change", option.value);
  close();
}

function moveActive(step) {
  if (!open.value) {
    open.value = true;
    return;
  }
  const count = normalizedOptions.value.length;
  if (count === 0) {
    activeIndex.value = -1;
    return;
  }
  activeIndex.value = (activeIndex.value + step + count) % count;
  nextTick(() => {
    scrollActiveOptionIntoView();
  });
}

function handleControlKeydown(event) {
  switch (event.key) {
    case "ArrowDown":
      event.preventDefault();
      moveActive(1);
      break;
    case "ArrowUp":
      event.preventDefault();
      moveActive(-1);
      break;
    case "Enter":
    case " ":
      event.preventDefault();
      if (open.value && activeIndex.value >= 0) {
        choose(normalizedOptions.value[activeIndex.value]);
      } else {
        open.value = true;
      }
      break;
    case "Escape":
      close();
      break;
    default:
      break;
  }
}

function updateMenuPosition() {
  const trigger = rootRef.value;
  if (!trigger) {
    return;
  }

  const triggerRect = trigger.getBoundingClientRect();
  const menuRect = menuRef.value?.getBoundingClientRect();
  const menuHeight = menuRect?.height || DEFAULT_MENU_HEIGHT;
  const availableBelow = window.innerHeight - triggerRect.bottom - VIEWPORT_MARGIN;
  const availableAbove = triggerRect.top - VIEWPORT_MARGIN;
  const shouldOpenTop = availableBelow < Math.min(menuHeight, DEFAULT_MENU_HEIGHT) && availableAbove > availableBelow;

  placement.value = shouldOpenTop ? "top" : "bottom";

  const maxHeight = Math.max(
    120,
    Math.min(
      DEFAULT_MENU_HEIGHT,
      shouldOpenTop ? availableAbove - MENU_OFFSET : availableBelow - MENU_OFFSET,
    ),
  );

  const measuredHeight = Math.min(menuHeight, maxHeight);
  const top = shouldOpenTop
    ? Math.max(VIEWPORT_MARGIN, triggerRect.top - measuredHeight - MENU_OFFSET)
    : Math.min(window.innerHeight - VIEWPORT_MARGIN - measuredHeight, triggerRect.bottom + MENU_OFFSET);

  const width = triggerRect.width;
  const left = Math.min(
    Math.max(VIEWPORT_MARGIN, triggerRect.left),
    window.innerWidth - VIEWPORT_MARGIN - width,
  );

  menuPosition.value = {
    top,
    left,
    width,
    maxHeight,
  };
}

function scrollActiveOptionIntoView() {
  if (!menuRef.value || activeIndex.value < 0) {
    return;
  }

  const option = menuRef.value.querySelectorAll(".ued-select__option")[activeIndex.value];
  option?.scrollIntoView({ block: "nearest" });
}

function handlePointerDown(event) {
  const target = event.target;
  if (rootRef.value?.contains(target) || menuRef.value?.contains(target)) {
    return;
  }
  close();
}

function handleWindowChange() {
  if (!open.value) {
    return;
  }
  updateMenuPosition();
}

onMounted(() => {
  document.addEventListener("pointerdown", handlePointerDown);
  window.addEventListener("resize", handleWindowChange);
  window.addEventListener("scroll", handleWindowChange, true);
});

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", handlePointerDown);
  window.removeEventListener("resize", handleWindowChange);
  window.removeEventListener("scroll", handleWindowChange, true);
});
</script>
