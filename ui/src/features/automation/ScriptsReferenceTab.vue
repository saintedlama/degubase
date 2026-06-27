<template>
  <div class="flex-1 overflow-y-auto bg-zinc-50 dark:bg-[#1e1e1e]">
    <div class="max-w-2xl px-6 py-5 flex flex-col gap-7">

      <!-- Event context -->
      <section>
        <div class="flex items-center gap-3 mb-3">
          <span class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]">Event context</span>
          <div class="flex-1 h-px bg-border-1" />
        </div>
        <div class="border border-border-1 rounded-lg overflow-hidden text-[12px]">
          <div v-for="(row, i) in eventContextRows" :key="i"
            class="flex items-start gap-0 border-b border-zinc-100 dark:border-[#2d2d2d] last:border-0">
            <span class="px-3 py-1.5 font-mono text-brand-600 dark:text-brand-400 w-44 shrink-0">{{ row.name }}</span>
            <span class="px-3 py-1.5 text-zinc-400 dark:text-[#6d6d6d] w-24 shrink-0">{{ row.type }}</span>
            <span class="px-3 py-1.5 text-text-2 flex-1">{{ row.desc }}</span>
          </div>
        </div>
        <p class="mt-2 text-[11px] text-zinc-400 dark:text-[#6d6d6d]">
          Flat aliases also available: <code class="font-mono">event_type</code>, <code class="font-mono">row_id</code>, <code class="font-mono">row</code>, <code class="font-mono">table_name</code>, <code class="font-mono">table_code</code>
        </p>
      </section>

      <!-- event.data fields (dynamic) -->
      <section>
        <div class="flex items-center gap-3 mb-3">
          <span class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]">event.data — row fields</span>
          <div class="flex-1 h-px bg-border-1" />
        </div>
        <div v-if="refLoading" class="text-[12px] text-zinc-400 dark:text-[#6d6d6d]">Loading columns…</div>
        <div v-else-if="!scopeTables.length" class="text-[12px] text-zinc-400 dark:text-[#6d6d6d]">No tables in scope.</div>
        <div v-else class="flex flex-col gap-3">
          <div v-for="tbl in scopeTables" :key="tbl.id" class="border border-border-1 rounded-lg overflow-hidden">
            <div class="flex items-center gap-2 px-3 py-2 bg-zinc-100 dark:bg-[#252526] border-b border-border-1">
              <span class="text-[12px] font-semibold text-text-1">{{ tbl.name }}</span>
              <span class="text-[11px] font-mono text-zinc-400 dark:text-[#6d6d6d]">{{ tbl.code }}</span>
            </div>
            <div v-if="!tableColumns[tbl.id]" class="px-3 py-2 text-[12px] text-zinc-400 dark:text-[#6d6d6d]">Loading…</div>
            <div v-else-if="!tableColumns[tbl.id].length" class="px-3 py-2 text-[12px] text-zinc-400 dark:text-[#6d6d6d]">No columns.</div>
            <div v-else>
              <div
                v-for="col in tableColumns[tbl.id]"
                :key="col.id"
                class="flex items-start gap-0 border-b border-zinc-100 dark:border-[#2d2d2d] last:border-0 text-[12px]"
              >
                <span class="px-3 py-1.5 font-mono text-brand-600 dark:text-brand-400 w-52 shrink-0 truncate">event.data["{{ col.name }}"]</span>
                <span class="px-3 py-1.5 text-zinc-400 dark:text-[#6d6d6d] w-28 shrink-0">{{ col.type }}</span>
                <span class="px-3 py-1.5 text-text-2 flex-1 flex flex-wrap gap-1">
                  <template v-if="colChoices(col).length">
                    <span
                      v-for="c in colChoices(col)"
                      :key="c"
                      class="px-1.5 py-0.5 rounded text-[10px] bg-zinc-100 dark:bg-[#2d2d2d] text-text-2"
                    >{{ c }}</span>
                  </template>
                </span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Logging -->
      <section>
        <div class="flex items-center gap-3 mb-3">
          <span class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]">Logging</span>
          <div class="flex-1 h-px bg-border-1" />
        </div>
        <pre class="bg-zinc-900 dark:bg-[#141414] text-zinc-100 rounded-lg px-4 py-3 text-[12px] font-mono overflow-x-auto leading-relaxed">log.info(<span class="text-green-400">"message"</span>)
log.warn(<span class="text-green-400">"message"</span>)
log.error(<span class="text-green-400">"message"</span>)
log(<span class="text-green-400">"message"</span>)          <span class="text-zinc-500">-- shorthand for log.info</span></pre>
      </section>

      <!-- HTTP -->
      <section>
        <div class="flex items-center gap-3 mb-3">
          <span class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]">HTTP</span>
          <div class="flex-1 h-px bg-border-1" />
        </div>
        <pre class="bg-zinc-900 dark:bg-[#141414] text-zinc-100 rounded-lg px-4 py-3 text-[12px] font-mono overflow-x-auto leading-relaxed"><span class="text-zinc-500">-- GET request</span>
