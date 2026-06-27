<template>
  <div class="flex-1 overflow-y-auto px-5 py-4 flex flex-col gap-3 min-h-0">
    <div v-if="loading" class="text-xs text-text-2">Loading…</div>
    <div v-else-if="!executions.length" class="text-xs text-text-2">No executions recorded yet.</div>

    <div
      v-for="ex in executions"
      :key="ex.id"
      class="border border-border-1 rounded-lg overflow-hidden shrink-0"
    >
      <div class="flex items-center gap-3 px-4 py-2.5 bg-zinc-50 dark:bg-[#252526]">
        <span
          class="text-[11px] font-bold px-1.5 py-0.5 rounded shrink-0"
          :class="ex.success
            ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
            : 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400'"
        >{{ ex.success ? 'OK' : 'FAIL' }}</span>
        <span class="text-[12px] text-text-1 font-medium">{{ eventLabel(ex.event_type) }}</span>
        <span v-if="tableName(ex.table_code)" class="text-[11px] text-text-2">{{ tableName(ex.table_code) }}</span>
        <RouterLink
          v-if="ex.row_id && ex.table_code"
          :to="`/workspaces/${workspaceCode}/tables/${ex.table_code}/rows/${ex.row_id}`"
          class="text-[11px] text-brand-600 dark:text-brand-400 hover:underline shrink-0"
        >record #{{ ex.row_id }}</RouterLink>
        <span v-else-if="ex.row_id" class="text-[11px] text-text-2">record #{{ ex.row_id }}</span>
        <span v-else class="text-[11px] text-text-2">(bulk)</span>
        <span class="ml-auto text-[11px] text-text-2 shrink-0">{{ formatTs(ex.started_at) }}</span>
        <span class="text-[11px] text-text-2 shrink-0">{{ ex.duration_ms }}ms</span>
      </div>

      <div v-if="ex.logs?.length" class="divide-y divide-zinc-100 dark:divide-[#3c3c3c]">
        <div
          v-for="(entry, i) in ex.logs"
          :key="i"
          class="flex items-start gap-2.5 px-4 py-1.5 font-mono text-[12px]"
        >
          <span
            class="shrink-0 text-[10px] font-bold uppercase mt-0.5 w-9"
            :class="{
              'text-text-2': entry.level === 'info',
              'text-amber-600 dark:text-amber-400': entry.level === 'warn',
              'text-red-600 dark:text-red-400':    entry.level === 'error',
            }"
          >{{ entry.level }}</span>
          <span class="text-text-1 break-all whitespace-pre-wrap">{{ entry.message }}</span>
          <span class="ml-auto shrink-0 text-[11px] text-zinc-300 dark:text-[#6d6d6d]">{{ formatTs(entry.ts) }}</span>
        </div>
      </div>
      <div v-else class="px-4 py-2 text-[12px] text-text-2 italic">no log output</div>
    </div>
  </div>
</template>

<script setup>
import { ref, inject, onActivated, watchEffect } from 'vue'
import { api } from '../../api/client.js'

const workspaceCode = inject('workspaceCode')
const form = inject('scriptForm')
const tables = inject('tables')
const eventLabel = inject('eventLabel')
const formatTs = inject('formatTs')

const executions = ref([])
const loading = ref(false)

function tableName(code) {
  if (!code) return ''
  return tables.value.find(t => t.code === code)?.name ?? code
}

async function fetch() {
  if (!form.value?.id) return
  loading.value = true
  try {
    executions.value = await api.listScriptExecutions(workspaceCode, form.value.id, { limit: 50 })
  } finally {
    loading.value = false
  }
}

onActivated(fetch)
watchEffect(() => { if (form.value?.id) fetch() })
</script>
