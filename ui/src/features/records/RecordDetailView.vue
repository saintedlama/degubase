<template>
  <div class="flex flex-col flex-1 min-h-0 overflow-hidden">
    <div v-if="loading" class="flex-1 flex items-center justify-center text-sm text-zinc-400">Loading…</div>
    <div v-else-if="!currentRow || !table" class="flex-1 flex items-center justify-center text-sm text-zinc-400">Row not found.</div>
    <RecordView
      v-else
      :workspace-code="route.params.workspaceCode"
      :table-code="route.params.tableCode"
      :table="table"
      :row="currentRow"
      :row-index="navIndex"
      :prev-row-id="prevRowId"
      :next-row-id="nextRowId"
      :auto-focus="route.query.new === '1'"
      @close="goBack"
      @navigate="navigateTo"
      @row-updated="onRowUpdated"
      @row-deleted="onRowDeleted"
      @row-created="onRowCreated"
      @column-updated="onColumnUpdated"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../../api/client.js'
import RecordView from './RecordView.vue'
import { useTableEvents } from './useTableEvents.js'

const route = useRoute()
const router = useRouter()

const table = ref(null)
const currentRow = ref(null)  // the displayed record — always fetched directly by ID
const navRows = ref([])       // row list used only for prev/next navigation
const loading = ref(true)

const rowId = computed(() => Number(route.params.rowId))
const navIndex = computed(() => navRows.value.findIndex(r => r.id === rowId.value))
const prevRowId = computed(() => navIndex.value > 0 ? navRows.value[navIndex.value - 1].id : null)
const nextRowId = computed(() => navIndex.value < navRows.value.length - 1 ? navRows.value[navIndex.value + 1].id : null)

async function loadCurrentRow() {
  currentRow.value = await api.getRow(
    route.params.workspaceCode,
    route.params.tableCode,
    rowId.value,
  )
}

async function load() {
  loading.value = true
  try {
    // Fetch the current record by ID directly — never rely on list pagination for this.
    // listRows is fetched in parallel only to populate prev/next navigation.
    const [t, row, r] = await Promise.all([
      api.getTable(route.params.workspaceCode, route.params.tableCode),
      api.getRow(route.params.workspaceCode, route.params.tableCode, rowId.value),
      api.listRows(route.params.workspaceCode, route.params.tableCode),
    ])
    table.value = t
    currentRow.value = row
    navRows.value = r?.data ?? []
  } finally {
    loading.value = false
  }
}

// Reload table + nav list when the workspace/table changes
watch(() => `${route.params.workspaceCode}/${route.params.tableCode}`, load, { immediate: true })

// When navigating between records in the same table, only refetch the current record
watch(rowId, (id, oldId) => {
  if (id && oldId && id !== oldId) loadCurrentRow()
})

useTableEvents(
  () => route.params.workspaceCode,
  () => route.params.tableCode,
  {
    onUpdated(updatedRow) {
      if (updatedRow.id === rowId.value) currentRow.value = updatedRow
      const idx = navRows.value.findIndex(r => r.id === updatedRow.id)
      if (idx >= 0) navRows.value[idx] = updatedRow
    },
    onCreated(newRow) {
      if (!navRows.value.find(r => r.id === newRow.id)) navRows.value.push(newRow)
    },
    onDeleted(id) {
      const idx = navRows.value.findIndex(r => r.id === id)
      if (idx >= 0) navRows.value.splice(idx, 1)
      if (id === rowId.value) {
        const nextId = navRows.value[idx]?.id ?? navRows.value[idx - 1]?.id ?? null
        if (nextId) navigateTo(nextId)
        else goBack()
      }
    },
  }
)

function goBack() {
  if (route.query.returnTo) {
    router.push(decodeURIComponent(route.query.returnTo))
    return
  }
  const viewCode = route.query.view || table.value?.views?.[0]?.code
  if (viewCode) {
    router.push(`/workspaces/${route.params.workspaceCode}/tables/${route.params.tableCode}/views/${viewCode}`)
  } else {
    router.back()
  }
}

function navigateTo(targetRowId) {
  const viewQuery = route.query.view ? `?view=${route.query.view}` : ''
  router.push(`/workspaces/${route.params.workspaceCode}/tables/${route.params.tableCode}/rows/${targetRowId}${viewQuery}`)
}

function onRowUpdated(updatedRow) {
  currentRow.value = { ...(currentRow.value ?? {}), ...updatedRow }
  const idx = navRows.value.findIndex(r => r.id === updatedRow.id)
  if (idx >= 0) navRows.value[idx] = { ...navRows.value[idx], ...updatedRow }
}

function onRowDeleted(deletedRow) {
  const idx = navRows.value.findIndex(r => r.id === deletedRow.id)
  if (idx >= 0) navRows.value.splice(idx, 1)
  const nextId = navRows.value[idx]?.id ?? navRows.value[idx - 1]?.id ?? null
  if (nextId) navigateTo(nextId)
  else goBack()
}

function onRowCreated(newRow) {
  navRows.value.push(newRow)
  navigateTo(newRow.id)
}

function onColumnUpdated(updatedCol) {
  if (!table.value) return
  table.value = {
    ...table.value,
    columns: table.value.columns.map(c => c.id === updatedCol.id ? updatedCol : c),
  }
}
</script>
