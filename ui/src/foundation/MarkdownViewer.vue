<template>
  <div class="annotation-md" ref="containerEl" v-html="rendered" />
</template>

<script setup>
import { computed, ref, watch, nextTick } from 'vue'
import MarkdownIt from 'markdown-it'
import mermaid from 'mermaid'

mermaid.initialize({ startOnLoad: false, theme: 'neutral', securityLevel: 'loose' })

const md = new MarkdownIt({ breaks: true, linkify: true })
const containerEl = ref(null)

const props = defineProps({
  content: { type: String, default: '' },
})

const rendered = computed(() => props.content ? md.render(props.content) : '')

let diagramCounter = 0

async function renderMermaidBlocks() {
  if (!containerEl.value) return
  const blocks = containerEl.value.querySelectorAll('code.language-mermaid')
  for (const block of blocks) {
    const source = block.textContent ?? ''
    if (!source.trim()) continue
    const id = `mermaid-md-${++diagramCounter}`
    const wrapper = block.closest('pre') ?? block
    try {
      const result = await mermaid.render(id, source)
      const div = document.createElement('div')
      div.className = 'mermaid-block'
      div.innerHTML = result.svg
      wrapper.replaceWith(div)
    } catch (e) {
      const errDiv = document.createElement('div')
      errDiv.className = 'mermaid-error'
      errDiv.textContent = e?.message ?? String(e)
      wrapper.replaceWith(errDiv)
    }
  }
}

watch(rendered, () => nextTick(renderMermaidBlocks), { immediate: true })
</script>

<style>
.annotation-md p { margin: 0 0 0.25em; }
.annotation-md p:last-child { margin-bottom: 0; }
.annotation-md strong { font-weight: 600; }
.annotation-md em { font-style: italic; }
.annotation-md code { font-family: ui-monospace, monospace; font-size: 0.9em; background: rgba(0,0,0,.06); border-radius: 3px; padding: 0.05em 0.25em; }
.annotation-md ul { list-style: disc; padding-left: 1.2em; margin: 0.2em 0; }
.annotation-md ol { list-style: decimal; padding-left: 1.2em; margin: 0.2em 0; }
.annotation-md li { margin: 0.1em 0; }
.annotation-md a { color: var(--brand-600, #4f46e5); text-decoration: underline; }
.annotation-md .mermaid-block svg { max-width: 100%; height: auto; }
.annotation-md .mermaid-error { font-family: ui-monospace, monospace; font-size: 0.85em; color: #dc2626; white-space: pre-wrap; }
</style>
