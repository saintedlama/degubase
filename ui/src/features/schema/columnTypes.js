export const TYPE_FAMILIES = {
  'text': 'text', 'long-text': 'text', 'markdown': 'text', 'mermaid': 'text', 'email': 'text', 'url': 'text',
  'number': 'number', 'currency': 'number', 'percent': 'number', 'rating': 'number',
  'date': 'date', 'datetime': 'date',
  'checkbox': 'checkbox',
  'checklist': 'checklist',
  'single-select': 'select', 'multi-select': 'select',
  'file': 'file', 'image': 'file',
  'symbol': 'emoji', 'emoji': 'emoji',
  'created-at': 'system', 'updated-at': 'system',
  'row-link': 'link',
}

export const ALL_TYPES = [
  { value: 'text',          label: 'Text',          family: 'text' },
  { value: 'long-text',     label: 'Long Text',     family: 'text' },
  { value: 'markdown',      label: 'Markdown',      family: 'text' },
  { value: 'mermaid',       label: 'Mermaid',       family: 'text' },
  { value: 'email',         label: 'Email',         family: 'text' },
  { value: 'url',           label: 'URL',           family: 'text' },
  { value: 'number',        label: 'Number',        family: 'number' },
  { value: 'currency',      label: 'Currency',      family: 'number' },
  { value: 'percent',       label: 'Percent',       family: 'number' },
  { value: 'rating',        label: 'Rating',        family: 'number' },
  { value: 'date',          label: 'Date',          family: 'date' },
  { value: 'datetime',      label: 'Date & Time',   family: 'date' },
  { value: 'checkbox',      label: 'Checkbox',      family: 'checkbox' },
  { value: 'checklist',     label: 'Checklist',     family: 'checklist' },
  { value: 'single-select', label: 'Single Select', family: 'select' },
  { value: 'multi-select',  label: 'Multi Select',  family: 'select' },
  { value: 'file',          label: 'File',          family: 'file' },
  { value: 'image',         label: 'Image',         family: 'file' },
  { value: 'emoji',         label: 'Emoji',         family: 'emoji' },
  { value: 'created-at',    label: 'Created At',    family: 'system' },
  { value: 'updated-at',    label: 'Updated At',    family: 'system' },
  { value: 'row-link',      label: 'Row Link',      family: 'link' },
]

export function compatibleTypes(family) {
  return ALL_TYPES.filter(t => t.family === family)
}

export const TYPE_GROUPS = [
  { label: 'Text',   types: ALL_TYPES.filter(t => t.family === 'text') },
  { label: 'Number', types: ALL_TYPES.filter(t => t.family === 'number') },
  { label: 'Date',   types: ALL_TYPES.filter(t => t.family === 'date') },
  { label: 'File',   types: ALL_TYPES.filter(t => t.family === 'file') },
  { label: 'Other',  types: ALL_TYPES.filter(t => ['checkbox', 'checklist', 'single-select', 'multi-select', 'emoji', 'created-at', 'updated-at'].includes(t.value)) },
  { label: 'Relations', types: ALL_TYPES.filter(t => t.family === 'link') },
]

import {
  RiText, RiAlignLeft, RiMarkdownLine, RiFlowChart, RiMailLine, RiLink,
  RiHashtag, RiMoneyDollarCircleLine, RiPercentLine, RiStarLine,
  RiCalendarLine, RiCalendarEventLine,
  RiCheckboxLine, RiCheckboxMultipleLine,
  RiListRadio, RiListCheck2,
  RiFileLine, RiImageLine,
  RiEmotionLine,
  RiTimeLine, RiRefreshLine,
  RiNodeTree,
} from '@remixicon/vue'

export const TYPE_ICONS = {
  'text':          RiText,
  'long-text':     RiAlignLeft,
  'markdown':      RiMarkdownLine,
  'mermaid':       RiFlowChart,
  'email':         RiMailLine,
  'url':           RiLink,
  'number':        RiHashtag,
  'currency':      RiMoneyDollarCircleLine,
  'percent':       RiPercentLine,
  'rating':        RiStarLine,
  'date':          RiCalendarLine,
  'datetime':      RiCalendarEventLine,
  'checkbox':      RiCheckboxLine,
  'checklist':     RiCheckboxMultipleLine,
  'single-select': RiListRadio,
  'multi-select':  RiListCheck2,
  'file':          RiFileLine,
  'image':         RiImageLine,
  'emoji':         RiEmotionLine,
  'created-at':    RiTimeLine,
  'updated-at':    RiRefreshLine,
  'row-link':      RiNodeTree,
}

export function typeIcon(type) { return TYPE_ICONS[type] ?? RiText }
