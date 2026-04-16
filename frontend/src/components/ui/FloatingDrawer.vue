<script setup>
import UEDDrawer from "@/components/layout/UEDDrawer.vue";

const props = defineProps({
  visible: {
    type: Boolean,
    default: false,
  },
  title: {
    type: String,
    default: "",
  },
  description: {
    type: String,
    default: "",
  },
  kicker: {
    type: String,
    default: "",
  },
  width: {
    type: String,
    default: "420px",
  },
  closeOnScrim: {
    type: Boolean,
    default: true,
  },
});

const emit = defineEmits(["close", "update:visible"]);

function handleClose() {
  emit("close");
  emit("update:visible", false);
}
</script>

<template>
  <UEDDrawer
    :visible="visible"
    :title="title"
    :description="description"
    :kicker="kicker"
    :width="width"
    :close-on-scrim="closeOnScrim"
    @close="handleClose"
    @update:visible="(value) => emit('update:visible', value)"
  >
    <template v-if="$slots.summary" #summary>
      <slot name="summary" />
    </template>

    <slot />

    <template v-if="$slots.footer" #footer>
      <slot name="footer" />
    </template>
  </UEDDrawer>
</template>