<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-199" @click="close" />

    <div
      ref="floatingEl"
      tabindex="-1"
      class="fixed z-200 flex flex-col rounded-xl overflow-hidden bg-surface-1 border border-border-1 shadow-[0_8px_32px_rgba(0,0,0,.18),0_2px_8px_rgba(0,0,0,.10)] focus:outline-none"
      :style="panelStyle"
      @click.stop
      @keydown.escape.stop="close"
      @keydown.tab="onTabKey"
    >
      <!-- Header -->
      <div class="flex items-center gap-2 px-3 h-9 bg-[#f5f5f8] dark:bg-[#2d2d30] border-b border-border-1 shrink-0">
        <span class="text-[10px] font-bold tracking-widest uppercase text-text-2 flex-1 truncate">{{ colName }}</span>
        <button
          v-if="colType === 'row-link' && localVal"
          class="flex items-center justify-center w-5 h-5 bg-transparent border-none text-text-2 cursor-pointer rounded hover:text-brand-600 hover:bg-border-1 transition-colors"
          title="Open linked record"
          @click="navigateToLinked"
        >
          <RiArrowRightUpLine size="13" />
        </button>
        <button
          class="flex items-center justify-center w-5 h-5 bg-transparent border-none text-text-2 cursor-pointer rounded hover:text-text-1 hover:bg-border-1 transition-colors"
          title="Close (Esc)"
          @click="close"
        >
          <RiCloseLine size="13" />
        </button>
      </div>

      <!-- Long text -->
      <textarea
        v-if="colType === 'long-text'"
        ref="longTextRef"
        v-model="localVal"
        class="flex-1 min-h-0 w-full resize-none outline-none border-none bg-surface-1 px-4 py-3 text-[13px] leading-relaxed text-text-1 font-[inherit] placeholder-zinc-300 dark:placeholder-[#6a6a6a]"
        placeholder="Enter text…"
      />

      <!-- Markdown -->
      <div v-else-if="colType === 'markdown'" class="flex-1 min-h-0 overflow-auto">
        <MarkdownEditor v-model="localVal" />
      </div>

      <!-- Multi-select -->
      <div v-else-if="colType === 'multi-select'" class="flex-1 min-h-0 overflow-auto">
        <MultiSelectEditor
          :model-value="localVal"
          :choices="columnChoices"
          :column="column"
          @update:model-value="localVal = $event"
        />
      </div>

      <!-- Checklist -->
      <div v-else-if="colType === 'checklist'" class="flex-1 min-h-0 overflow-auto">
        <ChecklistEditor :model-value="localVal ?? []" @update:model-value="localVal = $event" />
      </div>

      <!-- Row link picker -->
      <div v-else-if="colType === 'row-link'" class="flex-1 min-h-0 overflow-auto">
        <RowLinkPicker
          :model-value="localVal"
          :workspace-code="workspaceCode"
          :target-table-code="columnOptions.targetTableCode"
          :display-column-code="columnOptions.displayColumnCode"
          @update:model-value="localVal = $event"
        />
      </div>

      <!-- Referencing records list -->
      <div v-else-if="colType === 'referencing'" class="flex-1 min-h-0 overflow-auto">
        <div class="p-3 flex flex-col gap-1">
          <a
            v-for="ref in referencingRecords"
            :key="ref.sourceRowId"
            class="flex items-center gap-1.5 px-2 py-1.5 rounded text-[12px] text-text-1 no-underline hover:bg-border-1 transition-colors cursor-pointer"
            @click="navigateToRef(ref)"
          >
            <RiNodeTree size="11" class="text-text-3 shrink-0" />
            <span class="text-text-2 text-[11px] shrink-0">{{ ref.sourceTableName }}</span>
            <span class="truncate">{{ ref.sourceRowLabel }}</span>
          </a>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { computePosition, autoUpdate, flip, shift, size, offset as offsetMiddleware } from '@floating-ui/dom'
import MarkdownEditor from '../../foundation/MarkdownEditor.vue'
import MultiSelectEditor from '../views/MultiSelectEditor.vue'
import ChecklistEditor from './ChecklistEditor.vue'
import RowLinkPicker from './RowLinkPicker.vue'
import { RiCloseLine, RiArrowRightUpLine, RiNodeTree } from '@remixicon/vue'

const props = defineProps({
  modelValue:     { default: null },
  colType:        { type: String, required: true },
  colName:        { type: String, default: '' },
  column:         { type: Object, default: () => ({}) },
  anchorRect:     { type: Object, default: null },
  workspaceCode:  { type: String, default: '' },
  referencingRecords: { type: Array, default: () => [] },
})
const emit = defineEmits(['close', 'navigate'])

const route = useRoute()
const router = useRouter()

const localVal    = ref(props.modelValue)
const longTextRef = ref(null)
const floatingEl  = ref(null)
const floatLeft   = ref(0)
const floatTop    = ref(0)
const positioned  = ref(false)

watch(() => props.modelValue, v => { localVal.value = v })

