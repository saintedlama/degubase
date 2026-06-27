<template>
  <div class="px-3 pt-3 pb-1 flex flex-col gap-2.5">
    <div>
      <label class="popup-label">Name</label>
      <input
        ref="nameInput"
        v-model="form.name"
        data-testid="column-name-input"
        class="popup-input"
        placeholder="Column name"
        @keydown.enter.prevent="save"
        @keydown.escape="$emit('cancel')"
      />
    </div>
    <div>
      <label class="popup-label">Type</label>
      <select
        v-model="form.type"
        class="popup-select"
        :disabled="!isNew && family === 'system'"
      >
        <!-- Create mode: all types in groups -->
        <template v-if="isNew">
          <optgroup v-for="group in TYPE_GROUPS" :key="group.label" :label="group.label">
            <option v-for="t in group.types" :key="t.value" :value="t.value">{{ t.label }}</option>
          </optgroup>
        </template>
        <!-- Edit mode: only types in the same family -->
        <template v-else>
          <option v-for="t in compatibleTypes(family)" :key="t.value" :value="t.value">{{ t.label }}</option>
        </template>
      </select>
    </div>

    <div v-if="family === 'link'" class="flex flex-col gap-2.5">
      <div class="flex flex-col gap-1.5">
        <label class="popup-label">Target table</label>
        <select v-model="form.targetTableCode" class="popup-select" @change="onTargetTableChange">
          <option value="">— same table (hierarchy) —</option>
          <option v-for="t in tables" :key="t.code" :value="t.code">{{ t.name }}</option>
        </select>
      </div>
      <div class="flex flex-col gap-1.5">
        <label class="popup-label">Display column</label>
        <select v-model="form.displayColumnCode" class="popup-select" :disabled="!targetTableColumns.length">
          <option value="">— auto (first text column) —</option>
          <option v-for="c in targetTableColumns" :key="c.code" :value="c.code">{{ c.name }}</option>
        </select>
      </div>
    </div>

    <div v-if="family === 'select'" class="flex flex-col gap-2">
      <!-- Palette selector -->
      <div>
        <label class="popup-label">Palette</label>
        <div class="flex flex-col gap-1.5">
          <div v-for="group in PALETTE_GROUPS" :key="group.label" class="flex flex-col gap-0.5">
            <span class="text-[9px] font-bold tracking-widest uppercase text-text-3">{{ group.label }}</span>
            <div class="flex gap-1">
              <button
                v-for="(palette, li) in group.palettes"
                :key="li"
                class="flex gap-px p-1 rounded border transition-colors cursor-pointer"
                :class="activePalette === group.offset + li
                  ? 'border-brand-500 bg-brand-50 dark:bg-brand-900/20'
                  : 'border-border-1 hover:border-zinc-400 dark:hover:border-[#5a5a5a]'"
                :title="group.names[li]"
                @click="applyPalette(group.offset + li)"
              >
                <span
                  v-for="color in palette.slice(0, 4)"
                  :key="color"
                  class="block w-2.5 h-2.5 rounded-sm"
                  :style="{ background: color }"
                />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Choices list -->
      <div>
        <label class="popup-label">Choices</label>
        <div class="flex flex-col gap-1 max-h-48 overflow-y-auto">
          <template v-for="(item, idx) in form.choiceItems" :key="idx">
            <div class="flex items-center gap-1">
              <button
                class="w-5 h-5 rounded shrink-0 border border-black/10 cursor-pointer hover:scale-110 transition-transform"
                :style="{ background: item.color }"
                @click="colorPickerIdx = colorPickerIdx === idx ? -1 : idx"
              />
              <input
                v-model="item.value"
                class="choice-name-input popup-input flex-1 py-0.75 text-[12px]"
                placeholder="Option name"
              />
              <button class="choice-reorder-btn" :disabled="idx === 0" @click="moveChoice(idx, 'up')">
                <RiArrowUpSLine size="12" />
              </button>
              <button class="choice-reorder-btn" :disabled="idx === form.choiceItems.length - 1" @click="moveChoice(idx, 'down')">
                <RiArrowDownSLine size="12" />
              </button>
              <button
                class="flex items-center justify-center w-4 h-4 text-zinc-300 dark:text-[#5a5a5a] hover:text-red-500 bg-transparent border-none cursor-pointer rounded shrink-0 transition-colors"
                @click="removeChoice(idx)"
              >
                <RiCloseLine size="11" />
              </button>
            </div>

            <div v-if="colorPickerIdx === idx" class="flex flex-wrap gap-0.5 p-1.5 ml-6 bg-zinc-50 dark:bg-[#2a2a2a] rounded border border-border-1">
              <button
                v-for="color in ALL_PALETTE_COLORS"
                :key="color"
                class="w-4 h-4 rounded border cursor-pointer transition-all hover:scale-110"
                :class="item.color === color ? 'border-zinc-500 dark:border-zinc-300 scale-110' : 'border-transparent'"
                :style="{ background: color }"
                @click="setChoiceColor(idx, color)"
              />
            </div>
          </template>
        </div>

        <button
          class="flex items-center gap-1 mt-1.5 text-[11px] text-text-2 hover:text-text-1 bg-transparent border-none cursor-pointer px-0.5 py-0.5 rounded transition-colors"
          @click="addChoice"
        >
          <RiAddLine size="12" />
          Add option
        </button>
      </div>
    </div>
  </div>

  <div class="flex items-center justify-end gap-2 px-3 py-2.5 border-t border-zinc-100 dark:border-[#3c3c3c] mt-1">
    <button class="popup-btn-cancel" @click="$emit('cancel')">Cancel</button>
    <SpinnerButton
      data-testid="save-column-btn"
      class="popup-btn-save"
      :loading="saving"
      :disabled="!form.name.trim()"
      :idle="isNew ? 'Add Column' : 'Save'"
      busy="Saving…"
      @click="save"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, nextTick, onMounted } from 'vue'
