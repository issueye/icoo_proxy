<script setup>
import { cn } from "@/lib/utils"
const props = defineProps({
  title: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: "",
  },
  icon: {
    type: [Object, Function],
    default: null,
  },
  compact: {
    type: Boolean,
    default: false,
  },
})
</script>

<template>
  <div :class="cn('page-header', compact ? 'page-header--compact' : '')">
    <div class="page-header__copy">
      <div class="page-header__title-row">
        <div v-if="icon" class="page-header__icon">
          <component :is="icon" :size="compact ? 16 : 18" class="text-primary flex-shrink-0" />
        </div>
        <h2 :class="cn('page-header__title', compact ? 'page-header__title--compact' : '')">
          {{ title }}
        </h2>
      </div>
      <p v-if="description" class="page-header__description">
        {{ description }}
      </p>
    </div>

    <div v-if="$slots.actions" class="page-header__actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--ui-border-subtle);
}

.page-header__copy {
  min-width: 0;
  flex: 1;
}

.page-header__title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.page-header__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  border: 1px solid color-mix(in srgb, var(--color-accent) 18%, var(--ui-border-default));
  background: var(--color-accent-soft);
  box-shadow: var(--shadow-rest);
  flex-shrink: 0;
}

.page-header__title {
  margin: 0;
  font-size: var(--text-title-1);
  line-height: var(--line-title-1);
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--color-text-primary);
}

.page-header__title--compact {
  font-size: var(--text-title-2);
  line-height: var(--line-title-2);
}

.page-header__description {
  margin: 6px 0 0;
  max-width: 760px;
  font-size: var(--text-body);
  line-height: 1.6;
  color: var(--color-text-secondary);
}

.page-header__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.page-header--compact {
  margin-bottom: 10px;
}

@media (max-width: 720px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
  }

  .page-header__title {
    font-size: 1.12rem;
  }
}
</style>
