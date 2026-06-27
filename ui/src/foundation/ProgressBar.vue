<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed top-0 left-0 right-0 z-9999 h-0.5 pointer-events-none"
    >
      <div
        class="h-full"
        :class="failed ? 'bg-red-500' : 'bg-brand-500'"
        :style="{
          width: pct + '%',
          transition: pct >= 100
            ? 'width 0.15s ease-in'
            : 'width 0.3s ease-out',
        }"
      />
    </div>
  </Teleport>
</template>

<script setup>
import { useProgress } from './useProgress.js'
const { visible, pct, failed } = useProgress()
</script>