import { api } from '../../api/client.js'
import { PALETTE_GROUPS, PALETTES, ALL_PALETTE_COLORS } from '../views/palettes.js'
import { TYPE_FAMILIES, TYPE_GROUPS, compatibleTypes } from './columnTypes.js'
import { RiCloseLine, RiAddLine, RiArrowUpSLine, RiArrowDownSLine } from '@remixicon/vue'
import SpinnerButton from '../../foundation/SpinnerButton.vue'

const props = defineProps({
  column:        { type: Object, default: null },
  workspaceCode: { type: String, required: true },
  tableCode:     { type: String, required: true },
})
const emit = defineEmits(['saved', 'cancel'])

const nameInput     = ref(null)
const form          = reactive({ name: '', type: 'text', choiceItems: [], targetTableCode: '', displayColumnCode: '' })
const activePalette = ref(0)
const tables        = ref([])
const targetTableColumns = ref([])
const colorPickerIdx = ref(-1)
const saving        = ref(false)
let originalDisplayColumnCode = ''

const isNew  = computed(() => !props.column)
const family = computed(() => TYPE_FAMILIES[form.type] ?? 'text')

async function loadTargetTableColumns(code) {
  if (!code) { targetTableColumns.value = []; return }
  try {
    targetTableColumns.value = await api.listColumns(props.workspaceCode, code)
  } catch { targetTableColumns.value = [] }
}

function onTargetTableChange() {
  form.displayColumnCode = ''
  loadTargetTableColumns(form.targetTableCode)
}

async function fetchAllRows(workspaceCode, tableCode) {
  const rows = []
  let page = 1
  while (true) {
    const res = await api.listRows(workspaceCode, tableCode, { page, pageSize: 100 })
    rows.push(...(res?.data ?? []))
    if (rows.length >= (res?.total ?? 0)) break
    page++
  }
  return rows
}

async function refreshRowLinkLabels() {
  const colCode = props.column.code
  const targetCode = form.targetTableCode || props.tableCode
  const textTypes = ['text', 'long-text', 'markdown', 'email', 'url', 'number', 'currency', 'percent']

  const [sourceRows, targetRows, targetCols] = await Promise.all([
    fetchAllRows(props.workspaceCode, props.tableCode),
    fetchAllRows(props.workspaceCode, targetCode),
    api.listColumns(props.workspaceCode, targetCode),
  ])

  const targetRowMap = new Map(targetRows.map(r => [r.id, r]))

  let effectiveLabelCode = form.displayColumnCode || null
  if (!effectiveLabelCode) {
    const labelCol = [...targetCols]
      .sort((a, b) => a.position - b.position)
      .find(c => textTypes.includes(c.type))
    effectiveLabelCode = labelCol?.code ?? null
  }

  function computeLabel(targetRow) {
    if (effectiveLabelCode) {
      const v = targetRow.data?.[effectiveLabelCode]
      if (v != null && v !== '') return String(v)
    }
    for (const v of Object.values(targetRow.data ?? {})) {
      if (v && typeof v === 'string') return v
    }
    return `Row #${targetRow.id}`
  }

  await Promise.all(
    sourceRows
      .filter(row => row.data?.[colCode]?.id)
      .map(row => {
        const linkVal = row.data[colCode]
        const targetRow = targetRowMap.get(linkVal.id)
        if (!targetRow) return null
        const newLabel = computeLabel(targetRow)
        if (newLabel === linkVal.label) return null
        return api.patchRow(props.workspaceCode, props.tableCode, row.id, {
          data: { [colCode]: { ...linkVal, label: newLabel } },
        })
      })
      .filter(Boolean)
  )
}

