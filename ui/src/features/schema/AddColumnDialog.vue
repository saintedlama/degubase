<template>
  <Backdrop @dismiss="$emit('cancel')">
    <div data-testid="add-column-dialog" class="bg-surface-1 border border-border-1 rounded-xl w-80 max-w-[calc(100vw-32px)] shadow-xl flex flex-col overflow-hidden">

      <!-- Header -->
      <div class="flex items-center justify-between px-4 pt-4 pb-0">
        <h3 class="text-[14px] font-bold text-text-1">Add Column</h3>
        <button
          class="flex items-center justify-center w-6.5 h-6.5 bg-transparent border-none text-text-2 cursor-pointer rounded-md hover:text-text-1 hover:bg-border-1 transition-colors"
          @click="$emit('cancel')"
        >
          <RiCloseLine size="15" />
        </button>
      </div>

      <!-- Form -->
      <ColumnEditForm
        :column="null"
        :workspace-code="workspaceCode"
        :table-code="tableCode"
        @saved="onSaved"
        @cancel="$emit('cancel')"
      />
    </div>
  </Backdrop>
</template>

<script setup>
import ColumnEditForm from './ColumnEditForm.vue'
import { RiCloseLine } from '@remixicon/vue'
import Backdrop from '../../foundation/Backdrop.vue'

defineProps({
  workspaceCode: { type: String, required: true },
  tableCode:     { type: String, required: true },
})
const emit = defineEmits(['confirm', 'cancel'])

function onSaved(newCol) {
  emit('confirm', newCol)
}
</script>
