<template>
  <section class="ued-page-section ued-panel" :class="sectionClass">
    <header v-if="title || $slots.header || $slots.actions" class="ued-page-section__header">
      <div class="ued-page-section__copy">
        <slot name="header">
          <h3 v-if="title" class="ued-title-2">{{ title }}</h3>
          <p v-if="description" class="ued-page-section__description">{{ description }}</p>
        </slot>
      </div>
      <div v-if="$slots.actions" class="ued-page-section__actions">
        <slot name="actions" />
      </div>
    </header>

    <div class="ued-page-section__body">
      <slot />
    </div>
  </section>
</template>

<script setup>
import { computed } from "vue"
import { cn } from "@/lib/utils"

const props = defineProps({
  title: {
    type: String,
    default: "",
  },
  description: {
    type: String,
    default: "",
  },
  muted: {
    type: Boolean,
    default: false,
  },
})

const sectionClass = computed(() => cn(props.muted && "ued-panel--muted"))
</script>

<style scoped>
.ued-page-section {
  overflow: hidden;
}

.ued-page-section__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ued-space-12);
  padding: var(--ued-panel-padding);
  border-bottom: 1px solid var(--ued-border-subtle);
}

.ued-page-section__copy {
  min-width: 0;
  flex: 1;
}

.ued-page-section__description {
  margin: 4px 0 0;
  font-size: var(--ued-text-body);
  line-height: 1.6;
  color: var(--ued-text-secondary);
}

.ued-page-section__actions {
  display: flex;
  align-items: center;
  gap: var(--ued-space-8);
  flex-wrap: wrap;
}

.ued-page-section__body {
  padding: var(--ued-panel-padding);
}
</style>