<template>
  <button
    v-bind="$attrs"
    class="inline-flex items-center gap-1.5"
    :disabled="loading || !!$attrs.disabled"
  >
    <Spinner v-if="loading" :size="spinnerSize" />
    {{ loading ? (busy || idle) : idle }}
  </button>
</template>

<script setup>
import Spinner from './Spinner.vue'

defineOptions({ inheritAttrs: false })

const props = defineProps({
  loading:    { type: Boolean, default: false },
  idle:       { type: String,  required: true },
  busy:       { type: String,  default: null },
  spinnerSize:{ type: Number,  default: 11 },
})
</script>
