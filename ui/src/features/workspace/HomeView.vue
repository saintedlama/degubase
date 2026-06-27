<template>
  <div class="min-h-screen flex flex-col bg-zinc-100 dark:bg-[#1e1e1e]">

    <AppTopNav />

    <!-- Content -->
    <div class="flex-1 p-10 max-w-[960px] w-full mx-auto">
      <div class="flex items-start justify-between mb-7">
        <div>
          <h1 class="text-[22px] font-bold text-text-1 mb-1">Workspaces</h1>
          <p class="text-[13px] text-text-2">Choose a workspace to get started, or create a new one.</p>
        </div>
        <button
          class="inline-flex items-center gap-1.5 px-3.5 py-1.75 bg-brand-600 text-white rounded-md text-[13px] font-medium border-none cursor-pointer hover:bg-brand-700 transition-colors shrink-0"
          @click="openCreate"
        >
          <RiAddLine size="15" />
          New workspace
        </button>
      </div>

      <div v-if="loading" class="text-text-2 text-sm py-8">Loading…</div>
      <div v-else-if="error" class="text-red-500 text-sm py-8">{{ error }}</div>
      <div v-else-if="workspaces.length === 0" class="flex items-center gap-2 text-text-2 text-sm py-8">
        No workspaces yet.
        <button class="text-brand-600 bg-transparent border-none cursor-pointer p-0 hover:underline text-sm" @click="openCreate">Create one to get started →</button>
      </div>

      <div v-else class="grid gap-3.5" style="grid-template-columns: repeat(auto-fill, minmax(260px, 1fr))">
        <RouterLink
          v-for="ws in workspaces"
          :key="ws.code"
          :to="`/workspaces/${ws.code}`"
          class="bg-surface-1 border border-border-1 rounded-xl p-5 flex items-start gap-3.5 shadow-sm hover:border-brand-600 hover:shadow-[0_0_0_3px_rgba(210,105,30,0.12)] dark:hover:shadow-[0_0_0_3px_rgba(210,105,30,0.12)] transition-all no-underline"
        >
          <div
            class="w-9.5 h-9.5 rounded-[9px] flex items-center justify-center text-base font-bold shrink-0"
            :style="workspaceAvatarStyle(ws.name)"
          >
            {{ ws.name[0]?.toUpperCase() }}
          </div>
          <div class="flex-1 min-w-0 flex flex-col gap-1">
            <span class="text-sm font-semibold text-text-1">{{ ws.name }}</span>
            <span v-if="ws.context" class="text-xs text-text-2 truncate">{{ ws.context }}</span>
          </div>
          <div class="flex gap-0.5 shrink-0" @click.stop>
            <IconButton
              :border="false"
              title="Edit"
              class="w-6.5 h-6.5 rounded-md hover:bg-border-1"
              @click="openEdit(ws)"
            >
              <RiPencilLine size="14" />
            </IconButton>
            <IconButton
              :border="false"
              danger
              title="Delete"
              class="w-6.5 h-6.5 rounded-md hover:bg-red-50 dark:hover:bg-red-950/40"
              @click="deleteWorkspace(ws)"
            >
              <RiDeleteBin6Line size="14" />
            </IconButton>
          </div>
        </RouterLink>
      </div>
    </div>
  </div>

  <WorkspaceFormView
    v-if="dialogCode !== undefined"
    :code="dialogCode"
    @close="dialogCode = undefined"
    @saved="onSaved"
  />
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../../api/client.js'
import AppTopNav from '../../foundation/AppTopNav.vue'
import WorkspaceFormView from './WorkspaceFormView.vue'
import { RiAddLine, RiPencilLine, RiDeleteBin6Line } from '@remixicon/vue'
import { workspaceAvatarStyle } from '../views/palettes.js'
import IconButton from '../../foundation/IconButton.vue'
import { useConfirm } from '../../foundation/useConfirm.js'

const { confirm } = useConfirm()
const workspaces = ref([])
const loading = ref(true)
const error = ref(null)

// undefined = hidden, null = create new, string = edit (workspace code)
const dialogCode = ref(undefined)

onMounted(load)

async function load() {
  try {
    workspaces.value = await api.listWorkspaces()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function openCreate() { dialogCode.value = null }
function openEdit(ws) { dialogCode.value = ws.code }

function onSaved(ws) {
  const idx = workspaces.value.findIndex(w => w.id === ws.id)
  if (idx !== -1) workspaces.value[idx] = ws
}

async function deleteWorkspace(ws) {
  if (!await confirm({ title: 'Delete workspace?', message: `"${ws.name}" and all its tables will be permanently deleted. This cannot be undone.`, confirmLabel: 'Delete' })) return
  await api.deleteWorkspace(ws.code)
  workspaces.value = workspaces.value.filter(w => w.code !== ws.code)
}
</script>
