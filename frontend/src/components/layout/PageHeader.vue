<script setup>
import { cn } from "@/lib/utils"
import Separator from "@/components/ui/Separator.vue"

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
  gap: 14px;
  margin-bottom: 14px;
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
  border-radius: var(--radius-md);
  border: 1px solid color-mix(in srgb, var(--color-border) 88%, transparent);
  background: color-mix(in srgb, var(--color-accent) 10%, var(--color-bg-secondary));
  box-shadow: var(--shadow-sm);
  flex-shrink: 0;
}

.page-header__title {
  margin: 0;
  font-size: 1.28rem;
  line-height: 1.1;
  font-weight: 800;
  letter-spacing: -0.03em;
  color: var(--color-text-primary);
}

.page-header__title--compact {
  font-size: 0.98rem;
}

.page-header__description {
  margin: 6px 0 0;
  max-width: 700px;
  font-size: 0.84rem;
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
