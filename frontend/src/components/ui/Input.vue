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
      return "input-control--sm"
    case "default":
      return "input-control--md"
    case "lg":
      return "input-control--lg"
    default:
      return "input-control--md"
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
    <Label v-if="label" :class="cn('mb-1.5')">
      {{ label }}
    </Label>

    <div
      :class="cn(
        'input-shell',
        {
          'is-error': error,
          'is-disabled': disabled,
        }
      )"
    >
      <div class="flex w-full items-center">
        <span
          v-if="prefix"
          class="input-affix input-affix--leading"
        >{{ prefix }}</span>

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
            inputSize,
            attrs.class
          )"
        />

        <button
          v-if="clearable && hasValue && !disabled"
          type="button"
          @click="clearInput"
          class="input-action mr-1 p-0.5"
          tabindex="-1"
        >
          <X :size="14" />
        </button>

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

        <span
          v-if="suffix"
          class="input-affix input-affix--trailing"
        >{{ suffix }}</span>
      </div>
    </div>

    <p
      v-if="description && !error"
      class="input-meta"
    >
      {{ description }}
    </p>

    <p
      v-if="error"
      class="input-meta input-meta--error"
    >
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.input-control--sm {
  height: 28px;
}

.input-control--md {
  height: 30px;
}

.input-control--lg {
  height: 34px;
}

.input-affix--leading {
  padding-left: 10px;
  padding-right: 4px;
  font-size: 12px;
}

.input-affix--trailing {
  padding-left: 4px;
  padding-right: 10px;
  font-size: 12px;
}

.input-meta {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-muted);
}

.input-meta--error {
  color: var(--color-error);
}
</style>
