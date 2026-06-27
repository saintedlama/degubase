import { ref, computed, watch } from 'vue'

export function useCardFields(viewConfig, availableColumns, emit) {
  const localCardFields = ref(viewConfig.value?.cardFields ?? null)

  watch(() => viewConfig.value?.cardFields, (fields) => {
    localCardFields.value = fields ?? null
  })

  function defaultCardFields() {
    return availableColumns.value
      .filter(c => c.type !== 'created-at' && c.type !== 'updated-at')
      .slice(0, 4)
      .map(c => ({ col: c.code, showLabel: true, truncate: true, style: 'default' }))
  }

  function normalize(fields) {
    return (fields ?? defaultCardFields()).map(f =>
      typeof f === 'string' ? { col: f, showLabel: true, truncate: true, style: 'default' } : f
    )
  }

  function getFieldConfig(colCode) {
    return normalize(localCardFields.value).find(f => f.col === colCode)
      ?? { col: colCode, showLabel: true, truncate: true, style: 'default' }
  }

  function isCardField(colCode) {
    const fields = localCardFields.value ?? defaultCardFields()
    return fields.some(f => (typeof f === 'string' ? f : f.col) === colCode)
  }

  function ensureCardField(colCode) {
    let fields = normalize(localCardFields.value)
    if (!fields.some(f => f.col === colCode)) {
      fields = [...fields, { col: colCode, showLabel: true, truncate: true, style: 'default' }]
      localCardFields.value = fields
      emit('card-fields-change', fields)
    }
  }

  function toggleCardField(colCode) {
    let fields = normalize(localCardFields.value)
    const exists = fields.find(f => f.col === colCode)
    if (exists) {
      fields = fields.filter(f => f.col !== colCode)
    } else {
      fields = [...fields, { col: colCode, showLabel: true, truncate: true, style: 'default' }]
    }
    localCardFields.value = fields
    emit('card-fields-change', fields)
  }

  function updateFieldConfig(colCode, updates) {
    let fields = normalize(localCardFields.value)
    const idx = fields.findIndex(f => f.col === colCode)
    if (idx !== -1) {
      fields[idx] = { ...fields[idx], ...updates }
      localCardFields.value = fields
      emit('card-fields-change', fields)
    }
  }

  function isTruncatable(col) {
    return col.type === 'long-text' || col.type === 'markdown'
  }

  const visibleCardFields = computed(() => {
    const fields = normalize(localCardFields.value)
    const fieldMap = new Map(fields.map(f => [f.col, f]))
    // Iterate availableColumns so columnOrder drives display order,
    // using localCardFields only as a lookup for visibility and per-field config.
    return (availableColumns.value ?? [])
      .map(col => {
        const f = fieldMap.get(col.code)
        if (!f) return null
        return { col, showLabel: f.showLabel ?? true, truncate: f.truncate ?? true, style: f.style ?? 'default' }
      })
      .filter(Boolean)
  })

  return {
    getFieldConfig,
    isCardField,
    ensureCardField,
    toggleCardField,
    updateFieldConfig,
    isTruncatable,
    visibleCardFields,
  }
}
