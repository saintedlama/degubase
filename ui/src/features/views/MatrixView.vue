<template>
    <div
        class="flex flex-col flex-1 min-h-0 overflow-hidden bg-zinc-100 dark:bg-[#1e1e1e]"
    >
        <!-- Need at least 2 single-select columns -->
        <EmptyState
            v-if="selectColumns.length < 2"
            message="Add at least two single-select columns to use the matrix view."
        >
            <template #icon
                ><RiGridLine
                    size="36"
                    class="text-zinc-300 dark:text-[#454545]"
            /></template>
        </EmptyState>

        <template v-else>
            <!-- Sub-toolbar -->
            <TableToolbar
                :table="table"
                :view-config="viewConfig"
                @update:search="searchDebounced = $event"
                @sort-change="emit('sort-change', $event)"
                @filter-change="emit('filter-change', $event)"
                @add-record="addRecord(null, null)"
                @bulk-action="showBulkDialog = true"
                @export-csv="exportCsv"
                @import-csv="showImportDialog = true"
            >
                <template #before>
                    <span class="text-[11px] text-text-2">Columns</span>
                    <select
                        v-model="localXCol"
                        class="h-6 px-2 text-[11px] rounded border border-border-1 bg-surface-2 text-text-1 outline-none cursor-pointer"
                        @change="onXColChange"
                    >
                        <option
                            v-for="col in selectColumns"
                            :key="col.code"
                            :value="col.code"
                            :disabled="col.code === localYCol"
                        >
                            {{ col.name }}
                        </option>
                    </select>
                    <span class="text-[11px] text-text-2">Rows</span>
                    <select
                        v-model="localYCol"
                        class="h-6 px-2 text-[11px] rounded border border-border-1 bg-surface-2 text-text-1 outline-none cursor-pointer"
                        @change="onYColChange"
                    >
                        <option
                            v-for="col in selectColumns"
                            :key="col.code"
                            :value="col.code"
                            :disabled="col.code === localXCol"
                        >
                            {{ col.name }}
                        </option>
                    </select>
                </template>
                <template #fields>
                    <CardFieldsFlapout
                        :columns="nonAxisColumns"
                        :is-card-field="isCardField"
                        :get-field-config="getFieldConfig"
                        @toggle-field="toggleCardField"
                        @move-field="moveFieldInList"
                        @update-config="updateFieldConfig"
                        @ensure-field="ensureCardField"
                    >
                        <template #prepend>
                            <div
                                class="flex items-center gap-1 py-0.5 mb-2 border-b border-zinc-100 dark:border-[#3c3c3c] pb-2"
                            >
                                <input
                                    id="toggle-axis-labels"
                                    type="checkbox"
                                    :checked="showAxisLabels"
                                    class="w-3 h-3 accent-brand-600 cursor-pointer shrink-0"
                                    @change="
                                        emit(
                                            'axis-labels-change',
                                            !$event.target.checked,
                                        )
                                    "
                                />
                                <label
                                    for="toggle-axis-labels"
                                    class="flex-1 text-[12px] text-text-1 cursor-pointer"
                                    >Show axis labels</label
                                >
                            </div>
                        </template>
                    </CardFieldsFlapout>
                </template>
            </TableToolbar>

            <!-- Matrix grid -->
            <div class="flex-1 min-h-0 overflow-auto">
                <div class="flex w-full p-4 gap-2">
                    <!-- Y-axis label column -->
                    <div
                        v-if="showAxisLabels"
                        class="shrink-0 w-6 flex items-center justify-center self-stretch"
                    >
                        <span
                            class="text-[10px] font-semibold text-text-3 uppercase tracking-wider whitespace-nowrap"
                            style="transform: rotate(90deg)"
                            >{{ yCol?.name }} →</span
                        >
                    </div>

                    <!-- Grid: x-label row + x-header row + y-rows -->
                    <div class="flex-1 flex flex-col gap-2">
                        <!-- X-axis label row -->
                        <div v-if="showAxisLabels" class="flex gap-2 py-1">
                            <div class="shrink-0 w-8" />
                            <div class="flex-1 flex items-center px-2">
                                <span
                                    class="text-[10px] font-semibold text-text-3 uppercase tracking-wider truncate"
                                    >{{ xCol?.name }} →</span
                                >
                            </div>
                        </div>

                        <!-- X-axis header row -->
                        <div class="flex gap-2">
                            <!-- Corner spacer -->
                            <div class="shrink-0 w-8" />
                            <!-- X column headers -->
                            <div
                                v-for="xEntry in xEntries"
                                :key="xEntry.key"
                                class="flex-1 min-w-52 flex items-center gap-1.5 px-2 py-1"
                            >
                                <span
                                    v-if="xEntry.value != null"
                                    class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium truncate"
                                    :style="xEntry.pillStyle"
                                    >{{ xEntry.label }}</span
                                >
                                <span
                                    v-else
                                    class="text-[12px] font-medium text-text-3 italic"
                                    >No value</span
                                >
                                <span
                                    class="text-[10px] text-text-3 ml-auto shrink-0"
                                    >{{ xColumnCount(xEntry.key) }}</span
                                >
                            </div>
                        </div>

                        <!-- Y-axis rows -->
                        <div
                            v-for="yEntry in yEntries"
                            :key="yEntry.key"
                            class="flex gap-2"
                        >
                            <!-- Y row header -->
                            <div
                                class="shrink-0 w-8 flex items-center justify-center py-2"
                            >
                                <div style="transform: rotate(90deg)">
                                    <span
                                        v-if="yEntry.value != null"
                                        class="inline-block px-1.5 py-0.5 rounded-full text-[11px] font-medium whitespace-nowrap"
                                        :style="yEntry.pillStyle"
                                        >{{ yEntry.label }}</span
                                    >
                                    <span
                                        v-else
                                        class="text-[12px] font-medium text-text-3 italic whitespace-nowrap"
                                        >No value</span
                                    >
                                </div>
                            </div>

                            <!-- Cells -->
                            <div
                                v-for="xEntry in xEntries"
                                :key="xEntry.key"
                                class="flex-1 min-w-52 flex flex-col rounded-lg p-1.5 gap-1.5 min-h-20 transition-all"
                                :class="
                                    dragOverCell ===
                                    cellKey(xEntry.key, yEntry.key)
                                        ? 'bg-brand-50 dark:bg-brand-900/20 ring-1 ring-inset ring-brand-300 dark:ring-brand-700/60'
                                        : 'bg-zinc-200/60 dark:bg-[#2a2a2a]'
                                "
                                @dragover.prevent="
                                    dragOverCell = cellKey(
                                        xEntry.key,
                                        yEntry.key,
                                    )
                                "
                                @dragleave="
                                    onDragLeave($event, xEntry.key, yEntry.key)
                                "
                                @drop.prevent="
                                    onDrop($event, xEntry.value, yEntry.value)
                                "
                            >
                                <!-- Cards -->
                                <div
                                    v-for="row in cellRows(
                                        xEntry.key,
                                        yEntry.key,
                                    )"
                                    :key="row.id"
                                    draggable="true"
                                    class="bg-surface-1 rounded-lg border border-border-1 cursor-pointer shadow-sm hover:shadow-md hover:border-zinc-300 dark:hover:border-[#555] transition-all overflow-hidden select-none p-2.5 shrink-0"
                                    :class="
                                        draggingRowId === row.id
                                            ? 'opacity-30'
                                            : ''
                                    "
                                    @dragstart="onDragStart($event, row)"
                                    @dragend="onDragEnd"
                                    @click="emit('open-row', { row })"
                                >
                                    <template
                                        v-for="field in visibleCardFields"
                                        :key="field.col.id"
                                    >
                                        <div
                                            v-if="hasValue(row, field.col)"
                                            :class="
                                                field.col.type === 'image'
                                                    ? 'first:-mt-2.5 last:-mb-2.5'
                                                    : 'mb-1 last:mb-0'
                                            "
                                        >
                                            <CardCellRenderer
                                                :row="row"
                                                :field="field"
                                                :compact="true"
                                                :workspace-code="workspaceCode"
                                                :table-code="tableCode"
                                            />
                                        </div>
                                    </template>
                                    <div
                                        v-if="
                                            !visibleCardFields.some((f) =>
                                                hasValue(row, f.col),
                                            )
                                        "
                                        class="text-[11px] text-zinc-300 dark:text-[#555] italic"
                                    >
                                        Empty
                                    </div>
                                </div>

                                <!-- Add record in cell -->
                                <button
                                    class="mt-auto flex items-center gap-1 px-1.5 py-1 text-[11px] text-text-3 hover:text-text-1 bg-transparent border-none cursor-pointer rounded-md hover:bg-zinc-300/50 dark:hover:bg-[#3c3c3c] transition-colors w-full"
                                    @click="
                                        addRecord(xEntry.value, yEntry.value)
                                    "
                                >
                                    <RiAddLine size="12" />
                                    Add
                                </button>
                            </div>
                        </div>
                    </div>
                    <!-- end grid flex-col -->
                </div>
                <!-- end outer flex -->

                <!-- Load more -->
                <div
                    v-if="hasMore"
                    class="flex justify-center px-4 py-2 border-t border-border-1"
                >
                    <button
                        class="inline-flex items-center gap-1.5 px-4 py-1.5 text-[12px] font-medium text-text-2 bg-surface-1 border border-border-1 rounded-md cursor-pointer hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#555] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        :disabled="loadingMore"
                        @click="loadMore"
                    >
                        <span v-if="loadingMore">Loading…</span>
                        <span v-else
                            >Load more ({{ rows.length }} / {{ total }})</span
                        >
                    </button>
                </div>
            </div>
        </template>

        <BulkActionDialog
            v-if="showBulkDialog"
            :workspace-code="workspaceCode"
            :table-code="tableCode"
            :columns="table.columns ?? []"
            :initial-filters="viewConfig?.filters ?? []"
            @done="onBulkDone"
            @cancel="showBulkDialog = false"
        />

        <ImportCsvDialog
            v-if="showImportDialog"
            :workspace-code="workspaceCode"
            :table-code="tableCode"
            @done="onImportDone"
            @cancel="showImportDialog = false"
        />
    </div>
