<template>
  <div class="flex-1 flex flex-col overflow-hidden">

    <!-- Top bar -->
    <div class="flex items-center justify-between px-6 py-3 border-b border-border-1 bg-white dark:bg-[#1e1e1e] shrink-0">
      <h1 class="text-[15px] font-bold text-text-1">Scripts</h1>
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 bg-brand-600 text-white text-[13px] font-medium rounded-md border-none cursor-pointer hover:bg-brand-700 transition-colors"
        @click="newScript"
      >
        <RiAddLine size="14" />
        New script
      </button>
    </div>

    <div class="flex-1 flex overflow-hidden">

      <!-- Script list: full-width on mobile when nothing selected, fixed sidebar on desktop -->
      <div
        class="flex-col border-r border-border-1 bg-zinc-50 dark:bg-[#252526] overflow-y-auto"
        :class="form
          ? 'hidden md:flex md:w-64 md:shrink-0'
          : 'flex flex-1 md:flex-none md:w-64 md:shrink-0'"
      >
        <div v-if="loading" class="px-4 py-3 text-xs text-text-2">Loading…</div>
        <div v-else-if="!scripts.length" class="px-4 py-3 text-xs text-text-2">No scripts yet.</div>
        <button
          v-for="sc in scripts"
          :key="sc.id"
          class="flex flex-col items-start gap-0.5 px-4 py-3 text-left border-none cursor-pointer border-b border-border-1 transition-colors"
          :class="selected?.id === sc.id
            ? 'bg-brand-50 dark:bg-brand-900/20'
            : 'bg-transparent hover:bg-zinc-100 dark:hover:bg-[#2d2d30]'"
          @click="selectScript(sc)"
        >
          <div class="flex items-center gap-2 w-full min-w-0">
            <span
              class="w-1.5 h-1.5 rounded-full shrink-0"
              :class="sc.enabled ? 'bg-green-500' : 'bg-zinc-300 dark:bg-zinc-600'"
            />
            <span class="text-[13px] font-medium text-text-1 truncate flex-1">{{ sc.name }}</span>
          </div>
          <span class="text-[11px] text-text-2 pl-3.5">{{ eventLabel(sc.event_type) }}</span>
          <span v-if="sc.table_ids?.length" class="text-[11px] text-text-2 pl-3.5 truncate w-full">
            {{ sc.table_ids.map(tableNameById).join(', ') }}
          </span>
        </button>
      </div>

      <!-- Editor panel: hidden on mobile when nothing selected, full-width otherwise -->
      <div
        class="flex-col min-h-0"
        :class="form ? 'flex flex-1' : 'hidden md:flex md:flex-1'"
      >
        <div v-if="!form" class="flex-1 flex items-center justify-center text-sm text-text-2">
          Select a script or create a new one.
        </div>

        <template v-else>
          <!-- Back button (mobile only) -->
          <button
            class="md:hidden flex items-center gap-1.5 px-4 py-2.5 text-[13px] text-text-2 bg-zinc-50 dark:bg-[#252526] border-b border-border-1 shrink-0 hover:text-text-1 transition-colors"
            @click="goBack"
          >
            <RiArrowLeftLine size="14" />
            Scripts
          </button>

          <!-- Script settings bar -->
          <div class="flex flex-wrap items-end gap-3 px-5 py-3 border-b border-border-1 bg-white dark:bg-[#1e1e1e] shrink-0">

            <div class="flex flex-col gap-1 min-w-36">
              <label class="text-[10px] font-semibold text-text-2 tracking-wide uppercase">Name</label>
              <input v-model="form.name" class="form-input" placeholder="Script name" />
            </div>

            <div class="flex flex-col gap-1">
              <label class="text-[10px] font-semibold text-text-2 tracking-wide uppercase">Event</label>
              <select v-model="form.event_type" class="form-input">
                <option value="record.created">Record created</option>
                <option value="record.updated">Record updated</option>
                <option value="record.deleted">Record deleted</option>
              </select>
            </div>

            <!-- Table scope multi-select -->
            <div class="flex flex-col gap-1 relative" ref="tableScopeEl">
              <label class="text-[10px] font-semibold text-text-2 tracking-wide uppercase">Table scope</label>
              <button
                type="button"
                class="form-input flex items-center gap-2 min-w-40 text-left"
                @click="tableScopeOpen = !tableScopeOpen"
              >
                <span class="flex-1 truncate text-[13px]">{{ tableScopeLabel }}</span>
                <RiArrowDownSLine size="13" class="text-zinc-400 shrink-0 transition-transform" :class="tableScopeOpen ? 'rotate-180' : ''" />
              </button>
              <div
                v-if="tableScopeOpen"
                class="absolute top-full left-0 z-50 mt-1 w-56 bg-surface-1 border border-border-1 rounded-lg shadow-lg py-1 max-h-52 overflow-y-auto"
              >
                <label class="flex items-center gap-2 px-3 py-1.5 cursor-pointer hover:bg-surface-2">
                  <input
                    type="checkbox"
                    class="accent-brand-600"
                    :checked="form.table_ids.length === 0"
                    @change="form.table_ids = []"
                  />
                  <span class="text-[13px] text-text-1">All tables</span>
                </label>
                <div class="my-1 border-t border-zinc-100 dark:border-[#3c3c3c]" />
                <label
                  v-for="tbl in tables"
                  :key="tbl.id"
                  class="flex items-center gap-2 px-3 py-1.5 cursor-pointer hover:bg-surface-2"
                >
                  <input
                    type="checkbox"
                    class="accent-brand-600"
                    :checked="form.table_ids.includes(tbl.id)"
                    @change="toggleTableScope(tbl.id)"
                  />
                  <span class="text-[13px] text-text-1 truncate">{{ tbl.name }}</span>
                </label>
              </div>
            </div>

            <label class="flex items-center gap-2 cursor-pointer pb-1.5">
              <span class="text-[12px] text-text-2">Enabled</span>
              <button
                type="button"
                class="relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border-2 border-transparent transition-colors cursor-pointer"
                :class="form.enabled ? 'bg-brand-600' : 'bg-zinc-300 dark:bg-zinc-600'"
                @click="form.enabled = !form.enabled"
              >
                <span
                  class="inline-block h-3.5 w-3.5 rounded-full bg-white shadow-sm transform transition-transform"
                  :class="form.enabled ? 'translate-x-4' : 'translate-x-0.5'"
                />
              </button>
            </label>

            <div class="flex items-center gap-2 ml-auto">
              <button
                v-if="form.id"
                class="px-3 py-1.5 text-[12px] font-medium text-red-500 bg-transparent border border-red-200 dark:border-red-900/40 rounded-md cursor-pointer hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                @click="confirmDelete"
              >Delete</button>
              <button
                class="px-3 py-1.5 text-[12px] font-medium text-text-2 bg-surface-2 border border-border-1 rounded-md cursor-pointer hover:bg-border-1 transition-colors"
                @click="cancelEdit"
              >Cancel</button>
              <SpinnerButton
                :loading="saving"
                :disabled="!form.name"
                idle="Save"
                busy="Saving…"
                class="px-3 py-1.5 text-[12px] font-medium bg-brand-600 text-white border-none rounded-md cursor-pointer hover:bg-brand-700 disabled:opacity-40 disabled:cursor-default transition-colors"
                @click="saveScript"
              />
            </div>
          </div>

          <p v-if="saveError" class="px-5 py-1.5 text-xs text-red-500 bg-red-50 dark:bg-red-900/10 shrink-0">{{ saveError }}</p>

          <!-- Tab bar -->
          <div class="flex items-center gap-0 border-b border-border-1 bg-white dark:bg-[#1e1e1e] shrink-0 px-5">
            <button
              class="px-3 py-2 text-[12px] font-medium border-b-2 -mb-px transition-colors border-none bg-transparent cursor-pointer"
              :class="activeTab === 'code'
                ? 'border-brand-500 text-brand-600 dark:text-brand-400'
                : 'border-transparent text-text-2 hover:text-text-1'"
              @click="navigateTab('code')"
            >Code</button>
            <button
              v-if="form.id"
              class="px-3 py-2 text-[12px] font-medium border-b-2 -mb-px transition-colors border-none bg-transparent cursor-pointer"
              :class="activeTab === 'executions'
                ? 'border-brand-500 text-brand-600 dark:text-brand-400'
                : 'border-transparent text-text-2 hover:text-text-1'"
              @click="navigateTab('executions')"
            >Executions</button>
            <button
              class="px-3 py-2 text-[12px] font-medium border-b-2 -mb-px transition-colors border-none bg-transparent cursor-pointer"
              :class="activeTab === 'reference'
                ? 'border-brand-500 text-brand-600 dark:text-brand-400'
                : 'border-transparent text-text-2 hover:text-text-1'"
              @click="navigateTab('reference')"
            >Reference</button>
          </div>

          <!-- Router outlet for tab content -->
          <router-view v-slot="{ Component }">
            <keep-alive>
              <component :is="Component" />
            </keep-alive>
          </router-view>

        </template>
      </div>

    </div>

    <!-- Environment variables panel -->
    <div class="shrink-0 border-t border-border-1 bg-white dark:bg-[#1e1e1e]">
      <button
        class="flex items-center gap-2 w-full px-5 py-3 text-left border-none bg-transparent cursor-pointer"
        @click="envOpen = !envOpen"
      >
        <RiArrowDownSLine
          size="14"
          class="text-zinc-400 transition-transform"
          :class="envOpen ? 'rotate-0' : '-rotate-90'"
        />
        <span class="text-[13px] font-semibold text-text-1">Environment variables</span>
        <span class="text-[11px] text-text-2 ml-1">({{ envVars.length }})</span>
      </button>

      <div v-if="envOpen" class="px-5 pb-4 flex flex-col gap-2">

        <!-- Existing env vars -->
        <div
          v-for="ev in envVars"
          :key="ev.id"
          class="flex items-center gap-2"
        >
          <input :value="ev.key" class="form-input w-44 font-mono" readonly />

          <!-- Secret: value is never returned; offer a replace field -->
          <template v-if="ev.is_secret">
            <span class="text-[11px] font-semibold px-1.5 py-0.5 rounded bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 shrink-0">secret</span>
            <input
              class="form-input flex-1 font-mono"
              placeholder="enter new value to replace"
              @change="updateEnvVar(ev, $event.target.value)"
            />
          </template>

          <!-- Normal: value visible and editable -->
          <input
            v-else
            :value="ev.value"
            class="form-input flex-1 font-mono"
            @change="updateEnvVar(ev, $event.target.value)"
          />

          <button
            class="flex items-center justify-center w-7 h-7 text-zinc-400 hover:text-red-500 bg-transparent border-none cursor-pointer rounded transition-colors shrink-0"
            title="Delete"
            @click="deleteEnvVar(ev)"
          ><RiDeleteBinLine size="14" /></button>
        </div>

        <!-- Add new -->
        <div class="flex items-center gap-2 mt-1">
          <input v-model="newEnv.key" class="form-input w-44 font-mono" placeholder="KEY" @keydown.enter="addEnvVar" />
          <input v-model="newEnv.value" class="form-input flex-1 font-mono" placeholder="value" @keydown.enter="addEnvVar" />
          <label class="flex items-center gap-1.5 shrink-0 cursor-pointer">
            <input v-model="newEnv.is_secret" type="checkbox" class="accent-brand-600" />
            <span class="text-[12px] text-text-2">Secret</span>
          </label>
          <button
            :disabled="!newEnv.key"
            class="flex items-center gap-1 px-3 py-1.5 text-[12px] font-medium bg-brand-600 text-white border-none rounded-md cursor-pointer hover:bg-brand-700 disabled:opacity-40 disabled:cursor-default transition-colors shrink-0"
            @click="addEnvVar"
          >
            <RiAddLine size="13" />
            Add
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, provide } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../../api/client.js'
import { RiAddLine, RiArrowDownSLine, RiArrowLeftLine, RiDeleteBinLine } from '@remixicon/vue'
import SpinnerButton from '../../foundation/SpinnerButton.vue'
import { useConfirm } from '../../foundation/useConfirm.js'

