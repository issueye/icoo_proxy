<template>
  <span class="model-brand-icon" :class="`model-brand-icon--${normalizedIcon}`" aria-hidden="true">
    <svg v-if="normalizedIcon === 'gpt'" viewBox="0 0 32 32">
      <path d="M16 5.2a6.1 6.1 0 0 1 10.4 4.3 6.1 6.1 0 0 1-1.1 11.8A6.1 6.1 0 0 1 15 27.7 6.1 6.1 0 0 1 5.6 22a6.1 6.1 0 0 1 .8-11.7A6.1 6.1 0 0 1 16 5.2Z" />
      <path d="m10.2 12.4 5.8-3.3 5.8 3.3V19L16 22.3 10.2 19Z" />
    </svg>
    <svg v-else-if="normalizedIcon === 'deepseek'" viewBox="0 0 32 32">
      <path d="M5 18.6c2.6-5.8 7.3-8.7 14.1-8.7 3.7 0 6.3 1 7.9 2.9-1.2 6.1-5.6 9.8-12.2 9.8-4.2 0-7.5-1.3-9.8-4Z" />
      <path d="M19.7 9.9c.6-2.2 2-3.8 4.3-4.9.5 2.2.1 4.2-1.2 6" />
      <circle cx="21.8" cy="14.1" r="1.1" />
    </svg>
    <svg v-else-if="normalizedIcon === 'glm'" viewBox="0 0 32 32">
      <path d="M16 4 27 10.2v11.6L16 28 5 21.8V10.2Z" />
      <path d="m10.1 13.2 5.9-3.4 5.9 3.4v5.6L16 22.2l-5.9-3.4Z" />
    </svg>
    <svg v-else-if="normalizedIcon === 'claude'" viewBox="0 0 32 32">
      <path d="M16 4.5v23M4.5 16h23M7.9 7.9l16.2 16.2M24.1 7.9 7.9 24.1M10.1 5.7l11.8 20.6M26.3 10.1 5.7 21.9M21.9 5.7 10.1 26.3M5.7 10.1l20.6 11.8" />
    </svg>
    <svg v-else viewBox="0 0 32 32">
      <rect x="7" y="7" width="18" height="18" rx="4" />
      <path d="M12 16h8M16 12v8" />
    </svg>
  </span>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  icon: { type: String, default: "custom" },
});

const normalizedIcon = computed(() => ["gpt", "deepseek", "glm", "claude"].includes(props.icon)
  ? props.icon
  : "custom");
</script>

<style scoped>
.model-brand-icon {
  display: inline-grid;
  width: 28px;
  height: 28px;
  flex: 0 0 28px;
  place-items: center;
  border: 1px solid var(--ued-color-border-light);
  border-radius: var(--ued-radius-md);
  background: var(--ued-color-muted);
  color: var(--ued-color-text-secondary);
}

.model-brand-icon svg {
  width: 16px;
  height: 16px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.model-brand-icon--gpt { background: #ecf8f4; color: #087f5b; }
.model-brand-icon--deepseek { background: #edf4ff; color: #2563eb; }
.model-brand-icon--glm { background: #f1efff; color: #6741d9; }
.model-brand-icon--claude { background: #fff2eb; color: #c2410c; }
.model-brand-icon--deepseek circle { fill: currentColor; stroke: none; }
</style>
