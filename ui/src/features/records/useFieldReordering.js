export function useFieldReordering(columnsRef, columnOrderRef, emit) {
  function move(col, dir) {
    const cols = columnsRef.value
    const idx = cols.findIndex(c => c.code === col.code)
    if (idx === -1) return
    const newOrder = (columnOrderRef.value ?? cols.map(c => c.code)).slice()
    if (dir === 'up' && idx > 0) {
      ;[newOrder[idx - 1], newOrder[idx]] = [newOrder[idx], newOrder[idx - 1]]
    } else if (dir === 'down' && idx < newOrder.length - 1) {
      ;[newOrder[idx], newOrder[idx + 1]] = [newOrder[idx + 1], newOrder[idx]]
    } else {
      return
    }
    emit('column-order-change', newOrder)
  }

  return { moveFieldInList: move }
}
