<script setup>
import { ref, computed, useAttrs } from "vue"
import { cn } from "@/lib/utils"
import { Search, X } from "lucide-vue-next"

defineOptions({
  inheritAttrs: false,
})

const props = defineProps({
  modelValue: {
    type: String,
    default: "",
  },
  placeholder: {
    type: String,
    default: "搜索...",
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  size: {
    type: String,
    default: "default",
  },
})

const emit = defineEmits(["update:modelValue", "clear", "search"])
const attrs = useAttrs()

const inputRef = ref(null)

const hasValue = computed(() => {
  return props.modelValue !== "" && props.modelValue !== null
})

const inputSize = computed(() => {
  switch (props.size) {
    case "sm":
      return "h-8 text-xs"
    case "default":
      return "h-9 text-sm"
    case "lg":
      return "h-10 text-sm"
    default:
      return "h-9 text-sm"
  }
})

const inputAttrs = computed(() => {
  const { class: _class, ...rest } = attrs
  return rest
})

function onInput(event) {
  emit("update:modelValue", event.target.value)
}

function clearInput() {
  emit("update:modelValue", "")
  emit("clear")
  inputRef.value?.focus()
}

function onKeydown(event) {
  if (event.key === "Enter") {
    emit("search", props.modelValue)
  }
  if (event.key === "Escape") {
    clearInput()
  }
}

// 暴露 focus 方法
function focus() {
  inputRef.value?.focus()
}

defineExpose({ focus })
</script>

<template>
  <div
    :class="cn(
      'input-shell',
      {
        'is-disabled': disabled,
      }
    )"
  >
    <!-- 搜索图标 -->
    <Search
      :size="14"
      class="absolute left-2.5 text-muted-foreground flex-shrink-0 pointer-events-none"
    />

    <!-- 输入框 -->
    <input
      ref="inputRef"
      v-bind="inputAttrs"
      type="text"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      @input="onInput"
      @keydown="onKeydown"
      :class="cn(
        'input-control',
        'pl-8 pr-7',
        'disabled:cursor-not-allowed',
        inputSize,
        attrs.class
      )"
    />

    <!-- 清除按钮 -->
    <button
      v-if="hasValue && !disabled"
      type="button"
      @click="clearInput"
      class="input-action absolute right-2 p-0.5"
      tabindex="-1"
    >
      <X :size="12" />
    </button>
  </div>
</template>
