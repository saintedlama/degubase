<template>
  <div class="flex-1 overflow-y-auto p-4 bg-[#282c34]">
    <!-- Editing: live preview via /preview endpoint (reflects unsaved changes) -->
    <template v-if="editing">
      <div v-if="fetchingPreview" class="flex items-center gap-2 text-zinc-400 text-[12px]">
        <RiLoader4Line size="13" class="animate-spin" /> Loading…
      </div>
      <pre v-else class="text-[12px] font-mono leading-relaxed whitespace-pre-wrap text-zinc-100">{{ previewMarkdown }}</pre>
    </template>
    <!-- Viewing: current saved skill via /context endpoint -->
    <template v-else>
      <div v-if="loadingContext" class="flex items-center gap-2 text-zinc-400 text-[12px]">
        <RiLoader4Line size="13" class="animate-spin" /> Loading…
      </div>
      <pre v-else class="text-[12px] font-mono leading-relaxed whitespace-pre-wrap text-zinc-100">{{ skillContext }}</pre>
    </template>
  </div>
</template>

<script setup>
import { ref, inject, watchEffect, onActivated } from 'vue'
import { RiLoader4Line } from '@remixicon/vue'

const workspaceCode = inject('workspaceCode')
const form = inject('skillForm')
const editing = inject('editing')
const previewMarkdown = inject('previewMarkdown')
const fetchingPreview = inject('fetchingPreview')

const skillContext = ref('')
const loadingContext = ref(false)

async function fetchContext() {
  if (!form.value?.id || editing.value) return
  loadingContext.value = true
  skillContext.value = ''
  try {
    const resp = await fetch(`/api/workspaces/${workspaceCode}/skills/${form.value.id}/context`)
    skillContext.value = await resp.text()
  } catch {
    skillContext.value = '(failed to load)'
  } finally {
    loadingContext.value = false
  }
}

watchEffect(() => {
  if (!editing.value && form.value?.id) {
    fetchContext()
  }
})

onActivated(() => {
  if (!editing.value && form.value?.id) {
    fetchContext()
  }
})
</script>
