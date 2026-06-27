<template>
  <div class="flex flex-col flex-1 min-h-0 overflow-hidden bg-surface-1">

    <RecordToolbar
      :row="row"
      :row-title="rowTitle"
      :save-status="saveStatus"
      :workspace-code="workspaceCode"
      :table-code="tableCode"
      :prev-row-id="prevRowId"
      :next-row-id="nextRowId"
      @close="close"
      @create-row="createRow"
      @duplicate-row="duplicateRow"
      @delete-row="deleteRow"
      @navigate="navigate"
    />

    <!-- Form body -->
    <div ref="formEl" class="flex-1 overflow-auto">
      <div class="max-w-2xl mx-auto px-8 py-8 flex flex-col gap-7">

        <div v-for="col in sortedColumns" :key="col.id">
          <!-- Field label -->
          <div class="flex items-center gap-1.5 mb-2">
            <span class="flex items-center justify-center w-5 h-5 text-text-2 bg-border-1 rounded shrink-0">
              <component :is="typeIcon(col.type)" size="12" />
            </span>
            <label class="text-xs font-semibold text-text-2 tracking-wide">{{ col.name }}</label>
          </div>


          <FieldEditor
            v-model="formData[col.code]"
            :col="col"
            :row="row"
            :workspace-code="workspaceCode"
            :table-code="tableCode"
            :row-id="row.id"
            variant="form"
            @commit="onFieldCommit(col)"
            @file-upload="onFileUpload(col, $event)"
            @file-remove="onFileRemove(col)"
            @lightbox="lightboxSrc = $event.src; lightboxAlt = $event.alt"
            @column-updated="emit('column-updated', $event)"
          />
        </div>

        <!-- Referencing records panel -->
        <div class="border-t border-border-1 pt-6 mt-2">
          <button
            class="flex items-center gap-2 w-full text-left bg-transparent border-none cursor-pointer p-0 group"
            @click="toggleRefs"
          >
            <RiLinksLine size="14" class="text-text-2 shrink-0" />
            <span class="text-xs font-semibold text-text-2 tracking-wide group-hover:text-zinc-700 dark:group-hover:text-[#d4d4d4] transition-colors">
              Referenced by
              <span v-if="referencingTotal > 0" class="font-normal text-text-3">({{ referencingTotal }})</span>
            </span>
            <RiArrowDownSLine v-if="!refsOpen" size="14" class="ml-auto text-text-2" />
            <RiArrowUpSLine v-else size="14" class="ml-auto text-text-2" />
          </button>

          <div v-if="refsOpen" class="mt-3">
            <div v-if="refsLoading" class="text-xs text-text-3 py-2">Loading…</div>
            <div v-else-if="!referencingRecords.length" class="text-xs text-text-3 py-2">Not referenced by any records.</div>
            <div v-else class="flex flex-col gap-1">
              <a
                v-for="ref in referencingRecords"
                :key="ref.sourceRowId"
                class="flex items-center gap-2 px-3 py-2 rounded-lg text-[12px] text-text-1 no-underline hover:bg-border-1 transition-colors cursor-pointer"
                @click="navigateToRef(ref)"
              >
                <RiNodeTree size="11" class="text-text-3 shrink-0" />
                <span class="text-text-2 text-[11px] shrink-0">{{ ref.sourceTableName }}</span>
                <span class="truncate">{{ ref.sourceRowLabel }}</span>
              </a>
            </div>
          </div>
        </div>

        <!-- Activity panel -->
        <div class="border-t border-border-1 pt-6 mt-2">
          <button
            class="flex items-center gap-2 w-full text-left bg-transparent border-none cursor-pointer p-0 group"
            @click="toggleHistory"
          >
            <RiTimeLine size="14" class="text-text-2 shrink-0" />
            <span class="text-xs font-semibold text-text-2 tracking-wide group-hover:text-zinc-700 dark:group-hover:text-[#d4d4d4] transition-colors">
              Activity
              <span v-if="history.length" class="font-normal text-text-3">({{ history.length }})</span>
            </span>
            <RiArrowDownSLine v-if="!historyOpen" size="14" class="ml-auto text-text-2" />
            <RiArrowUpSLine v-else size="14" class="ml-auto text-text-2" />
          </button>

          <div v-if="historyOpen" class="mt-3 flex flex-col gap-3">
            <!-- Add annotation input -->
            <div class="flex flex-col gap-1.5" @keydown.ctrl.enter.prevent="submitAnnotation" @keydown.meta.enter.prevent="submitAnnotation">
              <MarkdownEditor v-model="newAnnotation" />
              <div class="flex items-center justify-between">
                <span class="text-[10px] text-text-3">Ctrl+Enter to submit · markdown supported</span>
                <button
                  class="text-xs font-medium px-2.5 py-1 rounded-md bg-brand-600 text-white hover:bg-brand-700 disabled:opacity-40 transition-colors"
                  :disabled="!newAnnotation.trim() || annotationSaving"
                  @click="submitAnnotation"
                >Add</button>
              </div>
            </div>

            <div v-if="historyLoading" class="text-xs text-text-3 py-2">Loading…</div>
            <div v-else-if="!history.length" class="text-xs text-text-3 py-2">No activity yet.</div>

            <!-- Change entry -->
            <template v-for="(entry, i) in history" :key="entry.id">
              <div
                v-if="entry.entry_type === 'change'"
                class="rounded-lg border border-border-1 bg-surface-2 px-3 py-3 flex flex-col gap-2"
              >
                <div class="flex items-center justify-between gap-2">
                  <div class="flex items-center gap-1.5 min-w-0">
                    <RiEditLine size="11" class="text-text-3 shrink-0" />
                    <span class="text-[11px] font-medium text-text-2" :title="formatDate(entry.changed_at)">
                      {{ relativeTime(entry.changed_at) }}
                    </span>
                  </div>
                  <button
                    class="text-[11px] font-medium text-text-2 hover:text-brand-600 dark:hover:text-brand-400 bg-transparent border-none cursor-pointer px-0 py-0 transition-colors shrink-0"
                    @click="revert(entry)"
                  >Revert</button>
                </div>
                <div v-if="diffFor(entry, i).length" class="flex flex-col gap-1">
                  <div v-for="change in diffFor(entry, i)" :key="change.col.id" class="text-[11px] leading-snug">
                    <span class="font-semibold text-zinc-600 dark:text-[#c9c9c9]">{{ change.col.name }}: </span>
                    <span class="text-red-500 dark:text-red-400 line-through mr-1">{{ displayVal(change.before) }}</span>
                    <span class="text-emerald-600 dark:text-emerald-400">{{ displayVal(change.after) }}</span>
                  </div>
                </div>
                <div v-else class="text-[11px] text-text-3">No tracked field changes.</div>
                <!-- Inline annotation on change entry -->
                <div v-if="editingId !== entry.id" class="flex items-start gap-1.5 mt-0.5">
                  <MarkdownViewer
                    v-if="entry.annotation"
                    :content="entry.annotation"
                    class="flex-1 text-[11px] italic text-text-2 leading-snug"
                  />
                  <button
                    class="text-[10px] text-text-3 hover:text-brand-600 dark:hover:text-brand-400 bg-transparent border-none cursor-pointer px-0 py-0 shrink-0 transition-colors"
                    @click="startEdit(entry)"
                  >{{ entry.annotation ? 'Edit note' : '+ Add note' }}</button>
                </div>
                <div v-else class="flex flex-col gap-1.5 mt-0.5">
                  <MarkdownEditor v-model="editText" />
                  <div class="flex gap-1.5 justify-end">
                    <button class="text-[11px] px-2 py-1 rounded bg-brand-600 text-white hover:bg-brand-700 transition-colors" @click="saveEdit(entry)">Save</button>
                    <button class="text-[11px] px-2 py-1 rounded bg-border-1 text-text-2 hover:bg-zinc-300 dark:hover:bg-[#4a4a4a] transition-colors" @click="cancelEdit">Cancel</button>
                  </div>
                </div>
              </div>

              <!-- Annotation entry -->
              <div
                v-else
                class="rounded-lg border border-amber-200 dark:border-[#4a3f2a] bg-amber-50 dark:bg-[#2e2a1e] px-3 py-3 flex flex-col gap-1.5"
              >
                <div class="flex items-center justify-between gap-2">
                  <div class="flex items-center gap-1.5 min-w-0">
                    <RiStickyNoteLine size="11" class="text-amber-500 dark:text-amber-400 shrink-0" />
                    <span class="text-[11px] font-medium text-text-2" :title="formatDate(entry.changed_at)">
                      {{ relativeTime(entry.changed_at) }}
                    </span>
                  </div>
                  <div class="flex items-center gap-1.5 shrink-0">
                    <button
                      class="text-[10px] text-text-3 hover:text-brand-600 dark:hover:text-brand-400 bg-transparent border-none cursor-pointer px-0 py-0 transition-colors"
                      @click="startEdit(entry)"
                    >Edit</button>
                    <button
                      class="text-[10px] text-text-3 hover:text-red-500 dark:hover:text-red-400 bg-transparent border-none cursor-pointer px-0 py-0 transition-colors"
                      @click="deleteAnnotation(entry)"
                    >Delete</button>
                  </div>
                </div>
                <div v-if="editingId !== entry.id">
                  <MarkdownViewer
                    :content="entry.annotation"
                    class="text-[11px] text-text-1 leading-snug"
                  />
                </div>
                <div v-else class="flex flex-col gap-1.5">
                  <MarkdownEditor v-model="editText" />
                  <div class="flex gap-1.5 justify-end">
                    <button class="text-[11px] px-2 py-1 rounded bg-brand-600 text-white hover:bg-brand-700 transition-colors" @click="saveEdit(entry)">Save</button>
                    <button class="text-[11px] px-2 py-1 rounded bg-border-1 text-text-2 hover:bg-zinc-300 dark:hover:bg-[#4a4a4a] transition-colors" @click="cancelEdit">Cancel</button>
                  </div>
                </div>
              </div>
            </template>
          </div>
        </div>

      </div>
    </div>
  </div>

  <ImageLightbox
    v-if="lightboxSrc"
    :src="lightboxSrc"
    :alt="lightboxAlt"
    @close="lightboxSrc = null"
  />
