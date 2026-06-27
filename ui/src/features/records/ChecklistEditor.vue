<template>
  <div class="px-3 py-2 flex flex-col gap-0.5">
    <div
      v-for="(item, idx) in modelValue ?? []"
      :key="idx"
      class="flex items-center gap-2 py-1"
    >
      <input
        type="checkbox"
        :checked="item.checked"
        class="w-3.5 h-3.5 accent-brand-600 cursor-pointer shrink-0"
        @change="toggle(idx, $event.target.checked)"
      />
      <span
        class="flex-1 text-[13px] leading-snug"
        :class="item.checked ? 'line-through text-text-3' : 'text-text-1'"
      >{{ item.text }}</span>
      <button
        class="flex items-center justify-center w-4 h-4 bg-transparent border-none text-zinc-300 dark:text-[#555] cursor-pointer rounded hover:text-red-400 transition-colors"
        @click="remove(idx)"
      ><RiDeleteBin6Line size="11" /></button>
    </div>

    <div class="flex items-center gap-2 mt-1 pt-1 border-t border-zinc-100 dark:border-[#3c3c3c]">
      <span class="text-[13px] text-zinc-300 dark:text-[#555] shrink-0">+</span>
      <input
        ref="addInputRef"
        v-model="newText"
        type="text"
        placeholder="Add item…"
        class="flex-1 text-[13px] bg-transparent border-none outline-none text-text-1 placeholder-zinc-300 dark:placeholder-[#555] font-[inherit]"
        @keydown.enter.prevent="addItem"
        @blur="addItem"
      />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { RiDeleteBin6Line } from '@remixicon/vue'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:modelValue'])

const newText = ref('')
const addInputRef = ref(null)

function toggle(idx, checked) {
  const next = (props.modelValue ?? []).map((item, i) =>
    i === idx ? { ...item, checked } : item
  )
  emit('update:modelValue', next)
}

function remove(idx) {
  const next = (props.modelValue ?? []).filter((_, i) => i !== idx)
  emit('update:modelValue', next)
}

function addItem() {
  const text = newText.value.trim()
  if (!text) return
  const next = [...(props.modelValue ?? []), { text, checked: false }]
  emit('update:modelValue', next)
  newText.value = ''
}
</script>
