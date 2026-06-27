<template>
  <div
    class="fixed inset-0 bg-black/35 flex items-center justify-center z-200 backdrop-blur-sm"
    @mousedown.self="down = true"
    @click.self="dismiss"
  >
    <slot />
  </div>
</template>

<script setup>
import { ref } from 'vue'

const emit = defineEmits(['dismiss'])
const down = ref(false)

function dismiss() {
  if (down.value) {
    down.value = false
    emit('dismiss')
  }
}
</script>
