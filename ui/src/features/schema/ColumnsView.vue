<template>
  <div class="flex flex-col flex-1 min-h-0 overflow-hidden bg-zinc-100 dark:bg-[#1e1e1e]">

    <!-- Panel toolbar -->
    <div class="flex items-center gap-2 px-5 h-10 bg-surface-1 border-b border-border-1 shrink-0">
      <span class="text-xs font-medium text-brand-600 dark:text-brand-400">Columns</span>
      <div class="ml-auto flex items-center gap-1.5">
        <button
          class="inline-flex items-center gap-1 h-6 px-2 text-[11px] font-medium text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-700/50 rounded cursor-pointer hover:bg-brand-100 dark:hover:bg-brand-900/50 transition-colors"
          @click="startAdd"
        >
          <RiAddLine size="12" />
          Add column
        </button>
      </div>
    </div>

    <!-- Scrollable column list -->
    <div class="flex-1 overflow-y-auto px-5 py-3">
      <div class="max-w-xl flex flex-col gap-2">

        <div v-if="loading" class="text-[12px] text-text-2 py-6 text-center">Loading…</div>

        <template v-else>
          <!-- Existing columns -->
          <div class="bg-surface-1 border border-border-1 rounded-lg overflow-hidden">
            <div
              v-for="col in columns"
              :key="col.id"
              class="border-b border-zinc-100 dark:border-[#2d2d30] last:border-b-0"
            >
              <div class="flex items-center gap-2 px-3 py-2">
                <component :is="typeIcon(col.type)" size="14" class="text-text-2 shrink-0" />
                <span class="flex-1 text-[13px] font-medium text-text-1">{{ col.name }}</span>
                <span class="text-[11px] text-zinc-400 dark:text-[#6d6d6d]">{{ typeLabelMap[col.type] ?? col.type }}</span>
                <button
                  class="flex items-center justify-center w-6 h-6 rounded bg-transparent border-none cursor-pointer transition-colors"
                  :class="editingId === col.id
                    ? 'text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30'
                    : 'text-zinc-300 dark:text-[#555] hover:text-text-1 hover:bg-border-1'"
                  title="Edit column"
                  @click="toggleEdit(col)"
                >
                  <RiPencilLine size="12" />
                </button>
                <button
                  class="flex items-center justify-center w-6 h-6 rounded bg-transparent border-none text-zinc-300 dark:text-[#555] hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/40 cursor-pointer transition-colors"
                  title="Delete column"
                  @click="confirmDelete(col)"
                >
                  <RiDeleteBin6Line size="12" />
                </button>
              </div>

              <div v-if="editingId === col.id" class="border-t border-zinc-100 dark:border-[#3c3c3c]">
                <ColumnEditForm
                  :column="col"
                  :workspace-code="workspaceCode"
                  :table-code="tableCode"
                  @saved="onSaved"
                  @cancel="editingId = null"
                />
              </div>
            </div>
          </div>

          <!-- Add new column form -->
          <div
            v-if="addingNew"
            class="bg-surface-1 border border-brand-300 dark:border-brand-700/60 rounded-lg overflow-hidden"
          >
            <div class="px-3 py-2 border-b border-zinc-100 dark:border-[#3c3c3c]">
              <span class="text-[11px] font-semibold text-text-2 uppercase tracking-wide">New column</span>
            </div>
            <ColumnEditForm
              :workspace-code="workspaceCode"
              :table-code="tableCode"
              @saved="onAdded"
              @cancel="addingNew = false"
            />
          </div>
        </template>

      </div>
    </div>
  </div>

</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../../api/client.js'
import ColumnEditForm from './ColumnEditForm.vue'
import { typeIcon, ALL_TYPES } from './columnTypes.js'
import { RiAddLine, RiPencilLine, RiDeleteBin6Line } from '@remixicon/vue'
import { useConfirm } from '../../foundation/useConfirm.js'

const { confirm } = useConfirm()

const props = defineProps({
  workspaceCode: { type: String, required: true },
  tableCode:     { type: String, required: true },
})

const emit = defineEmits(['columns-changed'])

const columns   = ref([])
const loading   = ref(true)
const editingId = ref(null)
const addingNew = ref(false)

const typeLabelMap = Object.fromEntries(ALL_TYPES.map(t => [t.value, t.label]))

onMounted(load)

async function load() {
  loading.value = true
  try {
    columns.value = await api.listColumns(props.workspaceCode, props.tableCode)
  } finally {
    loading.value = false
  }
}

function toggleEdit(col) {
  editingId.value = editingId.value === col.id ? null : col.id
  addingNew.value = false
}

function startAdd() {
  addingNew.value = true
  editingId.value = null
}

async function onSaved() {
  editingId.value = null
  await load()
  emit('columns-changed')
}

async function onAdded() {
  addingNew.value = false
  await load()
  emit('columns-changed')
}

async function confirmDelete(col) {
  editingId.value = null
  const ok = await confirm({
    title: 'Delete column?',
    message: `Deleting "${col.name}" will permanently remove all data stored in this column. This cannot be undone.`,
    confirmLabel: 'Delete',
    danger: true,
  })
  if (!ok) return
  await api.deleteColumn(props.workspaceCode, props.tableCode, col.code)
  await load()
  emit('columns-changed')
}
</script>
