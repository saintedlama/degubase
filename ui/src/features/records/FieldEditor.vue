<template>
  <!-- ── Cell variant: inline table cell editing ─────────────── -->
  <template v-if="variant === 'cell'">
    <input
      v-if="isNumber"
      v-model.number="localVal"
      type="number"
      class="cell-editor"
      @keyup.enter="emit('commit')"
      @keydown.tab.prevent="emit('tab', $event)"
      @keyup.escape="emit('cancel')"
      @blur="emit('commit')"
    />
    <input
      v-else-if="col.type === 'date'"
      v-model="localVal"
      type="date"
      class="cell-editor"
      @keyup.enter="emit('commit')"
      @keydown.tab.prevent="emit('tab', $event)"
      @keyup.escape="emit('cancel')"
      @blur="emit('commit')"
    />
    <input
      v-else-if="col.type === 'datetime'"
      v-model="localVal"
      type="datetime-local"
      class="cell-editor"
      @keyup.enter="emit('commit')"
      @keydown.tab.prevent="emit('tab', $event)"
      @keyup.escape="emit('cancel')"
      @blur="emit('commit')"
    />
    <SelectPicker
      v-else-if="col.type === 'single-select'"
      :model-value="localVal"
      :choices="choices"
      :col="col"
      variant="cell"
      :workspace-code="workspaceCode"
      :table-code="tableCode"
      @update:model-value="val => emit('update:modelValue', val)"
      @commit="emit('commit')"
      @cancel="emit('cancel')"
      @tab="emit('tab', $event)"
      @column-updated="emit('column-updated', $event)"
    />
    <EmojiPicker
      v-else-if="col.type === 'symbol' || col.type === 'emoji'"
      :model-value="localVal"
      variant="cell"
      @update:model-value="val => emit('update:modelValue', val)"
      @commit="emit('commit')"
      @cancel="emit('cancel')"
      @tab="emit('tab', $event)"
    />
    <input
      v-else
      v-model="localVal"
      class="cell-editor"
      @keyup.enter="emit('commit')"
      @keydown.tab.prevent="emit('tab', $event)"
      @keyup.escape="emit('cancel')"
      @blur="emit('commit')"
    />
  </template>

  <!-- ── Form variant: record detail form fields ─────────────── -->
  <template v-else>
    <!-- Read-only system timestamps -->
    <p v-if="col.type === 'created-at'" class="text-sm text-text-2">{{ formatDate(row.created_at) }}</p>
    <p v-else-if="col.type === 'updated-at'" class="text-sm text-text-2">{{ formatDate(row.updated_at) }}</p>

    <!-- Markdown WYSIWYG -->
    <MarkdownEditor
      v-else-if="col.type === 'markdown'"
      :model-value="localVal"
      @update:model-value="val => { emit('update:modelValue', val); emit('commit') }"
    />

    <!-- Mermaid diagram editor -->
    <MermaidEditor
      v-else-if="col.type === 'mermaid'"
      :model-value="localVal"
      @update:model-value="val => { emit('update:modelValue', val); emit('commit') }"
    />

    <!-- Long text -->
    <textarea
      v-else-if="col.type === 'long-text'"
      v-model="localVal"
      rows="5"
      class="form-input resize-y font-[inherit] leading-relaxed"
      @change="emit('commit')"
    />

    <!-- Checkbox -->
    <label v-else-if="col.type === 'checkbox'" class="inline-flex items-center gap-2 cursor-pointer">
      <input
        type="checkbox"
        :checked="!!localVal"
        class="w-4 h-4 rounded accent-brand-600 cursor-pointer"
        @change="e => { emit('update:modelValue', e.target.checked); emit('commit') }"
      />
      <span class="text-sm text-text-1">{{ localVal ? 'Yes' : 'No' }}</span>
    </label>

    <!-- Single-select -->
    <SelectPicker
      v-else-if="col.type === 'single-select'"
      :model-value="localVal"
      :choices="choices"
      :col="col"
      variant="form"
      :workspace-code="workspaceCode"
      :table-code="tableCode"
      @update:model-value="val => { emit('update:modelValue', val); emit('commit') }"
      @column-updated="emit('column-updated', $event)"
    />

    <!-- Emoji / Symbol -->
    <EmojiPicker
      v-else-if="col.type === 'symbol' || col.type === 'emoji'"
      :model-value="localVal"
      variant="form"
      @update:model-value="val => { emit('update:modelValue', val); emit('commit') }"
    />

    <!-- Multi-select -->
    <MultiSelectEditor
      v-else-if="col.type === 'multi-select'"
      :model-value="localVal"
      :choices="choices"
      :column="col"
      :workspace-code="workspaceCode"
      :table-code="tableCode"
      class="border border-(--input-border) rounded-lg overflow-hidden"
      @update:model-value="val => { emit('update:modelValue', val); emit('commit') }"
      @column-updated="emit('column-updated', $event)"
    />

    <!-- Date -->
    <input
      v-else-if="col.type === 'date'"
      v-model="localVal"
      type="date"
      class="form-input w-48"
      @change="emit('commit')"
    />

    <!-- Datetime -->
    <input
      v-else-if="col.type === 'datetime'"
      v-model="localVal"
      type="datetime-local"
      class="form-input w-64"
      @change="emit('commit')"
    />

    <!-- Rating -->
    <div v-else-if="col.type === 'rating'" class="flex items-center gap-0.5">
      <button
        v-for="star in 5"
        :key="star"
        class="text-2xl leading-none bg-transparent border-none p-1 cursor-pointer transition-colors"
        :class="star <= (localVal || 0) ? 'text-amber-400' : 'text-zinc-200 dark:text-[#3c3c3c] hover:text-amber-200'"
        @click="setRating(star)"
      >★</button>
      <span class="ml-1 text-sm text-text-2">{{ localVal || 0 }} / 5</span>
    </div>

    <!-- Number types (currency/percent with prefix/suffix) -->
    <div v-else-if="isNumber" class="flex items-center gap-1.5">
      <span v-if="col.type === 'currency'" class="text-sm text-text-2">$</span>
      <input
        v-model.number="localVal"
        type="number"
        class="form-input w-40"
        @change="emit('commit')"
      />
      <span v-if="col.type === 'percent'" class="text-sm text-text-2">%</span>
    </div>

    <!-- File / Image -->
    <template v-else-if="col.type === 'file' || col.type === 'image'">
      <!-- Single hidden input used for both initial upload and replace -->
      <input
        ref="fileInputEl"
        type="file"
        class="hidden"
        :accept="col.type === 'image' ? 'image/*' : undefined"
        @change="emit('file-upload', $event)"
      />

      <!-- File exists: preview + actions -->
      <div v-if="fileMeta" class="flex items-center gap-3">
        <img
          v-if="col.type === 'image' && isImageMime(fileMeta.mimeType)"
          :src="thumbnailUrl(workspaceCode, tableCode, rowId, fileMeta.fileId)"
          class="w-24 h-24 object-cover rounded-lg border border-border-1 cursor-zoom-in"
          :alt="fileMeta.filename"
          @click.stop="emit('lightbox', { src: fileUrl(workspaceCode, tableCode, rowId, fileMeta.fileId), alt: fileMeta.filename })"
        />
        <div v-else class="flex items-center gap-3 p-3 bg-surface-2 rounded-lg border border-border-1">
          <span class="text-xl">📄</span>
          <div class="min-w-0">
            <div class="text-sm font-medium text-text-1 truncate max-w-80">{{ fileMeta.filename }}</div>
            <div class="text-xs text-text-2">{{ formatFileSize(fileMeta.size) }}</div>
          </div>
        </div>
        <div class="flex items-center gap-1">
          <IconButton
            v-if="col.type === 'image' && isImageMime(fileMeta.mimeType)"
            class="w-7 h-7"
            title="View full size"
            @click.stop="emit('lightbox', { src: fileUrl(workspaceCode, tableCode, rowId, fileMeta.fileId), alt: fileMeta.filename })"
          >
            <RiFullscreenLine size="14" />
          </IconButton>
          <a
            :href="fileUrl(workspaceCode, tableCode, rowId, fileMeta.fileId)"
            :download="fileMeta.filename"
            class="flex items-center justify-center w-7 h-7 bg-transparent border border-border-1 text-text-2 cursor-pointer rounded-md hover:text-brand-600 hover:border-brand-400 transition-colors"
            title="Download"
          >
            <RiDownloadLine size="14" />
          </a>
          <IconButton
            class="w-7 h-7"
            title="Replace"
            @click="fileInputEl?.click()"
          >
            <RiUploadLine size="14" />
          </IconButton>
          <IconButton
            class="w-7 h-7"
            title="Remove"
            danger
            @click="emit('file-remove')"
          >
            <RiCloseLine size="14" />
          </IconButton>
        </div>
      </div>

      <!-- No file: upload prompt -->
      <label
        v-else
        class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-text-2 bg-surface-2 border border-dashed border-zinc-300 dark:border-[#4a4a4a] rounded-lg cursor-pointer hover:border-brand-400 hover:text-brand-600 transition-colors"
        @click.prevent="fileInputEl?.click()"
      >
        <RiUploadLine size="16" />
        {{ col.type === 'image' ? 'Upload Image' : 'Upload File' }}
      </label>
    </template>

    <!-- Checklist -->
    <ChecklistEditor
      v-else-if="col.type === 'checklist'"
      :model-value="localVal ?? []"
      class="border border-(--input-border) rounded-lg overflow-hidden"
      @update:model-value="val => { emit('update:modelValue', val); emit('commit') }"
    />

    <!-- Row link -->
    <RowLinkPicker
      v-else-if="col.type === 'row-link'"
      :model-value="localVal"
      :workspace-code="workspaceCode"
      :target-table-code="rowLinkTargetCode"
      class="border border-(--input-border) rounded-lg overflow-hidden"
      @update:model-value="val => { emit('update:modelValue', val); emit('commit') }"
    />

    <!-- Text, email, url -->
    <input
      v-else
      v-model="localVal"
      :type="col.type === 'email' ? 'email' : col.type === 'url' ? 'url' : 'text'"
      class="form-input"
      @change="emit('commit')"
    />
  </template>
</template>

<script setup>
import { computed, ref } from 'vue'
import { fileUrl, thumbnailUrl } from '../../api/client.js'
import MarkdownEditor from '../../foundation/MarkdownEditor.vue'
import MermaidEditor from '../../foundation/MermaidEditor.vue'
import MultiSelectEditor from '../views/MultiSelectEditor.vue'
import SelectPicker from '../views/SelectPicker.vue'
import EmojiPicker from '../views/EmojiPicker.vue'
import ChecklistEditor from './ChecklistEditor.vue'
import RowLinkPicker from './RowLinkPicker.vue'
import { RiUploadLine, RiDownloadLine, RiCloseLine, RiFullscreenLine } from '@remixicon/vue'
import IconButton from '../../foundation/IconButton.vue'

const props = defineProps({
  modelValue: { default: null },
  col: { type: Object, required: true },
  variant: { type: String, default: 'cell' },
  row: { type: Object, default: null },
  workspaceCode: { type: String, default: '' },
  tableCode: { type: String, default: '' },
  rowId: { type: [Number, String], default: null },
})

const emit = defineEmits(['update:modelValue', 'commit', 'cancel', 'tab', 'file-upload', 'file-remove', 'lightbox', 'column-updated'])

const fileInputEl = ref(null)

const localVal = computed({
  get: () => props.modelValue,
  set: val => emit('update:modelValue', val),
})

const isNumber = computed(() =>
  ['number', 'currency', 'percent'].includes(props.col.type)
)

const rowLinkTargetCode = computed(() => {
  if (props.col.type !== 'row-link') return ''
  const opts = typeof props.col.options === 'string' ? JSON.parse(props.col.options) : props.col.options
  return opts?.targetTableCode || props.tableCode
})

const choices = computed(() => {
  const opts = typeof props.col.options === 'string' ? JSON.parse(props.col.options) : props.col.options
  return opts?.choices ?? []
})

const fileMeta = computed(() => {
  const v = props.modelValue
  if (!v || typeof v !== 'object') return null
  return v
})

function isImageMime(mime) { return mime && mime.startsWith('image/') }

function formatDate(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

function formatFileSize(bytes) {
  if (bytes == null) return ''
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function setRating(star) {
  const next = star === (props.modelValue || 0) ? 0 : star
  emit('update:modelValue', next)
  emit('commit')
}
</script>

<style scoped>
.cell-editor {
  display: block;
  width: 100%;
  height: 34px;
  min-height: 34px;
  background: transparent;
  border: none;
  outline: none;
  color: var(--cell-text);
  font-size: 13px;
  font-family: inherit;
  padding: 0 10px;
  box-sizing: border-box;
  line-height: 34px;
  resize: none;
}
.cell-editor option { background: var(--cell-option-bg); }

.form-input {
  width: 100%;
  background: var(--input-bg);
  border: 1px solid var(--input-border);
  border-radius: 8px;
  color: var(--text-1);
  font-size: 14px;
  font-family: inherit;
  padding: 8px 12px;
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
  box-sizing: border-box;
}
.form-input:focus {
  border-color: var(--color-brand-600);
  box-shadow: 0 0 0 3px rgba(210, 105, 30, 0.12);
  background: var(--input-bg-focus);
}
</style>
