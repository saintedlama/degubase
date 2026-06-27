<template>
  <div class="flex flex-col">
    <div class="px-3 py-2 border-b border-border-1">
      <input
        ref="searchInput"
        v-model="search"
        placeholder="Search rows…"
        class="w-full text-[12px] bg-transparent outline-none text-text-1 placeholder:text-text-3"
      />
    </div>
    <div class="overflow-auto" style="max-height: 280px">
      <div v-if="loading" class="px-3 py-4 text-[12px] text-text-3">Loading…</div>
      <template v-else>
        <button
          v-if="modelValue"
          class="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-3 hover:bg-border-1 cursor-pointer border-none bg-transparent transition-colors"
          @click="emit('update:modelValue', null)"
        >
          <RiCloseLine size="12" />
          Clear link
        </button>
        <button
          v-for="row in filteredRows"
          :key="row.id"
          class="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] hover:bg-border-1 cursor-pointer border-none bg-transparent transition-colors text-left"
          :class="modelValue?.id === row.id ? 'text-brand-600 dark:text-brand-400' : 'text-text-1'"
          @click="select(row)"
        >
          <span class="truncate flex-1">{{ labelOf(row) }}</span>
          <RiCheckLine v-if="modelValue?.id === row.id" size="12" class="shrink-0 text-brand-500" />
        </button>
        <div v-if="!filteredRows.length" class="px-3 py-4 text-[12px] text-text-3">No rows found</div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../../api/client.js'
import { RiCloseLine, RiCheckLine } from '@remixicon/vue'

const props = defineProps({
  modelValue: { default: null },
  workspaceCode: { type: String, required: true },
  targetTableCode: { type: String, required: true },
  displayColumnCode: { type: String, default: null },
})
const emit = defineEmits(['update:modelValue'])

const searchInput = ref(null)
const search = ref('')
const rows = ref([])
const labelColCode = ref(null)
const loading = ref(true)

onMounted(async () => {
  try {
    const [cols, res] = await Promise.all([
      api.listColumns(props.workspaceCode, props.targetTableCode),
      api.listRows(props.workspaceCode, props.targetTableCode, { pageSize: 100 }),
    ])
    const textTypes = ['text', 'long-text', 'markdown', 'email', 'url', 'number', 'currency', 'percent']
    const labelCol = [...cols]
      .sort((a, b) => a.position - b.position)
      .find(c => textTypes.includes(c.type))
    labelColCode.value = labelCol?.code ?? null
    rows.value = res.data ?? []
  } catch { /* ignore */ } finally {
    loading.value = false
  }
  searchInput.value?.focus()
})

function labelOf(row) {
  const code = props.displayColumnCode || labelColCode.value
  if (code != null) {
    const v = row.data?.[code]
    if (v != null && v !== '') return String(v)
  }
  for (const v of Object.values(row.data ?? {})) {
    if (v && typeof v === 'string') return v
  }
  return `Row #${row.id}`
}

const filteredRows = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter(r => labelOf(r).toLowerCase().includes(q))
})

function select(row) {
  emit('update:modelValue', { id: row.id, label: labelOf(row) })
}
</script>
