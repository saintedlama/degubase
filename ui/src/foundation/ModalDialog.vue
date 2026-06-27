<template>
  <Backdrop @dismiss="$emit('cancel')">
    <div class="bg-surface-1 border border-border-1 rounded-xl w-100 max-w-[calc(100vw-32px)] shadow-xl flex flex-col overflow-hidden">

      <!-- Header -->
      <div class="flex items-center justify-between px-5 pt-5 pb-0">
        <h3 class="text-[15px] font-bold text-text-1">{{ title }}</h3>
        <button
          class="flex items-center justify-center w-6.5 h-6.5 bg-transparent border-none text-text-2 cursor-pointer rounded-md hover:text-text-1 hover:bg-border-1 transition-colors"
          @click="$emit('cancel')"
        >
          <RiCloseLine size="16" />
        </button>
      </div>

      <!-- Body -->
      <div class="p-5 flex flex-col gap-3.5">
        <slot />
      </div>

      <!-- Footer -->
      <div class="flex justify-end gap-2 px-5 py-3.5 border-t border-border-1 bg-surface-2">
        <button
          class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-surface-2 text-text-2 border border-zinc-300 dark:border-[#3c3c3c] cursor-pointer hover:bg-border-1 transition-colors"
          @click="$emit('cancel')"
        >Cancel</button>
        <button
          :disabled="confirmDisabled"
          class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 disabled:opacity-40 disabled:cursor-default transition-colors"
          @click="$emit('confirm')"
        >{{ confirmLabel }}</button>
      </div>
    </div>
  </Backdrop>
</template>

<script setup>
import { RiCloseLine } from '@remixicon/vue'
import Backdrop from './Backdrop.vue'

defineProps({
  title: { type: String, required: true },
  confirmLabel: { type: String, default: 'Save' },
  confirmDisabled: { type: Boolean, default: false },
})
defineEmits(['confirm', 'cancel'])
</script>
