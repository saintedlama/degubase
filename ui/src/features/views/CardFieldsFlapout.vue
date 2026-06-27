<template>
  <Flapout align="right">
    <template #trigger="{ open, toggle, setAnchor }">
      <button
        :ref="setAnchor"
        class="inline-flex items-center gap-1.5 h-6 px-2 text-[11px] font-medium rounded border transition-colors cursor-pointer"
        :class="open
          ? 'text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 border-brand-300 dark:border-brand-700/50'
          : 'text-text-2 bg-transparent border-border-1 hover:text-text-1 hover:bg-border-1'"
        @click.stop="toggle"
      >
        <RiLayoutColumnLine size="12" />
        Fields
      </button>
    </template>

    <PanelCard class="p-3 w-80 max-h-96 overflow-y-auto">
      <slot name="prepend" />

      <div class="text-[10px] font-semibold text-text-3 mb-1.5 uppercase tracking-wide">Fields on cards</div>
      <div v-if="columns.length === 0" class="text-[12px] text-text-3 italic">No other fields available</div>
      <div v-for="(col, idx) in columns" :key="col.code" class="flex items-center gap-1 py-0.5 group/field">
        <input
          type="checkbox"
          :id="`cf-${col.code}`"
          :checked="isCardField(col.code)"
          class="w-3 h-3 accent-brand-600 cursor-pointer shrink-0"
          @change="emit('toggle-field', col.code)"
        />
        <label :for="`cf-${col.code}`" class="flex-1 text-[12px] text-text-1 cursor-pointer truncate min-w-0">{{ col.name }}</label>
        <div class="flex items-center gap-0.5">
          <button
            :disabled="idx === 0"
            class="flex items-center justify-center w-4 h-4 rounded bg-transparent border-none text-text-3 hover:text-text-1 hover:bg-border-1 transition-colors disabled:opacity-30 disabled:cursor-default cursor-pointer"
            title="Move up"
            @click.stop="emit('move-field', col, 'up')"
          ><RiArrowUpSLine size="11" /></button>
          <button
            :disabled="idx === columns.length - 1"
            class="flex items-center justify-center w-4 h-4 rounded bg-transparent border-none text-text-3 hover:text-text-1 hover:bg-border-1 transition-colors disabled:opacity-30 disabled:cursor-default cursor-pointer"
            title="Move down"
            @click.stop="emit('move-field', col, 'down')"
          ><RiArrowDownSLine size="11" /></button>
        </div>
        <ToggleIconButton
          :active="getFieldConfig(col.code).style === 'title'"
          title="Title style"
          class="shrink-0"
          @click.stop="emit('ensure-field', col.code); emit('update-config', col.code, { style: getFieldConfig(col.code).style === 'title' ? 'default' : 'title' })"
        ><RiFontSize size="12" /></ToggleIconButton>
        <ToggleIconButton
          :active="getFieldConfig(col.code).showLabel !== false"
          title="Show label"
          class="shrink-0"
          @click.stop="emit('ensure-field', col.code); emit('update-config', col.code, { showLabel: getFieldConfig(col.code).showLabel === false })"
        ><RiPriceTag3Line size="12" /></ToggleIconButton>
        <ToggleIconButton
          v-if="isTruncatable(col)"
          :active="getFieldConfig(col.code).truncate !== false"
          title="Truncate text"
          class="shrink-0"
          @click.stop="emit('ensure-field', col.code); emit('update-config', col.code, { truncate: getFieldConfig(col.code).truncate === false })"
        ><RiFileReduceLine size="12" /></ToggleIconButton>
      </div>
    </PanelCard>
  </Flapout>
</template>

<script setup>
import Flapout from '../../foundation/Flapout.vue'
import PanelCard from '../../foundation/PanelCard.vue'
import ToggleIconButton from '../../foundation/ToggleIconButton.vue'
import { RiLayoutColumnLine, RiFontSize, RiPriceTag3Line, RiFileReduceLine, RiArrowUpSLine, RiArrowDownSLine } from '@remixicon/vue'

defineProps({
  columns:       { type: Array,    required: true },
  isCardField:   { type: Function, required: true },
  getFieldConfig:{ type: Function, required: true },
  isTruncatable: { type: Function, default: () => false },
})

const emit = defineEmits(['toggle-field', 'move-field', 'update-config', 'ensure-field'])
</script>
