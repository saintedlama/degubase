<template>
  <div class="flex-1 flex flex-col overflow-y-auto">

    <!-- Top bar -->
    <div class="flex items-center justify-between px-6 py-3 border-b border-border-1 bg-white dark:bg-[#1e1e1e] shrink-0">
      <div>
        <h1 class="text-[15px] font-bold text-text-1">MCP Server</h1>
        <p class="text-[12px] text-text-2 mt-0.5">Connect AI agents to this workspace via the Model Context Protocol.</p>
      </div>
      <label class="flex items-center gap-2.5 cursor-pointer">
        <span class="text-[13px] text-text-2">{{ workspace?.mcp_enabled ? 'Enabled' : 'Disabled' }}</span>
        <button
          type="button"
          class="relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border-2 border-transparent transition-colors cursor-pointer"
          :class="workspace?.mcp_enabled ? 'bg-brand-600' : 'bg-zinc-300 dark:bg-zinc-600'"
          :disabled="toggling"
          @click="toggleMCP"
        >
          <span
            class="inline-block h-3.5 w-3.5 rounded-full bg-white shadow-sm transform transition-transform"
            :class="workspace?.mcp_enabled ? 'translate-x-4' : 'translate-x-0.5'"
          />
        </button>
      </label>
    </div>

    <div class="flex-1 px-6 py-5 flex flex-col gap-6 max-w-2xl">

      <!-- Error -->
      <p v-if="error" class="text-xs text-red-500 bg-red-50 dark:bg-red-900/10 px-3 py-2 rounded-md">{{ error }}</p>

      <!-- Endpoint info (shown when enabled) -->
      <div v-if="workspace?.mcp_enabled" class="flex flex-col gap-2 p-4 rounded-lg border border-border-1 bg-zinc-50 dark:bg-[#252526]">
        <p class="text-[12px] font-semibold text-text-1">SSE endpoint</p>
        <p class="text-[12px] text-text-2">Add this URL to your MCP client configuration:</p>
        <div class="flex items-center gap-2">
          <code class="flex-1 text-[12px] font-mono bg-white dark:bg-[#1e1e1e] border border-border-1 rounded px-3 py-1.5 text-text-1 break-all select-all">{{ sseUrl }}</code>
          <button
            class="shrink-0 flex items-center gap-1.5 px-3 py-1.5 text-[12px] font-medium bg-surface-2 border border-border-1 rounded-md cursor-pointer hover:bg-border-1 transition-colors"
            @click="copyUrl"
          >
            <RiFileCopyLine v-if="!copied" size="13" />
            <RiCheckLine v-else size="13" class="text-green-500" />
            {{ copied ? 'Copied' : 'Copy' }}
          </button>
        </div>
        <p class="text-[11px] text-text-2 mt-1">Authentication: use a workspace API token as the Bearer token.</p>
      </div>

      <!-- Tables section -->
      <div class="flex flex-col gap-3">
        <div class="flex items-baseline justify-between">
          <h2 class="text-[13px] font-semibold text-text-1">Table descriptions</h2>
          <p class="text-[11px] text-text-2">Descriptions help the AI agent understand each table's purpose.</p>
        </div>

        <div v-if="loading" class="text-xs text-text-2 py-2">Loading…</div>
        <div v-else-if="!tables.length" class="text-xs text-text-2 py-2">No tables yet.</div>

        <div
          v-for="tbl in tables"
          :key="tbl.code"
          class="flex flex-col gap-2 p-4 rounded-lg border border-border-1 bg-white dark:bg-[#1e1e1e]"
        >
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span class="text-sm shrink-0 text-text-2">
                <i v-if="tbl.icon" :class="[resolveTableIcon(tbl.icon), 'text-sm leading-none']" />
                <RiTableView v-else size="14" />
              </span>
              <span class="text-[13px] font-medium text-text-1 truncate">{{ tbl.name }}</span>
              <span class="text-[11px] font-mono text-text-2 shrink-0">{{ tbl.code }}</span>
            </div>
            <button
              v-if="editingCode !== tbl.code"
              class="shrink-0 text-[11px] text-brand-600 dark:text-brand-400 hover:underline cursor-pointer bg-transparent border-none"
              @click="startEdit(tbl)"
            >
              {{ tbl.context ? 'Edit' : 'Add description' }}
            </button>
          </div>

          <!-- Display mode -->
          <p
            v-if="editingCode !== tbl.code && tbl.context"
            class="text-[12px] text-text-2 leading-relaxed"
          >{{ tbl.context }}</p>
          <p
            v-else-if="editingCode !== tbl.code"
            class="text-[12px] text-text-2 italic"
          >No description — agents won't know what this table is for.</p>

          <!-- Edit mode -->
          <template v-if="editingCode === tbl.code">
            <textarea
              v-model="editDraft"
              rows="3"
              class="form-input w-full resize-y text-[12px]"
              placeholder="Describe this table's purpose for AI agents…"
              autofocus
            />
            <p v-if="editError" class="text-xs text-red-500">{{ editError }}</p>
            <div class="flex items-center gap-2 justify-end">
              <button
                class="px-3 py-1.5 text-[12px] font-medium text-text-2 bg-surface-2 border border-border-1 rounded-md cursor-pointer hover:bg-border-1 transition-colors"
                @click="cancelEdit"
              >Cancel</button>
              <SpinnerButton
                :loading="editSaving"
                idle="Save"
                busy="Saving…"
                class="px-3 py-1.5 text-[12px] font-medium bg-brand-600 text-white border-none rounded-md cursor-pointer hover:bg-brand-700 disabled:opacity-40 transition-colors"
                @click="saveEdit(tbl)"
              />
            </div>
          </template>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../../api/client.js'
