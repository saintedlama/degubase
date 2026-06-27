<template>
  <Teleport to="body">
    <div
      class="fixed inset-0 z-400 bg-black/85 flex items-center justify-center p-4"
      @click="$emit('close')"
      @keydown.escape.stop="$emit('close')"
    >
      <button
        class="absolute top-4 right-4 flex items-center justify-center w-9 h-9 rounded-full bg-black/50 border border-white/20 text-white/80 hover:text-white hover:bg-black/70 transition-colors cursor-pointer"
        title="Close (Esc)"
        @click.stop="$emit('close')"
      >
        <RiCloseLine size="18" />
      </button>
      <img
        :src="src"
        :alt="alt"
        class="max-w-full max-h-full object-contain rounded select-none"
        @click.stop
      />
    </div>
  </Teleport>
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { RiCloseLine } from '@remixicon/vue'

defineProps({
  src: { type: String, required: true },
  alt: { type: String, default: '' },
})
const emit = defineEmits(['close'])

function onKeydown(e) {
  if (e.key === 'Escape') emit('close')
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>
