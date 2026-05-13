<template>
  <label class="ued-field">
    <span v-if="label" class="ued-field__label" :class="{ 'ued-field__label--sr-only': hideLabel }">
      {{ label }}
      <span v-if="required" class="ued-field__required" aria-hidden="true">*</span>
    </span>
    <slot />
    <p v-if="hint && !error" class="ued-field__hint">{{ hint }}</p>
    <p v-if="error" class="ued-field__error">{{ error }}</p>
  </label>
</template>

<script setup>
defineProps({
  label: {
    type: String,
    default: "",
  },
  hint: {
    type: String,
    default: "",
  },
  error: {
    type: String,
    default: "",
  },
  required: {
    type: Boolean,
    default: false,
  },
  hideLabel: {
    type: Boolean,
    default: false,
  },
});
</script>

<style scoped>
.ued-field__label--sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>
