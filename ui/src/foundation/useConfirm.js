import { ref } from 'vue'

// Singleton — one dialog at a time.
const state = ref(null) // { title, message, confirmLabel, danger, resolve }

export function useConfirm() {
  function confirm({ title, message, confirmLabel = 'Confirm', danger = true } = {}) {
    return new Promise((resolve) => {
      state.value = { title, message, confirmLabel, danger, resolve }
    })
  }

  function accept() {
    state.value?.resolve(true)
    state.value = null
  }

  function dismiss() {
    state.value?.resolve(false)
    state.value = null
  }

  return { state, confirm, accept, dismiss }
}