</template>

<script setup>
import { ref, computed, watch } from "vue";
import { api } from "../../api/client.js";
import { getChoiceColor, textColorForBg } from "./palettes.js";
import { hasValue } from "../records/cellHelpers.js";
import Flapout from "../../foundation/Flapout.vue";
import BulkActionDialog from "./BulkActionDialog.vue";
import ImportCsvDialog from "./ImportCsvDialog.vue";
import TableToolbar from "./TableToolbar.vue";
import CardCellRenderer from "../records/CardCellRenderer.vue";
import { RiAddLine, RiGridLine } from "@remixicon/vue";
import { useViewRows } from "../records/useViewRows.js";
import { useSortedColumns } from "../records/useSortedColumns.js";
import { useCardFields } from "./useCardFields.js";
import { useFieldReordering } from "../records/useFieldReordering.js";
import { useNotifications } from "../../foundation/useNotifications.js";
import EmptyState from "../../foundation/EmptyState.vue";
import CardFieldsFlapout from "./CardFieldsFlapout.vue";

const props = defineProps({
    workspaceCode: { type: String, required: true },
    tableCode: { type: String, required: true },
    table: { type: Object, required: true },
    viewConfig: { type: Object, default: () => ({}) },
});
const emit = defineEmits([
    "open-row",
    "reload-table",
    "sort-change",
    "filter-change",
    "x-col-change",
    "y-col-change",
    "axis-labels-change",
    "card-fields-change",
    "column-order-change",
]);

