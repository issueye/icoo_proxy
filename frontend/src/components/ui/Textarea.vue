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
          'textarea-control',
          {
            'textarea-control--error': error,
          },
          attrs.class
        )"
      />
    </div>

    <div class="textarea-meta-row">
      <p
        v-if="description && !error"
        class="textarea-meta"
      >
        {{ description }}
      </p>

      <p
        v-if="maxLength"
        class="textarea-meta"
      >
        {{ String(modelValue).length }}/{{ maxLength }}
      </p>
    </div>

    <p
      v-if="error"
      class="textarea-meta textarea-meta--error"
    >
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.textarea-control {
  display: flex;
  width: 100%;
  border: 1px solid var(--ui-border-default);
  border-radius: var(--radius-sm);
  background: var(--ui-bg-surface);
  padding: 8px 10px;
  font-size: 13px;
  color: var(--color-text-primary);
  line-height: 1.6;
  resize: none;
  transition: border-color 0.14s ease, box-shadow 0.14s ease;
}

.textarea-control::placeholder {
  color: var(--color-text-muted);
}

.textarea-control:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: var(--shadow-focus);
}

.textarea-control:disabled {
  cursor: not-allowed;
  opacity: 0.6;
  background: var(--ui-bg-surface-muted);
}

.textarea-control:read-only {
  opacity: 0.8;
}

.textarea-control--error {
  border-color: var(--color-error);
  box-shadow: 0 0 0 3px rgba(196, 43, 28, 0.12);
}

.textarea-meta-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 6px;
}

.textarea-meta {
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-muted);
}

.textarea-meta--error {
  margin-top: 6px;
  color: var(--color-error);
}
</style>