import { resolveTableIcon } from './tableIcons.js'
import { RiFileCopyLine, RiCheckLine, RiTableView } from '@remixicon/vue'
import SpinnerButton from '../../foundation/SpinnerButton.vue'

const route = useRoute()
const workspaceCode = route.params.workspaceCode

const workspace = ref(null)
const tables = ref([])
const loading = ref(true)
const error = ref('')
const toggling = ref(false)

const editingCode = ref(null)
const editDraft = ref('')
const editSaving = ref(false)
const editError = ref('')

const copied = ref(false)

const sseUrl = computed(() => {
  const base = window.location.origin
  return `${base}/api/workspaces/${workspaceCode}/mcp/sse`
})

onMounted(async () => {
  try {
    const [ws, tbls] = await Promise.all([
      api.getWorkspace(workspaceCode),
      api.listTables(workspaceCode),
    ])
    workspace.value = ws
    tables.value = tbls ?? []
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})

async function toggleMCP() {
  if (!workspace.value || toggling.value) return
  toggling.value = true
  error.value = ''
  try {
    const newVal = !workspace.value.mcp_enabled
    const updated = await api.updateWorkspace(workspaceCode, {
      name: workspace.value.name,
      context: workspace.value.context,
      mcp_enabled: newVal,
    })
    workspace.value = updated
  } catch (e) {
    error.value = e.message
  } finally {
    toggling.value = false
  }
}

function copyUrl() {
  navigator.clipboard.writeText(sseUrl.value).then(() => {
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  })
}

function startEdit(tbl) {
  editingCode.value = tbl.code
  editDraft.value = tbl.context ?? ''
  editError.value = ''
}

function cancelEdit() {
  editingCode.value = null
  editDraft.value = ''
  editError.value = ''
}

async function saveEdit(tbl) {
  editSaving.value = true
  editError.value = ''
  try {
    const updated = await api.updateTable(workspaceCode, tbl.code, {
      name: tbl.name,
      context: editDraft.value.trim(),
      icon: tbl.icon,
    })
    const idx = tables.value.findIndex(t => t.code === tbl.code)
    if (idx !== -1) tables.value[idx] = { ...tables.value[idx], context: updated.context }
    cancelEdit()
  } catch (e) {
    editError.value = e.message
  } finally {
    editSaving.value = false
  }
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
</style>
