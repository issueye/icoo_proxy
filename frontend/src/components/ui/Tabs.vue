<script setup>
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from "radix-vue"
import { cn } from "@/lib/utils"

defineProps({
  defaultValue: {
    type: String,
    default: undefined,
  },
  modelValue: {
    type: String,
    default: undefined,
  },
})

const emit = defineEmits(["update:modelValue"])

const onUpdate = (value) => {
  emit("update:modelValue", value)
}
</script>

<template>
  <TabsRoot
    :model-value="modelValue"
    :default-value="defaultValue"
    @update:model-value="onUpdate"
  >
    <slot name="list" :TabsList="TabsList" :TabsTrigger="TabsTrigger" :cn="cn" />
    <slot />
  </TabsRoot>
</template>
