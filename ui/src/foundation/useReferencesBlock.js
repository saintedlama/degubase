import { ref } from 'vue'

// Singleton — one dialog at a time.
const state = ref(null) // { workspaceCode, references: [{tableCode, tableName, rowId}] }

export function useReferencesBlock() {
  function showBlock({ workspaceCode, references }) {
    state.value = { workspaceCode, references }
  }

  function dismiss() {
    state.value = null
  }

  return { state, showBlock, dismiss }
}
