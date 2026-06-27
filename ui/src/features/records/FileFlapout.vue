<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-199" @click="$emit('close')" />

    <div
      class="fixed z-200 flex flex-col rounded-xl overflow-hidden bg-surface-1 border border-border-1 shadow-[0_8px_32px_rgba(0,0,0,.18),0_2px_8px_rgba(0,0,0,.10)]"
      :style="panelStyle"
      @click.stop
      @keydown.escape.stop="$emit('close')"
    >
      <!-- Header -->
      <div class="flex items-center gap-2 px-3 h-9 bg-[#f5f5f8] dark:bg-[#2d2d30] border-b border-border-1 shrink-0">
        <span class="text-[10px] font-bold tracking-widest uppercase text-text-2 flex-1 truncate">{{ colName }}</span>
        <button
          class="flex items-center justify-center w-5 h-5 bg-transparent border-none text-text-2 cursor-pointer rounded hover:text-text-1 hover:bg-border-1 transition-colors"
          title="Close (Esc)"
          @click="$emit('close')"
        ><RiCloseLine size="13" /></button>
      </div>

      <!-- Image preview -->
      <div v-if="isImage && meta" class="bg-zinc-100 dark:bg-[#1e1e1e] flex items-center justify-center p-3 shrink-0">
        <img
          :src="thumbnailUrl(workspaceCode, tableCode, rowId, meta.fileId)"
          :alt="meta.filename"
          class="max-h-48 max-w-full object-contain rounded"
        />
      </div>

      <!-- File icon (non-image) -->
      <div v-else-if="meta" class="flex items-center justify-center py-6 text-5xl shrink-0">📄</div>

      <!-- Empty state -->
      <div v-else class="flex items-center justify-center py-6 text-zinc-300 dark:text-[#454545] shrink-0">
        <RiUploadLine size="32" />
      </div>

      <!-- Metadata -->
      <div v-if="meta" class="px-3 py-2 border-b border-zinc-100 dark:border-[#3c3c3c] shrink-0">
        <p class="text-[12px] font-medium text-text-1 truncate">{{ meta.filename }}</p>
        <p class="text-[11px] text-text-3 mt-0.5">{{ formatFileSize(meta.size) }} · {{ meta.mimeType }}</p>
      </div>

      <!-- Actions -->
      <div class="flex flex-col gap-1 p-2 shrink-0">
        <!-- View full size -->
        <button
          v-if="isImage && meta"
          class="flex items-center gap-2 px-3 py-1.5 rounded-md text-[12px] text-text-1 hover:bg-border-1 transition-colors bg-transparent border-none cursor-pointer text-left"
          @click.stop="lightboxOpen = true"
        >
          <RiFullscreenLine size="13" class="text-text-3 shrink-0" />
          View full size
        </button>

        <!-- Download -->
        <a
          v-if="meta"
          :href="fileUrl(workspaceCode, tableCode, rowId, meta.fileId)"
          :download="meta.filename"
          class="flex items-center gap-2 px-3 py-1.5 rounded-md text-[12px] text-text-1 hover:bg-border-1 transition-colors no-underline"
          @click.stop
        >
          <RiDownloadLine size="13" class="text-text-3 shrink-0" />
          Download
        </a>

        <!-- Replace / Upload -->
        <label
          class="flex items-center gap-2 px-3 py-1.5 rounded-md text-[12px] cursor-pointer transition-colors"
          :class="uploading
            ? 'text-text-3 cursor-wait'
            : 'text-text-1 hover:bg-border-1'"
        >
          <RiUploadLine size="13" class="text-text-3 shrink-0" />
          {{ uploading ? 'Uploading…' : (meta ? 'Replace file' : 'Upload file') }}
          <input
            type="file"
            class="hidden"
            :accept="colType === 'image' ? 'image/*' : undefined"
            :disabled="uploading"
            @change="onFileSelected"
          />
        </label>

        <!-- Remove -->
        <button
          v-if="meta"
          class="flex items-center gap-2 px-3 py-1.5 rounded-md text-[12px] text-red-500 hover:bg-red-50 dark:hover:bg-red-950/30 bg-transparent border-none cursor-pointer transition-colors text-left"
          :disabled="uploading"
          @click="removeFile"
        >
          <RiDeleteBinLine size="13" class="shrink-0" />
          Remove file
        </button>
      </div>
    </div>
  </Teleport>

  <ImageLightbox
    v-if="lightboxOpen && isImage && meta"
    :src="fileUrl(workspaceCode, tableCode, rowId, meta.fileId)"
    :alt="meta?.filename"
    @close="lightboxOpen = false"
  />
</template>

<script setup>
import { ref, computed } from 'vue'
import { api, fileUrl, thumbnailUrl } from '../../api/client.js'
import { RiCloseLine, RiDownloadLine, RiUploadLine, RiDeleteBinLine, RiFullscreenLine } from '@remixicon/vue'
import ImageLightbox from './ImageLightbox.vue'
import { useNotifications } from '../../foundation/useNotifications.js'

const lightboxOpen = ref(false)
const { notify } = useNotifications()

const props = defineProps({
  meta:           { type: Object, default: null },   // { fileId, filename, size, mimeType }
  colType:        { type: String, required: true },
  colName:        { type: String, default: '' },
  workspaceCode:  { type: String, required: true },
  tableCode:      { type: String, required: true },
  rowId:          { type: Number, required: true },
  colId:          { type: [Number, String], required: true },
  colCode:        { type: String, required: true },
  anchorRect:     { type: Object, default: null },
})
const emit = defineEmits(['close', 'rowUpdated'])

const uploading = ref(false)
const isImage = computed(() => props.colType === 'image')

const panelStyle = computed(() => {
  const W = 280
  const m = 6
  if (!props.anchorRect) {
    return { width: W + 'px', left: '50%', top: '50%', transform: 'translate(-50%,-50%)' }
  }
  const { left, top, bottom } = props.anchorRect
  const vw = window.innerWidth
  const vh = window.innerHeight
  const x = Math.min(Math.max(left, m), vw - W - m)
  const approxH = props.meta ? (isImage.value ? 380 : 260) : 160
  const y = bottom + approxH + m <= vh ? bottom + m : Math.max(m, top - approxH - m)
  return { width: W + 'px', left: x + 'px', top: y + 'px' }
})

function formatFileSize(bytes) {
  if (bytes == null) return ''
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

async function onFileSelected(event) {
  const file = event.target.files?.[0]
  if (!file) return
  uploading.value = true
  try {
    const updatedRow = await api.uploadFile(
      props.workspaceCode, props.tableCode, props.rowId, props.colCode, file
    )
    emit('rowUpdated', updatedRow)
    emit('close')
  } catch (e) {
    notify(e.message || 'File upload failed')
  } finally {
    uploading.value = false
  }
}

async function removeFile() {
  uploading.value = true
  try {
    const updatedRow = await api.patchRow(
      props.workspaceCode, props.tableCode, props.rowId,
      { data: { [props.colCode]: null } }
    )
    emit('rowUpdated', updatedRow)
    emit('close')
  } catch (e) {
    notify(e.message || 'File removal failed')
  } finally {
    uploading.value = false
  }
}
</script>
