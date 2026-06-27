<template>
  <Flapout ref="flapout">
    <template #trigger="{ toggle, setAnchor }">
      <button
        :ref="setAnchor"
        :class="variant === 'cell' ? 'cell-trigger' : 'form-trigger'"
        @click.stop="toggle"
        @keydown.tab.prevent="flapout?.close(); emit('tab', $event)"
        @keydown.escape.stop="flapout?.close(); emit('cancel')"
      >
        <PillBadge v-if="modelValue" :col="col" :value="modelValue" />
        <span v-else class="text-text-3">—</span>
        <RiArrowDownSLine v-if="variant === 'form'" size="14" class="ml-auto shrink-0 text-text-3" />
      </button>
    </template>

    <!-- Dropdown panel -->
    <PanelCard class="min-w-44 max-h-72 overflow-y-auto py-1">
      <!-- Clear -->
      <button
        class="option-row"
        :class="!modelValue ? 'bg-border-1' : ''"
        @click="pick('')"
      >
        <span class="text-text-3">—</span>
      </button>

      <!-- Choices -->
      <button
        v-for="c in choices"
        :key="c"
        class="option-row"
        :class="c === modelValue ? 'bg-border-1' : ''"
        @click="pick(c)"
      >
        <PillBadge :col="col" :value="c" />
        <RiCheckLine v-if="c === modelValue" size="12" class="ml-auto shrink-0 text-text-3" />
      </button>

      <!-- Add option (only when workspace/table codes are available) -->
      <template v-if="workspaceCode && tableCode">
        <div class="mx-2 my-1 border-t border-border-1" />

        <div v-if="adding" class="flex items-center gap-1 px-2 py-1">
          <input
            ref="addEl"
            v-model="newName"
            class="flex-1 text-[12px] bg-transparent border border-border-1 rounded px-2 py-1 outline-none focus:border-brand-400 text-text-1 font-[inherit]"
            placeholder="New option…"
            @keyup.enter="confirmAdd"
            @keyup.escape.stop="cancelAdd"
          />
          <button
            class="text-[11px] px-2 py-1 rounded bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 transition-colors shrink-0"
            @click="confirmAdd"
          >Add</button>
          <button
            class="text-[11px] px-2 py-1 rounded bg-transparent border border-border-1 text-text-2 cursor-pointer hover:bg-border-1 transition-colors shrink-0"
            @click="cancelAdd"
          >✕</button>
        </div>
        <button v-else class="option-row text-text-3 hover:text-brand-600 dark:hover:text-brand-400" @click.stop="startAdd">
          <span class="text-sm leading-none">+</span>
          Add option
        </button>
      </template>
    </PanelCard>
  </Flapout>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import Flapout from '../../foundation/Flapout.vue'
import { PALETTES } from './palettes.js'
import { api } from '../../api/client.js'
import { RiArrowDownSLine, RiCheckLine } from '@remixicon/vue'
import PillBadge from '../../foundation/PillBadge.vue'
import PanelCard from '../../foundation/PanelCard.vue'

const props = defineProps({
  modelValue:    { default: '' },
  choices:       { type: Array, default: () => [] },
  col:           { type: Object, required: true },
  variant:       { type: String, default: 'form' },
  workspaceCode: { type: String, default: '' },
  tableCode:     { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'commit', 'cancel', 'tab', 'column-updated'])

const flapout = ref(null)
const adding = ref(false)
const newName = ref('')
const addEl = ref(null)

// In cell mode, open the picker immediately when the editor mounts
onMounted(() => {
  if (props.variant === 'cell') nextTick(() => flapout.value?.toggle())
})

function pick(value) {
  emit('update:modelValue', value)
  flapout.value?.close()
  emit('commit')
}

function startAdd() {
  adding.value = true
  nextTick(() => addEl.value?.focus())
}

function cancelAdd() {
  adding.value = false
  newName.value = ''
}

async function confirmAdd() {
  const name = newName.value.trim()
  if (!name) return

  const opts = typeof props.col.options === 'string'
    ? JSON.parse(props.col.options)
    : (props.col.options ?? {})

  const currentChoices = opts.choices ?? []
  if (currentChoices.includes(name)) {
    pick(name)
    cancelAdd()
    return
  }

  const pi = opts.palette ?? 0
  const palette = PALETTES[pi] ?? PALETTES[0]
  const color = palette[currentChoices.length % palette.length]

  const updated = await api.updateColumn(
    props.workspaceCode, props.tableCode, props.col.code,
    {
      name: props.col.name,
      type: props.col.type,
      options: {
        ...opts,
        choices: [...currentChoices, name],
        choiceColors: { ...(opts.choiceColors ?? {}), [name]: color },
      },
    },
  )
  emit('column-updated', updated)
  pick(name)
  cancelAdd()
}
</script>

<style scoped>
.cell-trigger {
  display: flex;
  align-items: center;
  gap: 4px;
  width: 100%;
  height: 34px;
  background: transparent;
  border: none;
  outline: none;
  padding: 0 10px;
  cursor: pointer;
  font-size: 13px;
  font-family: inherit;
  color: var(--cell-text);
  text-align: left;
  box-sizing: border-box;
}

.form-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  background: var(--input-bg);
  border: 1px solid var(--input-border);
  border-radius: 8px;
  color: var(--text-1);
  font-size: 14px;
  font-family: inherit;
  padding: 8px 12px;
  outline: none;
  cursor: pointer;
  text-align: left;
  box-sizing: border-box;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.form-trigger:focus {
  border-color: var(--color-brand-600);
  box-shadow: 0 0 0 3px rgba(210, 105, 30, 0.12);
}

.option-row {
  display: flex;
  align-items: center;
  gap: 6px;
  width: calc(100% - 8px);
  margin: 0 4px;
  padding: 5px 8px;
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 13px;
  font-family: inherit;
  text-align: left;
  color: var(--text-1);
  border-radius: 5px;
  box-sizing: border-box;
  transition: background 0.1s;
}
.option-row:hover {
  background: rgb(243 244 246 / 1);
}
:global(.dark) .option-row:hover {
  background: #3c3c3c;
}
</style>
