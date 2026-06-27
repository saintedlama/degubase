<template>
  <div class="flex-1 flex flex-col overflow-hidden">

    <!-- Top bar -->
    <div class="flex items-center justify-between px-6 py-3 border-b border-border-1 bg-white dark:bg-[#1e1e1e] shrink-0">
      <h1 class="text-[15px] font-bold text-text-1">Users</h1>
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 bg-brand-600 text-white text-[13px] font-medium rounded-md border-none cursor-pointer hover:bg-brand-700 transition-colors"
        @click="showCreate = true"
      >
        <RiAddLine size="14" />
        Add user
      </button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto px-6 py-6">
      <div class="max-w-2xl flex flex-col gap-4">

        <div v-if="loading" class="text-sm text-text-2">Loading…</div>
        <div v-else-if="!users.length" class="text-sm text-text-2">No users yet.</div>
        <div v-else class="flex flex-col divide-y divide-zinc-200 dark:divide-[#3c3c3c] border border-border-1 rounded-xl overflow-hidden">
          <div
            v-for="u in users"
            :key="u.id"
            class="flex items-center gap-3 px-4 py-3 bg-surface-1"
          >
            <div class="w-8 h-8 rounded-full bg-brand-100 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400 flex items-center justify-center text-sm font-bold shrink-0">
              {{ (u.name || u.username)[0].toUpperCase() }}
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-[13px] font-medium text-text-1 truncate">
                {{ u.name || u.username }}
                <span v-if="u.is_admin" class="ml-1.5 text-[10px] font-semibold text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/20 px-1.5 py-px rounded">Admin</span>
              </div>
              <div class="text-[11px] text-text-2 truncate">{{ u.username }}</div>
            </div>
            <button
              v-if="u.id !== currentUser?.id"
              class="flex items-center justify-center w-7 h-7 text-text-2 hover:text-red-500 bg-transparent border-none cursor-pointer rounded transition-colors"
              title="Remove user"
              @click="removeUser(u)"
            >
              <RiDeleteBinLine size="14" />
            </button>
          </div>
        </div>

      </div>
    </div>

  </div>

  <!-- Add user dialog -->
  <ModalDialog
    v-if="showCreate"
    title="Add user"
    confirm-label="Create"
    :confirm-disabled="!form.username || !form.password"
    @confirm="createUser"
    @cancel="closeCreate"
  >
    <div class="flex flex-col gap-1.5">
      <label class="text-xs font-semibold text-text-2 tracking-wide">Name</label>
      <input v-model="form.name" type="text" placeholder="Name" autofocus class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all" />
    </div>
    <div class="flex flex-col gap-1.5">
      <label class="text-xs font-semibold text-text-2 tracking-wide">Username</label>
      <input v-model="form.username" type="text" placeholder="username" class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all" />
    </div>
    <div class="flex flex-col gap-1.5">
      <label class="text-xs font-semibold text-text-2 tracking-wide">Password</label>
      <input v-model="form.password" type="password" placeholder="••••••••" class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all" />
    </div>
    <label class="flex items-center gap-2 cursor-pointer">
      <input v-model="form.is_admin" type="checkbox" class="w-4 h-4 rounded accent-brand-600" />
      <span class="text-[13px] text-text-1">Admin</span>
    </label>
    <p v-if="createError" class="text-xs text-red-500">{{ createError }}</p>
  </ModalDialog>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { api } from '../../api/client.js'
import { useAuth } from './useAuth.js'
import { RiAddLine, RiDeleteBinLine } from '@remixicon/vue'
import ModalDialog from '../../foundation/ModalDialog.vue'
import { useConfirm } from '../../foundation/useConfirm.js'

const { user: currentUser } = useAuth()

const users = ref([])
const loading = ref(true)
const showCreate = ref(false)
const createError = ref('')
const form = reactive({ name: '', username: '', password: '', is_admin: false })
const { confirm } = useConfirm()

onMounted(load)

async function load() {
  loading.value = true
  try { users.value = await api.listUsers() }
  finally { loading.value = false }
}

async function createUser() {
  createError.value = ''
  try {
    await api.createUser({ name: form.name, username: form.username, password: form.password, is_admin: form.is_admin })
    closeCreate()
    await load()
  } catch (e) {
    createError.value = e.message || 'Failed to create user.'
  }
}

function closeCreate() {
  showCreate.value = false
  createError.value = ''
  Object.assign(form, { name: '', username: '', password: '', is_admin: false })
}

async function removeUser(u) {
  if (!await confirm({ title: 'Remove user?', message: `${u.name || u.username} will be removed from this workspace.`, confirmLabel: 'Remove' })) return
  try {
    await api.deleteUser(u.id)
    await load()
  } catch (e) {
    alert(e.message)
  }
}
</script>

