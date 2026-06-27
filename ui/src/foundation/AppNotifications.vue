<template>
  <Teleport to="body">
    <div class="fixed bottom-4 right-4 z-[500] flex flex-col gap-2 items-end pointer-events-none">
      <TransitionGroup name="toast">
        <div
          v-for="n in notifications"
          :key="n.id"
          class="pointer-events-auto flex items-start gap-2.5 min-w-64 max-w-sm px-3.5 py-3 rounded-lg shadow-lg border text-[13px] leading-snug"
          :class="typeClass(n.type)"
        >
          <component :is="typeIcon(n.type)" size="15" class="shrink-0 mt-px" />
          <span class="flex-1">{{ n.message }}</span>
          <button
            class="shrink-0 opacity-60 hover:opacity-100 bg-transparent border-none cursor-pointer p-0 leading-none mt-px"
            @click="dismiss(n.id)"
          >
            <RiCloseLine size="14" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup>
import { RiCloseLine, RiErrorWarningLine, RiCheckLine, RiInformationLine } from '@remixicon/vue'
import { useNotifications } from './useNotifications.js'

const { notifications, dismiss } = useNotifications()

function typeClass(type) {
  if (type === 'error')   return 'bg-red-50 dark:bg-red-950/60 border-red-200 dark:border-red-800/60 text-red-800 dark:text-red-200'
  if (type === 'success') return 'bg-emerald-50 dark:bg-emerald-950/60 border-emerald-200 dark:border-emerald-800/60 text-emerald-800 dark:text-emerald-200'
  return 'bg-brand-50 dark:bg-brand-900/40 border-brand-200 dark:border-brand-700/60 text-brand-800 dark:text-brand-200'
}

function typeIcon(type) {
  if (type === 'error')   return RiErrorWarningLine
  if (type === 'success') return RiCheckLine
  return RiInformationLine
}
</script>

<style scoped>
.toast-enter-active { transition: all 0.2s ease-out; }
.toast-leave-active { transition: all 0.2s ease-in; }
.toast-enter-from   { opacity: 0; transform: translateX(1rem); }
.toast-leave-to     { opacity: 0; transform: translateX(1rem); }
</style>
