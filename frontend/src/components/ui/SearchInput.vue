<script setup>
import { ref, computed } from "vue"
import { cn } from "@/lib/utils"
import { Search, X } from "lucide-vue-next"

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
      return "h-11 text-base"
    default:
      return "h-9 text-sm"
  }
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
      'relative flex items-center w-full',
      'border border-border rounded-md',
      'bg-background transition-all',
      'focus-within:outline-none focus-within:ring-1 focus-within:ring-ring focus-within:border-ring',
      {
        'opacity-50': disabled,
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
      type="text"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      @input="onInput"
      @keydown="onKeydown"
      :class="cn(
        'w-full bg-transparent border-0 outline-none',
        'pl-8 pr-7',
        'placeholder:text-muted-foreground',
        'disabled:cursor-not-allowed',
        inputSize,
        $attrs.class
      )"
    />

    <!-- 清除按钮 -->
    <button
      v-if="hasValue && !disabled"
      type="button"
      @click="clearInput"
      class="absolute right-2 p-0.5 text-muted-foreground hover:text-foreground transition-colors"
      tabindex="-1"
    >
      <X :size="12" />
    </button>
  </div>
</template>
