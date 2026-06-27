<template>
  <div ref="rootRef">
    <!-- Trigger -->
    <button
      ref="triggerRef"
      type="button"
      class="flex items-center gap-2 px-3 py-2 w-full bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-[13px] text-text-1 cursor-pointer hover:border-brand-500 transition-colors text-left"
      @click="toggle"
    >
      <i v-if="resolvedIcon" :class="[resolvedIcon, 'text-base shrink-0']" />
      <span v-else class="text-text-3">No icon</span>
      <span v-if="currentEntry" class="flex-1 text-xs text-text-2">{{ currentEntry.label }}</span>
      <span class="ml-auto text-text-3 text-[11px]">▾</span>
    </button>

    <!-- Flapout teleported to body to escape overflow:hidden dialog containers -->
    <Teleport to="body">
      <div
        v-if="open"
        ref="flapoutRef"
        class="fixed z-9999 bg-surface-1 border border-border-1 rounded-xl shadow-lg p-2"
        :style="pos.upward
          ? { bottom: pos.bottom + 'px', left: pos.left + 'px', width: pos.width + 'px' }
          : { top: pos.top + 'px',    left: pos.left + 'px', width: pos.width + 'px' }"
      >
        <div class="grid grid-cols-6 gap-1">
          <!-- Clear -->
          <button
            type="button"
            class="flex items-center justify-center w-9 h-9 rounded-lg text-base border transition-colors"
            :class="!modelValue ? 'border-brand-500 bg-brand-50 dark:bg-brand-900/20 text-brand-600' : 'border-transparent text-text-2 hover:bg-border-1'"
            title="No icon"
            @click="select('')"
          >✕</button>
          <!-- Icons -->
          <button
            v-for="entry in TABLE_ICONS"
            :key="entry.name"
            type="button"
            class="flex items-center justify-center w-9 h-9 rounded-lg text-base border transition-colors"
            :class="modelValue === entry.name ? 'border-brand-500 bg-brand-50 dark:bg-brand-900/20 text-brand-600' : 'border-transparent text-text-2 hover:bg-border-1 hover:text-text-1'"
            :title="entry.label"
            @click="select(entry.name)"
          >
            <i :class="entry.icon" />
          </button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onUnmounted } from 'vue'
import { TABLE_ICONS, resolveTableIcon } from '../features/workspace/tableIcons.js'

const props = defineProps({
  modelValue: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const open = ref(false)
const rootRef = ref(null)
const triggerRef = ref(null)
const flapoutRef = ref(null)
const pos = ref({ top: 0, bottom: 0, left: 0, width: 0, upward: false })

const resolvedIcon = computed(() => resolveTableIcon(props.modelValue))
const currentEntry = computed(() => TABLE_ICONS.find(e => e.name === props.modelValue) ?? null)

function handleDocClick(e) {
  if (!rootRef.value?.contains(e.target) && !flapoutRef.value?.contains(e.target)) {
    open.value = false
    document.removeEventListener('click', handleDocClick, true)
  }
}

onUnmounted(() => document.removeEventListener('click', handleDocClick, true))

function toggle() {
  if (!open.value && triggerRef.value) {
    const rect = triggerRef.value.getBoundingClientRect()
    const upward = window.innerHeight - rect.bottom < 300 && rect.top > window.innerHeight - rect.bottom
    pos.value = upward
      ? { upward: true,  bottom: window.innerHeight - rect.top + 4, left: rect.left, width: rect.width }
      : { upward: false, top: rect.bottom + 4,                      left: rect.left, width: rect.width }
    // Defer so this toggle's own click doesn't immediately close it
    setTimeout(() => document.addEventListener('click', handleDocClick, true), 0)
  } else {
    open.value = false
    document.removeEventListener('click', handleDocClick, true)
    return
  }
  open.value = true
}

function select(name) {
  emit('update:modelValue', name)
  open.value = false
  document.removeEventListener('click', handleDocClick, true)
}
</script>
