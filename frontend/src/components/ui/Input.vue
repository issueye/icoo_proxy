<script setup>
import { computed, ref, useAttrs } from "vue"
import { cn } from "@/lib/utils"
import { X, Eye, EyeOff } from "lucide-vue-next"
import Label from "./Label.vue"

defineOptions({
  inheritAttrs: false,
})

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
const attrs = useAttrs()

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
      return "h-10 px-3.5 text-sm"
    default:
      return "h-9 px-3 text-sm"
  }
})

const inputAttrs = computed(() => {
  const { class: _class, ...rest } = attrs
  return rest
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
    <div
      :class="cn(
        'input-shell',
        {
          'is-error': error,
          'is-disabled': disabled,
        }
      )"
    >
      <div
        class="flex w-full items-center"
      >
        <!-- 前置内容 -->
        <span
          v-if="prefix"
          class="input-affix pl-2.5 pr-1 text-xs"
        >{{ prefix }}</span>

        <!-- 输入框 -->
        <input
          v-bind="inputAttrs"
          :type="inputType"
          :value="modelValue"
          :placeholder="placeholder"
          :disabled="disabled"
          :readonly="readonly"
          @input="onInput"
          @keydown="onKeydown"
          :class="cn(
            'input-control',
            'disabled:cursor-not-allowed disabled:opacity-50',
            'read-only:opacity-70',
            inputSize,
            attrs.class
          )"
        />

        <!-- 清除按钮 -->
        <button
          v-if="clearable && hasValue && !disabled"
          type="button"
          @click="clearInput"
          class="input-action mr-1 p-0.5"
          tabindex="-1"
        >
          <X :size="14" />
        </button>

        <!-- 密码显示切换 -->
        <button
          v-if="showPasswordToggle"
          type="button"
          @click="togglePassword"
          class="input-action mr-1 p-0.5"
          tabindex="-1"
        >
          <Eye v-if="!showPassword" :size="14" />
          <EyeOff v-else :size="14" />
        </button>

        <!-- 后置内容 -->
        <span
          v-if="suffix"
          class="input-affix pr-2.5 pl-1 text-xs"
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
