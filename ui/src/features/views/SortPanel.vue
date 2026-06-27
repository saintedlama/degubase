<template>
  <Flapout align="left">
    <template #trigger="{ open, toggle, setAnchor }">
      <button
        :ref="setAnchor"
        class="inline-flex items-center gap-1.5 h-6 px-2 text-[11px] font-medium rounded border transition-colors cursor-pointer"
        :class="sorts.length > 0
          ? 'text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 border-brand-300 dark:border-brand-700/50'
          : 'text-text-2 bg-transparent border-border-1 hover:text-text-1 hover:bg-border-1'"
        @click.stop="toggle"
      >
        <RiArrowUpDownLine size="12" />
        Sort
        <span
          v-if="sorts.length"
          class="inline-flex items-center justify-center min-w-4 h-4 px-1 rounded-full bg-brand-100 dark:bg-brand-900/50 text-brand-600 dark:text-brand-400 text-[10px] font-bold"
        >{{ sorts.length }}</span>
      </button>
    </template>

    <PanelCard class="p-3 w-72">
      <div class="text-[10px] font-semibold text-text-3 uppercase tracking-wide mb-2">Sort</div>

      <div v-if="sorts.length === 0" class="text-[11px] text-text-3 italic mb-2">No sort applied.</div>

      <div v-for="(s, idx) in sorts" :key="idx" class="flex items-center gap-1 mb-2">
        <span class="text-[10px] text-text-3 w-8 shrink-0">{{ idx === 0 ? 'By' : 'Then' }}</span>

        <select
          class="sort-ctl flex-1 min-w-0"
          :value="s.col"
          @change="updateSort(idx, 'col', $event.target.value)"
        >
          <option v-for="col in sortableColumns" :key="col.code" :value="col.code">{{ col.name }}</option>
        </select>

        <select
          class="sort-ctl w-24 shrink-0"
          :value="s.dir"
          @change="updateSort(idx, 'dir', $event.target.value)"
        >
          <option value="asc">{{ isSelectCol(s.col) ? 'Defined' : 'Asc' }}</option>
          <option value="desc">{{ isSelectCol(s.col) ? 'Reversed' : 'Desc' }}</option>
        </select>

        <button
          class="flex items-center justify-center w-5 h-5 shrink-0 bg-transparent border-none text-text-2 cursor-pointer rounded hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/40 transition-colors"
          @click="removeSort(idx)"
        >
          <RiCloseLine size="12" />
        </button>
      </div>

      <div class="flex items-center gap-2 mt-1">
        <button
          v-if="availableColumns.length > 0"
          class="inline-flex items-center gap-1 text-[11px] font-medium text-text-2 bg-transparent border-none cursor-pointer px-0 rounded hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
          @click="addSort"
        >
          <RiAddLine size="12" />
          Add sort level
        </button>
        <button
          v-if="sorts.length"
          class="ml-auto text-[11px] text-text-2 bg-transparent border-none cursor-pointer hover:text-red-500 transition-colors"
          @click="$emit('update:sorts', [])"
        >
          Clear
        </button>
      </div>
    </PanelCard>
  </Flapout>
</template>

<script setup>
import { computed } from 'vue'
import Flapout from '../../foundation/Flapout.vue'
import { RiArrowUpDownLine, RiCloseLine, RiAddLine } from '@remixicon/vue'
import PanelCard from '../../foundation/PanelCard.vue'

const SORTABLE_TYPES = [
  'text', 'long-text', 'number', 'currency', 'percent', 'rating',
  'date', 'datetime', 'created-at', 'updated-at',
  'single-select', 'multi-select', 'checkbox',
]

const props = defineProps({
  columns: { type: Array, default: () => [] },
  sorts: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:sorts'])

const sortableColumns = computed(() =>
  [...props.columns]
    .filter(c => SORTABLE_TYPES.includes(c.type))
    .sort((a, b) => a.position - b.position)
)

const usedColCodes = computed(() => new Set(props.sorts.map(s => s.col)))
const availableColumns = computed(() =>
  sortableColumns.value.filter(c => !usedColCodes.value.has(c.code))
)

function isSelectCol(colCode) {
  const t = props.columns.find(c => c.code === colCode)?.type
  return t === 'single-select' || t === 'multi-select'
}

function updateSort(idx, key, value) {
  const updated = props.sorts.map((s, i) => i === idx ? { ...s, [key]: value } : s)
  emit('update:sorts', updated)
}

function removeSort(idx) {
  emit('update:sorts', props.sorts.filter((_, i) => i !== idx))
}

function addSort() {
  const col = availableColumns.value[0]
  if (!col) return
  emit('update:sorts', [...props.sorts, { col: col.code, dir: 'asc' }])
}
</script>

<style scoped>
.sort-ctl {
  height: 24px;
  padding: 0 5px;
  font-size: 11px;
  border-radius: 4px;
  border: 1px solid var(--border-1);
  background: var(--surface-2);
  color: var(--text-1);
  outline: none;
  cursor: pointer;
}
.sort-ctl:focus {
  border-color: var(--color-brand-600);
}
</style>
