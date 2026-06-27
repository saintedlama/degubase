<template>
  <div class="flex-1 flex flex-col items-center justify-center gap-2 text-sm text-text-2">
    <slot name="icon" />
    <span>{{ message }}</span>
    <slot name="action" />
  </div>
</template>

<script setup>
defineProps({
  message: { type: String, default: '' },
})
</script>