const columnChoices = computed(() => {
  const opts = typeof props.column?.options === 'string'
    ? JSON.parse(props.column.options)
    : props.column?.options
  return opts?.choices ?? []
})

const columnOptions = computed(() => {
  const opts = typeof props.column?.options === 'string'
    ? JSON.parse(props.column.options)
    : props.column?.options
  return opts ?? {}
})

// Panel width by type; variable-height types use floating-ui size() for max-height
const panelWidth = computed(() => {
  if (props.colType === 'multi-select') return 260
  if (props.colType === 'checklist')    return 280
  if (props.colType === 'row-link')     return 280
  if (props.colType === 'referencing')  return 300
  return 460
})

const isVariableHeight = computed(() =>
  ['multi-select', 'checklist', 'row-link', 'referencing'].includes(props.colType)
)

async function place() {
  const el = floatingEl.value
  if (!el) return

  if (!props.anchorRect) {
    // No anchor: center on screen
    positioned.value = true
    return
  }

  // Use a virtual element so floating-ui can position against the cell rect
  const virtualAnchor = { getBoundingClientRect: () => props.anchorRect }

  const middleware = [
    offsetMiddleware(4),
    flip({ padding: 8 }),
    shift({ padding: 8 }),
  ]

  if (isVariableHeight.value) {
    // Let floating-ui constrain panel height to the available viewport space
    middleware.push(size({
      padding: 8,
      apply({ availableHeight, elements }) {
        elements.floating.style.maxHeight = Math.max(80, availableHeight) + 'px'
      },
    }))
  }

  const { x, y } = await computePosition(virtualAnchor, el, {
    placement: 'bottom-start',
    strategy: 'fixed',
    middleware,
  })

  floatLeft.value = x
  floatTop.value = y
  positioned.value = true
}

const panelStyle = computed(() => {
  if (!props.anchorRect) {
    // Centered fallback when no anchor is available
    const s = { width: panelWidth.value + 'px', left: '50%', top: '50%', transform: 'translate(-50%,-50%)' }
    if (!isVariableHeight.value) s.height = '360px'
    return s
  }

  const s = {
    width:      panelWidth.value + 'px',
    left:       floatLeft.value + 'px',
    top:        floatTop.value + 'px',
    visibility: positioned.value ? 'visible' : 'hidden',
  }
  if (!isVariableHeight.value) s.height = '360px'
  return s
})

// autoUpdate re-runs place() via ResizeObserver whenever the panel grows (e.g.
// checklist items added) — handles both dynamic repositioning and flip/maxHeight
// recalculation when the panel height changes after initial positioning.
let stopAutoUpdate = null

function startAutoUpdate() {
  stopAutoUpdate?.()
  stopAutoUpdate = null
  const el = floatingEl.value
  if (!el) { positioned.value = false; return }
  if (!props.anchorRect) { positioned.value = true; return }
  const virtualAnchor = { getBoundingClientRect: () => props.anchorRect }
  stopAutoUpdate = autoUpdate(virtualAnchor, el, place)
}

watch(floatingEl, (el) => {
  if (!el) { stopAutoUpdate?.(); stopAutoUpdate = null; positioned.value = false; return }
  startAutoUpdate()
  // Focus the panel once it becomes visible so Tab keydowns are captured.
  // Must wait for positioned=true because browsers ignore focus() on visibility:hidden elements.
  // Long-text is excluded — it auto-focuses its own textarea in onMounted.
  if (props.colType !== 'long-text') {
    const stopFocusWatch = watch(positioned, (isPositioned) => {
      if (isPositioned) {
        stopFocusWatch()
        el.focus({ preventScroll: true })
      }
    }, { flush: 'post' })
  }
}, { flush: 'post' })

watch(() => props.anchorRect, startAutoUpdate)

onUnmounted(() => { stopAutoUpdate?.() })

function close() { emit('close', localVal.value) }

function navigateToLinked() {
  const targetCode = columnOptions.value.targetTableCode
  const rowId = localVal.value?.id
  if (!targetCode || !rowId) return
  emit('close', localVal.value)
  router.push(`/workspaces/${props.workspaceCode}/tables/${targetCode}/rows/${rowId}?returnTo=${encodeURIComponent(route.fullPath)}`)
}

function navigateToRef(ref) {
  emit('close', localVal.value)
  router.push(`/workspaces/${props.workspaceCode}/tables/${ref.sourceTableCode}/rows/${ref.sourceRowId}?returnTo=${encodeURIComponent(route.fullPath)}`)
}

function onTabKey(event) {
  // Let CodeMirror handle Tab for indentation in the markdown editor
  if (props.colType === 'markdown') return
  event.preventDefault()
  emit('navigate', { dir: event.shiftKey ? 'prev' : 'next', value: localVal.value })
}

onMounted(() => {
  if (props.colType === 'long-text') {
    nextTick(() => {
      longTextRef.value?.focus()
      const len = longTextRef.value?.value?.length ?? 0
      longTextRef.value?.setSelectionRange(len, len)
    })
  }
})
</script>
