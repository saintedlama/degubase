<template>
  <Flapout align="left">
    <template #trigger="{ open, toggle, setAnchor }">
      <button
        :ref="setAnchor"
        class="inline-flex items-center gap-1.5 h-6 px-2 text-[11px] font-medium rounded border transition-colors cursor-pointer"
        :class="filters.length > 0
          ? 'text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 border-brand-300 dark:border-brand-700/50'
          : 'text-text-2 bg-transparent border-border-1 hover:text-text-1 hover:bg-border-1'"
        @click.stop="toggle"
      >
        <RiFilterLine size="12" />
        Filter
        <span
          v-if="filters.length"
          class="inline-flex items-center justify-center min-w-4 h-4 px-1 rounded-full bg-brand-100 dark:bg-brand-900/50 text-brand-600 dark:text-brand-400 text-[10px] font-bold"
        >{{ filters.length }}</span>
      </button>
    </template>

    <PanelCard class="p-3 w-80">
      <div class="text-[10px] font-semibold text-text-3 uppercase tracking-wide mb-2">Filters</div>

      <div v-if="filters.length === 0" class="text-[11px] text-text-3 italic mb-2">No filters applied.</div>

      <div v-for="f in filters" :key="f.id" class="mb-2">
        <!-- Main row: column, operator, value (or spacer), delete -->
        <div class="flex items-center gap-1">
          <select
            class="filter-ctl flex-1 min-w-0"
            :value="f.col"
            @change="changeCol(f, $event.target.value)"
          >
            <option v-for="col in sortedColumns" :key="col.code" :value="col.code">{{ col.name }}</option>
          </select>

          <select
            class="filter-ctl w-28 shrink-0"
            :value="f.op"
            @change="changeOp(f, $event.target.value)"
          >
            <option v-for="op in getColOps(colByCode(f.col))" :key="op.value" :value="op.value">{{ op.label }}</option>
          </select>

          <template v-if="opNeedsValue(f.op) && !(isMultiValueOp(f.op) && isSelectType(colByCode(f.col)))">
            <!-- Single-select for is/is_not operators -->
            <select
              v-if="isSelectType(colByCode(f.col))"
              class="filter-ctl w-24 shrink-0"
              :value="f.value"
              @change="changeVal(f, $event.target.value)"
            >
              <option value="">—</option>
              <option v-for="c in getColumnChoices(colByCode(f.col))" :key="c" :value="c">{{ c }}</option>
            </select>
            <!-- Date input with "now" toggle -->
            <div
              v-else-if="isDateType(colByCode(f.col))"
              class="flex items-center gap-0.5 w-28 shrink-0"
            >
              <span
                v-if="f.value === 'now'"
                class="flex-1 text-center text-[10px] font-semibold text-brand-600 dark:text-brand-400 border border-brand-300 dark:border-brand-700/50 rounded bg-brand-50 dark:bg-brand-900/30 px-1 leading-5"
              >now</span>
              <input
                v-else
                :value="f.value"
                :type="filterInputType(colByCode(f.col))"
                class="filter-val flex-1 min-w-0"
                placeholder="Value"
                @input="changeVal(f, $event.target.value)"
              />
              <button
                class="filter-now-btn shrink-0"
                :class="{ active: f.value === 'now' }"
                :title="f.value === 'now' ? 'Clear now' : 'Use current date/time'"
                @click="changeVal(f, f.value === 'now' ? '' : 'now')"
              >now</button>
            </div>
            <!-- Default text/number input -->
            <input
              v-else
              :value="f.value"
              :type="filterInputType(colByCode(f.col))"
              class="filter-val w-24 shrink-0"
              placeholder="Value"
              @input="changeVal(f, $event.target.value)"
            />
          </template>
          <div v-else-if="isMultiValueOp(f.op) && isSelectType(colByCode(f.col))" class="flex-1" />

          <button
            class="flex items-center justify-center w-5 h-5 shrink-0 bg-transparent border-none text-text-2 cursor-pointer rounded hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/40 transition-colors"
            @click="remove(f.id)"
          >
            <RiCloseLine size="12" />
          </button>
        </div>

        <!-- Pill chips row for in/not_in operators -->
        <div
          v-if="isMultiValueOp(f.op) && isSelectType(colByCode(f.col))"
          class="flex flex-wrap gap-1 mt-1.5 pl-0.5"
        >
          <button
            v-for="c in getColumnChoices(colByCode(f.col))"
            :key="c"
            class="filter-pill"
            :class="{ active: parseJsonArray(f.value).includes(c) }"
            @click="toggleChip(f, c)"
          >{{ c }}</button>
        </div>
      </div>

      <div class="flex items-center gap-2 mt-1">
        <button
          class="inline-flex items-center gap-1 text-[11px] font-medium text-text-2 bg-transparent border-none cursor-pointer px-0 rounded hover:text-brand-600 dark:hover:text-brand-400 transition-colors"
          @click="add"
        >
          <RiAddLine size="12" />
          Add filter
        </button>
        <button
          v-if="filters.length"
          class="ml-auto text-[11px] text-text-2 bg-transparent border-none cursor-pointer hover:text-red-500 transition-colors"
          @click="$emit('update:filters', [])"
        >
          Clear all
        </button>
      </div>
    </PanelCard>
  </Flapout>
