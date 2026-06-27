export const EMOJIS = [
  { name: 'check',       label: 'Check',       emoji: '✅' },
  { name: 'cross',       label: 'Cross',       emoji: '❌' },
  { name: 'warning',     label: 'Warning',     emoji: '⚠️' },
  { name: 'info',        label: 'Info',        emoji: 'ℹ️' },
  { name: 'flag',        label: 'Flag',        emoji: '🚩' },
  { name: 'star',        label: 'Star',        emoji: '⭐' },
  { name: 'fire',        label: 'Fire',        emoji: '🔥' },
  { name: 'bolt',        label: 'Bolt',        emoji: '⚡' },
  { name: 'pin',         label: 'Pin',         emoji: '📌' },
  { name: 'lock',        label: 'Lock',        emoji: '🔒' },
  { name: 'heart',       label: 'Heart',       emoji: '❤️' },
  { name: 'thumbs-up',   label: 'Thumbs up',   emoji: '👍' },
  { name: 'thumbs-down', label: 'Thumbs down', emoji: '👎' },
  { name: 'bookmark',    label: 'Bookmark',    emoji: '🔖' },
  { name: 'eye',         label: 'Eye',         emoji: '👁️' },
  { name: 'question',    label: 'Question',    emoji: '❓' },
]

export function getEmoji(name) {
  return EMOJIS.find(e => e.name === name) ?? null
}

// Resolve display emoji for both old 'symbol' and new 'emoji' column types.
// Semantic names are identical, so the lookup works for both.
export function resolveEmoji(value) {
  return getEmoji(value)?.emoji ?? null
}
