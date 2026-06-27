<template>
  <svg
    class="animate-spin shrink-0"
    viewBox="0 0 24 24"
    fill="none"
    :width="size"
    :height="size"
    aria-hidden="true"
  >
    <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" class="opacity-25" />
    <path fill="currentColor" class="opacity-75" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
  </svg>
</template>

<script setup>
defineProps({ size: { type: Number, default: 12 } })
</script>
