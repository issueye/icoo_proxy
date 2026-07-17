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
        :class="[`is-${placement}`, { 'ued-select__menu--searchable': searchable }]"
        :style="menuStyle"
        role="listbox"
        tabindex="-1"
      >
        <div v-if="searchable" class="ued-select__search">
          <input
            ref="searchInputRef"
            v-model="searchKeyword"
            type="text"
            class="ued-select__search-input"
            placeholder="搜索…"
            @keydown="handleControlKeydown"
          />
        </div>
        <div class="ued-select__options">
          <button
            v-for="(option, index) in filteredOptions"
            :key="option.value"
            type="button"
            class="ued-select__option"
            :class="{ 'is-selected': option.value === modelValue, 'is-active': visibleOptions[index] === activeOption }"
            role="option"
            :aria-selected="option.value === modelValue"
            @mouseenter="activeOption = visibleOptions[index]"
            @click="choose(option)"
          >
            <span>{{ option.label }}</span>
            <span v-if="option.value === modelValue" class="ued-select__check" aria-hidden="true">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5" /></svg>
            </span>
          </button>
          <div v-if="filteredOptions.length === 0" class="ued-select__empty">
            {{ searchKeyword ? "无匹配项" : "暂无选项" }}
          </div>
        </div>
      </div>
    </Teleport>
  </UFormField>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import UFormField from "./UFormField.vue";

const emit = defineEmits(["update:modelValue", "change", "search"]);

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
  searchable: {
    type: Boolean,
    default: false,
  },
});

const rootRef = ref(null);
const menuRef = ref(null);
const searchInputRef = ref(null);
const open = ref(false);
const activeOption = ref(null);
const searchKeyword = ref("");
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
const SEARCH_ROW_HEIGHT = 40;

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

const filteredOptions = computed(() => {
  const keyword = searchKeyword.value.trim().toLowerCase();
  if (!keyword) {
    return normalizedOptions.value;
  }
  return normalizedOptions.value.filter((option) =>
    String(option.label).toLowerCase().includes(keyword),
  );
});

// Track the actual visible options so keyboard navigation stays in sync
// after filtering.
const visibleOptions = computed(() => filteredOptions.value);

const menuStyle = computed(() => ({
  top: `${menuPosition.value.top}px`,
  left: `${menuPosition.value.left}px`,
  width: `${menuPosition.value.width}px`,
  maxHeight: `${menuPosition.value.maxHeight}px`,
}));

function resetActive() {
  const selectedIndex = visibleOptions.value.findIndex((option) => option.value === props.modelValue);
  activeOption.value = selectedIndex >= 0
    ? visibleOptions.value[selectedIndex]
    : visibleOptions.value[0] ?? null;
}

watch(
  open,
  async (value) => {
    if (!value) {
      return;
    }

    searchKeyword.value = "";
    resetActive();

    await refreshMenuPosition();
    scrollActiveOptionIntoView();

    if (props.searchable) {
      await nextTick();
      searchInputRef.value?.focus();
    }
  },
  { flush: "post" },
);

watch(searchKeyword, async () => {
  if (!open.value) return;
  resetActive();
  await refreshMenuPosition();
  scrollActiveOptionIntoView();
  emit("search", searchKeyword.value);
});

watch(
  () => props.options,
  async () => {
    if (!open.value) {
      return;
    }
    await refreshMenuPosition();
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
    await refreshMenuPosition();
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
  const count = visibleOptions.value.length;
  if (count === 0) {
    activeOption.value = null;
    return;
  }
  let index = visibleOptions.value.findIndex((option) => option === activeOption.value);
  if (index < 0) index = 0;
  index = (index + step + count) % count;
  activeOption.value = visibleOptions.value[index];
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
      if (open.value && activeOption.value) {
        choose(activeOption.value);
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
    maxHeight: props.searchable ? maxHeight - SEARCH_ROW_HEIGHT : maxHeight,
  };
}

async function refreshMenuPosition() {
  await nextTick();
  if (!open.value) return;
  updateMenuPosition();
  await nextTick();
  if (!open.value) return;
  updateMenuPosition();
}

function scrollActiveOptionIntoView() {
  if (!menuRef.value || !activeOption.value) {
    return;
  }

  const index = visibleOptions.value.findIndex((option) => option === activeOption.value);
  if (index < 0) return;

  const option = menuRef.value.querySelectorAll(".ued-select__option")[index];
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

<style scoped>
.ued-select__search {
  position: sticky;
  top: 0;
  z-index: 1;
  padding: var(--ued-space-2, 3px);
  border-bottom: 1px solid var(--ued-color-divider);
  background: var(--ued-color-bg-card);
}

.ued-select__search-input {
  width: 100%;
  height: var(--ued-size-sm);
  padding: 0 6px;
  border: 1px solid var(--ued-color-input);
  border-radius: var(--ued-radius-md);
  background: var(--ued-color-bg-card);
  color: var(--ued-color-text);
  font-size: var(--ued-font-size-sm);
  outline: none;
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.ued-select__search-input:hover {
  border-color: var(--ued-color-text-muted);
}

.ued-select__search-input:focus {
  border-color: var(--ued-color-primary);
  box-shadow: 0 0 0 2px var(--ued-color-focus-ring-soft);
}

.ued-select__options {
  overflow-y: auto;
}
</style>

