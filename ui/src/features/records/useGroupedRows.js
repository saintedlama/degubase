import { ref, watch } from 'vue'
import { api } from '../../api/client.js'
import { useTableEvents } from './useTableEvents.js'

const PAGE_SIZE = 200

export function useGroupedRows(workspaceCodeFn, tableCodeFn, groupByColFn, filtersFn, sortsFn, searchFn) {
  const groups = ref([]) // [{ value, total, rows, loadingMore }]

  function buildParams() {
    const params = { groupBy: groupByColFn() }
    const filters = filtersFn()
    const sorts = sortsFn()
    const q = searchFn?.() ?? ''
    if (filters.length) params.filters = JSON.stringify(filters)
    if (sorts.length) params.sort = JSON.stringify(sorts)
    if (q) params.q = q
    return params
  }

  async function loadGroups() {
    const ws = workspaceCodeFn()
    const table = tableCodeFn()
    const groupBy = groupByColFn()
    if (!ws || !table || !groupBy) { groups.value = []; return }
    const result = await api.listGroupedRows(ws, table, buildParams())
    groups.value = result.map(g => ({ ...g, loadingMore: false }))
  }

  function groupHasMore(group) {
    return group.rows.length < group.total
  }

  async function loadMoreGroup(value) {
    const group = groups.value.find(g => g.value === value)
    if (!group || group.loadingMore || !groupHasMore(group)) return
    group.loadingMore = true
    try {
      const groupBy = groupByColFn()
      const extraFilter = value != null
        ? { col: groupBy, op: 'is', value }
        : { col: groupBy, op: 'is_empty', value: '' }
      const result = await api.listRows(workspaceCodeFn(), tableCodeFn(), {
        filters: JSON.stringify([...filtersFn(), extraFilter]),
        sort: sortsFn().length ? JSON.stringify(sortsFn()) : undefined,
        q: searchFn?.() || undefined,
        page: Math.ceil(group.rows.length / PAGE_SIZE) + 1,
        pageSize: PAGE_SIZE,
      })
      group.rows = [...group.rows, ...result.data]
      group.total = result.total
    } finally {
      group.loadingMore = false
    }
  }

  useTableEvents(workspaceCodeFn, tableCodeFn, {
    onCreated(row) {
      const groupBy = groupByColFn()
      if (!groupBy) return
      const val = row.data?.[groupBy] ?? null
      const group = groups.value.find(g => g.value === val)
      if (group) {
        if (!group.rows.find(r => r.id === row.id)) { group.rows.push(row); group.total++ }
      } else {
        groups.value.push({ value: val, total: 1, rows: [row], loadingMore: false })
      }
    },
    onUpdated(row) {
      const groupBy = groupByColFn()
      if (!groupBy) return
      const newVal = row.data?.[groupBy] ?? null
      for (const group of groups.value) {
        const i = group.rows.findIndex(r => r.id === row.id)
        if (i < 0) continue
        if (group.value === newVal) {
          group.rows[i] = row
        } else {
          group.rows.splice(i, 1); group.total--
          const dest = groups.value.find(g => g.value === newVal)
          if (dest) { dest.rows.push(row); dest.total++ }
          else groups.value.push({ value: newVal, total: 1, rows: [row], loadingMore: false })
        }
        return
      }
    },
    onDeleted(id) {
      if (!groupByColFn()) return
      for (const group of groups.value) {
        const i = group.rows.findIndex(r => r.id === id)
        if (i >= 0) { group.rows.splice(i, 1); group.total--; return }
      }
    },
  })

  watch(tableCodeFn, loadGroups, { immediate: true })
  watch(groupByColFn, loadGroups)
  watch(filtersFn, loadGroups, { deep: true })
  watch(sortsFn, loadGroups, { deep: true })
  if (searchFn) watch(searchFn, loadGroups)

  return { groups, groupHasMore, loadGroups, loadMoreGroup }
}
