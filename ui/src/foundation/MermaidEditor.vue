<template>
  <div class="flex flex-col border border-border-1 rounded-lg overflow-hidden focus-within:border-brand-400 focus-within:ring-3 focus-within:ring-brand-100 dark:focus-within:ring-brand-900/20 transition-all">
    <!-- Toolbar -->
    <div class="flex items-center gap-2 px-2 py-1.5 bg-surface-2 border-b border-border-1">
      <span class="text-[11px] text-text-2 font-medium">Mermaid</span>
      <div class="flex items-center rounded-md border border-border-1 overflow-hidden bg-surface-1 ml-auto">
        <button
          class="px-2.5 py-0.5 text-[11px] font-medium transition-colors"
          :class="mode === 'split' ? 'bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400' : 'text-text-2 hover:text-text-1'"
          @click="mode = 'split'"
        >Split</button>
        <button
          class="px-2.5 py-0.5 text-[11px] font-medium border-l border-border-1 transition-colors"
          :class="mode === 'code' ? 'bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400' : 'text-text-2 hover:text-text-1'"
          @click="mode = 'code'"
        >Code</button>
        <button
          class="px-2.5 py-0.5 text-[11px] font-medium border-l border-border-1 transition-colors"
          :class="mode === 'preview' ? 'bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400' : 'text-text-2 hover:text-text-1'"
          @click="mode = 'preview'"
        >Preview</button>
      </div>
    </div>

    <!-- Editor body -->
    <div class="flex min-h-52" :class="mode === 'split' ? 'divide-x divide-border-1' : ''">
      <!-- Code pane -->
      <textarea
        v-if="mode !== 'preview'"
        v-model="localVal"
        class="flex-1 p-3.5 font-mono text-[13px] leading-relaxed text-text-1 bg-surface-1 resize-none outline-none min-h-52"
        :class="mode === 'split' ? 'w-1/2' : 'w-full'"
        placeholder="graph TD&#10;  A[Start] --> B[End]"
        spellcheck="false"
        @input="onInput"
      />

      <!-- Preview pane -->
      <div
        v-if="mode !== 'code'"
        class="flex-1 flex items-center justify-center p-4 bg-surface-1 overflow-auto"
        :class="mode === 'split' ? 'w-1/2' : 'w-full'"
      >
        <div v-if="error" class="text-[12px] text-red-500 font-mono whitespace-pre-wrap">{{ error }}</div>
        <div v-else-if="svg" v-html="svg" class="mermaid-preview max-w-full" />
        <div v-else class="text-[12px] text-text-3">Preview will appear here</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import mermaid from 'mermaid'

mermaid.initialize({ startOnLoad: false, theme: 'neutral', securityLevel: 'loose' })

const props = defineProps({
  modelValue: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const mode = ref('split')
const localVal = ref(props.modelValue ?? '')
const svg = ref('')
const error = ref('')
let renderCounter = 0

watch(() => props.modelValue, val => {
  if (val !== localVal.value) localVal.value = val ?? ''
})

async function render(source) {
  if (!source?.trim()) {
    svg.value = ''
    error.value = ''
    return
  }
  const id = `mermaid-editor-${++renderCounter}`
  try {
    const result = await mermaid.render(id, source)
    svg.value = result.svg
    error.value = ''
  } catch (e) {
    svg.value = ''
    error.value = e?.message ?? String(e)
  }
}

function onInput() {
  emit('update:modelValue', localVal.value)
  render(localVal.value)
}

onMounted(() => render(localVal.value))
</script>

<style scoped>
.mermaid-preview :deep(svg) {
  max-width: 100%;
  height: auto;
}
</style>
