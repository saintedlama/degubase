<template>
  <div class="flex-1 overflow-hidden p-4 bg-[#282c34]">
    <LuaEditor v-model="form.code" class="h-full" />
  </div>
</template>

<script setup>
import { inject } from 'vue'
import LuaEditor from './LuaEditor.vue'

const form = inject('scriptForm')
</script>
