<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick, useAttrs } from "vue"
import { cn } from "@/lib/utils"
import { ChevronDown, Check, X, Search } from "lucide-vue-next"
import Label from "./Label.vue"

defineOptions({
  inheritAttrs: false,
})

const props = defineProps({
  modelValue: {
    type: [String, Number, Boolean, Array],
    default: "",
  },
  options: {
    type: Array,
    default: () => [],
  },
  placeholder: {
    type: String,
    default: "请选择...",
  },
  label: {
    type: String,
    default: "",
  },
  description: {
    type: String,
    default: "",
  },
  error: {
    type: String,
    default: "",
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  clearable: {
    type: Boolean,
    default: false,
  },
  searchable: {
    type: Boolean,
    default: false,
  },
  multiple: {
    type: Boolean,
    default: false,
  },
  size: {
    type: String,
    default: "default",
  },
  optionLabelKey: {
    type: String,
    default: "label",
  },
  optionValueKey: {
    type: String,
    default: "value",
  },
  optionDisabledKey: {
    type: String,
    default: "disabled",
  },
  optionGroupKey: {
    type: String,
    default: "group",
  },
})

const emit = defineEmits(["update:modelValue", "change"])
const attrs = useAttrs()

const isOpen = ref(false)
const searchQuery = ref("")
const selectRef = ref(null)
const triggerRef = ref(null)
const dropdownWidth = ref(0)

const normalizedOptions = computed(() => {
  return props.options.map((option) => {
    if (typeof option === "object") {
      return {
        label: option[props.optionLabelKey] ?? String(option),
        value: option[props.optionValueKey] ?? option,
        disabled: option[props.optionDisabledKey] ?? false,
        group: option[props.optionGroupKey],
      }
    }
    return {
      label: String(option),
      value: option,
      disabled: false,
      group: undefined,
    }
  })
})

const groupedOptions = computed(() => {
  const groups = {}
  normalizedOptions.value.forEach((option) => {
    const group = option.group || "__ungrouped__"
    if (!groups[group]) {
      groups[group] = []
    }
    groups[group].push(option)
  })
  return groups
})

const filteredOptions = computed(() => {
  if (!searchQuery.value) {
    return normalizedOptions.value
  }
  const query = searchQuery.value.toLowerCase()
  return normalizedOptions.value.filter((option) =>
    option.label.toLowerCase().includes(query)
  )
})

const filteredGroupedOptions = computed(() => {
  if (!searchQuery.value) {
    return groupedOptions.value
  }
  const query = searchQuery.value.toLowerCase()
  const groups = {}
  normalizedOptions.value.forEach((option) => {
    if (option.label.toLowerCase().includes(query)) {
      const group = option.group || "__ungrouped__"
      if (!groups[group]) {
        groups[group] = []
      }
      groups[group].push(option)
    }
  })
  return groups
})

const selectedLabel = computed(() => {
  if (props.multiple && Array.isArray(props.modelValue)) {
    if (props.modelValue.length === 0) {
      return props.placeholder
    }
    const labels = props.modelValue
      .map((val) => {
        const option = normalizedOptions.value.find((o) => o.value === val)
        return option?.label
      })
      .filter(Boolean)
    return labels.join(", ")
  }

  const option = normalizedOptions.value.find((o) => o.value === props.modelValue)
  return option?.label || props.placeholder
})

const isSelected = computed(() => {
  if (props.multiple && Array.isArray(props.modelValue)) {
    return props.modelValue.length > 0
  }
  return props.modelValue !== "" && props.modelValue !== null && props.modelValue !== undefined
})

const hasValue = computed(() => {
  if (props.multiple && Array.isArray(props.modelValue)) {
    return props.modelValue.length > 0
  }
  return props.modelValue !== "" && props.modelValue !== null && props.modelValue !== undefined
})

const selectSize = computed(() => {
  switch (props.size) {
    case "sm":
      return "select-trigger--sm"
    case "default":
      return "select-trigger--md"
    case "lg":
      return "select-trigger--lg"
    default:
      return "select-trigger--md"
  }
})

const dropdownStyle = computed(() => {
  const triggerWidth = dropdownWidth.value || 0

  return {
    width: `${Math.max(triggerWidth, 136)}px`,
    maxWidth: "calc(100vw - 24px)",
  }
})

function updateDropdownWidth() {
  dropdownWidth.value = triggerRef.value?.offsetWidth || 0
}

function toggleDropdown() {
  if (!props.disabled) {
    isOpen.value = !isOpen.value
    if (isOpen.value) {
      nextTick(() => {
        updateDropdownWidth()
        if (!props.searchable) {
          return
        }
        const searchInput = selectRef.value?.querySelector('input[type="text"]')
        searchInput?.focus()
      })
    }
  }
}

function selectOption(option) {
  if (option.disabled) return

  if (props.multiple) {
    const currentValue = Array.isArray(props.modelValue) ? [...props.modelValue] : []
    const index = currentValue.indexOf(option.value)
    if (index > -1) {
      currentValue.splice(index, 1)
    } else {
      currentValue.push(option.value)
    }
    emit("update:modelValue", currentValue)
    emit("change", currentValue)
  } else {
    emit("update:modelValue", option.value)
    emit("change", option.value)
    isOpen.value = false
    searchQuery.value = ""
  }
}

function clearSelection() {
  if (props.multiple) {
    emit("update:modelValue", [])
    emit("change", [])
  } else {
    emit("update:modelValue", "")
    emit("change", "")
  }
}

function isSelectedValue(value) {
  if (props.multiple && Array.isArray(props.modelValue)) {
    return props.modelValue.includes(value)
  }
  return props.modelValue === value
}

function handleClickOutside(event) {
  if (selectRef.value && !selectRef.value.contains(event.target)) {
    isOpen.value = false
    searchQuery.value = ""
  }
}

onMounted(() => {
  document.addEventListener("click", handleClickOutside)
  window.addEventListener("resize", updateDropdownWidth)
  updateDropdownWidth()
})

onUnmounted(() => {
  document.removeEventListener("click", handleClickOutside)
  window.removeEventListener("resize", updateDropdownWidth)
})
</script>

<template>
  <div ref="selectRef" class="relative w-full">
    <!-- 标签 -->
    <Label v-if="label" :class="cn('mb-1.5')">
      {{ label }}
    </Label>

    <!-- 触发器 -->
    <div
      ref="triggerRef"
      :class="cn(
        'input-shell relative cursor-pointer',
        {
          'is-error': error,
          'is-disabled cursor-not-allowed': disabled,
        },
        attrs.class
      )"
      @click="toggleDropdown"
    >
      <!-- 选中值显示 -->
      <div
        :class="cn(
          'input-control select-trigger flex items-center gap-1 truncate',
          selectSize,
          { 'text-muted-foreground': !isSelected }
        )"
      >
        <span class="truncate">{{ selectedLabel }}</span>
      </div>

      <!-- 清除按钮 -->
      <button
        v-if="clearable && hasValue && !disabled"
        type="button"
        @click.stop="clearSelection"
        class="input-action mr-1 p-0.5"
        tabindex="-1"
      >
        <X :size="14" />
      </button>

      <!-- 下拉箭头 -->
      <ChevronDown
        :size="14"
        class="input-affix mr-2 transition-transform"
        :class="{ 'rotate-180': isOpen }"
      />
    </div>

    <!-- 描述文字 -->
    <p v-if="description && !error" class="select-meta">
      {{ description }}
    </p>

    <p v-if="error" class="select-meta select-meta--error">
      {{ error }}
    </p>

    <!-- 下拉菜单 -->
    <div
      v-if="isOpen"
      :style="dropdownStyle"
      :class="cn(
        'absolute left-0 top-full z-50 mt-1 origin-top-left',
        'select-dropdown',
        'max-h-60 overflow-y-auto'
      )"
    >
      <div v-if="searchable" class="p-2 border-b border-border">
        <div class="input-shell">
          <Search :size="12" class="input-affix absolute left-2 top-1/2 -translate-y-1/2" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索..."
            class="input-control select-search-input"
            @click.stop
          />
        </div>
      </div>

      <div class="py-1">
        <template v-if="Object.keys(filteredGroupedOptions).length > 1 || ('__ungrouped__' in filteredGroupedOptions && Object.keys(filteredGroupedOptions).length > 1)">
          <template v-for="(options, group) in filteredGroupedOptions" :key="group">
            <div
              v-if="group !== '__ungrouped__'"
              class="select-group-label"
            >
              {{ group }}
            </div>
            <div
              v-for="option in options"
              :key="String(option.value)"
              :class="cn(
                'select-option',
                {
                  'select-option--selected': isSelectedValue(option.value),
                  'opacity-50 cursor-not-allowed': option.disabled,
                  'pl-6': group !== '__ungrouped__',
                }
              )"
              @click.stop="selectOption(option)"
            >
              <div
                v-if="multiple"
                :class="cn(
                  'select-check',
                  isSelectedValue(option.value)
                    ? 'select-check--selected'
                    : ''
                )"
              >
                <Check v-if="isSelectedValue(option.value)" :size="12" />
              </div>
              <span class="truncate flex-1">{{ option.label }}</span>
            </div>
          </template>
        </template>

        <template v-else>
          <div
            v-for="option in filteredOptions"
            :key="String(option.value)"
            :class="cn(
              'select-option',
              {
                'select-option--selected': isSelectedValue(option.value),
                'opacity-50 cursor-not-allowed': option.disabled,
              }
            )"
            @click.stop="selectOption(option)"
          >
            <div
              v-if="multiple"
              :class="cn(
                'select-check',
                isSelectedValue(option.value)
                  ? 'select-check--selected'
                  : ''
              )"
            >
              <Check v-if="isSelectedValue(option.value)" :size="12" />
            </div>
            <span class="truncate flex-1">{{ option.label }}</span>
          </div>
        </template>

        <div
          v-if="filteredOptions.length === 0"
          class="select-empty"
        >
          无匹配结果
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.select-trigger--sm {
  height: 28px;
}

.select-trigger--md {
  height: 30px;
}

.select-trigger--lg {
  height: 34px;
}

.select-meta {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-muted);
}

.select-meta--error {
  color: var(--color-error);
}

.select-dropdown {
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface);
  box-shadow: var(--shadow-panel);
}

.select-group-label {
  padding: 4px 12px;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-muted);
}

.select-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  font-size: 13px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: background-color 0.14s ease, color 0.14s ease;
}

.select-option:hover {
  background: var(--ui-bg-surface-hover);
  color: var(--color-text-primary);
}

.select-option--selected {
  background: var(--color-accent-soft);
  color: var(--color-accent);
}

.select-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: 1px solid var(--ui-border-default);
  border-radius: 4px;
}

.select-check--selected {
  border-color: var(--color-accent);
  background: var(--color-accent);
  color: #fff;
}

.select-search-input {
  height: 28px;
  padding-left: 28px;
  padding-right: 8px;
  font-size: 12px;
}

.select-empty {
  padding: 14px 12px;
  font-size: 12px;
  text-align: center;
  color: var(--color-text-muted);
}
</style>
