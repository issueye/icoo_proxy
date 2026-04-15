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
      return "h-8 px-2.5 text-xs"
    case "default":
      return "h-9 px-3 text-sm"
    case "lg":
      return "h-10 px-3.5 text-sm"
    default:
      return "h-9 px-3 text-sm"
  }
})

const dropdownStyle = computed(() => {
  const triggerWidth = dropdownWidth.value || 0
  const clampedWidth = Math.min(Math.max(triggerWidth, 136), 240)

  return {
    width: `${clampedWidth}px`,
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
          'input-control flex items-center gap-1 truncate',
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
    <p v-if="description && !error" class="mt-1 text-xs text-muted-foreground">
      {{ description }}
    </p>

    <!-- 错误提示 -->
    <p v-if="error" class="mt-1 text-xs text-error">
      {{ error }}
    </p>

    <!-- 下拉菜单 -->
    <div
      v-if="isOpen"
      :style="dropdownStyle"
      :class="cn(
        'absolute left-0 top-full z-50 mt-1 origin-top-left',
        'border border-border rounded-md',
        'bg-popover shadow-lg',
        'max-h-60 overflow-y-auto'
      )"
    >
      <!-- 搜索框 -->
      <div v-if="searchable" class="p-2 border-b border-border">
        <div class="input-shell">
          <Search :size="12" class="input-affix absolute left-2 top-1/2 -translate-y-1/2" />
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索..."
            class="input-control h-7 pl-7 pr-2 text-xs"
            @click.stop
          />
        </div>
      </div>

      <!-- 选项列表 -->
      <div class="py-1">
        <!-- 分组显示 -->
        <template v-if="Object.keys(filteredGroupedOptions).length > 1 || ('__ungrouped__' in filteredGroupedOptions && Object.keys(filteredGroupedOptions).length > 1)">
          <template v-for="(options, group) in filteredGroupedOptions" :key="group">
            <!-- 分组标题 -->
            <div
              v-if="group !== '__ungrouped__'"
              class="px-3 py-1 text-[11px] font-medium text-muted-foreground truncate"
            >
              {{ group }}
            </div>
            <!-- 分组选项 -->
            <div
              v-for="option in options"
              :key="String(option.value)"
              :class="cn(
                'flex items-center gap-2 px-3 py-1.5 cursor-pointer transition-colors text-sm',
                'hover:bg-accent hover:text-accent-foreground',
                {
                  'bg-accent/10 text-foreground': isSelectedValue(option.value),
                  'opacity-50 cursor-not-allowed': option.disabled,
                  'pl-6': group !== '__ungrouped__',
                }
              )"
              @click.stop="selectOption(option)"
            >
              <!-- 多选勾选框 -->
              <div
                v-if="multiple"
                :class="cn(
                  'flex-shrink-0 w-4 h-4 rounded border flex items-center justify-center',
                  isSelectedValue(option.value)
                    ? 'bg-primary border-primary text-primary-foreground'
                    : 'border-border'
                )"
              >
                <Check v-if="isSelectedValue(option.value)" :size="12" />
              </div>
              <span class="truncate flex-1">{{ option.label }}</span>
            </div>
          </template>
        </template>

        <!-- 无分组或搜索结果 -->
        <template v-else>
          <div
            v-for="option in filteredOptions"
            :key="String(option.value)"
            :class="cn(
              'flex items-center gap-2 px-3 py-1.5 cursor-pointer transition-colors text-sm',
              'hover:bg-accent hover:text-accent-foreground',
              {
                'bg-accent/10 text-foreground': isSelectedValue(option.value),
                'opacity-50 cursor-not-allowed': option.disabled,
              }
            )"
            @click.stop="selectOption(option)"
          >
            <!-- 多选勾选框 -->
            <div
              v-if="multiple"
              :class="cn(
                'flex-shrink-0 w-4 h-4 rounded border flex items-center justify-center',
                isSelectedValue(option.value)
                  ? 'bg-primary border-primary text-primary-foreground'
                  : 'border-border'
              )"
            >
              <Check v-if="isSelectedValue(option.value)" :size="12" />
            </div>
            <span class="truncate flex-1">{{ option.label }}</span>
          </div>
        </template>

        <!-- 无结果提示 -->
        <div
          v-if="filteredOptions.length === 0"
          class="px-3 py-4 text-xs text-center text-muted-foreground truncate"
        >
          无匹配结果
        </div>
      </div>
    </div>
  </div>
</template>