onMounted(async () => {
  if (props.column) {
    const col  = props.column
    form.name  = col.name
    form.type  = col.type
    const opts = col.options
      ? (typeof col.options === 'string' ? JSON.parse(col.options) : col.options)
      : null
    const pi = opts?.palette ?? 0
    activePalette.value = pi
    const choices = opts?.choices ?? []
    const colors  = opts?.choiceColors ?? {}
    form.choiceItems = choices.map((c, i) => ({
      value: c,
      color: colors[c] || PALETTES[pi][i % PALETTES[pi].length],
    }))
    form.targetTableCode = opts?.targetTableCode ?? ''
    const savedDisplayCode = opts?.displayColumnCode ?? ''

    await Promise.all([
      form.targetTableCode
        ? loadTargetTableColumns(form.targetTableCode)
        : Promise.resolve(),
      api.listTables(props.workspaceCode).then(t => { tables.value = t }).catch(() => {}),
    ])

    // Set after options are in DOM so Vue correctly renders the selected <option>
    form.displayColumnCode = savedDisplayCode
    originalDisplayColumnCode = savedDisplayCode

    nextTick(() => nameInput.value?.select())
  } else {
    try {
      tables.value = await api.listTables(props.workspaceCode)
    } catch { /* non-fatal */ }
    nextTick(() => nameInput.value?.focus())
  }
})

async function save() {
  const name = form.name.trim()
  if (!name || saving.value) return
  saving.value = true
  try {
    const payload = { name, type: form.type }
    if (family.value === 'select') {
      const items = form.choiceItems.filter(i => i.value.trim())
      const choices = items.map(i => i.value.trim())
      const choiceColors = {}
      items.forEach(i => { choiceColors[i.value.trim()] = i.color })
      payload.options = { choices, choiceColors, palette: activePalette.value }
    }
    if (family.value === 'link') {
      payload.options = {
        targetTableCode: form.targetTableCode || props.tableCode,
        ...(form.displayColumnCode ? { displayColumnCode: form.displayColumnCode } : {}),
      }
    }
    if (isNew.value) {
      const newCol = await api.createColumn(props.workspaceCode, props.tableCode, payload)
      emit('saved', newCol)
    } else {
      await api.updateColumn(props.workspaceCode, props.tableCode, props.column.code, payload)
      if (family.value === 'link' && form.displayColumnCode !== originalDisplayColumnCode) {
        await refreshRowLinkLabels()
      }
      emit('saved')
    }
  } catch (e) {
    alert(e.message)
  } finally {
    saving.value = false
  }
}

function applyPalette(pi) {
  activePalette.value = pi
  colorPickerIdx.value = -1
  form.choiceItems.forEach((item, i) => {
    item.color = PALETTES[pi][i % PALETTES[pi].length]
  })
}

function addChoice() {
  const pi = activePalette.value
  const nextColor = PALETTES[pi][form.choiceItems.length % PALETTES[pi].length]
  colorPickerIdx.value = -1
  form.choiceItems.push({ value: '', color: nextColor })
  nextTick(() => {
    const inputs = document.querySelectorAll('.choice-name-input')
    inputs[inputs.length - 1]?.focus()
  })
}

function removeChoice(idx) {
  colorPickerIdx.value = -1
  form.choiceItems.splice(idx, 1)
}

function moveChoice(idx, dir) {
  const items = form.choiceItems
  colorPickerIdx.value = -1
  if (dir === 'up' && idx > 0) {
    ;[items[idx - 1], items[idx]] = [items[idx], items[idx - 1]]
  } else if (dir === 'down' && idx < items.length - 1) {
    ;[items[idx], items[idx + 1]] = [items[idx + 1], items[idx]]
  }
}

function setChoiceColor(idx, color) {
  form.choiceItems[idx].color = color
  colorPickerIdx.value = -1
}
</script>

<style scoped>
.popup-label {
  display: block;
  font-size: 10px;
  font-weight: 600;
  color: var(--text-2);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 4px;
}
.popup-input,
.popup-select {
  width: 100%;
  box-sizing: border-box;
  padding: 5px 8px;
  font-size: 12px;
  border-radius: 5px;
  border: 1px solid var(--border-1);
  background: var(--input-bg);
  color: var(--text-1);
  outline: none;
  font-family: inherit;
  transition: border-color 0.15s;
}
.popup-input:focus,
.popup-select:focus { border-color: var(--color-brand-600); }
.popup-select:disabled { opacity: 0.5; cursor: default; }
.choice-reorder-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--text-3);
  cursor: pointer;
  border-radius: 3px;
  transition: background 0.1s, color 0.1s;
  flex-shrink: 0;
}
.choice-reorder-btn:hover:not(:disabled) {
  background: var(--toolbar-hover-bg);
  color: var(--text-1);
}
.choice-reorder-btn:disabled { opacity: 0.2; cursor: default; }
.popup-btn-cancel {
  padding: 4px 11px;
  font-size: 12px;
  background: transparent;
  border: 1px solid var(--border-1);
  border-radius: 5px;
  color: var(--text-2);
  cursor: pointer;
  transition: background 0.1s;
}
.popup-btn-cancel:hover { background: var(--toolbar-hover-bg); }
.popup-btn-save {
  padding: 4px 11px;
  font-size: 12px;
  background: var(--color-brand-600);
  border: none;
  border-radius: 5px;
  color: white;
  cursor: pointer;
  transition: background 0.1s;
}
.popup-btn-save:hover:not(:disabled) { background: var(--color-brand-700); }
.popup-btn-save:disabled { opacity: 0.4; cursor: default; }
</style>
