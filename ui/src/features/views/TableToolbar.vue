<template>
  <div class="flex flex-wrap items-center gap-2 px-5 min-h-10 py-1 bg-surface-1 border-b border-border-1 shrink-0">
    <slot name="before" />

    <div class="ml-auto flex flex-wrap items-center gap-1.5">
      <!-- Search -->
      <div class="relative flex items-center">
        <RiSearchLine size="12" class="absolute left-2 text-text-3 pointer-events-none" />
        <input
          v-model="searchRaw"
          type="search"
          placeholder="Search…"
          class="h-6 pl-6 pr-2 text-[11px] rounded border border-border-1 bg-surface-2 text-text-1 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] outline-none focus:border-brand-400 dark:focus:border-brand-600 w-32 transition-all focus:w-48"
        />
      </div>

      <SortPanel
        :columns="table.columns ?? []"
        :sorts="viewConfig?.sort ?? []"
        @update:sorts="$emit('sort-change', $event)"
      />
      <FilterPanel
        :columns="table.columns ?? []"
        :filters="viewConfig?.filters ?? []"
        @update:filters="$emit('filter-change', $event)"
      />

      <slot name="fields" />

      <button
        class="inline-flex items-center gap-1 h-6 px-2 text-[11px] font-medium text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-700/50 rounded cursor-pointer hover:bg-brand-100 dark:hover:bg-brand-900/50 transition-colors"
        @click="$emit('add-record')"
      >
        <RiAddLine size="12" />
        Add record
      </button>

      <!-- Secondary actions -->
      <Flapout align="right">
        <template #trigger="{ toggle, setAnchor }">
          <button
            :ref="setAnchor"
            class="flex items-center justify-center w-6 h-6 rounded border border-border-1 bg-transparent text-text-2 cursor-pointer hover:text-text-1 hover:bg-border-1 transition-colors"
            title="More actions"
            @click.stop="toggle"
          >
            <RiMoreLine size="13" />
          </button>
        </template>
        <template #default="{ close }">
          <div class="bg-surface-1 border border-border-1 rounded-lg shadow-xl py-1 w-44">
            <button
              class="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-1 hover:bg-border-1 transition-colors cursor-pointer border-none bg-transparent"
              @click="$emit('bulk-action'); close()"
            >
              <RiEditBoxLine size="13" class="shrink-0 text-text-2" />
              Bulk edit
            </button>
            <button
              class="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-1 hover:bg-border-1 transition-colors cursor-pointer border-none bg-transparent"
              @click="$emit('export-csv'); close()"
            >
              <RiDownloadLine size="13" class="shrink-0 text-text-2" />
              Export CSV
            </button>
            <button
              class="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-1 hover:bg-border-1 transition-colors cursor-pointer border-none bg-transparent"
              @click="$emit('import-csv'); close()"
            >
              <RiUploadLine size="13" class="shrink-0 text-text-2" />
              Import CSV
            </button>
          </div>
        </template>
      </Flapout>
    </div>

    <slot name="after" />
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import SortPanel from './SortPanel.vue'
import FilterPanel from './FilterPanel.vue'
import Flapout from '../../foundation/Flapout.vue'
import { RiSearchLine, RiEditBoxLine, RiDownloadLine, RiUploadLine, RiAddLine, RiMoreLine } from '@remixicon/vue'

const props = defineProps({
  table: { type: Object, required: true },
  viewConfig: { type: Object, default: () => ({}) },
})
const emit = defineEmits(['update:search', 'sort-change', 'filter-change', 'add-record', 'bulk-action', 'export-csv', 'import-csv'])

const searchRaw = ref('')
let searchTimer = null
watch(searchRaw, (val) => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => emit('update:search', val), 300)
})
</script>
