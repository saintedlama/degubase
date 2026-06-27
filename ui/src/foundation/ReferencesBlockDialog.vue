<template>
  <Teleport to="body">
    <Backdrop v-if="state" @dismiss="dismiss">
      <div
        class="bg-surface-1 border border-border-1 rounded-xl shadow-xl p-6 flex flex-col gap-4"
        style="width: min(480px, calc(100vw - 32px))"
      >
        <h3 class="text-[15px] font-bold text-text-1">Cannot delete record</h3>
        <p class="text-[13px] text-text-2 leading-relaxed">
          This record is referenced by {{ state.references.length === 1 ? '1 other record' : `${state.references.length} other records` }} and cannot be deleted. Remove the links first:
        </p>
        <ul class="flex flex-col gap-1">
          <li v-for="ref in state.references" :key="`${ref.tableCode}-${ref.rowId}`">
            <RouterLink
              :to="`/workspaces/${state.workspaceCode}/tables/${ref.tableCode}/rows/${ref.rowId}`"
              class="text-[13px] text-brand-600 dark:text-brand-400 hover:underline"
              @click="dismiss"
            >
              {{ ref.tableName }} #{{ ref.rowId }}
            </RouterLink>
          </li>
        </ul>
        <div class="flex justify-end">
          <button
            class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-surface-2 text-text-2 border border-zinc-300 dark:border-[#3c3c3c] cursor-pointer hover:bg-border-1 transition-colors"
            @click="dismiss"
          >Close</button>
        </div>
      </div>
    </Backdrop>
  </Teleport>
</template>

<script setup>
import Backdrop from './Backdrop.vue'
import { useReferencesBlock } from './useReferencesBlock.js'

const { state, dismiss } = useReferencesBlock()
</script>