const { notify } = useNotifications();
const { rows, total, hasMore, loadingMore, loadMore, searchDebounced } =
    useViewRows(() => props);
const showBulkDialog = ref(false);
const showImportDialog = ref(false);

function onBulkDone(count) {
    showBulkDialog.value = false;
    notify(`${count} record${count !== 1 ? "s" : ""} updated`);
}

function exportCsv() {
    const params = {};
    if (props.viewConfig?.filters?.length)
        params.filters = JSON.stringify(props.viewConfig.filters);
    if (props.viewConfig?.sort?.length)
        params.sort = JSON.stringify(props.viewConfig.sort);
    const url = api.exportCsvUrl(props.workspaceCode, props.tableCode, params);
    const a = document.createElement("a");
    a.href = url;
    a.download = "";
    a.click();
}

function onImportDone(count) {
    showImportDialog.value = false;
    notify(`${count} record${count !== 1 ? "s" : ""} imported`);
}

// ── Columns ───────────────────────────────────────────────────────────────────

const sortedColumns = useSortedColumns(
    computed(() => props.table.columns ?? []),
    computed(() => props.viewConfig?.columnOrder),
);

const selectColumns = computed(() =>
    sortedColumns.value.filter((c) => c.type === "single-select"),
);

const localXCol = ref(
    props.viewConfig?.xCol ?? selectColumns.value[0]?.code ?? null,
);
const localYCol = ref(
    props.viewConfig?.yCol ?? selectColumns.value[1]?.code ?? null,
);

