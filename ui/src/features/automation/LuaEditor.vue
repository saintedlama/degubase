<template>
  <div ref="el" class="lua-editor" />
</template>

<script setup>
import { ref, watch, onMounted, onBeforeUnmount, shallowRef } from 'vue'
import { EditorState } from '@codemirror/state'
import { EditorView, lineNumbers, highlightActiveLineGutter, highlightActiveLine, keymap } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { StreamLanguage, indentUnit } from '@codemirror/language'
import { lua } from '@codemirror/legacy-modes/mode/lua'
import { oneDark } from '@codemirror/theme-one-dark'

const props = defineProps({
  modelValue: { type: String, default: '' },
  readonly: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const el = ref(null)
const view = shallowRef(null)

const updateListener = EditorView.updateListener.of((update) => {
  if (update.docChanged) {
    emit('update:modelValue', update.state.doc.toString())
  }
})

const baseTheme = EditorView.theme({
  '&': { height: '100%', fontSize: '13px' },
  '.cm-scroller': { fontFamily: 'ui-monospace, "Cascadia Code", Menlo, monospace', overflow: 'auto' },
  '.cm-content': { padding: '10px 0' },
  '.cm-line': { padding: '0 14px' },
  '.cm-gutters': { paddingLeft: '4px' },
})

function buildState(doc) {
  return EditorState.create({
    doc,
    extensions: [
      lineNumbers(),
      highlightActiveLineGutter(),
      highlightActiveLine(),
      history(),
      StreamLanguage.define(lua),
      oneDark,
      baseTheme,
      indentUnit.of('  '),
      keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
      updateListener,
      EditorState.readOnly.of(props.readonly),
    ],
  })
}

onMounted(() => {
  view.value = new EditorView({
    state: buildState(props.modelValue ?? ''),
    parent: el.value,
  })
})

watch(() => props.modelValue, (val) => {
  if (!view.value) return
  const current = view.value.state.doc.toString()
  if (current !== val) {
    view.value.dispatch({
      changes: { from: 0, to: current.length, insert: val ?? '' },
    })
  }
})

onBeforeUnmount(() => view.value?.destroy())
</script>

<style scoped>
.lua-editor {
  height: 100%;
  min-height: 200px;
  border-radius: 6px;
  overflow: hidden;
}
</style>
