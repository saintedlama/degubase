export function getColOps(col) {
  if (!col) return []
  const t = col.type
  if (['text', 'long-text', 'markdown', 'email', 'url'].includes(t)) return [
    { value: 'contains', label: 'contains' },
    { value: 'not_contains', label: "doesn't contain" },
    { value: 'is_empty', label: 'is empty' },
    { value: 'is_not_empty', label: 'is not empty' },
  ]
  if (['number', 'currency', 'percent', 'rating'].includes(t)) return [
    { value: 'eq', label: 'equals' },
    { value: 'gt', label: 'greater than' },
    { value: 'lt', label: 'less than' },
    { value: 'is_empty', label: 'is empty' },
    { value: 'is_not_empty', label: 'is not empty' },
  ]
  if (['date', 'datetime', 'created-at', 'updated-at'].includes(t)) return [
    { value: 'before', label: 'is before' },
    { value: 'after', label: 'is after' },
    { value: 'on', label: 'is on' },
    { value: 'is_empty', label: 'is empty' },
    { value: 'is_not_empty', label: 'is not empty' },
  ]
  if (t === 'checkbox') return [
    { value: 'is_true', label: 'is checked' },
    { value: 'is_false', label: 'is not checked' },
  ]
  if (['single-select', 'multi-select'].includes(t)) return [
    { value: 'is', label: 'is' },
    { value: 'is_not', label: 'is not' },
    { value: 'in', label: 'is any of' },
    { value: 'not_in', label: 'is none of' },
    { value: 'is_empty', label: 'is empty' },
    { value: 'is_not_empty', label: 'is not empty' },
  ]
  return [
    { value: 'contains', label: 'contains' },
    { value: 'is_empty', label: 'is empty' },
    { value: 'is_not_empty', label: 'is not empty' },
  ]
}

export const NO_VALUE_OPS = ['is_empty', 'is_not_empty', 'is_true', 'is_false']
export function opNeedsValue(op) { return !NO_VALUE_OPS.includes(op) }
export function isMultiValueOp(op) { return op === 'in' || op === 'not_in' }

export function isSelectType(col) {
  return col && ['single-select', 'multi-select'].includes(col.type)
}

export function isDateType(col) {
  return col && ['date', 'datetime', 'created-at', 'updated-at'].includes(col.type)
}

export function filterInputType(col) {
  if (!col) return 'text'
  if (['number', 'currency', 'percent', 'rating'].includes(col.type)) return 'number'
  if (col.type === 'datetime') return 'datetime-local'
  if (['date', 'created-at', 'updated-at'].includes(col.type)) return 'date'
  return 'text'
}

export function getColumnChoices(col) {
  if (!col) return []
  const opts = typeof col.options === 'string' ? JSON.parse(col.options) : col.options
  return opts?.choices ?? []
}
