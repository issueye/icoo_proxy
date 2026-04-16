<script setup>
import { computed } from "vue"
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
  kicker: {
    type: String,
    default: "",
  },
  icon: {
    type: [Object, Function],
    default: null,
  },
  divided: {
    type: Boolean,
    default: false,
  },
  compact: {
    type: Boolean,
    default: false,
  },
})

const wrapperClass = computed(() =>
  cn(
    "ued-page-head",
    props.divided && "ued-page-head--divided",
    props.compact && "ued-page-head--compact",
  ),
)
</script>

<template>
  <header :class="wrapperClass">
    <div class="ued-page-head__main">
      <div v-if="kicker" class="ued-kicker">{{ kicker }}</div>

      <div class="ued-page-head__title-row">
        <div v-if="icon" class="ued-page-head__icon">
          <component :is="icon" :size="compact ? 16 : 18" />
        </div>
        <div class="ued-page-head__copy">
          <h2 :class="compact ? 'ued-title-2' : 'ued-title-1'">{{ title }}</h2>
          <p v-if="description" class="ued-page-head__description">
            {{ description }}
          </p>
        </div>
      </div>
    </div>

    <div v-if="$slots.actions" class="ued-page-head__actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<style scoped>
.ued-page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--ued-space-16);
}

.ued-page-head--divided {
  padding-bottom: 12px;
  border-bottom: 1px solid var(--ued-border-subtle);
}

.ued-page-head--compact {
  gap: var(--ued-space-12);
}

.ued-page-head__main {
  min-width: 0;
  flex: 1;
}

.ued-page-head__title-row {
  display: flex;
  align-items: flex-start;
  gap: var(--ued-space-10);
  min-width: 0;
}

.ued-page-head__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 1px solid color-mix(in srgb, var(--ued-accent) 18%, var(--ued-border-default));
  border-radius: var(--ued-radius-sm);
  background: var(--ued-accent-soft);
  color: var(--ued-accent);
  flex-shrink: 0;
}

.ued-page-head__copy {
  min-width: 0;
}

.ued-page-head__description {
  margin: 4px 0 0;
  font-size: var(--ued-text-body);
  line-height: 1.6;
  color: var(--ued-text-secondary);
}

.ued-page-head__actions {
  display: flex;
  align-items: center;
  gap: var(--ued-space-8);
  flex-wrap: wrap;
  flex-shrink: 0;
}

@media (max-width: 720px) {
  .ued-page-head {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>