</template>

<script setup>
import { ref, computed, reactive, watch, onMounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../../api/client.js'
import FieldEditor from './FieldEditor.vue'
import ImageLightbox from './ImageLightbox.vue'
import MarkdownEditor from '../../foundation/MarkdownEditor.vue'
import MarkdownViewer from '../../foundation/MarkdownViewer.vue'
import { useNotifications } from '../../foundation/useNotifications.js'
import { useConfirm } from '../../foundation/useConfirm.js'
import { useReferencesBlock } from '../../foundation/useReferencesBlock.js'
import RecordToolbar from './RecordToolbar.vue'
import {
  RiTimeLine, RiEditLine, RiStickyNoteLine, RiLinksLine, RiArrowDownSLine, RiArrowUpSLine, RiNodeTree,
} from '@remixicon/vue'
import { typeIcon } from '../schema/columnTypes.js'

const props = defineProps({
  workspaceCode: { type: String, required: true },
  tableCode: { type: String, required: true },
  table: { type: Object, required: true },
  row: { type: Object, required: true },
  rowIndex: { type: Number, default: 0 },
  prevRowId: { type: Number, default: null },
  nextRowId: { type: Number, default: null },
  autoFocus: { type: Boolean, default: false },
})
const emit = defineEmits(['close', 'row-updated', 'navigate', 'row-deleted', 'row-created', 'column-updated'])

const route = useRoute()
const router = useRouter()

const lightboxSrc = ref(null)
const lightboxAlt = ref('')
const { notify } = useNotifications()
const { confirm } = useConfirm()
const { showBlock } = useReferencesBlock()

let revisionId = null  // tracks the current editing session's revision

const formEl = ref(null)

const sortedColumns = computed(() =>
  [...(props.table.columns ?? [])].sort((a, b) => a.position - b.position)
)

const rowTitle = computed(() => {
  const firstText = sortedColumns.value.find(c =>
    ['text', 'long-text', 'email', 'url'].includes(c.type)
  )
  if (!firstText) return ''
  return props.row.data?.[firstText.code] ?? ''
})

onMounted(() => {
  if (!props.autoFocus) return
  nextTick(() => {
    const first = formEl.value?.querySelector(
      'input.form-input:not([type="file"]):not([type="checkbox"]), textarea.form-input'
    )
    first?.focus()
  })
})

const formData = reactive({})
let dirty = false
let syncTimer = null
const initialData = {}

function initForm() {
  clearTimeout(syncTimer)
  dirty = false
  for (const col of props.table.columns ?? []) {
    if (col.type === 'created-at' || col.type === 'updated-at') continue
    const val = props.row.data?.[col.code]
    const init = val != null ? val : (col.type === 'multi-select' || col.type === 'checklist' ? [] : '')
    formData[col.code] = init
    initialData[col.code] = JSON.stringify(init)
  }
}
initForm()
watch(() => props.row, initForm, { deep: false })

const saveStatus = ref('')

function scheduleSync() {
  dirty = true
  saveStatus.value = 'Saving…'
  clearTimeout(syncTimer)
  syncTimer = setTimeout(save, 600)
}

function dirtyAndSave() {
  dirty = true
  save()
}

function onFieldCommit(col) {
  if (col.type === 'markdown') scheduleSync()
  else dirtyAndSave()
}

async function save() {
  clearTimeout(syncTimer)
  if (!dirty) return
  dirty = false
  saveStatus.value = 'Saving…'
  try {
    const changed = {}
    for (const [key, val] of Object.entries(formData)) {
      if (JSON.stringify(val) !== initialData[key]) {
        changed[key] = val
        initialData[key] = JSON.stringify(val)
      }
    }
    const { row: updated, revisionId: newRevId } = await api.patchRow(
      props.workspaceCode,
      props.tableCode,
      props.row.id,
      { data: changed },
      revisionId,
    )
    revisionId = newRevId
    props.row.data = updated.data
    props.row.updated_at = updated.updated_at
    emit('row-updated', updated)
    saveStatus.value = 'Saved'
    setTimeout(() => { saveStatus.value = '' }, 2000)
    if (historyLoaded.value) {
      historyLoaded.value = false
      loadHistory()
    }
  } catch (e) {
    dirty = true
    saveStatus.value = 'Error saving'
    console.error('Row save failed:', e)
  }
}

async function onFileUpload(col, event) {
  const file = event.target.files?.[0]
  if (!file) return
  try {
    const updated = await api.uploadFile(
      props.workspaceCode, props.tableCode, props.row.id, col.code, file
    )
    props.row.data = updated.data
    props.row.updated_at = updated.updated_at
    formData[col.code] = updated.data?.[col.code]
    emit('row-updated', updated)
  } catch (e) {
    notify(e.message || 'File upload failed')
  }
  event.target.value = ''
}

async function onFileRemove(col) {
  const newData = { ...(props.row.data ?? {}), [col.code]: null }
  try {
    const { row: updated, revisionId: newRevId } = await api.updateRow(props.workspaceCode, props.tableCode, props.row.id, { data: newData }, null)
    revisionId = newRevId
    props.row.data = updated.data
    props.row.updated_at = updated.updated_at
    formData[col.code] = null
    emit('row-updated', updated)
  } catch (e) {
    notify(e.message || 'File removal failed')
  }
}

// ── Referencing records ──────────────────────────────────────────────────────

const refsOpen = ref(false)
const refsLoading = ref(false)
const refsLoaded = ref(false)
const referencingRecords = ref([])
const referencingTotal = ref(0)

async function loadRefs() {
  refsLoading.value = true
  try {
    const res = await api.listReferencingRows(
      props.workspaceCode,
      props.tableCode,
      props.row.id,
      { page_size: 100 }
    )
    referencingRecords.value = res.data ?? []
    referencingTotal.value = res.total ?? 0
    refsLoaded.value = true
  } catch (e) {
    console.error('Failed to load referencing records:', e)
  } finally {
    refsLoading.value = false
  }
}

async function toggleRefs() {
  refsOpen.value = !refsOpen.value
  if (refsOpen.value && !refsLoaded.value) {
    await loadRefs()
  }
}

function navigateToRef(ref) {
  router.push(`/workspaces/${props.workspaceCode}/tables/${ref.sourceTableCode}/rows/${ref.sourceRowId}?returnTo=${encodeURIComponent(route.fullPath)}`)
}

// ── Activity (history + annotations) ─────────────────────────────────────────

const historyOpen = ref(false)
const historyLoaded = ref(false)
const historyLoading = ref(false)
const history = ref([])

// new annotation
const newAnnotation = ref('')
const annotationSaving = ref(false)

// inline edit state
const editingId = ref(null)
const editText = ref('')

async function loadHistory() {
  historyLoading.value = true
  try {
    const entries = await api.listRowHistory(props.workspaceCode, props.tableCode, props.row.id)
    history.value = entries ?? []
    historyLoaded.value = true
  } catch (e) {
    console.error('Failed to load history:', e)
  } finally {
    historyLoading.value = false
  }
}

async function toggleHistory() {
  historyOpen.value = !historyOpen.value
  if (historyOpen.value && !historyLoaded.value) {
    await loadHistory()
  }
}

async function submitAnnotation() {
  const text = newAnnotation.value.trim()
  if (!text) return
  annotationSaving.value = true
  try {
    const entry = await api.createAnnotation(props.workspaceCode, props.tableCode, props.row.id, text)
    history.value = [entry, ...history.value]
    newAnnotation.value = ''
  } catch (e) {
    notify(e.message || 'Failed to add annotation')
  } finally {
    annotationSaving.value = false
  }
}

function startEdit(entry) {
  editingId.value = entry.id
  editText.value = entry.annotation ?? ''
}

function cancelEdit() {
  editingId.value = null
  editText.value = ''
}

async function saveEdit(entry) {
  const text = editText.value.trim()
  try {
    const updated = await api.updateAnnotation(
      props.workspaceCode, props.tableCode, props.row.id, entry.id, text
    )
    const idx = history.value.findIndex(h => h.id === entry.id)
    if (idx !== -1) history.value[idx] = updated
  } catch (e) {
    notify(e.message || 'Failed to save')
  }
  cancelEdit()
}

async function deleteAnnotation(entry) {
  try {
    await api.deleteAnnotation(props.workspaceCode, props.tableCode, props.row.id, entry.id)
    history.value = history.value.filter(h => h.id !== entry.id)
  } catch (e) {
    notify(e.message || 'Failed to delete annotation')
  }
}

function formatDate(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

function relativeTime(iso) {
  const diff = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}h ago`
  return `${Math.floor(hrs / 24)}d ago`
}

function displayVal(val) {
  if (val == null || val === '') return '(empty)'
  if (typeof val === 'boolean') return val ? 'Yes' : 'No'
  if (typeof val === 'object') return JSON.stringify(val)
  return String(val)
}

function diffFor(entry, index) {
  const before = typeof entry.data === 'string' ? JSON.parse(entry.data) : (entry.data ?? {})
  const afterRaw = index === 0 ? props.row.data : history.value[index - 1].data
  const after = typeof afterRaw === 'string' ? JSON.parse(afterRaw) : (afterRaw ?? {})
  return sortedColumns.value
    .filter(col => {
      const bVal = before[col.code] ?? ''
      const aVal = after[col.code] ?? ''
      return JSON.stringify(bVal) !== JSON.stringify(aVal)
    })
    .map(col => ({
      col,
      before: before[col.code] ?? '',
      after: after[col.code] ?? '',
    }))
}

async function revert(entry) {
  if (!await confirm({ title: 'Revert to this version?', message: 'The record will be restored to this snapshot.', confirmLabel: 'Revert', danger: false })) return
  revisionId = null  // revert is a deliberate new action, start a fresh revision
  const snapshot = typeof entry.data === 'string' ? JSON.parse(entry.data) : (entry.data ?? {})
  for (const col of props.table.columns ?? []) {
    if (col.type === 'created-at' || col.type === 'updated-at') continue
    formData[col.code] = col.code in snapshot ? snapshot[col.code] : (col.type === 'multi-select' ? [] : '')
  }
  dirty = true
  await save()
  historyLoaded.value = false
  await loadHistory()
}

watch(() => props.row.id, () => {
  history.value = []
  historyLoaded.value = false
  historyOpen.value = false
  revisionId = null
  newAnnotation.value = ''
  cancelEdit()
  // Reset referencing state when row changes
  referencingRecords.value = []
  refsLoaded.value = false
  refsOpen.value = false
  referencingTotal.value = 0
})

async function close() {
  clearTimeout(syncTimer)
  if (dirty) await save()
  emit('close')
}

async function navigate(targetRowId) {
  clearTimeout(syncTimer)
  if (dirty) await save()
  emit('navigate', targetRowId)
}

async function deleteRow() {
  if (!await confirm({ title: 'Delete record?', message: 'This record will be permanently deleted. This cannot be undone.', confirmLabel: 'Delete' })) return
  try {
    await api.deleteRow(props.workspaceCode, props.tableCode, props.row.id)
    emit('row-deleted', props.row)
  } catch (e) {
    if (e.status === 409 && e.code === 'referenced') {
      showBlock({ workspaceCode: props.workspaceCode, references: e.body.references })
    } else {
      notify(e.message || 'Delete failed')
    }
  }
}

async function createRow() {
  try {
    const newRow = await api.createRow(props.workspaceCode, props.tableCode, { data: {} })
    emit('row-created', newRow)
  } catch (e) {
    notify(e.message || 'Failed to create record')
  }
}

async function duplicateRow() {
  try {
    const newRow = await api.createRow(props.workspaceCode, props.tableCode, { data: { ...props.row.data } })
    emit('row-created', newRow)
  } catch (e) {
    notify(e.message || 'Failed to duplicate record')
  }
}
</script>
