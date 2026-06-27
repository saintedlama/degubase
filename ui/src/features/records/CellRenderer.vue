<template>
  <!-- Checkbox (direct interaction, no edit mode) -->
  <label v-if="col.type === 'checkbox'" class="flex items-center justify-center h-8.5 cursor-pointer" @click.stop>
    <input
      type="checkbox"
      :checked="!!value"
      class="w-3.5 h-3.5 accent-brand-600 cursor-pointer"
      @change="emit('toggle', $event.target.checked)"
    />
  </label>

  <!-- System timestamps -->
  <span v-else-if="col.type === 'created-at' || col.type === 'updated-at'" class="block px-2.5 leading-8.5 truncate text-text-3 text-[13px]">{{ formatDate(value) }}</span>

  <!-- File with content -->
  <div v-else-if="col.type === 'file' && fileMeta" class="flex items-center gap-2 px-2.5 h-8.5 overflow-hidden">
    <span class="text-xs shrink-0">📄</span>
    <a
      :href="fileUrl(workspaceCode, tableCode, row.id, fileMeta.fileId)"
      :download="fileMeta.filename"
      class="text-[13px] text-brand-600 dark:text-brand-400 truncate hover:underline"
      @click.stop
    >{{ fileMeta.filename }}</a>
  </div>

  <!-- Image with content -->
  <div v-else-if="col.type === 'image' && fileMeta" class="flex items-center gap-2 px-2.5 h-8.5">
    <img
      v-if="isImageMime(fileMeta.mimeType)"
      :src="thumbnailUrl(workspaceCode, tableCode, row.id, fileMeta.fileId)"
      class="h-7 w-7 object-cover rounded"
      :alt="fileMeta.filename"
    />
    <a
      :href="fileUrl(workspaceCode, tableCode, row.id, fileMeta.fileId)"
      :download="fileMeta.filename"
      class="text-[13px] text-brand-600 dark:text-brand-400 truncate hover:underline"
      @click.stop
    >{{ fileMeta.filename }}</a>
  </div>

  <!-- File/Image upload placeholder (no content yet) -->
  <label
    v-else-if="col.type === 'file' || col.type === 'image'"
    class="flex items-center justify-center h-8.5 cursor-pointer text-text-3 hover:text-brand-500 transition-colors"
    @click.stop
  >
    <RiUploadLine size="14" />
    <input
      type="file"
      class="hidden"
      :accept="col.type === 'image' ? 'image/*' : undefined"
      @change="emit('upload', $event)"
    />
  </label>

  <!-- Single-select pill -->
  <div v-else-if="col.type === 'single-select' && value" class="px-2 flex items-center h-8.5">
    <PillBadge :col="col" :value="value" class="max-w-30" />
  </div>

  <!-- Multi-select pills -->
  <div v-else-if="col.type === 'multi-select' && value?.length" class="px-2 flex items-center gap-1 h-8.5 overflow-hidden">
    <PillBadge
      v-for="v in [value].flat()"
      :key="v"
      :col="col"
      :value="v"
      class="max-w-24 shrink-0"
    />
  </div>

  <!-- Rating stars (direct interaction, no edit mode) -->
  <div v-else-if="col.type === 'rating'" class="flex items-center gap-px px-2.5 h-8.5" @click.stop>
    <button
      v-for="star in 5"
      :key="star"
      class="text-base leading-none bg-transparent border-none p-px cursor-pointer transition-colors"
      :class="star <= (value || 0) ? 'text-amber-400' : 'text-zinc-200 dark:text-[#454545] hover:text-amber-200'"
      @click="emit('rate', star === (value || 0) ? 0 : star)"
    >★</button>
  </div>

  <!-- Symbol / Emoji -->
  <div v-else-if="(col.type === 'symbol' || col.type === 'emoji') && value" class="flex items-center px-2.5 h-8.5">
    <span class="text-base leading-none" :title="getEmoji(value)?.label">{{ resolveEmoji(value) }}</span>
  </div>

  <!-- Checklist progress -->
  <div v-else-if="col.type === 'checklist' && value?.length" class="flex items-center gap-1.5 px-2.5 h-8.5">
    <span class="text-[12px] text-text-2">
      {{ value.filter(i => i.checked).length }} / {{ value.length }}
    </span>
    <div class="flex-1 h-1 bg-border-1 rounded-full overflow-hidden">
      <div
        class="h-full bg-brand-500 rounded-full transition-all"
        :style="{ width: (value.filter(i => i.checked).length / value.length * 100) + '%' }"
      />
    </div>
  </div>

  <!-- Row link chip -->
  <div v-else-if="col.type === 'row-link' && value" class="flex items-center gap-1 px-2.5 h-8.5 overflow-hidden">
    <RiNodeTree size="11" class="text-text-3 shrink-0" />
    <span class="text-[13px] text-text-1 truncate flex-1">{{ value.label }}</span>
    <span
      v-if="refCount > 0"
      ref="badgeEl"
      class="inline-flex items-center gap-0.5 px-1 py-0.5 rounded text-[10px] leading-none bg-surface-2 border border-border-1 text-text-2 cursor-pointer hover:bg-border-1 transition-colors shrink-0 ml-0.5"
      @click.stop="handleRefClick"
    >↗ {{ refCount }}</span>
    <button
      class="shrink-0 flex items-center justify-center w-4 h-4 rounded text-text-3 hover:text-brand-600 bg-transparent border-none cursor-pointer transition-colors"
      title="Open linked record"
      @click.stop="navigateToLinked(value.id)"
    >
      <RiArrowRightUpLine size="11" />
    </button>
  </div>

  <!-- Mermaid diagram indicator -->
  <div v-else-if="col.type === 'mermaid' && value" class="flex items-center gap-1.5 px-2.5 h-8.5">
    <RiFlowChart size="13" class="text-text-3 shrink-0" />
    <span class="text-[13px] text-text-2 truncate">Diagram</span>
  </div>

  <!-- Default text display -->
  <span v-else class="block px-2.5 leading-8.5 truncate text-text-1 text-[13px]">{{ displayText }}</span>

  <!-- Referencing records flapout -->
  <CellFlapout
    v-if="showRefsFlapout"
    col-type="referencing"
    :col-name="'Referencing Records'"
    :anchor-rect="badgeRect"
    :workspace-code="workspaceCode"
    :referencing-records="refsList"
    @close="showRefsFlapout = false"
  />
