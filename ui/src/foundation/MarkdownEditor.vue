<template>
  <Teleport to="body" :disabled="!isFullscreen">
    <div
      :class="isFullscreen
        ? 'fixed inset-0 z-[300] flex flex-col bg-surface-1'
        : 'flex flex-col border border-border-1 rounded-lg overflow-hidden focus-within:border-brand-400 focus-within:ring-3 focus-within:ring-brand-100 dark:focus-within:ring-brand-900/20 transition-all'"
    >
      <!-- Toolbar -->
      <div class="flex flex-wrap items-center gap-1 px-2 py-1.5 bg-surface-2 border-b border-border-1">

        <!-- Format buttons (WYSIWYG only) -->
        <template v-if="!rawMode">
          <button v-for="btn in formatButtons" :key="btn.label"
            class="toolbar-btn"
            :class="{ 'active': btn.active?.() }"
            :title="btn.title"
            @mousedown.prevent="btn.action"
            v-html="btn.label"
          />
          <div class="w-px h-4 bg-border-1 mx-0.5 shrink-0" />
          <button
            class="toolbar-btn"
            :class="{ 'active': editor?.isActive('bulletList') }"
            title="Bullet list"
            @mousedown.prevent="editor?.chain().focus().toggleBulletList().run()"
          ><RiListUnordered size="15" /></button>
          <button
            class="toolbar-btn"
            :class="{ 'active': editor?.isActive('orderedList') }"
            title="Ordered list"
            @mousedown.prevent="editor?.chain().focus().toggleOrderedList().run()"
          ><RiListOrdered size="15" /></button>
          <button
            class="toolbar-btn"
            :class="{ 'active': editor?.isActive('blockquote') }"
            title="Blockquote"
            @mousedown.prevent="editor?.chain().focus().toggleBlockquote().run()"
          ><RiDoubleQuotesL size="15" /></button>
          <div class="w-px h-4 bg-border-1 mx-0.5 shrink-0" />
          <button
            class="toolbar-btn font-mono text-[11px]"
            :class="{ 'active': editor?.isActive('code') }"
            title="Inline code"
            @mousedown.prevent="editor?.chain().focus().toggleCode().run()"
          >`c`</button>
          <button
            class="toolbar-btn font-mono text-[10px] tracking-tighter"
            :class="{ 'active': editor?.isActive('codeBlock') }"
            title="Code block"
            @mousedown.prevent="editor?.chain().focus().toggleCodeBlock().run()"
          >```</button>
        </template>

        <!-- Undo / Redo (WYSIWYG only) -->
        <template v-if="!rawMode">
          <div class="w-px h-4 bg-border-1 mx-0.5 shrink-0" />
          <button
            class="toolbar-btn"
            :disabled="!editor?.can().undo()"
            title="Undo (Ctrl+Z)"
            @mousedown.prevent="editor?.chain().focus().undo().run()"
          ><RiArrowGoBackLine size="14" /></button>
          <button
            class="toolbar-btn"
            :disabled="!editor?.can().redo()"
            title="Redo (Ctrl+Shift+Z)"
            @mousedown.prevent="editor?.chain().focus().redo().run()"
          ><RiArrowGoForwardLine size="14" /></button>
        </template>

        <!-- Right-side actions -->
        <div class="flex items-center gap-1 ml-auto shrink-0">
          <!-- Download -->
          <button
            class="toolbar-btn"
            title="Download as .md"
            @mousedown.prevent="downloadMarkdown"
          ><RiDownloadLine size="14" /></button>

          <!-- Full-screen toggle -->
          <button
            class="toolbar-btn"
            :title="isFullscreen ? 'Exit full screen (Esc)' : 'Full screen'"
            @mousedown.prevent="toggleFullscreen"
          >
            <RiFullscreenExitLine v-if="isFullscreen" size="14" />
            <RiFullscreenLine v-else size="14" />
          </button>

          <div class="w-px h-4 bg-border-1 mx-0.5" />

          <!-- Mode toggle -->
          <div class="flex items-center rounded-md border border-border-1 overflow-hidden bg-surface-1">
            <button
              class="px-2.5 py-0.5 text-[11px] font-medium transition-colors"
              :class="!rawMode ? 'bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400' : 'text-text-2 hover:text-text-1'"
              @click="setMode(false)"
            >Rich</button>
            <button
              class="px-2.5 py-0.5 text-[11px] font-medium transition-colors border-l border-border-1"
              :class="rawMode ? 'bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400' : 'text-text-2 hover:text-text-1'"
              @click="setMode(true)"
            >Markdown</button>
          </div>
        </div>
      </div>

      <!-- WYSIWYG editor -->
      <EditorContent v-if="!rawMode" :editor="editor" :class="['tiptap-wrap', isFullscreen && 'fullscreen']" />

      <!-- Raw markdown textarea -->
      <textarea
        v-else
        v-model="rawContent"
        :class="['w-full p-3.5 font-mono text-[13px] leading-relaxed text-text-1 bg-surface-1 resize-none outline-none', isFullscreen ? 'flex-1 min-h-0' : 'min-h-52']"
        placeholder="Enter markdown…"
        @input="onRawInput"
      />
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import { Markdown } from 'tiptap-markdown'
import {
  RiListUnordered, RiListOrdered, RiDoubleQuotesL,
  RiArrowGoBackLine, RiArrowGoForwardLine,
  RiDownloadLine, RiFullscreenLine, RiFullscreenExitLine,
} from '@remixicon/vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const rawMode = ref(false)
const rawContent = ref(props.modelValue ?? '')
const hasFocus = ref(false)
const isFullscreen = ref(false)

const editor = useEditor({
  extensions: [
    StarterKit,
    Markdown.configure({ html: false, transformPastedText: true, transformCopiedText: true }),
  ],
  content: props.modelValue ?? '',
  onFocus() { hasFocus.value = true },
  onBlur() { hasFocus.value = false },
  onUpdate({ editor }) {
    if (!rawMode.value) {
      emit('update:modelValue', editor.storage.markdown.getMarkdown())
    }
  },
})

watch(() => props.modelValue, (val) => {
  if (rawMode.value) {
    rawContent.value = val ?? ''
    return
  }
  if (hasFocus.value) return
  const current = editor.value?.storage.markdown.getMarkdown()
  if (current !== val) {
    editor.value?.commands.setContent(val ?? '')
  }
})

const formatButtons = computed(() => [
  { label: '<b>B</b>', title: 'Bold', active: () => editor.value?.isActive('bold'), action: () => editor.value?.chain().focus().toggleBold().run() },
  { label: '<i>I</i>', title: 'Italic', active: () => editor.value?.isActive('italic'), action: () => editor.value?.chain().focus().toggleItalic().run() },
  { label: '<s>S</s>', title: 'Strikethrough', active: () => editor.value?.isActive('strike'), action: () => editor.value?.chain().focus().toggleStrike().run() },
  { label: 'H1', title: 'Heading 1', active: () => editor.value?.isActive('heading', { level: 1 }), action: () => editor.value?.chain().focus().toggleHeading({ level: 1 }).run() },
  { label: 'H2', title: 'Heading 2', active: () => editor.value?.isActive('heading', { level: 2 }), action: () => editor.value?.chain().focus().toggleHeading({ level: 2 }).run() },
  { label: 'H3', title: 'Heading 3', active: () => editor.value?.isActive('heading', { level: 3 }), action: () => editor.value?.chain().focus().toggleHeading({ level: 3 }).run() },
])

function setMode(toRaw) {
  if (toRaw === rawMode.value) return
  if (toRaw) {
    rawContent.value = editor.value?.storage.markdown.getMarkdown() ?? ''
  } else {
    editor.value?.commands.setContent(rawContent.value)
  }
  rawMode.value = toRaw
}

function onRawInput() {
  emit('update:modelValue', rawContent.value)
}

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
}

function downloadMarkdown() {
  const content = rawMode.value
    ? rawContent.value
    : (editor.value?.storage.markdown.getMarkdown() ?? '')
  const blob = new Blob([content], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'document.md'
  a.click()
  URL.revokeObjectURL(url)
}

function onKeydown(e) {
  if (e.key === 'Escape' && isFullscreen.value) {
    isFullscreen.value = false
  }
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  editor.value?.destroy()
  document.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
.toolbar-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 24px;
  height: 24px;
  padding: 0 4px;
  border-radius: 4px;
  border: none;
  background: transparent;
  color: var(--toolbar-text);
  font-size: 12px;
  cursor: pointer;
  transition: background 0.1s, color 0.1s;
  flex-shrink: 0;
}
.toolbar-btn:hover, .toolbar-btn.active {
  background: var(--toolbar-hover-bg);
  color: var(--toolbar-hover-text);
}
.toolbar-btn:disabled {
  opacity: 0.35;
  cursor: default;
  pointer-events: none;
}
</style>

<style>
/* Global: TipTap content styles */
.tiptap-wrap .tiptap {
  outline: none;
  min-height: 200px;
  padding: 14px 16px;
  font-size: 14px;
  line-height: 1.75;
  color: var(--text-1);
  background: var(--surface-1);
}
.tiptap-wrap.fullscreen {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
.tiptap-wrap.fullscreen .tiptap {
  min-height: 100%;
}
.tiptap-wrap .tiptap > * + * { margin-top: 0.5em; }
.tiptap-wrap .tiptap h1 { font-size: 1.5em; font-weight: 700; line-height: 1.3; }
.tiptap-wrap .tiptap h2 { font-size: 1.25em; font-weight: 650; line-height: 1.35; }
.tiptap-wrap .tiptap h3 { font-size: 1.1em; font-weight: 600; line-height: 1.4; }
.tiptap-wrap .tiptap ul { list-style: disc; padding-left: 1.5em; }
.tiptap-wrap .tiptap ol { list-style: decimal; padding-left: 1.5em; }
.tiptap-wrap .tiptap blockquote {
  border-left: 3px solid var(--blockquote-border);
  padding-left: 1em;
  color: var(--blockquote-text);
  font-style: italic;
}
.tiptap-wrap .tiptap code {
  background: var(--code-bg);
  border: 1px solid var(--code-border);
  border-radius: 4px;
  padding: 0.1em 0.35em;
  font-family: ui-monospace, monospace;
  font-size: 0.875em;
  color: var(--code-text);
}
.tiptap-wrap .tiptap pre {
  background: var(--codeblock-bg);
  color: var(--codeblock-text);
  border-radius: 8px;
  padding: 1em 1.25em;
  overflow-x: auto;
  font-family: ui-monospace, monospace;
  font-size: 0.875em;
  line-height: 1.6;
}
.tiptap-wrap .tiptap pre code {
  background: none;
  border: none;
  padding: 0;
  color: inherit;
  font-size: inherit;
}
.tiptap-wrap .tiptap strong { font-weight: 600; }
.tiptap-wrap .tiptap em { font-style: italic; }
.tiptap-wrap .tiptap s { text-decoration: line-through; }
.tiptap-wrap .tiptap p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  color: var(--placeholder);
  pointer-events: none;
  float: left;
  height: 0;
}
</style>
