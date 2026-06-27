<template>
  <button
    class="w-4.5 h-4.5 flex items-center justify-center rounded border transition-colors cursor-pointer"
    :class="active
      ? 'text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 border-brand-300 dark:border-brand-700/50'
      : 'text-zinc-300 dark:text-[#555] border-transparent hover:text-zinc-500 dark:hover:text-[#9d9d9d]'"
    :title="title"
    v-bind="$attrs"
  >
    <slot />
  </button>
</template>

<script setup>
defineProps({
  active: { type: Boolean, default: false },
  title:  { type: String, default: '' },
})
</script>
