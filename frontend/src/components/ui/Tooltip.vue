<script setup>
import {
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
  TooltipContent,
  TooltipArrow,
} from "radix-vue"
import { cn } from "@/lib/utils"

defineProps({
  content: {
    type: String,
    default: "",
  },
  side: {
    type: String,
    default: "top",
  },
  delayDuration: {
    type: Number,
    default: 200,
  },
})
</script>

<template>
  <TooltipProvider :delay-duration="delayDuration">
    <TooltipRoot>
      <TooltipTrigger as-child>
        <slot />
      </TooltipTrigger>
      <TooltipContent
        :side="side"
        :class="cn(
          'z-50 overflow-hidden rounded-md border bg-popover px-3 py-1.5 text-sm text-popover-foreground shadow-md animate-in fade-in-0 zoom-in-95 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
          $attrs.class
        )"
      >
        <slot name="content">{{ content }}</slot>
        <TooltipArrow :class="'fill-popover'" />
      </TooltipContent>
    </TooltipRoot>
  </TooltipProvider>
</template>
