<template>
  <div class="flex items-center gap-2 px-5 h-13 bg-surface-1 border-b border-border-1 shrink-0">
    <button
      class="inline-flex items-center gap-1 text-xs font-medium text-text-2 hover:text-text-1 transition-colors bg-transparent border-none cursor-pointer px-2 py-1 rounded hover:bg-border-1 shrink-0"
      @click="$emit('close')"
    >
      <RiArrowLeftSLine size="16" />
      Back
    </button>
    <div class="w-px h-4 bg-border-1 shrink-0" />
    <span class="text-xs text-text-2 shrink-0">Record #{{ row.id }}</span>
    <span v-if="rowTitle" class="text-sm font-medium text-text-1 truncate">{{ rowTitle }}</span>

    <!-- Save status -->
    <span class="ml-auto shrink-0 text-xs transition-opacity duration-300"
      :class="saveStatus === 'Saved' ? 'text-emerald-500' : saveStatus ? 'text-text-2' : 'opacity-0 text-zinc-400'"
    >{{ saveStatus || 'Saved' }}</span>

    <!-- Copy link -->
    <button
      class="shrink-0 inline-flex items-center gap-1 text-xs font-medium text-text-2 hover:text-text-1 bg-transparent border-none cursor-pointer px-2 py-1 rounded hover:bg-border-1 transition-colors"
      :title="linkCopied ? 'Copied!' : 'Copy link to this record'"
      @click="copyLink"
    >
      <RiLink v-if="!linkCopied" size="14" />
      <RiCheckLine v-else size="14" class="text-emerald-500" />
      <span>{{ linkCopied ? 'Copied' : 'Copy link' }}</span>
    </button>

    <div class="w-px h-4 bg-border-1 shrink-0" />

    <!-- New record -->
    <IconButton
      :border="false"
      title="New record"
      class="shrink-0 w-7 h-7 rounded hover:bg-border-1"
      @click="$emit('create-row')"
    >
      <RiAddLine size="15" />
    </IconButton>

    <!-- Duplicate record -->
    <IconButton
      :border="false"
      title="Duplicate record"
      class="shrink-0 w-7 h-7 rounded hover:bg-border-1"
      @click="$emit('duplicate-row')"
    >
      <RiFileCopyLine size="14" />
    </IconButton>

    <!-- Delete record -->
    <IconButton
      :border="false"
      danger
      title="Delete record"
      class="shrink-0 w-7 h-7 rounded hover:bg-red-50 dark:hover:bg-red-950/40"
      @click="$emit('delete-row')"
    >
      <RiDeleteBin6Line size="14" />
    </IconButton>

    <!-- Prev / Next navigation -->
    <div v-if="prevRowId != null || nextRowId != null" class="flex items-center gap-0.5 shrink-0 border border-border-1 rounded-md overflow-hidden">
      <button
        class="flex items-center justify-center w-7 h-7 bg-transparent border-none transition-colors"
        :class="prevRowId != null ? 'text-text-2 hover:text-text-1 hover:bg-border-1 cursor-pointer' : 'text-zinc-200 dark:text-[#4a4a4a] cursor-default'"
        :disabled="prevRowId == null"
        title="Previous record"
        @click="prevRowId != null && $emit('navigate', prevRowId)"
      >
        <RiArrowUpSLine size="16" />
      </button>
      <div class="w-px h-4 bg-border-1" />
      <button
        class="flex items-center justify-center w-7 h-7 bg-transparent border-none transition-colors"
        :class="nextRowId != null ? 'text-text-2 hover:text-text-1 hover:bg-border-1 cursor-pointer' : 'text-zinc-200 dark:text-[#4a4a4a] cursor-default'"
        :disabled="nextRowId == null"
        title="Next record"
        @click="nextRowId != null && $emit('navigate', nextRowId)"
      >
        <RiArrowDownSLine size="16" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import IconButton from '../../foundation/IconButton.vue'
import {
  RiArrowLeftSLine, RiArrowUpSLine, RiArrowDownSLine, RiLink, RiCheckLine,
  RiAddLine, RiFileCopyLine, RiDeleteBin6Line,
} from '@remixicon/vue'

const props = defineProps({
  row: { type: Object, required: true },
  rowTitle: { type: String, default: '' },
  saveStatus: { type: String, default: '' },
  workspaceCode: { type: String, required: true },
  tableCode: { type: String, required: true },
  prevRowId: { type: Number, default: null },
  nextRowId: { type: Number, default: null },
})
defineEmits(['close', 'create-row', 'duplicate-row', 'delete-row', 'navigate'])

const linkCopied = ref(false)
function copyLink() {
  const url = `${location.origin}/workspaces/${props.workspaceCode}/tables/${props.tableCode}/rows/${props.row.id}`
  navigator.clipboard.writeText(url).then(() => {
    linkCopied.value = true
    setTimeout(() => { linkCopied.value = false }, 2000)
  })
}
</script>
