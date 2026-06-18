<template>
  <section class="content-panel">
    <div class="panel-header">
      <div class="panel-header__heading">
        <h3 class="panel-title">{{ title }}</h3>
        <p v-if="description" class="panel-description">{{ description }}</p>
      </div>
      <slot name="header-extra" />
    </div>
    <div class="panel-body">
      <slot />
    </div>
  </section>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: "",
  },
});
</script>
