<script setup>
import { computed, ref } from "vue"
import { cn } from "@/lib/utils"
import { X, Eye, EyeOff } from "lucide-vue-next"
import Label from "./Label.vue"

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: "",
  },
  type: {
    type: String,
    default: "text",
  },
  placeholder: {
    type: String,
    default: "",
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
  readonly: {
    type: Boolean,
    default: false,
  },
  clearable: {
    type: Boolean,
    default: false,
  },
  showPasswordToggle: {
    type: Boolean,
    default: false,
  },
  size: {
    type: String,
    default: "default",
  },
  prefix: {
    type: String,
    default: "",
  },
  suffix: {
    type: String,
    default: "",
  },
})

const emit = defineEmits(["update:modelValue", "clear", "keydown"])

const showPassword = ref(false)

const inputType = computed(() => {
  if (props.showPasswordToggle && props.type === "password") {
    return showPassword.value ? "text" : "password"
  }
  return props.type
})

const hasValue = computed(() => {
  return props.modelValue !== "" && props.modelValue !== null && props.modelValue !== undefined
})

const inputSize = computed(() => {
  switch (props.size) {
    case "sm":
      return "h-8 px-2.5 text-xs"
    case "default":
      return "h-9 px-3 text-sm"
    case "lg":
      return "h-11 px-4 text-base"
    default:
      return "h-9 px-3 text-sm"
  }
})

function onInput(event) {
  emit("update:modelValue", event.target.value)
}

function onKeydown(event) {
  emit("keydown", event)
}

function clearInput() {
  emit("update:modelValue", "")
  emit("clear")
}

function togglePassword() {
  showPassword.value = !showPassword.value
}
</script>

<template>
  <div class="w-full">
    <!-- 标签 -->
    <Label v-if="label" :class="cn('mb-1.5')">
      {{ label }}
    </Label>

    <!-- 输入框容器 -->
    <div class="relative">
      <div
        :class="cn(
          'flex items-center w-full border rounded-md transition-all',
          'bg-background text-foreground',
          'border-border',
          'focus-within:outline-none focus-within:ring-1 focus-within:ring-ring focus-within:border-ring',
          {
            'border-error ring-1 ring-error': error,
            'opacity-50 cursor-not-allowed': disabled,
          }
        )"
      >
        <!-- 前置内容 -->
        <span
          v-if="prefix"
          class="flex-shrink-0 pl-2.5 pr-1 text-muted-foreground text-xs"
        >{{ prefix }}</span>

        <!-- 输入框 -->
        <input
          :type="inputType"
          :value="modelValue"
          :placeholder="placeholder"
          :disabled="disabled"
          :readonly="readonly"
          @input="onInput"
          @keydown="onKeydown"
          :class="cn(
            'flex-1 bg-transparent border-0 outline-none',
            'placeholder:text-muted-foreground',
            'disabled:cursor-not-allowed disabled:opacity-50',
            'read-only:opacity-70',
            inputSize,
            $attrs.class
          )"
        />

        <!-- 清除按钮 -->
        <button
          v-if="clearable && hasValue && !disabled"
          type="button"
          @click="clearInput"
          class="flex-shrink-0 p-0.5 mr-1 text-muted-foreground hover:text-foreground transition-colors"
          tabindex="-1"
        >
          <X :size="14" />
        </button>

        <!-- 密码显示切换 -->
        <button
          v-if="showPasswordToggle"
          type="button"
          @click="togglePassword"
          class="flex-shrink-0 p-0.5 mr-1 text-muted-foreground hover:text-foreground transition-colors"
          tabindex="-1"
        >
          <Eye v-if="!showPassword" :size="14" />
          <EyeOff v-else :size="14" />
        </button>

        <!-- 后置内容 -->
        <span
          v-if="suffix"
          class="flex-shrink-0 pr-2.5 pl-1 text-muted-foreground text-xs"
        >{{ suffix }}</span>
      </div>
    </div>

    <!-- 描述文字 -->
    <p
      v-if="description && !error"
      class="mt-1 text-xs text-muted-foreground"
    >
      {{ description }}
    </p>

    <!-- 错误提示 -->
    <p
      v-if="error"
      class="mt-1 text-xs text-error"
    >
      {{ error }}
    </p>
  </div>
</template>
