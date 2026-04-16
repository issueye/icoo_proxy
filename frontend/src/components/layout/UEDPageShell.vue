<template>
  <div
    class="ued-page-shell"
    :style="{
      '--ued-shell-sidebar-width': sidebarWidth,
      '--ued-shell-aside-width': asideWidth,
      '--ued-shell-gap': gap,
    }"
  >
    <div class="ued-page-shell__frame" :class="frameClass">
      <aside v-if="$slots.sidebar" class="ued-page-shell__sidebar ued-panel ued-scroll">
        <slot name="sidebar" />
      </aside>

      <main class="ued-page-shell__main ued-scroll">
        <slot />
      </main>

      <aside v-if="$slots.aside" class="ued-page-shell__aside ued-panel ued-scroll">
        <slot name="aside" />
      </aside>
    </div>
  </div>
</template>

<script setup>
import { computed, useSlots } from "vue"
import { cn } from "@/lib/utils"

const props = defineProps({
  sidebarWidth: {
    type: String,
    default: "280px",
  },
  asideWidth: {
    type: String,
    default: "360px",
  },
  gap: {
    type: String,
    default: "16px",
  },
})

const slots = useSlots()

const frameClass = computed(() =>
  cn(
    slots.sidebar && !slots.aside && "has-sidebar",
    !slots.sidebar && slots.aside && "has-aside",
    slots.sidebar && slots.aside && "has-both",
  ),
)
</script>

<style scoped>
.ued-page-shell {
  width: 100%;
  height: 100%;
  padding: var(--ued-page-padding);
  background: var(--ued-bg-canvas);
}

.ued-page-shell__frame {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--ued-shell-gap, 16px);
  height: 100%;
  min-width: 0;
  min-height: 0;
}

.ued-page-shell__frame.has-sidebar {
  grid-template-columns: var(--ued-shell-sidebar-width, 280px) minmax(0, 1fr);
}

.ued-page-shell__frame.has-aside {
  grid-template-columns: minmax(0, 1fr) var(--ued-shell-aside-width, 360px);
}

.ued-page-shell__frame.has-both {
  grid-template-columns: var(--ued-shell-sidebar-width, 280px) minmax(0, 1fr) var(--ued-shell-aside-width, 360px);
}

.ued-page-shell__sidebar,
.ued-page-shell__aside,
.ued-page-shell__main {
  min-width: 0;
  min-height: 0;
}

.ued-page-shell__sidebar,
.ued-page-shell__aside {
  padding: var(--ued-panel-padding);
}

@media (max-width: 1100px) {
  .ued-page-shell__frame.has-both {
    grid-template-columns: var(--ued-shell-sidebar-width, 280px) minmax(0, 1fr);
  }

  .ued-page-shell__aside {
    grid-column: 1 / -1;
  }
}

@media (max-width: 860px) {
  .ued-page-shell__frame,
  .ued-page-shell__frame.has-sidebar,
  .ued-page-shell__frame.has-aside,
  .ued-page-shell__frame.has-both {
    grid-template-columns: 1fr;
  }
}
</style>