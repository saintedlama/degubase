export function getCellValue(row, col) {
  if (col.type === 'created-at') return row.created_at ?? null
  if (col.type === 'updated-at') return row.updated_at ?? null
  return row.data?.[col.code] ?? null
}

export function hasValue(row, col) {
  const v = getCellValue(row, col)
  return v != null && v !== '' && (!Array.isArray(v) || v.length > 0)
}

export function stripMarkdown(text) {
  if (!text) return ''
  return text
    .replace(/```[\s\S]*?```/g, '[code]')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/#{1,6}\s+/g, '')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/\*([^*]+)\*/g, '$1')
    .replace(/~~([^~]+)~~/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/^>\s*/gm, '')
    .replace(/\n+/g, ' ')
    .trim()
}

function formatDate(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

export function displayValue(row, col) {
  const v = getCellValue(row, col)
  if (v == null || v === '') return ''
  if (col.type === 'created-at' || col.type === 'updated-at') return formatDate(v)
  if (col.type === 'markdown') return stripMarkdown(String(v))
  if (col.type === 'checklist' && Array.isArray(v)) {
    const done = v.filter(i => i.checked).length
    return `${done} / ${v.length}`
  }
  if (Array.isArray(v)) return v.join(', ')
  if (typeof v === 'object' && v.filename) return v.filename
  if (typeof v === 'object' && v.label != null) return String(v.label)
  return String(v)
}

