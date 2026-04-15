<script setup>
import { cn } from "@/lib/utils"

const props = defineProps({
  icon: {
    type: [Object, Function],
    default: null,
  },
  iconColor: {
    type: String,
    default: "text-primary",
  },
  iconBg: {
    type: String,
    default: "bg-primary/10",
  },
  value: {
    type: [String, Number],
    required: true,
  },
  label: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: "",
  },
})
</script>

<template>
  <div class="metric-card page-surface-soft">
    <div class="metric-card__main">
      <div
        v-if="icon"
        :class="cn('metric-card__icon', iconBg)"
      >
        <component :is="icon" :size="18" :class="iconColor" />
      </div>

      <div class="metric-card__content">
        <p class="metric-card__value">{{ value }}</p>
        <p class="metric-card__label">{{ label }}</p>
        <p v-if="description" class="metric-card__description">
          {{ description }}
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.metric-card {
  min-width: 0;
  padding: var(--panel-padding);
}

.metric-card__main {
  display: flex;
  align-items: center;
  gap: var(--space-12);
}

.metric-card__icon {
  width: 34px;
  height: 34px;
  border-radius: var(--radius-sm);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.metric-card__content {
  min-width: 0;
  display: grid;
  gap: 2px;
}

.metric-card__value {
  margin: 0;
  font-size: 20px;
  line-height: 1.2;
  font-weight: 700;
  color: var(--color-text-primary);
}

.metric-card__label,
.metric-card__description {
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.metric-card__label {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.metric-card__description {
  font-size: 11px;
  color: var(--color-text-tertiary);
}
</style>
