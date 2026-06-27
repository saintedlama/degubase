<template>
  <Teleport to="body">
    <Backdrop v-if="state" @dismiss="dismiss">
      <div
        class="bg-surface-1 border border-border-1 rounded-xl shadow-xl p-6 flex flex-col gap-4"
        style="width: min(420px, calc(100vw - 32px))"
      >
        <h3 v-if="state.title" class="text-[15px] font-bold text-text-1">
          {{ state.title }}
        </h3>
        <p class="text-[13px] text-text-2 leading-relaxed">
          {{ state.message }}
        </p>
        <div class="flex justify-end gap-2">
          <button
            class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-surface-2 text-text-2 border border-zinc-300 dark:border-[#3c3c3c] cursor-pointer hover:bg-border-1 transition-colors"
            @click="dismiss"
          >Cancel</button>
          <button
            class="px-4 py-1.75 rounded-md text-[13px] font-medium border-none cursor-pointer transition-colors"
            :class="state.danger
              ? 'bg-red-600 text-white hover:bg-red-700'
              : 'bg-brand-600 text-white hover:bg-brand-700'"
            @click="accept"
          >{{ state.confirmLabel }}</button>
        </div>
      </div>
    </Backdrop>
  </Teleport>
</template>

<script setup>
import Backdrop from './Backdrop.vue'
import { useConfirm } from './useConfirm.js'

const { state, accept, dismiss } = useConfirm()
</script>
