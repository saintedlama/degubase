<template>
  <button
    class="flex items-center justify-center bg-transparent cursor-pointer rounded-md transition-colors"
    :class="[
      border ? 'border' : 'border-none',
      border
        ? (active
            ? 'border-brand-400 text-brand-600 dark:text-brand-400'
            : danger
              ? 'border-border-1 text-text-2 hover:text-red-500 hover:border-red-400'
              : 'border-border-1 text-text-2 hover:text-brand-600 hover:border-brand-400')
        : (active
            ? 'text-brand-600 dark:text-brand-400'
            : danger
              ? 'text-text-2 hover:text-red-500'
              : 'text-text-2 hover:text-text-1'),
    ]"
    :title="title"
    v-bind="$attrs"
  >
    <slot />
  </button>
</template>

<script setup>
defineProps({
  title:  { type: String, default: '' },
  danger: { type: Boolean, default: false },
  active: { type: Boolean, default: false },
  border: { type: Boolean, default: true },
})
</script>
