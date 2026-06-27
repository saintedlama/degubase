<template>
  <slot name="trigger" :open="isOpen" :toggle="toggle" :close="close" :set-anchor="setAnchor" />

  <Teleport to="body">
    <template v-if="isOpen">
      <div class="fixed inset-0 z-49" @click="close" />
      <div
        ref="floatingEl"
        class="fixed z-50"
        :style="[floatingStyles, isPositioned ? {} : { visibility: 'hidden' }]"
        @click.stop
      >
        <slot :close="close" />
      </div>
    </template>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { autoUpdate, flip, shift, offset as offsetMiddleware } from '@floating-ui/dom'
import { useFloating } from './useFloating.js'

const props = defineProps({
  align: { type: String, default: 'left' }, // 'left' | 'right'
  offset: { type: Number, default: 4 },
})

const isOpen = ref(false)
const anchorEl = ref(null)
const floatingEl = ref(null)

const placement = computed(() => props.align === 'right' ? 'bottom-end' : 'bottom-start')

const { floatingStyles, isPositioned } = useFloating(anchorEl, floatingEl, {
  placement,
  strategy: 'fixed',
  whileElementsMounted: autoUpdate,
  middleware: computed(() => [
    offsetMiddleware(props.offset),
    flip(),
    shift({ padding: 8 }),
  ]),
})

function setAnchor(el) {
  anchorEl.value = el instanceof Element ? el : (el?.$el ?? null)
}

function toggle() {
  if (isOpen.value) { close(); return }
  isOpen.value = true
}

function close() {
  isOpen.value = false
}

defineExpose({ toggle, close })
</script>
