import { ref, computed, watch } from 'vue'
import { api } from '../../api/client.js'

const PAGE_SIZE = 200

export function useRowPagination(workspaceCodeFn, tableCodeFn, filtersFn, sortsFn, searchFn, enabledFn = () => true) {
  const rows = ref([])
  const total = ref(0)
  const currentPage = ref(1)
  const loadingMore = ref(false)
  const hasMore = computed(() => rows.value.length < total.value)

  function buildParams(page) {
    const params = { page, pageSize: PAGE_SIZE }
    const filters = filtersFn()
    const sorts = sortsFn()
    const q = searchFn ? searchFn() : ''
    if (filters.length) params.filters = JSON.stringify(filters)
    if (sorts.length) params.sort = JSON.stringify(sorts)
    if (q) params.q = q
    return params
  }

  async function loadRows() {
    if (!enabledFn()) return
    currentPage.value = 1
    const result = await api.listRows(workspaceCodeFn(), tableCodeFn(), buildParams(1))
    rows.value = result.data
    total.value = result.total
  }

  async function loadMore() {
    if (!hasMore.value || loadingMore.value) return
    loadingMore.value = true
    try {
      const nextPage = currentPage.value + 1
      const result = await api.listRows(workspaceCodeFn(), tableCodeFn(), buildParams(nextPage))
      rows.value = [...rows.value, ...result.data]
      total.value = result.total
      currentPage.value = nextPage
    } finally {
      loadingMore.value = false
    }
  }

  watch(tableCodeFn, () => loadRows(), { immediate: true })
  watch(filtersFn, () => loadRows(), { deep: true })
  watch(sortsFn, () => loadRows(), { deep: true })
  if (searchFn) watch(searchFn, () => loadRows())
  watch(enabledFn, (enabled) => {
    if (enabled) loadRows()
    else { rows.value = []; total.value = 0 }
  })

  const rowEvents = {
    onCreated(row) {
      if (!rows.value.find(r => r.id === row.id)) {
        rows.value.push(row)
        total.value++
      }
    },
    onUpdated(row) {
      const i = rows.value.findIndex(r => r.id === row.id)
      if (i >= 0) rows.value[i] = row
    },
    onDeleted(id) {
      const existed = rows.value.some(r => r.id === id)
      rows.value = rows.value.filter(r => r.id !== id)
      if (existed) total.value--
    },
  }

  return { rows, total, hasMore, loadingMore, loadRows, loadMore, rowEvents }
}
