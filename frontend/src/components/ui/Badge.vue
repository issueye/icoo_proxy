<script setup>
import { cva } from "class-variance-authority"
import { cn } from "@/lib/utils"

const props = defineProps({
  variant: {
    type: String,
    default: "default",
  },
})

const badgeVariants = cva(
  "inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default:
          "border-primary/15 bg-primary/10 text-primary hover:bg-primary/15",
        secondary:
          "border-border bg-secondary text-secondary-foreground hover:bg-secondary/80",
        destructive:
          "border-destructive/20 bg-destructive/10 text-destructive hover:bg-destructive/15",
        outline: "border-border bg-transparent text-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)
</script>

<template>
  <div :class="cn(badgeVariants({ variant }), $attrs.class)">
    <slot />
  </div>
</template>
