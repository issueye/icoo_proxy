<script setup>
import { cva } from "class-variance-authority"
import { cn } from "@/lib/utils"

const props = defineProps({
  variant: {
    type: String,
    default: "default",
  },
  size: {
    type: String,
    default: "default",
  },
  as: {
    type: String,
    default: "button",
  },
  disabled: {
    type: Boolean,
    default: false,
  },
})

const buttonVariants = cva(
  "btn whitespace-nowrap disabled:pointer-events-none",
  {
    variants: {
      variant: {
        default: "btn-primary",
        brand: "btn-primary",
        destructive: "btn-danger",
        outline: "btn-secondary",
        secondary: "btn-secondary",
        ghost: "btn-ghost",
        link: "btn-link",
      },
      size: {
        default: "btn-md",
        sm: "btn-sm",
        lg: "btn-lg",
        icon: "btn-icon",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)
</script>

<template>
  <component
    :is="as"
    :class="cn(buttonVariants({ variant, size }), $attrs.class)"
    :disabled="disabled"
  >
    <slot />
  </component>
</template>
