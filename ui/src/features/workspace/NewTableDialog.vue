<template>
  <Backdrop @dismiss="$emit('cancel')">
    <div
      data-testid="new-table-dialog"
      class="bg-surface-1 border border-border-1 rounded-xl shadow-xl flex flex-col overflow-hidden"
      style="width: min(500px, calc(100vw - 32px))"
    >
      <!-- Header -->
      <div class="flex items-center justify-between px-5 pt-5 pb-0">
        <h3 class="text-[15px] font-bold text-text-1">New Table</h3>
        <button
          class="flex items-center justify-center w-6.5 h-6.5 bg-transparent border-none text-text-2 cursor-pointer rounded-md hover:text-text-1 hover:bg-border-1 transition-colors"
          @click="$emit('cancel')"
        >
          <RiCloseLine size="16" />
        </button>
      </div>

      <!-- Body -->
      <div class="px-5 pt-4 pb-5 flex flex-col gap-3.5">

        <!-- Name + Code row -->
        <div class="flex gap-2.5">
          <div class="flex-1 flex flex-col gap-1.5">
            <label class="text-xs font-semibold text-text-2 tracking-wide">Name</label>
            <input
              ref="nameInput"
              v-model="form.name"
              data-testid="table-name-input"
              type="text"
              placeholder="Table name"
              class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
              @keyup.enter="submit"
            />
          </div>
          <div class="flex flex-col gap-1.5" style="width:88px">
            <label class="text-xs font-semibold text-text-2 tracking-wide">Code</label>
            <input
              v-model="form.code"
              data-testid="table-code-input"
              type="text"
              maxlength="4"
              placeholder="CODE"
              class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all font-mono uppercase tracking-widest"
              @keyup.enter="submit"
              @input="codeManuallyEdited = true"
            />
          </div>
        </div>

        <!-- Description -->
        <div class="flex flex-col gap-1.5">
          <label class="text-xs font-semibold text-text-2 tracking-wide">
            Description <span class="font-normal text-text-3">(optional)</span>
          </label>
          <textarea
            v-model="form.context"
            placeholder="Describe this table…"
            rows="2"
            class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all resize-none"
          />
        </div>

        <!-- Icon -->
        <div class="flex flex-col gap-1.5">
          <label class="text-xs font-semibold text-text-2 tracking-wide">
            Icon <span class="font-normal text-text-3">(optional)</span>
          </label>
          <IconPicker v-model="form.icon" />
        </div>

        <!-- Template -->
        <div class="flex flex-col gap-2">
          <label class="text-xs font-semibold text-text-2 tracking-wide">Template</label>

          <!-- Category dropdown -->
          <select
            v-model="selectedCategory"
            data-testid="template-category-select"
            class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 cursor-pointer transition-all"
            @change="onCategoryChange"
          >
            <option value="blank">Blank</option>
            <option v-for="group in TEMPLATE_GROUPS" :key="group.label" :value="group.label">
              {{ group.label }}
            </option>
          </select>

          <!-- Blank hint -->
          <p v-if="selectedCategory === 'blank'" class="text-[12px] text-text-3 px-0.5 mt-0.5">
            A blank table with no predefined columns.
          </p>

          <!-- Template cards -->
          <div v-else class="grid grid-cols-3 gap-2 mt-0.5">
            <button
              v-for="tmpl in currentGroupTemplates"
              :key="tmpl.id"
              :data-testid="`template-${tmpl.id}`"
              class="tcard"
              :class="selectedId === tmpl.id
                ? 'border-brand-500 dark:border-brand-400 bg-brand-50 dark:bg-brand-900/20'
                : 'border-border-1 hover:border-brand-400 dark:hover:border-brand-400 hover:bg-brand-50/60 dark:hover:bg-brand-900/10'"
              @click="selectTemplate(tmpl.id)"
            >
              <span class="text-xl leading-none mb-0.5">{{ tmpl.icon }}</span>
              <span class="text-[12px] font-semibold text-text-1 leading-tight line-clamp-1">{{ tmpl.name }}</span>
              <span class="text-[10.5px] text-text-2 leading-snug line-clamp-2">{{ tmpl.description }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex justify-end gap-2 px-5 py-3.5 border-t border-border-1 bg-surface-2">
        <button
          class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-surface-2 text-text-2 border border-zinc-300 dark:border-[#3c3c3c] cursor-pointer hover:bg-border-1 transition-colors"
          @click="$emit('cancel')"
        >Cancel</button>
        <button
          data-testid="create-table-btn"
          :disabled="!form.name.trim() || submitting"
          class="px-4 py-1.75 rounded-md text-[13px] font-medium bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 disabled:opacity-40 disabled:cursor-default transition-colors"
          @click="submit"
        >Create</button>
      </div>
    </div>
  </Backdrop>
</template>

<script setup>
import { ref, reactive, computed, nextTick, onMounted, watch } from 'vue'
import { RiCloseLine } from '@remixicon/vue'
import { TEMPLATE_GROUPS, ALL_TEMPLATES } from './templates.js'
import Backdrop from '../../foundation/Backdrop.vue'
import IconPicker from '../../foundation/IconPicker.vue'

const emit = defineEmits(['confirm', 'cancel'])

const selectedCategory = ref('blank')
const selectedId = ref(null)
const form = reactive({ name: '', context: '', icon: '', code: '' })
const nameInput = ref(null)
const submitting = ref(false)
const codeManuallyEdited = ref(false)

function generateCode(name) {
  const letters = name.toUpperCase().replace(/[^A-Z0-9]/g, '')
  return (letters.slice(0, 4) + 'XXXX').slice(0, 4)
}

watch(() => form.name, (name) => {
  if (!codeManuallyEdited.value) {
    form.code = generateCode(name)
  }
})

onMounted(() => nextTick(() => nameInput.value?.focus()))

const currentGroupTemplates = computed(() =>
  TEMPLATE_GROUPS.find(g => g.label === selectedCategory.value)?.templates ?? []
)

const selectedTemplate = computed(() =>
  selectedId.value ? ALL_TEMPLATES.find(t => t.id === selectedId.value) : null
)

function onCategoryChange() {
  selectedId.value = null
}

function selectTemplate(id) {
  if (selectedId.value === id) {
    selectedId.value = null
    return
  }
  selectedId.value = id
  if (!form.name.trim()) {
    form.name = ALL_TEMPLATES.find(t => t.id === id)?.name ?? ''
  }
  nextTick(() => nameInput.value?.focus())
}

function submit() {
  if (!form.name.trim() || submitting.value) return
  submitting.value = true
  emit('confirm', {
    name: form.name.trim(),
    context: form.context.trim(),
    icon: form.icon.trim(),
    code: (form.code.trim() || generateCode(form.name)).toUpperCase(),
    template: selectedTemplate.value ?? null,
  })
}
</script>

<style scoped>
/* Layout only — colors are Tailwind dark: utilities on the elements */
.tcard {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  padding: 10px 11px;
  border-radius: 8px;
  border-width: 1.5px;
  border-style: solid;
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: border-color 0.12s, background 0.12s;
  min-height: 84px;
  width: 100%;
}
</style>
