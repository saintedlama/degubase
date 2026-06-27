<template>
  <Flapout ref="flapout">
    <template #trigger="{ toggle, setAnchor }">
      <button
        :ref="setAnchor"
        :class="variant === 'cell' ? 'cell-trigger' : 'form-trigger'"
        @click.stop="toggle"
        @keydown.tab.prevent="flapout?.close(); emit('tab', $event)"
        @keydown.escape.stop="flapout?.close(); emit('cancel')"
      >
        <span v-if="currentEmoji" class="text-lg leading-none">{{ currentEmoji.emoji }}</span>
        <span v-else class="text-text-3">—</span>
        <span v-if="variant === 'form' && currentEmoji" class="text-sm text-text-1">{{ currentEmoji.label }}</span>
        <RiArrowDownSLine v-if="variant === 'form'" size="14" class="ml-auto shrink-0 text-text-3" />
      </button>
    </template>
    <PanelCard class="p-2 w-52">
      <button class="flex items-center gap-2 w-full px-2 py-1 mb-1 rounded text-[11px] text-text-3 bg-transparent border-none cursor-pointer hover:bg-border-1 transition-colors" :class="!modelValue ? 'bg-border-1' : ''" @click="pick('')">
        <span class="w-4 text-center">—</span> None
      </button>
      <div class="grid grid-cols-4 gap-0.5">
        <button
          v-for="e in EMOJIS" :key="e.name"
          class="flex flex-col items-center gap-0.5 p-2 rounded cursor-pointer border-none transition-colors"
          :class="e.name === modelValue ? 'bg-border-1' : 'bg-transparent hover:bg-surface-2'"
          :title="e.label"
          @click="pick(e.name)"
        >
          <span class="text-lg leading-none">{{ e.emoji }}</span>
          <span class="text-[9px] leading-none truncate w-full text-center text-text-2">{{ e.label }}</span>
        </button>
      </div>
    </PanelCard>
  </Flapout>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import Flapout from '../../foundation/Flapout.vue'
import { EMOJIS, getEmoji } from './emojis.js'
import { RiArrowDownSLine } from '@remixicon/vue'
import PanelCard from '../../foundation/PanelCard.vue'

const props = defineProps({
  modelValue: { default: '' },
  variant:    { type: String, default: 'form' },
})

const emit = defineEmits(['update:modelValue', 'commit', 'cancel', 'tab'])

const flapout = ref(null)

const currentEmoji = computed(() => props.modelValue ? getEmoji(props.modelValue) : null)

onMounted(() => {
  if (props.variant === 'cell') nextTick(() => flapout.value?.toggle())
})

function pick(name) {
  emit('update:modelValue', name)
  flapout.value?.close()
  emit('commit')
}
</script>

<style scoped>
.cell-trigger {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
  height: 34px;
  background: transparent;
  border: none;
  outline: none;
  padding: 0 10px;
  cursor: pointer;
  font-family: inherit;
  box-sizing: border-box;
}

.form-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  background: var(--input-bg);
  border: 1px solid var(--input-border);
  border-radius: 8px;
  color: var(--text-1);
  font-size: 14px;
  font-family: inherit;
  padding: 8px 12px;
  outline: none;
  cursor: pointer;
  text-align: left;
  box-sizing: border-box;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.form-trigger:focus {
  border-color: var(--color-brand-600);
  box-shadow: 0 0 0 3px rgba(210, 105, 30, 0.12);
}
</style>
