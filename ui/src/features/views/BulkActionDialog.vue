<template>
  <Backdrop @dismiss="$emit('cancel')">
    <div class="bg-surface-1 border border-border-1 rounded-xl w-[540px] max-w-[calc(100vw-32px)] shadow-xl flex flex-col overflow-hidden max-h-[90vh]">

      <!-- Header -->
      <div class="flex items-center justify-between px-5 pt-5 pb-0 shrink-0">
        <h3 class="text-[15px] font-bold text-text-1">Bulk edit records</h3>
        <button
          class="flex items-center justify-center w-6.5 h-6.5 bg-transparent border-none text-text-2 cursor-pointer rounded-md hover:text-text-1 hover:bg-border-1 transition-colors"
          @click="$emit('cancel')"
        ><RiCloseLine size="16" /></button>
      </div>

      <div class="overflow-y-auto flex-1 min-h-0">
        <!-- Filters section -->
        <section class="px-5 pt-4 pb-0">
          <div class="text-[11px] font-semibold text-text-3 uppercase tracking-wide mb-2">Filter (which records to update)</div>

          <div v-if="filters.length === 0" class="text-[11px] text-text-3 italic mb-1">No filters — all records will be updated.</div>

          <div v-for="f in filters" :key="f.id" class="mb-1.5">
            <div class="flex items-center gap-1">
              <select class="bulk-ctl flex-1 min-w-0" :value="f.col" @change="changeFilterCol(f, $event.target.value)">
                <option v-for="col in sortedColumns" :key="col.code" :value="col.code">{{ col.name }}</option>
              </select>
              <select class="bulk-ctl w-28 shrink-0" :value="f.op" @change="changeFilterOp(f, $event.target.value)">
                <option v-for="op in getColOps(colByCode(f.col))" :key="op.value" :value="op.value">{{ op.label }}</option>
              </select>
              <template v-if="opNeedsValue(f.op)">
                <select
                  v-if="isSelectType(colByCode(f.col)) && !isMultiValueOp(f.op)"
                  class="bulk-ctl w-24 shrink-0"
                  :value="f.value"
                  @change="changeFilterVal(f, $event.target.value)"
                >
                  <option value="">—</option>
                  <option v-for="c in getColumnChoices(colByCode(f.col))" :key="c" :value="c">{{ c }}</option>
                </select>
                <input
                  v-else-if="!isMultiValueOp(f.op)"
                  :value="f.value"
                  :type="filterInputType(colByCode(f.col))"
                  class="bulk-ctl w-24 shrink-0"
                  placeholder="Value"
                  @input="changeFilterVal(f, $event.target.value)"
                />
                <div v-else class="flex-1" />
              </template>
              <button
                class="flex items-center justify-center w-5 h-5 shrink-0 bg-transparent border-none text-text-2 cursor-pointer rounded hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/40 transition-colors"
                @click="removeFilter(f.id)"
              ><RiCloseLine size="12" /></button>
            </div>
            <div v-if="isMultiValueOp(f.op) && isSelectType(colByCode(f.col))" class="flex flex-wrap gap-1 mt-1.5 pl-0.5">
              <button
                v-for="c in getColumnChoices(colByCode(f.col))"
                :key="c"
                class="bulk-pill"
                :class="{ active: parseJsonArray(f.value).includes(c) }"
                @click="toggleFilterChip(f, c)"
              >{{ c }}</button>
            </div>
          </div>

          <button class="inline-flex items-center gap-1 text-[11px] font-medium text-text-2 bg-transparent border-none cursor-pointer px-0 rounded hover:text-brand-600 dark:hover:text-brand-400 transition-colors mt-1 mb-3" @click="addFilter">
            <RiAddLine size="12" />
            Add filter
          </button>
        </section>

        <div class="mx-5 border-t border-border-1" />

        <!-- Updates section -->
        <section class="px-5 pt-4 pb-0">
          <div class="text-[11px] font-semibold text-text-3 uppercase tracking-wide mb-2">Set fields</div>

          <div v-if="updates.length === 0" class="text-[11px] text-text-3 italic mb-1">No fields set — add a field to update.</div>

          <div v-for="u in updates" :key="u.id" class="flex items-center gap-1 mb-1.5">
            <select class="bulk-ctl flex-1 min-w-0" :value="u.col" @change="changeUpdateCol(u, $event.target.value)">
              <option v-for="col in editableColumns" :key="col.code" :value="col.code">{{ col.name }}</option>
            </select>
            <span class="text-[11px] text-text-3 shrink-0">→</span>
            <!-- Value input depending on column type -->
            <select
              v-if="isSelectType(colByCode(u.col))"
              class="bulk-ctl w-36 shrink-0"
              :value="u.value"
              @change="changeUpdateVal(u, $event.target.value)"
            >
              <option value="">—</option>
              <option v-for="c in getColumnChoices(colByCode(u.col))" :key="c" :value="c">{{ c }}</option>
            </select>
            <input
              v-else-if="isCheckboxType(colByCode(u.col))"
              type="checkbox"
              :checked="u.value === true || u.value === 'true'"
              class="w-4 h-4 accent-brand-600 cursor-pointer shrink-0"
              @change="changeUpdateVal(u, $event.target.checked)"
            />
            <input
              v-else
              :value="u.value"
              :type="filterInputType(colByCode(u.col))"
              class="bulk-ctl w-36 shrink-0"
              placeholder="New value"
              @input="changeUpdateVal(u, $event.target.value)"
            />
            <button
              class="flex items-center justify-center w-5 h-5 shrink-0 bg-transparent border-none text-text-2 cursor-pointer rounded hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/40 transition-colors"
              @click="removeUpdate(u.id)"
            ><RiCloseLine size="12" /></button>
          </div>

          <button class="inline-flex items-center gap-1 text-[11px] font-medium text-text-2 bg-transparent border-none cursor-pointer px-0 rounded hover:text-brand-600 dark:hover:text-brand-400 transition-colors mt-1 mb-4" @click="addUpdate">
            <RiAddLine size="12" />
            Add field
          </button>
        </section>
      </div>

      <!-- Footer -->
      <div class="shrink-0 flex items-center gap-2 px-5 py-3.5 border-t border-border-1 bg-surface-2">
        <!-- Preview count -->
        <button
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-[12px] font-medium text-text-2 bg-transparent border border-border-1 cursor-pointer hover:text-text-1 hover:bg-border-1 transition-colors disabled:opacity-50"
          :disabled="loading"
          @click="preview"
        >
          <RiSearchLine size="12" />
          Preview
        </button>
        <span v-if="previewCount !== null" class="text-[12px] text-text-2">
          <span class="font-semibold text-text-1">{{ previewCount }}</span> record{{ previewCount !== 1 ? 's' : '' }} will be updated
        </span>

        <div class="ml-auto flex items-center gap-2">
          <button
            class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-surface-2 text-text-2 border border-zinc-300 dark:border-[#3c3c3c] cursor-pointer hover:bg-border-1 transition-colors"
            @click="$emit('cancel')"
          >Cancel</button>
          <button
            :disabled="updates.length === 0 || loading"
            class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 disabled:opacity-40 disabled:cursor-default transition-colors"
            @click="execute"
          >
            <span v-if="loading">Updating…</span>
            <span v-else>Update records</span>
          </button>
        </div>
      </div>
    </div>
  </Backdrop>
</template>

<script setup>
import { ref, computed } from 'vue'
import { RiCloseLine, RiAddLine, RiSearchLine } from '@remixicon/vue'
import Backdrop from '../../foundation/Backdrop.vue'
import { api } from '../../api/client.js'
import { getColOps, opNeedsValue, isMultiValueOp, isSelectType, isDateType, filterInputType, getColumnChoices } from './filterHelpers.js'

const props = defineProps({
  workspaceCode: { type: String, required: true },
  tableCode:     { type: String, required: true },
  columns:       { type: Array, default: () => [] },
  initialFilters:{ type: Array, default: () => [] },
})
const emit = defineEmits(['done', 'cancel'])

const VIRTUAL_TYPES = ['created-at', 'updated-at', 'image', 'file']

const sortedColumns = computed(() =>
  [...props.columns].sort((a, b) => a.position - b.position)
)
const editableColumns = computed(() =>
  sortedColumns.value.filter(c => !VIRTUAL_TYPES.includes(c.type))
)

function colByCode(code) {
  return props.columns.find(c => c.code === code) ?? null
}
function isCheckboxType(col) {
  return col?.type === 'checkbox'
}

// ── Filters ───────────────────────────────────────────────────────────────────

const filters = ref(props.initialFilters.map(f => ({ ...f, id: f.id ?? crypto.randomUUID() })))

function addFilter() {
  const col = sortedColumns.value[0]
  if (!col) return
  const ops = getColOps(col)
  filters.value.push({ id: crypto.randomUUID(), col: col.code, op: ops[0]?.value ?? 'contains', value: '' })
}
function removeFilter(id) {
  filters.value = filters.value.filter(f => f.id !== id)
}
function changeFilterCol(f, colCode) {
  const col = colByCode(colCode)
  const ops = getColOps(col)
  const idx = filters.value.findIndex(x => x.id === f.id)
  if (idx >= 0) filters.value[idx] = { ...f, col: colCode, op: ops[0]?.value ?? 'contains', value: '' }
}
function changeFilterOp(f, op) {
  let value = opNeedsValue(op) ? f.value : ''
  if (isMultiValueOp(op) !== isMultiValueOp(f.op)) value = ''
  const idx = filters.value.findIndex(x => x.id === f.id)
  if (idx >= 0) filters.value[idx] = { ...f, op, value }
}
function changeFilterVal(f, value) {
  const idx = filters.value.findIndex(x => x.id === f.id)
  if (idx >= 0) filters.value[idx] = { ...f, value }
}
function toggleFilterChip(f, choice) {
  const current = parseJsonArray(f.value)
  const next = current.includes(choice) ? current.filter(v => v !== choice) : [...current, choice]
  changeFilterVal(f, JSON.stringify(next))
}
function parseJsonArray(val) {
  if (!val) return []
  try { return JSON.parse(val) } catch { return [] }
}

// ── Updates ───────────────────────────────────────────────────────────────────

const updates = ref([])

function addUpdate() {
  const col = editableColumns.value[0]
  if (!col) return
  updates.value.push({ id: crypto.randomUUID(), col: col.code, value: '' })
}
function removeUpdate(id) {
  updates.value = updates.value.filter(u => u.id !== id)
}
function changeUpdateCol(u, colCode) {
  const idx = updates.value.findIndex(x => x.id === u.id)
  if (idx >= 0) updates.value[idx] = { ...u, col: colCode, value: '' }
}
function changeUpdateVal(u, value) {
  const idx = updates.value.findIndex(x => x.id === u.id)
  if (idx >= 0) updates.value[idx] = { ...u, value }
}

function buildData() {
  const data = {}
  for (const u of updates.value) {
    data[u.col] = u.value
  }
  return data
}

// ── Preview & execute ─────────────────────────────────────────────────────────

const previewCount = ref(null)
const loading = ref(false)

async function preview() {
  loading.value = true
  try {
    const res = await api.bulkPatch(props.workspaceCode, props.tableCode, filters.value, {}, true)
    previewCount.value = res.count
  } finally {
    loading.value = false
  }
}

async function execute() {
  loading.value = true
  try {
    const res = await api.bulkPatch(props.workspaceCode, props.tableCode, filters.value, buildData(), false)
    emit('done', res.updated)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.bulk-ctl {
  height: 24px;
  font-size: 11px;
  padding: 0 6px;
  border-radius: 4px;
  border: 1px solid var(--border-1);
  background: var(--surface-2);
  color: var(--text-1);
  outline: none;
}
.bulk-ctl:focus {
  border-color: var(--color-brand-400);
}
.bulk-pill {
  display: inline-flex;
  align-items: center;
  padding: 0 8px;
  height: 20px;
  font-size: 11px;
  border-radius: 9999px;
  border: 1px solid var(--border-1);
  background: var(--surface-2);
  color: var(--text-2);
  cursor: pointer;
  transition: all 0.1s;
}
.bulk-pill.active {
  background: var(--color-brand-100);
  border-color: var(--color-brand-300);
  color: var(--color-brand-700);
}
.dark .bulk-pill.active {
  background: color-mix(in srgb, var(--color-brand-900) 30%, transparent);
  border-color: color-mix(in srgb, var(--color-brand-700) 50%, transparent);
  color: var(--color-brand-400);
}
</style>
