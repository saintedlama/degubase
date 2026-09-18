<template>
  <div class="flex-1 flex flex-col items-center px-6 py-10 md:px-10 md:py-12">
    <div v-if="loading" class="text-text-2 text-sm">Loading…</div>
    <NotFoundView v-else-if="notFound" />
    <template v-else>
      <!-- Workspace header -->
      <div class="flex flex-col items-center gap-4 mb-8 w-full max-w-2xl">
        <div
          class="w-16 h-16 rounded-2xl flex items-center justify-center text-3xl font-bold"
          :style="workspaceAvatarStyle(workspace.name)"
        >
          {{ workspace.name[0]?.toUpperCase() }}
        </div>
        <div class="text-center">
          <h1 class="text-2xl font-bold text-text-1">{{ workspace.name }}</h1>
          <p v-if="workspace.context" class="text-sm text-text-2 mt-1 max-w-md">{{ workspace.context }}</p>
        </div>
      </div>

      <!-- Tables -->
      <div class="w-full max-w-2xl mb-10">
        <div class="flex items-center justify-between mb-3">
          <span class="text-[22px] font-bold text-text-1">Tables</span>
          <button
            class="flex items-center gap-1.5 px-3 py-1.5 bg-brand-600 hover:bg-brand-700 text-white text-[13px] font-medium rounded-lg transition-colors disabled:opacity-50 border-none cursor-pointer"
            :disabled="creating"
            @click="showCreateDialog = true"
          >
            <RiLoader4Line v-if="creating" size="14" class="animate-spin" />
            <RiAddLine v-else size="14" />
            {{ creating ? 'Creating…' : 'New table' }}
          </button>
        </div>

        <div v-if="tables.length === 0" class="text-sm text-text-2 py-8 text-center bg-surface-1 border border-border-1 rounded-xl">
          No tables yet. Create your first one.
        </div>
        <div v-else class="flex flex-col gap-1.5">
          <RouterLink
            v-for="tbl in tables"
            :key="tbl.code"
            :to="tableHref(tbl)"
            class="flex items-center gap-3 px-4 py-3 bg-surface-1 border border-border-1 rounded-xl hover:border-brand-500 dark:hover:border-brand-600 hover:shadow-sm transition-all no-underline"
          >
            <i v-if="tbl.icon" :class="[resolveTableIcon(tbl.icon), 'text-text-2 shrink-0 text-sm']" />
            <RiTableView v-else size="15" class="text-text-2 shrink-0" />
            <span class="flex-1 text-[14px] font-medium text-text-1">{{ tbl.name }}</span>
            <RiArrowRightSLine size="16" class="text-zinc-300 dark:text-[#6d6d6d] shrink-0" />
          </RouterLink>
        </div>
      </div>

    </template>
  </div>

  <NewTableDialog
    v-if="showCreateDialog"
    @confirm="createTable"
    @cancel="showCreateDialog = false"
  />
</template>

<script setup>
import { ref, watch, inject } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../../api/client.js'
import { RiAddLine, RiLoader4Line, RiTableView, RiArrowRightSLine } from '@remixicon/vue'
import { workspaceAvatarStyle } from '../views/palettes.js'
import NewTableDialog from './NewTableDialog.vue'
import NotFoundView from './NotFoundView.vue'
import { resolveTableIcon } from './tableIcons.js'
import { instantiateTemplate } from './templates.js'

const route = useRoute()
const router = useRouter()
const reloadSidebar = inject('reloadSidebar')

const workspace = ref(null)
const tables = ref([])
const loading = ref(false)
const notFound = ref(false)
const showCreateDialog = ref(false)
const creating = ref(false)

async function load(code) {
  loading.value = true
  notFound.value = false
  try {
    const [ws, tbls] = await Promise.all([api.getWorkspace(code), api.listTables(code)])
    workspace.value = ws
    tables.value = tbls ?? []
  } catch (e) {
    if (e.status === 404) notFound.value = true
    else throw e
  } finally {
    loading.value = false
  }
}

watch(() => route.params.workspaceCode, (code) => code && load(code), { immediate: true })

function tableHref(tbl) {
  const views = tbl.views ?? []
  const view = views.find(v => v.id === tbl.default_view_id) ?? views[0]
  return view
    ? `/workspaces/${route.params.workspaceCode}/tables/${tbl.code}/views/${view.code}`
    : `/workspaces/${route.params.workspaceCode}`
}

async function createTable({ name, context, icon, code, template }) {
  showCreateDialog.value = false
  creating.value = true
  try {
    const workspaceCode = route.params.workspaceCode
    const newTable = await api.createTable(workspaceCode, { name, context, icon, code })
    let targetView = newTable.views?.find(v => v.id === newTable.default_view_id) ?? null
    if (template) {
      targetView = (await instantiateTemplate(api, workspaceCode, newTable, template)) ?? targetView
    }
    reloadSidebar?.()
    if (targetView) {
      router.push(`/workspaces/${workspaceCode}/tables/${newTable.code}/views/${targetView.code}`)
    }
  } finally {
    creating.value = false
  }
}
</script>