const route = useRoute()
const router = useRouter()
const workspaceCode = route.params.workspaceCode
const { confirm } = useConfirm()

const scripts = ref([])
const tables = ref([])
const envVars = ref([])
const loading = ref(true)
const saving = ref(false)
const saveError = ref('')
const selected = ref(null)
const form = ref(null)
const envOpen = ref(false)
const newEnv = reactive({ key: '', value: '', is_secret: false })
const tableScopeOpen = ref(false)
const tableScopeEl = ref(null)
const tableColumns = ref({})
const refLoading = ref(false)

const VALID_TABS = ['code', 'executions', 'reference']

const activeTab = computed(() => {
  const segments = route.path.split('/')
  const last = segments[segments.length - 1]
  return VALID_TABS.includes(last) ? last : 'code'
})

const scopeTables = computed(() => {
  if (!form.value) return []
  if (!form.value.table_ids?.length) return tables.value
  return tables.value.filter(t => form.value.table_ids.includes(t.id))
})

const eventContextRows = computed(() => [
  { name: 'event.type',       type: 'string', desc: form.value?.event_type ?? '' },
  { name: 'event.row_id',     type: 'number', desc: 'ID of the row that triggered the script' },
  { name: 'event.table_name', type: 'string', desc: 'name of the table that triggered the script' },
  { name: 'event.table_code', type: 'string', desc: 'code of the triggering table' },
  { name: 'event.data',       type: 'table',  desc: 'row field values keyed by column name' },
])