</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, fileUrl, thumbnailUrl } from '../../api/client.js'
import { getCellValue, displayValue } from './cellHelpers.js'
import { RiUploadLine, RiNodeTree, RiArrowRightUpLine, RiFlowChart } from '@remixicon/vue'
import { getEmoji, resolveEmoji } from '../views/emojis.js'
import PillBadge from '../../foundation/PillBadge.vue'
import CellFlapout from './CellFlapout.vue'

const props = defineProps({
  col: { type: Object, required: true },
  row: { type: Object, required: true },
  workspaceCode: { type: String, required: true },
  tableCode: { type: String, required: true },
})

const emit = defineEmits(['toggle', 'rate', 'upload'])

const route = useRoute()
const router = useRouter()

const value = computed(() => getCellValue(props.row, props.col))

const rowLinkTargetCode = computed(() => {
  const opts = typeof props.col.options === 'string' ? JSON.parse(props.col.options) : props.col.options
  return opts?.targetTableCode ?? props.tableCode
})

function navigateToLinked(rowId) {
  router.push(`/workspaces/${props.workspaceCode}/tables/${rowLinkTargetCode.value}/rows/${rowId}?returnTo=${encodeURIComponent(route.fullPath)}`)
}

const fileMeta = computed(() => {
  const v = value.value
  if (!v || typeof v !== 'object') return null
  return v
})

function isImageMime(mime) { return mime && mime.startsWith('image/') }

function formatDate(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

const displayText = computed(() => displayValue(props.row, props.col))

// ── Referencing records badge ────────────────────────────────────────────────

const refCount = ref(0)
const refsList = ref([])
const showRefsFlapout = ref(false)
const badgeEl = ref(null)
const badgeRect = ref(null)

onMounted(async () => {
  if (props.col.type !== 'row-link' || !value.value?.id) return
  try {
    const res = await api.listReferencingRows(
      props.workspaceCode,
      rowLinkTargetCode.value,
      value.value.id,
      { page_size: 1 }
    )
    refCount.value = res.total ?? 0
  } catch (e) {
    console.error('Failed to fetch referencing count:', e)
  }
})

async function handleRefClick() {
  if (refCount.value === 1) {
    // Fetch the single record and navigate directly
    try {
      const res = await api.listReferencingRows(
        props.workspaceCode,
        rowLinkTargetCode.value,
        value.value.id,
        { page_size: 1 }
      )
      if (res.data?.length === 1) {
        const ref = res.data[0]
        router.push(`/workspaces/${props.workspaceCode}/tables/${ref.sourceTableCode}/rows/${ref.sourceRowId}?returnTo=${encodeURIComponent(route.fullPath)}`)
      }
    } catch (e) {
      console.error('Failed to navigate to referencing record:', e)
    }
    return
  }
  // Fetch full list and show flapout
  try {
    const res = await api.listReferencingRows(
      props.workspaceCode,
      rowLinkTargetCode.value,
      value.value.id,
      { page_size: 100 }
    )
    refsList.value = res.data ?? []
    if (badgeEl.value) {
      badgeRect.value = badgeEl.value.getBoundingClientRect()
    }
    showRefsFlapout.value = true
  } catch (e) {
    console.error('Failed to fetch referencing records:', e)
  }
}
</script>