watch(
    () => props.viewConfig?.xCol,
    (code) => {
        localXCol.value = code ?? selectColumns.value[0]?.code ?? null;
    },
);
watch(
    () => props.viewConfig?.yCol,
    (code) => {
        localYCol.value = code ?? selectColumns.value[1]?.code ?? null;
    },
);

watch(selectColumns, (cols) => {
    if (localXCol.value == null && cols.length > 0)
        localXCol.value = cols[0].code;
    if (localYCol.value == null && cols.length > 1)
        localYCol.value = cols[1].code;
});

const xCol = computed(
    () => props.table.columns?.find((c) => c.code === localXCol.value) ?? null,
);
const yCol = computed(
    () => props.table.columns?.find((c) => c.code === localYCol.value) ?? null,
);
const showAxisLabels = computed(() => !props.viewConfig?.hideAxisLabels);

function getChoices(col) {
    if (!col) return [];
    const opts =
        typeof col.options === "string" ? JSON.parse(col.options) : col.options;
    return opts?.choices ?? [];
}

const xChoices = computed(() => getChoices(xCol.value));
const yChoices = computed(() => getChoices(yCol.value));

// ── Matrix entries (choices + optional "No value") ────────────────────────────

function buildEntries(col, choices) {
    if (!col) return [];
    const colCode = col.code;
    const knownChoices = new Set(choices);

    const entries = choices.map((choice) => {
        const bg = getChoiceColor(col, choice);
        return {
            key: choice,
            value: choice,
            label: choice,
            pillStyle: { backgroundColor: bg, color: textColorForBg(bg) },
        };
    });

    const hasNoValue = rows.value.some((r) => {
        const v = r.data?.[colCode];
        return v == null || v === "" || !knownChoices.has(v);
    });
    if (hasNoValue || entries.length === 0) {
        entries.push({
            key: "__none__",
            value: null,
            label: "No value",
            pillStyle: {},
        });
    }
    return entries;
}

const xEntries = computed(() => buildEntries(xCol.value, xChoices.value));
const yEntries = computed(() => buildEntries(yCol.value, yChoices.value));

// ── Cell helpers ──────────────────────────────────────────────────────────────

