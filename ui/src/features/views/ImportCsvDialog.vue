<template>
  <Backdrop @dismiss="$emit('cancel')">
    <div class="bg-surface-1 border border-border-1 rounded-xl w-140 max-w-[calc(100vw-32px)] shadow-xl flex flex-col overflow-hidden max-h-[90vh]">

      <!-- Header -->
      <div class="flex items-center justify-between px-5 pt-5 pb-0 shrink-0">
        <h3 class="text-[15px] font-bold text-text-1">Import CSV</h3>
        <button
          class="flex items-center justify-center w-6.5 h-6.5 bg-transparent border-none text-text-2 cursor-pointer rounded-md hover:text-text-1 hover:bg-border-1 transition-colors"
          @click="$emit('cancel')"
        ><RiCloseLine size="16" /></button>
      </div>

      <div class="overflow-y-auto flex-1 min-h-0 px-5 py-4">

        <!-- Step 1: File picker -->
        <template v-if="step === 'pick'">
          <p class="text-[12px] text-text-2 mb-4">Upload a CSV file to import records into this table.</p>
          <label
            class="flex flex-col items-center justify-center gap-2 border-2 border-dashed border-border-1 rounded-lg p-8 cursor-pointer hover:border-brand-400 hover:bg-brand-50/50 dark:hover:bg-brand-900/10 transition-colors"
            :class="dragging ? 'border-brand-400 bg-brand-50/50 dark:bg-brand-900/10' : ''"
            @dragover.prevent="dragging = true"
            @dragleave="dragging = false"
            @drop.prevent="onFileDrop"
          >
            <RiUploadLine size="28" class="text-text-3" />
            <span class="text-[13px] font-medium text-text-1">Drop a CSV file or click to browse</span>
            <span class="text-[11px] text-text-3">Max 10 MB</span>
            <input ref="fileInput" type="file" accept=".csv,text/csv" class="hidden" @change="onFileChange" />
          </label>
          <p v-if="error" class="mt-3 text-[12px] text-red-500">{{ error }}</p>
        </template>

        <!-- Step 2: Column mapping -->
        <template v-else-if="step === 'map'">
          <div class="flex items-center gap-2 mb-4">
            <RiFileLine size="14" class="text-text-3 shrink-0" />
            <span class="text-[12px] text-text-1 font-medium truncate">{{ file?.name }}</span>
            <button class="ml-auto text-[11px] text-text-2 bg-transparent border-none cursor-pointer hover:text-brand-600 dark:hover:text-brand-400 shrink-0" @click="reset">Change file</button>
          </div>

          <div class="text-[11px] font-semibold text-text-3 uppercase tracking-wide mb-2">Map CSV columns to table fields</div>

          <div class="border border-border-1 rounded-lg overflow-hidden mb-4">
            <div class="grid text-[11px] font-semibold text-text-3 bg-surface-2 border-b border-border-1 px-3 py-1.5" style="grid-template-columns: 1fr auto 1fr">
              <span>CSV column</span>
              <span />
              <span>Table field</span>
            </div>
            <div
              v-for="header in preview.headers"
              :key="header"
              class="grid items-center gap-2 px-3 py-1.5 border-b border-border-1 last:border-b-0"
              style="grid-template-columns: 1fr auto 1fr"
            >
              <span class="text-[12px] text-text-1 truncate font-mono">{{ header }}</span>
              <RiArrowRightLine size="12" class="text-text-3 shrink-0" />
              <select
                class="h-6 text-[11px] px-1.5 rounded border border-border-1 bg-surface-2 text-text-1 outline-none"
                :value="mapping[header] ?? ''"
                @change="mapping[header] = $event.target.value || null"
              >
                <option value="">— skip —</option>
                <option v-for="col in preview.columns" :key="col.code" :value="col.code">{{ col.name }}</option>
              </select>
            </div>
          </div>

          <!-- Sample data preview -->
          <details v-if="preview.sample?.length" class="mb-1">
            <summary class="text-[11px] text-text-3 cursor-pointer select-none hover:text-text-2 transition-colors">
              Preview first {{ preview.sample.length }} row{{ preview.sample.length !== 1 ? 's' : '' }}
            </summary>
            <div class="mt-2 overflow-x-auto rounded border border-border-1">
              <table class="text-[11px] text-text-1 border-collapse min-w-full">
                <thead>
                  <tr>
                    <th v-for="h in preview.headers" :key="h" class="px-2 py-1 bg-surface-2 border-b border-border-1 text-left font-medium text-text-3 whitespace-nowrap">{{ h }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(row, i) in preview.sample" :key="i">
                    <td v-for="(cell, j) in row" :key="j" class="px-2 py-1 border-b border-border-1 last:border-b-0 whitespace-nowrap max-w-40 truncate">{{ cell }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </details>

          <p v-if="error" class="mt-2 text-[12px] text-red-500">{{ error }}</p>
        </template>

        <!-- Step 3: Done -->
        <template v-else-if="step === 'done'">
          <div class="flex flex-col items-center gap-3 py-6">
            <RiCheckboxCircleLine size="40" class="text-brand-600 dark:text-brand-400" />
            <p class="text-[14px] font-semibold text-text-1">Import complete</p>
            <p class="text-[13px] text-text-2">{{ importedCount }} record{{ importedCount !== 1 ? 's' : '' }} imported successfully.</p>
          </div>
        </template>

      </div>

      <!-- Footer -->
      <div class="shrink-0 flex items-center gap-2 px-5 py-3.5 border-t border-border-1 bg-surface-2">
        <button
          class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-surface-2 text-text-2 border border-zinc-300 dark:border-[#3c3c3c] cursor-pointer hover:bg-border-1 transition-colors"
          @click="step === 'done' ? $emit('done', importedCount) : $emit('cancel')"
        >{{ step === 'done' ? 'Close' : 'Cancel' }}</button>

        <div class="ml-auto">
          <button
            v-if="step === 'pick'"
            class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 disabled:opacity-40 disabled:cursor-default transition-colors"
            :disabled="!file || loading"
            @click="uploadForPreview"
          >
            <span v-if="loading">Analysing…</span>
            <span v-else>Next →</span>
          </button>
          <button
            v-else-if="step === 'map'"
            :disabled="!hasMappedColumn || loading"
            class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 disabled:opacity-40 disabled:cursor-default transition-colors"
            @click="runImport"
          >
            <span v-if="loading">Importing…</span>
            <span v-else>Import</span>
          </button>
        </div>
      </div>

    </div>
  </Backdrop>
</template>

<script setup>
import { ref, computed, reactive } from 'vue'
import { RiCloseLine, RiUploadLine, RiFileLine, RiArrowRightLine, RiCheckboxCircleLine } from '@remixicon/vue'
import Backdrop from '../../foundation/Backdrop.vue'
import { api } from '../../api/client.js'

const props = defineProps({
  workspaceCode: { type: String, required: true },
  tableCode:     { type: String, required: true },
})
const emit = defineEmits(['done', 'cancel'])

const step = ref('pick')   // 'pick' | 'map' | 'done'
const file = ref(null)
const preview = ref(null)  // ImportPreviewResponse
const mapping = reactive({})
const importedCount = ref(0)
const loading = ref(false)
const error = ref('')
const dragging = ref(false)
const fileInput = ref(null)

const hasMappedColumn = computed(() =>
  Object.values(mapping).some(v => v && v !== '')
)

function onFileChange(e) {
  const f = e.target.files?.[0]
  if (f) setFile(f)
}

function onFileDrop(e) {
  dragging.value = false
  const f = e.dataTransfer.files?.[0]
  if (f) setFile(f)
}

function setFile(f) {
  if (!f.name.endsWith('.csv') && f.type !== 'text/csv') {
    error.value = 'Please upload a .csv file'
    return
  }
  file.value = f
  error.value = ''
}

function reset() {
  step.value = 'pick'
  file.value = null
  preview.value = null
  Object.keys(mapping).forEach(k => delete mapping[k])
  importedCount.value = 0
  error.value = ''
  if (fileInput.value) fileInput.value.value = ''
}

async function uploadForPreview() {
  if (!file.value) return
  loading.value = true
  error.value = ''
  try {
    const res = await api.importCsvPreview(props.workspaceCode, props.tableCode, file.value)
    preview.value = res
    // Pre-populate mapping from suggestions
    Object.keys(mapping).forEach(k => delete mapping[k])
    for (const [header, code] of Object.entries(res.suggested_mapping ?? {})) {
      mapping[header] = code
    }
    step.value = 'map'
  } catch (e) {
    error.value = e.message || 'Failed to analyse CSV'
  } finally {
    loading.value = false
  }
}

async function runImport() {
  loading.value = true
  error.value = ''
  try {
    const effectiveMapping = Object.fromEntries(
      Object.entries(mapping).filter(([, v]) => v && v !== '')
    )
    const res = await api.importCsvConfirm(props.workspaceCode, props.tableCode, file.value, effectiveMapping)
    importedCount.value = res.imported
    step.value = 'done'
  } catch (e) {
    error.value = e.message || 'Import failed'
  } finally {
    loading.value = false
  }
}
</script>
