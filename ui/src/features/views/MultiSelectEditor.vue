<template>
  <div class="flex flex-col">
    <!-- Selected tags -->
    <div class="flex flex-wrap gap-1.5 px-3 pt-3 pb-2.5 min-h-10">
      <span
        v-for="v in modelValue"
        :key="v"
        class="inline-flex items-center gap-1 pl-2 pr-1 py-0.5 rounded-full text-[11px] font-medium"
        :style="choicePillStyle(column, v)"
      >
        {{ v }}
        <button
          type="button"
          class="flex items-center justify-center w-3.5 h-3.5 rounded-full opacity-60 hover:opacity-100 transition-opacity bg-black/10 border-none cursor-pointer leading-none"
          @click.stop="remove(v)"
        ><RiDeleteBin6Line size="9" /></button>
      </span>
      <span v-if="!modelValue?.length" class="text-[12px] text-text-3 self-center italic">Nothing selected</span>
    </div>

    <!-- Divider -->
    <div class="border-t border-border-1" />

    <!-- Choice list -->
    <div class="overflow-y-auto max-h-52 py-0.5">
      <label
        v-for="c in choices"
        :key="c"
        class="flex items-center gap-2.5 px-3 py-1.5 cursor-pointer hover:bg-surface-2 transition-colors"
      >
        <input
          type="checkbox"
          :checked="isSelected(c)"
          class="w-3.5 h-3.5 accent-brand-600 cursor-pointer shrink-0"
          @change="toggle(c)"
        />
        <PillBadge :col="column" :value="c" />
      </label>
      <div v-if="!choices?.length" class="px-3 py-2 text-[12px] text-text-3 italic">No options defined</div>
    </div>

    <!-- Add option -->
    <template v-if="workspaceCode && tableCode">
      <div class="border-t border-border-1" />
      <div v-if="addingOption" class="flex gap-1 px-3 py-2">
        <input
          ref="addInputEl"
          v-model="newOptionName"
          class="flex-1 text-[12px] bg-transparent border border-border-1 rounded px-2 py-1 outline-none focus:border-brand-400 text-text-1"
          placeholder="Option name…"
          @keyup.enter="confirmAdd"
          @keyup.escape="cancelAdd"
        />
        <button
          class="text-[11px] px-2 py-1 rounded bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 transition-colors"
          @click="confirmAdd"
        >Add</button>
        <button
          class="text-[11px] px-2 py-1 rounded bg-transparent border border-border-1 text-text-2 cursor-pointer hover:bg-border-1 transition-colors"
          @click="cancelAdd"
        >Cancel</button>
      </div>
      <button
        v-else
        class="flex items-center gap-1 px-3 py-2 text-[11px] text-text-3 hover:text-brand-600 dark:hover:text-brand-400 bg-transparent border-none cursor-pointer transition-colors w-full text-left"
        @click="startAdd"
      >
        <span class="text-base leading-none">+</span> Add option
      </button>
    </template>
  </div>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import { choicePillStyle, PALETTES } from './palettes.js'
import { api } from '../../api/client.js'
import PillBadge from '../../foundation/PillBadge.vue'
import { RiDeleteBin6Line } from '@remixicon/vue'

const props = defineProps({
  modelValue:    { type: Array,  default: () => [] },
  choices:       { type: Array,  default: () => [] },
  column:        { type: Object, default: () => ({}) },
  workspaceCode: { type: String, default: '' },
  tableCode:     { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue', 'column-updated'])

function isSelected(c) { return (props.modelValue ?? []).includes(c) }
function remove(v) { emit('update:modelValue', (props.modelValue ?? []).filter(x => x !== v)) }
function toggle(c) {
  const current = props.modelValue ?? []
  emit('update:modelValue', isSelected(c) ? current.filter(x => x !== c) : [...current, c])
}

const addingOption = ref(false)
const newOptionName = ref('')
const addInputEl = ref(null)

function startAdd() {
  addingOption.value = true
  nextTick(() => addInputEl.value?.focus())
}

function cancelAdd() {
  addingOption.value = false
  newOptionName.value = ''
}

async function confirmAdd() {
  const name = newOptionName.value.trim()
  if (!name) return

  const opts = typeof props.column.options === 'string'
    ? JSON.parse(props.column.options)
    : (props.column.options ?? {})

  const currentChoices = opts.choices ?? []
  if (currentChoices.includes(name)) {
    if (!isSelected(name)) toggle(name)
    cancelAdd()
    return
  }

  const pi = opts.palette ?? 0
  const palette = PALETTES[pi] ?? PALETTES[0]
  const color = palette[currentChoices.length % palette.length]

  const newOptions = {
    ...opts,
    choices: [...currentChoices, name],
    choiceColors: { ...(opts.choiceColors ?? {}), [name]: color },
  }

  const updated = await api.updateColumn(
    props.workspaceCode, props.tableCode, props.column.code,
    { name: props.column.name, type: props.column.type, options: newOptions },
  )
  emit('column-updated', updated)
  emit('update:modelValue', [...(props.modelValue ?? []), name])
  cancelAdd()
}
</script>