</template>

<script setup>
import { computed } from 'vue'
import Flapout from '../../foundation/Flapout.vue'
import { getColOps, opNeedsValue, isMultiValueOp, isSelectType, isDateType, filterInputType, getColumnChoices } from './filterHelpers.js'
import { RiFilterLine, RiCloseLine, RiAddLine } from '@remixicon/vue'
import PanelCard from '../../foundation/PanelCard.vue'

const props = defineProps({
  columns: { type: Array, default: () => [] },
  filters: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:filters'])

const sortedColumns = computed(() =>
  [...props.columns].sort((a, b) => a.position - b.position)
)

function colByCode(colCode) {
  return props.columns.find(c => c.code === colCode) ?? null
}

function changeCol(f, colCode) {
  const col = colByCode(colCode)
  const ops = getColOps(col)
  emit('update:filters', props.filters.map(x =>
    x.id === f.id ? { ...x, col: colCode, op: ops[0]?.value ?? 'contains', value: '' } : x
  ))
}

function changeOp(f, op) {
  let value = opNeedsValue(op) ? f.value : ''
  // Reset when switching between single-value and multi-value ops to avoid invalid values
  if (isMultiValueOp(op) !== isMultiValueOp(f.op)) value = ''
  emit('update:filters', props.filters.map(x =>
    x.id === f.id ? { ...x, op, value } : x
  ))
}

function changeVal(f, value) {
  emit('update:filters', props.filters.map(x =>
    x.id === f.id ? { ...x, value } : x
  ))
}

function toggleChip(f, choice) {
  const current = parseJsonArray(f.value)
  const next = current.includes(choice) ? current.filter(v => v !== choice) : [...current, choice]
  changeVal(f, JSON.stringify(next))
}

function parseJsonArray(val) {
  if (!val) return []
  try { return JSON.parse(val) } catch { return [] }
}

function remove(id) {
  emit('update:filters', props.filters.filter(f => f.id !== id))
}

function add() {
  const col = sortedColumns.value[0]
  if (!col) return
  const ops = getColOps(col)
  emit('update:filters', [...props.filters, {
    id: crypto.randomUUID(),
    col: col.code,
    op: ops[0]?.value ?? 'contains',
    value: '',
  }])
}
</script>

<style scoped>
.filter-ctl {
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
.filter-val {
  height: 24px;
  padding: 0 6px;
  font-size: 11px;
  border-radius: 4px;
  border: 1px solid var(--border-1);
  background: var(--surface-2);
  color: var(--text-1);
  outline: none;
}
.filter-ctl:focus, .filter-val:focus {
  border-color: var(--color-brand-600);
}
.filter-pill {
  height: 20px;
  padding: 0 7px;
  font-size: 10px;
  font-weight: 500;
  border-radius: 10px;
  border: 1px solid var(--border-1);
  background: var(--surface-2);
  color: var(--text-2);
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.1s, color 0.1s, border-color 0.1s;
}
.filter-pill:hover {
  border-color: var(--color-brand-600);
  color: var(--color-brand-600);
}
.filter-pill.active {
  background: color-mix(in srgb, var(--color-brand-600) 12%, transparent);
  border-color: var(--color-brand-600);
  color: var(--color-brand-600);
  font-weight: 600;
}
.filter-now-btn {
  height: 20px;
  padding: 0 4px;
  font-size: 10px;
  font-weight: 500;
  border-radius: 3px;
  border: 1px solid var(--border-1);
  background: var(--surface-2);
  color: var(--text-2);
  cursor: pointer;
  white-space: nowrap;
}
.filter-now-btn:hover {
  border-color: var(--color-brand-600);
  color: var(--color-brand-600);
}
.filter-now-btn.active {
  border-color: var(--color-brand-600);
  background: color-mix(in srgb, var(--color-brand-600) 15%, transparent);
  color: var(--color-brand-600);
}
</style>
