export const PALETTE_GROUPS = [
  {
    label: 'Qualitative', offset: 0,
    names: ['Set 1', 'Set 2', 'Pastel 2', 'Dark 2'],
    palettes: [
      ['#e41a1c','#377eb8','#4daf4a','#984ea3','#ff7f00','#ffff33','#a65628','#f781bf'],
      ['#66c2a5','#fc8d62','#8da0cb','#e78ac3','#a6d854','#ffd92f','#e5c494','#b3b3b3'],
      ['#b3e2cd','#fdcdac','#cbd5e8','#f4cae4','#e6f5c9','#fff2ae','#f1e2cc','#cccccc'],
      ['#1b9e77','#d95f02','#7570b3','#e7298a','#66a61e','#e6ab02','#a6761d','#666666'],
    ],
  },
  {
    label: 'Sequential', offset: 4,
    names: ['Blues', 'Greens', 'Reds', 'YlOrBr'],
    palettes: [
      ['#f7fbff','#deebf7','#c6dbef','#9ecae1','#6baed6','#4292c6','#2171b5','#084594'],
      ['#f7fcf5','#e5f5e0','#c7e9c0','#a1d99b','#74c476','#41ab5d','#238b45','#005a32'],
      ['#fff5f0','#fee0d2','#fcbba1','#fc9272','#fb6a4a','#ef3b2c','#cb181d','#99000d'],
      ['#ffffe5','#fff7bc','#fee391','#fec44f','#fe9929','#ec7014','#cc4c02','#8c2d04'],
    ],
  },
  {
    label: 'Diverging', offset: 8,
    names: ['Spectral', 'RdYlBu', 'RdYlGn', 'PiYG'],
    palettes: [
      ['#d53e4f','#f46d43','#fdae61','#fee08b','#e6f598','#abdda4','#66c2a5','#3288bd'],
      ['#d73027','#f46d43','#fdae61','#fee090','#e0f3f8','#abd9e9','#74add1','#4575b4'],
      ['#d73027','#f46d43','#fdae61','#fee08b','#d9ef8b','#a6d96a','#66bd63','#1a9850'],
      ['#c51b7d','#de77ae','#f1b6da','#fde0ef','#e6f5d0','#b8e186','#7fbc41','#4d9221'],
    ],
  },
]

export const PALETTES = PALETTE_GROUPS.flatMap(g => g.palettes)
export const ALL_PALETTE_COLORS = PALETTES.flat()

export function textColorForBg(hex) {
  if (!hex || hex.length < 7) return '#374151'
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return (0.299 * r + 0.587 * g + 0.114 * b) > 140 ? '#374151' : '#f9fafb'
}

export function getChoiceColor(col, value) {
  const opts = typeof col.options === 'string' ? JSON.parse(col.options) : col.options
  const stored = opts?.choiceColors?.[value]
  if (stored) return stored
  const choices = opts?.choices ?? []
  const pi = opts?.palette ?? 0
  const idx = choices.indexOf(value)
  return PALETTES[pi]?.[(idx >= 0 ? idx : 0) % PALETTES[pi].length] ?? PALETTES[0][0]
}

export function choicePillStyle(col, value) {
  const bg = getChoiceColor(col, value)
  return { backgroundColor: bg, color: textColorForBg(bg) }
}

export function workspaceAvatarStyle(name) {
  const colors = PALETTE_GROUPS[0].palettes.flat()
  const idx = [...(name || ' ')].reduce((acc, c) => acc + c.charCodeAt(0), 0) % colors.length
  const bg = colors[idx]
  return { backgroundColor: bg, color: textColorForBg(bg) }
}
