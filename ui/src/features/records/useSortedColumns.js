import { computed } from 'vue'

export function useSortedColumns(columns, columnOrder) {
  return computed(() => {
    const cols = [...(columns.value ?? [])]
    const order = columnOrder?.value
    if (order && order.length > 0) {
      const orderMap = new Map(order.map((code, i) => [code, i]))
      return cols.sort((a, b) => {
        const ai = orderMap.get(a.code)
        const bi = orderMap.get(b.code)
        if (ai != null && bi != null) return ai - bi
        if (ai != null) return -1
        if (bi != null) return 1
        return a.position - b.position
      })
    }
    return cols.sort((a, b) => a.position - b.position)
  })
}