function cellKey(xKey, yKey) {
    return `${xKey}|||${yKey}`;
}

function matchesEntry(val, entryKey, knownChoices) {
    if (entryKey === "__none__")
        return val == null || val === "" || !knownChoices.has(val);
    return val === entryKey;
}

function cellRows(xKey, yKey) {
    const xColCode = xCol.value?.code ?? null;
    const yColCode = yCol.value?.code ?? null;
    if (!xColCode || !yColCode) return [];
    const xKnown = new Set(xChoices.value);
    const yKnown = new Set(yChoices.value);
    return rows.value.filter(
        (r) =>
            matchesEntry(r.data?.[xColCode], xKey, xKnown) &&
            matchesEntry(r.data?.[yColCode], yKey, yKnown),
    );
}

function xColumnCount(xKey) {
    return yEntries.value.reduce(
        (sum, y) => sum + cellRows(xKey, y.key).length,
        0,
    );
}

// ── Card fields (excludes axis columns) ───────────────────────────────────────

const nonAxisColumns = computed(() =>
    sortedColumns.value.filter(
        (c) => c.code !== localXCol.value && c.code !== localYCol.value,
    ),
);

const {
    getFieldConfig,
    isCardField,
    ensureCardField,
    toggleCardField,
    updateFieldConfig,
    visibleCardFields,
} = useCardFields(
    computed(() => props.viewConfig),
    nonAxisColumns,
    emit,
);

const { moveFieldInList } = useFieldReordering(
    nonAxisColumns,
    computed(() => props.viewConfig),
    emit,
);

// ── Drag and drop ─────────────────────────────────────────────────────────────

const draggingRowId = ref(null);
const dragOverCell = ref(null);

function onDragStart(event, row) {
    draggingRowId.value = row.id;
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", String(row.id));
}

function onDragEnd() {
    draggingRowId.value = null;
    dragOverCell.value = null;
}

function onDragLeave(event, xKey, yKey) {
    if (!event.currentTarget.contains(event.relatedTarget)) {
        if (dragOverCell.value === cellKey(xKey, yKey))
            dragOverCell.value = null;
    }
}

async function onDrop(event, newXValue, newYValue) {
    dragOverCell.value = null;
    const rowId = Number(event.dataTransfer.getData("text/plain"));
    if (!rowId) return;
    const row = rows.value.find((r) => r.id === rowId);
    if (!row || !xCol.value || !yCol.value) return;
    draggingRowId.value = null;

    const xColCode = xCol.value.code;
    const yColCode = yCol.value.code;
    const prevX = row.data?.[xColCode] ?? null;
    const prevY = row.data?.[yColCode] ?? null;
    if (prevX === newXValue && prevY === newYValue) return;

    const newData = {
        ...(row.data ?? {}),
        [xColCode]: newXValue,
        [yColCode]: newYValue,
    };
    row.data = newData;
    try {
        const updated = await api.updateRow(
            props.workspaceCode,
            props.tableCode,
            row.id,
            { data: newData },
        );
        row.updated_at = updated.updated_at;
    } catch (e) {
        notify(e.message || "Save failed");
        row.data = {
            ...(row.data ?? {}),
            [xColCode]: prevX,
            [yColCode]: prevY,
        };
    }
}

// ── Add record ────────────────────────────────────────────────────────────────

async function addRecord(xValue, yValue) {
    const data = {};
    if (xCol.value && xValue != null) data[xCol.value.code] = xValue;
    if (yCol.value && yValue != null) data[yCol.value.code] = yValue;
    const row = await api.createRow(props.workspaceCode, props.tableCode, {
        data,
    });
    rows.value.push(row);
    emit("open-row", { row, isNew: true });
}

// ── Axis change ───────────────────────────────────────────────────────────────

function onXColChange() {
    emit("x-col-change", localXCol.value);
}
function onYColChange() {
    emit("y-col-change", localYCol.value);
}
</script>
