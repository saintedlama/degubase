<template>
  <div class="flex-1 flex flex-col overflow-hidden">

    <!-- Top bar -->
    <div class="flex items-center justify-between px-6 py-3 border-b border-border-1 bg-white dark:bg-[#1e1e1e] shrink-0">
      <h1 class="text-[15px] font-bold text-text-1">API Tokens</h1>
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 bg-brand-600 text-white text-[13px] font-medium rounded-md border-none cursor-pointer hover:bg-brand-700 transition-colors"
        @click="showCreate = true"
      >
        <RiAddLine size="14" />
        New token
      </button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto px-6 py-6">
      <div class="max-w-2xl flex flex-col gap-4">

        <p class="text-[12px] text-text-2">
          Tokens grant agents full access to this workspace. Each token is shown only once — store it securely.
        </p>

        <div v-if="loading" class="text-sm text-text-2">Loading…</div>
        <div v-else-if="!tokens.length" class="text-sm text-text-2">No tokens yet.</div>
        <div v-else class="flex flex-col divide-y divide-zinc-200 dark:divide-[#3c3c3c] border border-border-1 rounded-xl overflow-hidden">
          <div
            v-for="t in tokens"
            :key="t.id"
            class="flex items-center gap-3 px-4 py-3 bg-surface-1"
          >
            <RiKey2Line size="14" class="text-text-2 shrink-0" />
            <div class="flex-1 min-w-0">
              <div class="text-[13px] font-medium text-text-1">{{ t.name }}</div>
              <div class="text-[11px] text-text-2 font-mono">{{ t.prefix }}…</div>
            </div>
            <button
              class="flex items-center justify-center w-7 h-7 text-text-2 hover:text-red-500 bg-transparent border-none cursor-pointer rounded transition-colors"
              title="Revoke token"
              @click="revoke(t)"
            >
              <RiDeleteBinLine size="14" />
            </button>
          </div>
        </div>

      </div>
    </div>

  </div>

  <!-- New token dialog -->
  <ModalDialog
    v-if="showCreate"
    title="New API token"
    confirm-label="Create"
    :confirm-disabled="!tokenName.trim()"
    @confirm="create"
    @cancel="showCreate = false"
  >
    <div class="flex flex-col gap-1.5">
      <label class="text-xs font-semibold text-text-2 tracking-wide">Token name</label>
      <input
        ref="nameInput"
        v-model="tokenName"
        type="text"
        placeholder="e.g. Claude agent"
        autofocus
        class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
        @keyup.enter="create"
      />
      <p v-if="createError" class="text-xs text-red-500 mt-1">{{ createError }}</p>
    </div>
  </ModalDialog>

  <!-- Token reveal (one-time) -->
  <Backdrop v-if="newToken" @dismiss="closeCreate">
    <div class="bg-surface-1 border border-border-1 rounded-xl shadow-xl w-full max-w-md mx-4 p-6 flex flex-col gap-4">
      <div class="flex items-center gap-2">
        <RiCheckLine size="16" class="text-emerald-500 shrink-0" />
        <h2 class="text-[15px] font-bold text-text-1">Token created</h2>
      </div>
      <p class="text-[13px] text-text-2">Copy and save this token now — it won't be shown again.</p>
      <div class="flex items-center gap-2 bg-surface-2 border border-border-1 rounded-lg px-3 py-2.5">
        <code class="flex-1 text-[12px] font-mono text-text-1 break-all select-all">{{ newToken }}</code>
        <button
          class="shrink-0 flex items-center justify-center w-7 h-7 text-zinc-400 hover:text-brand-600 bg-transparent border-none cursor-pointer rounded transition-colors"
          :title="copied ? 'Copied!' : 'Copy'"
          @click="copyToken"
        >
          <RiCheckLine v-if="copied" size="14" class="text-emerald-500" />
          <RiFileCopyLine v-else size="14" />
        </button>
      </div>
      <div class="flex justify-end">
        <button
          class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 transition-colors"
          @click="closeCreate"
        >Done</button>
      </div>
    </div>
  </Backdrop>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../../api/client.js'
import { RiAddLine, RiKey2Line, RiDeleteBinLine, RiFileCopyLine, RiCheckLine } from '@remixicon/vue'
import ModalDialog from '../../foundation/ModalDialog.vue'
import Backdrop from '../../foundation/Backdrop.vue'
import { useConfirm } from '../../foundation/useConfirm.js'

const route = useRoute()
const workspaceCode = route.params.workspaceCode
const { confirm } = useConfirm()

const tokens = ref([])
const loading = ref(true)
const showCreate = ref(false)
const tokenName = ref('')
const newToken = ref('')
const createError = ref('')
const copied = ref(false)
const nameInput = ref(null)

onMounted(load)

async function load() {
  loading.value = true
  try { tokens.value = await api.listTokens(workspaceCode) }
  finally { loading.value = false }
}

async function create() {
  if (!tokenName.value.trim()) return
  createError.value = ''
  try {
    const result = await api.createToken(workspaceCode, { name: tokenName.value.trim() })
    newToken.value = result.token
    tokenName.value = ''
    await load()
  } catch (e) {
    createError.value = e.message || 'Failed to create token.'
  }
}

async function revoke(t) {
  if (!await confirm({ title: 'Revoke token?', message: `"${t.name}" will be permanently revoked. This cannot be undone.`, confirmLabel: 'Revoke' })) return
  try {
    await api.deleteToken(workspaceCode, t.id)
    await load()
  } catch (e) {
    alert(e.message)
  }
}

function closeCreate() {
  showCreate.value = false
  newToken.value = ''
  copied.value = false
}

async function copyToken() {
  await navigator.clipboard.writeText(newToken.value)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}
</script>