function colChoices(col) {
  if (col.type !== 'single-select' && col.type !== 'multi-select') return []
  try { return JSON.parse(col.options ?? '{}').choices ?? [] } catch { return [] }
}

const EVENT_LABELS = {
  'record.created': 'Record created',
  'record.updated': 'Record updated',
  'record.deleted': 'Record deleted',
}

function eventLabel(et) {
  return EVENT_LABELS[et] ?? et
}

function tableNameById(id) {
  return tables.value.find(t => t.id === id)?.name ?? `Table #${id}`
}

const tableScopeLabel = computed(() => {
  if (!form.value?.table_ids?.length) return 'All tables'
  if (form.value.table_ids.length === 1) return tableNameById(form.value.table_ids[0])
  return `${form.value.table_ids.length} tables`
})

function formatTs(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function toggleTableScope(tableId) {
  const ids = form.value.table_ids
  const idx = ids.indexOf(tableId)
  if (idx === -1) ids.push(tableId)
  else ids.splice(idx, 1)
}

onClickOutside(tableScopeEl, () => { tableScopeOpen.value = false })

// ── Provide shared state for child tab components ──
provide('workspaceCode', workspaceCode)
provide('scriptForm', form)
provide('tables', tables)
provide('envVars', envVars)
provide('scopeTables', scopeTables)
provide('eventContextRows', eventContextRows)
provide('eventLabel', eventLabel)
provide('formatTs', formatTs)
provide('colChoices', colChoices)
provide('tableColumns', tableColumns)
provide('refLoading', refLoading)

// ── Load scripts list & select from route ──
async function load() {
  loading.value = true
  try {
    const [sc, tbls, ev] = await Promise.all([
      api.listScripts(workspaceCode),
      api.listTables(workspaceCode),
      api.listScriptEnv(workspaceCode),
    ])
    scripts.value = sc
    tables.value = tbls
    envVars.value = ev

    const idParam = route.params.scriptId ? Number(route.params.scriptId) : null
    if (idParam) {
      const match = sc.find(s => s.id === idParam)
      if (match) setForm(match)
    }
  } finally {
    loading.value = false
  }
}

function setForm(sc) {
  selected.value = sc
  form.value = { ...sc, table_ids: [...(sc.table_ids ?? [])] }
  saveError.value = ''
  tableScopeOpen.value = false
  tableColumns.value = {}
}

function clearForm() {
  form.value = null
  selected.value = null
  saveError.value = ''
}

watch(() => route.params.scriptId, (id) => {
  if (id) {
    const match = scripts.value.find(s => s.id === Number(id))
    if (match) setForm(match)
  } else {
    clearForm()
  }
})

onMounted(load)

// ── Navigation ──
const BASE = computed(() => `/workspaces/${workspaceCode}/automations/scripts`)

function navigateTab(tab) {
  if (!form.value?.id) return
  router.replace(`${BASE.value}/${form.value.id}/${tab}`)
}

function selectScript(sc) {
  router.push(`${BASE.value}/${sc.id}/code`)
}

function newScript() {
  clearForm()
  router.replace(BASE.value)
}

function goBack() {
  clearForm()
  router.replace(BASE.value)
}

function cancelEdit() {
  clearForm()
  router.replace(BASE.value)
}

// ── Save / Delete ──
async function saveScript() {
  saveError.value = ''
  saving.value = true
  try {
    const payload = {
      name: form.value.name,
      event_type: form.value.event_type,
      table_ids: form.value.table_ids ?? [],
      code: form.value.code,
      enabled: form.value.enabled,
    }
    if (form.value.id) {
      const updated = await api.updateScript(workspaceCode, form.value.id, payload)
      const idx = scripts.value.findIndex(s => s.id === updated.id)
      if (idx !== -1) scripts.value[idx] = updated
      setForm(updated)
    } else {
      const created = await api.createScript(workspaceCode, payload)
      scripts.value.push(created)
      setForm(created)
      router.replace(`${BASE.value}/${created.id}/code`)
    }
  } catch (e) {
    saveError.value = e.message || 'Failed to save script.'
  } finally {
    saving.value = false
  }
}

async function confirmDelete() {
  const ok = await confirm({ title: 'Delete script?', message: `"${form.value.name}" will be permanently removed.`, confirmLabel: 'Delete' })
  if (!ok) return
  await api.deleteScript(workspaceCode, form.value.id)
  scripts.value = scripts.value.filter(s => s.id !== form.value.id)
  clearForm()
  router.replace(BASE.value)
}

// ── Env vars ──
async function addEnvVar() {
  if (!newEnv.key) return
  const ev = await api.upsertScriptEnv(workspaceCode, newEnv.key, newEnv.value, newEnv.is_secret)
  const idx = envVars.value.findIndex(e => e.id === ev.id)
  if (idx !== -1) envVars.value[idx] = ev
  else envVars.value.push(ev)
  newEnv.key = ''
  newEnv.value = ''
  newEnv.is_secret = false
}

async function updateEnvVar(ev, value) {
  if (!value) return
  const updated = await api.upsertScriptEnv(workspaceCode, ev.key, value, ev.is_secret)
  const idx = envVars.value.findIndex(e => e.id === ev.id)
  if (idx !== -1) envVars.value[idx] = updated
}

async function deleteEnvVar(ev) {
  await api.deleteScriptEnv(workspaceCode, ev.id)
  envVars.value = envVars.value.filter(e => e.id !== ev.id)
}
</script>

<style scoped>
.form-input {
  background: var(--input-bg);
  border: 1px solid var(--input-border);
  border-radius: 6px;
  color: var(--text-1);
  font-size: 13px;
  padding: 5px 9px;
  outline: none;
  box-sizing: border-box;
}
.form-input:focus {
  border-color: var(--color-brand-600);
  box-shadow: 0 0 0 3px rgba(210, 105, 30, 0.12);
}
select.form-input {
  cursor: pointer;
}
</style>
