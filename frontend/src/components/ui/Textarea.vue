<script setup>
import { computed, useAttrs } from "vue"
import { cn } from "@/lib/utils"
import Label from "./Label.vue"

defineOptions({
  inheritAttrs: false,
})

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: "",
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
  rows: {
    type: Number,
    default: 3,
  },
  maxLength: {
    type: Number,
    default: undefined,
  },
  autoResize: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(["update:modelValue"])
const attrs = useAttrs()

const textareaAttrs = computed(() => {
  const { class: _class, ...rest } = attrs
  return rest
})

function onInput(event) {
  const value = event.target.value
  emit("update:modelValue", value)

  // 自动调整高度
  if (props.autoResize) {
    const el = event.target
    el.style.height = "auto"
    el.style.height = el.scrollHeight + "px"
  }
}
</script>

<template>
  <div class="w-full">
    <!-- 标签 -->
    <Label v-if="label" :class="cn('mb-1.5')">
      {{ label }}
    </Label>

    <!-- 文本框容器 -->
    <div class="relative">
      <textarea
        v-bind="textareaAttrs"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :readonly="readonly"
        :rows="rows"
        :maxlength="maxLength"
        @input="onInput"
        :class="cn(
          'flex w-full rounded-md border border-border bg-background px-3 py-2 text-sm',
          'placeholder:text-muted-foreground',
          'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring focus-visible:border-ring',
          'transition-all',
          'disabled:cursor-not-allowed disabled:opacity-50',
          'read-only:opacity-70',
          'resize-none',
          {
            'border-error ring-1 ring-error': error,
          },
          attrs.class
        )"
      />
    </div>

    <!-- 底部信息栏 -->
    <div class="flex items-center justify-between mt-1">
      <!-- 描述文字 -->
      <p
        v-if="description && !error"
        class="text-xs text-muted-foreground"
      >
        {{ description }}
      </p>

      <!-- 字符计数 -->
      <p
        v-if="maxLength"
        class="text-xs text-muted-foreground"
      >
        {{ String(modelValue).length }}/{{ maxLength }}
      </p>
    </div>

    <!-- 错误提示 -->
    <p
      v-if="error"
      class="mt-1 text-xs text-error"
    >
      {{ error }}
    </p>
  </div>
</template>