<span class="text-blue-300">local</span> resp = http.get(<span class="text-green-400">"https://example.com/api"</span>)
<span class="text-zinc-500">-- resp.status  → 200</span>
<span class="text-zinc-500">-- resp.body    → response body as string</span>

<span class="text-zinc-500">-- POST request (JSON body)</span>
<span class="text-blue-300">local</span> resp = http.post(<span class="text-green-400">"https://example.com/api"</span>, <span class="text-green-400">'{"key":"value"}'</span>)

<span class="text-blue-300">if</span> resp.status ~= 200 <span class="text-blue-300">then</span>
  log.error(<span class="text-green-400">"request failed: "</span> .. tostring(resp.status))
  <span class="text-blue-300">return</span>
<span class="text-blue-300">end</span></pre>
      </section>

      <!-- DeguBase API -->
      <section>
        <div class="flex items-center gap-3 mb-3">
          <span class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]">DeguBase API</span>
          <div class="flex-1 h-px bg-border-1" />
        </div>
        <pre class="bg-zinc-900 dark:bg-[#141414] text-zinc-100 rounded-lg px-4 py-3 text-[12px] font-mono overflow-x-auto leading-relaxed"><span class="text-zinc-500">-- Patch a single field on the triggering row</span>
degubase.update_field(<span class="text-green-400">"column name"</span>, value)

<span class="text-zinc-500">-- Replace the full data of any row</span>
degubase.update_row(row_id, {<span class="text-green-400">field</span> = value, ...})

<span class="text-zinc-500">-- Fetch any row's data by ID</span>
<span class="text-blue-300">local</span> data = degubase.get_row(row_id)

<span class="text-zinc-500">-- Create a new row in the current table</span>
<span class="text-blue-300">local</span> new_id = degubase.create_row({<span class="text-green-400">field</span> = value, ...})</pre>
        <p class="mt-2 text-[11px] text-zinc-400 dark:text-[#6d6d6d]">
          <code class="font-mono">db.*</code> is an alias for <code class="font-mono">degubase.*</code>. Script-initiated writes never re-trigger scripts.
        </p>
      </section>

      <!-- Environment variables -->
      <section>
        <div class="flex items-center gap-3 mb-3">
          <span class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]">Environment variables</span>
          <div class="flex-1 h-px bg-border-1" />
        </div>
        <div v-if="!envVars.length" class="text-[12px] text-zinc-400 dark:text-[#6d6d6d]">No environment variables configured.</div>
        <div v-else class="border border-border-1 rounded-lg overflow-hidden mb-3">
          <div
            v-for="ev in envVars" :key="ev.id"
            class="flex items-center gap-0 border-b border-zinc-100 dark:border-[#2d2d2d] last:border-0 text-[12px]"
          >
            <span class="px-3 py-1.5 font-mono text-brand-600 dark:text-brand-400 flex-1">{{ ev.key }}</span>
            <span
              v-if="ev.is_secret"
              class="px-3 py-1.5 text-[10px] font-semibold text-amber-600 dark:text-amber-400"
            >secret</span>
            <span v-else class="px-3 py-1.5 font-mono text-zinc-400 dark:text-[#6d6d6d] truncate max-w-xs">{{ ev.value }}</span>
          </div>
        </div>
        <pre class="bg-zinc-900 dark:bg-[#141414] text-zinc-100 rounded-lg px-4 py-3 text-[12px] font-mono overflow-x-auto leading-relaxed"><span class="text-zinc-500">-- Access via function</span>
<span class="text-blue-300">local</span> key = env(<span class="text-green-400">"MY_KEY"</span>)

<span class="text-zinc-500">-- Or directly as a global (same name as the key)</span>
<span class="text-blue-300">local</span> key = MY_KEY</pre>
      </section>

    </div>
  </div>
</template>

<script setup>
import { ref, inject, watch, onActivated } from 'vue'
import { api } from '../../api/client.js'

const workspaceCode = inject('workspaceCode')
const form = inject('scriptForm')
const tables = inject('tables')
const envVars = inject('envVars')
const tableColumns = inject('tableColumns')
const refLoading = inject('refLoading')

const scopeTables = inject('scopeTables')
const eventContextRows = inject('eventContextRows')
const colChoices = inject('colChoices')

async function loadRefColumns() {
  refLoading.value = true
  try {
    const needed = scopeTables.value.filter(t => !tableColumns.value[t.id])
    await Promise.all(needed.map(async t => {
      const cols = await api.listColumns(workspaceCode, t.code)
      tableColumns.value[t.id] = cols
    }))
  } finally {
    refLoading.value = false
  }
}

onActivated(() => {
  if (form.value) loadRefColumns()
})

watch(() => form.value?.event_type, () => {
  if (form.value) loadRefColumns()
}, { immediate: true })
</script>
