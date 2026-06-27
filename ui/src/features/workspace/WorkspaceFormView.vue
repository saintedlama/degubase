<template>
  <Backdrop @dismiss="$emit('close')">
    <div
      class="bg-surface-1 border border-border-1 rounded-xl shadow-xl flex flex-col overflow-hidden"
      style="width: min(480px, calc(100vw - 32px))"
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-5 pt-5 pb-0">
        <h3 class="text-[15px] font-bold text-text-1">
          {{ isNew ? 'New Workspace' : 'Edit Workspace' }}
        </h3>
        <button
          class="flex items-center justify-center w-6.5 h-6.5 bg-transparent border-none text-text-2 cursor-pointer rounded-md hover:text-text-1 hover:bg-border-1 transition-colors"
          @click="$emit('close')"
        >
          <RiCloseLine size="16" />
        </button>
      </div>

      <!-- Body -->
      <div v-if="loading" class="px-5 py-6 text-sm text-text-2">Loading…</div>
      <form v-else class="px-5 pt-4 pb-5 flex flex-col gap-3.5" @submit.prevent="submit">
        <div class="flex gap-2.5">
          <div class="flex-1 flex flex-col gap-1.5">
            <label class="text-xs font-semibold text-text-2 tracking-wide">Name</label>
            <input
              ref="nameInput"
              v-model="form.name"
              data-testid="ws-name"
              type="text"
              placeholder="My Workspace"
              class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
            />
          </div>
          <div v-if="isNew" class="flex flex-col gap-1.5" style="width:88px">
            <label class="text-xs font-semibold text-text-2 tracking-wide">Code</label>
            <input
              v-model="form.code"
              data-testid="ws-code"
              type="text"
              maxlength="4"
              placeholder="CODE"
              class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all font-mono uppercase tracking-widest"
              @input="codeManuallyEdited = true"
            />
          </div>
        </div>
        <div class="flex flex-col gap-1.5">
          <label class="text-xs font-semibold text-text-2 tracking-wide">
            Context <span class="font-normal text-text-3">(optional)</span>
          </label>
          <textarea
            v-model="form.context"
            placeholder="Describe this workspace's purpose…"
            rows="3"
            class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all resize-none"
          />
        </div>
      </form>

      <!-- Footer -->
      <div class="flex justify-end gap-2 px-5 py-3.5 border-t border-border-1 bg-surface-2">
        <button
          class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-surface-2 text-text-2 border border-zinc-300 dark:border-[#3c3c3c] cursor-pointer hover:bg-border-1 transition-colors"
          @click="$emit('close')"
        >Cancel</button>
        <SpinnerButton
          :loading="saving"
          :disabled="!form.name.trim()"
          :idle="isNew ? 'Create Workspace' : 'Save changes'"
          busy="Saving…"
          class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 disabled:opacity-40 disabled:cursor-default transition-colors"
          @click="submit"
        />
      </div>
    </div>
  </Backdrop>
</template>

<script setup>
import { ref, reactive, computed, nextTick, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../../api/client.js'
import { RiCloseLine } from '@remixicon/vue'
import Backdrop from '../../foundation/Backdrop.vue'
import SpinnerButton from '../../foundation/SpinnerButton.vue'

const props = defineProps({ code: { type: String, default: null } })
const emit = defineEmits(['close', 'saved'])

const router = useRouter()
const isNew = computed(() => !props.code)
const loading = ref(!isNew.value)
const saving = ref(false)
const form = reactive({ name: '', context: '', code: '' })
const nameInput = ref(null)
const codeManuallyEdited = ref(false)

function generateCode(name) {
  const letters = name.toUpperCase().replace(/[^A-Z0-9]/g, '')
  return (letters.slice(0, 4) + 'XXXX').slice(0, 4)
}

watch(() => form.name, (name) => {
  if (!codeManuallyEdited.value) {
    form.code = generateCode(name)
  }
})

onMounted(async () => {
  if (!isNew.value) {
    const ws = await api.getWorkspace(props.code)
    if (ws) { form.name = ws.name; form.context = ws.context }
    loading.value = false
  }
  nextTick(() => nameInput.value?.focus())
})

async function submit() {
  if (!form.name.trim() || saving.value) return
  saving.value = true
  try {
    const data = { name: form.name.trim(), context: form.context.trim() }
    if (isNew.value) {
      const code = (form.code.trim() || generateCode(form.name)).toUpperCase()
      const ws = await api.createWorkspace({ ...data, code })
      emit('close')
      router.push(`/workspaces/${ws.code}`)
    } else {
      const ws = await api.updateWorkspace(props.code, data)
      emit('saved', ws)
      emit('close')
    }
  } finally {
    saving.value = false
  }
}
</script>
