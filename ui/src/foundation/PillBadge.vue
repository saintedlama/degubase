<template>
  <span
    class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium truncate"
    :style="choicePillStyle(col, value)"
  >{{ value }}</span>
</template>

<script setup>
import { choicePillStyle } from '../features/views/palettes.js'

defineProps({
  col:   { type: Object, required: true },
  value: { type: String, required: true },
})
</script>
