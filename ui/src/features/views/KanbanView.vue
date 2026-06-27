<template>
    <div
        class="flex flex-col flex-1 min-h-0 overflow-hidden bg-zinc-100 dark:bg-[#1e1e1e]"
    >
        <!-- No select columns available -->
        <EmptyState
            v-if="selectColumns.length === 0"
            message="Add a single-select column to use the kanban view."
        >
            <template #icon
                ><RiKanbanView
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
                        v-model="localLaneCode"
                        class="h-6 px-2 text-[11px] rounded border border-border-1 bg-surface-2 text-text-1 outline-none cursor-pointer"
                        @change="onLaneChange"
                    >
                        <option
                            v-for="col in selectColumns"
                            :key="col.code"
                            :value="col.code"
                        >
                            {{ col.name }}
                        </option>
                    </select>
                    <span class="text-[11px] text-text-2">Swimlanes</span>
                    <select
                        v-model="localGroupByCol"
                        class="h-6 px-2 text-[11px] rounded border border-border-1 bg-surface-2 text-text-1 outline-none cursor-pointer"
                        @change="onGroupByChange"
                    >
                        <option value="">None</option>
                        <option
                            v-for="col in groupableColumns"
                            :key="col.code"
                            :value="col.code"
                        >
                            {{ col.name }}
                        </option>
                    </select>

                    <!-- Show/hide columns -->
                    <Flapout v-if="laneChoices.length > 0" align="left">
                        <template #trigger="{ open, toggle, setAnchor }">
                            <button
                                :ref="setAnchor"
                                class="inline-flex items-center gap-1.5 h-6 px-2 text-[11px] font-medium rounded border transition-colors cursor-pointer"
                                :class="
                                    open
                                        ? 'text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 border-brand-300 dark:border-brand-700/50'
                                        : 'text-text-2 bg-transparent border-border-1 hover:text-text-1 hover:bg-border-1'
                                "
                                @click.stop="toggle"
                            >
                                <RiKanbanView size="10" />
                                Columns
                                <span
                                    v-if="hiddenColumnSet.size > 0"
                                    class="text-[10px] font-bold"
                                    >{{ hiddenColumnSet.size }} hidden</span
                                >
                            </button>
                        </template>
                        <PanelCard class="p-3 w-52">
                            <div
                                class="text-[10px] font-semibold text-text-3 mb-2 uppercase tracking-wide"
                            >
                                Show columns
                            </div>
                            <label
                                v-for="choice in laneChoices"
                                :key="choice"
                                class="flex items-center gap-2 py-1 cursor-pointer"
                            >
                                <input
                                    type="checkbox"
                                    :checked="!hiddenColumnSet.has(choice)"
                                    class="w-3.5 h-3.5 rounded border-zinc-300 dark:border-[#555] text-brand-500 focus:ring-brand-400"
                                    @change="toggleColumnVisibility(choice)"
                                />
                                <span class="text-[12px] text-text-1">{{
                                    choice
                                }}</span>
                            </label>
                        </PanelCard>
                    </Flapout>
                </template>
                <template #fields>
                    <CardFieldsFlapout
                        :columns="nonLaneColumns"
                        :is-card-field="isCardField"
                        :get-field-config="getFieldConfig"
                        :is-truncatable="isTruncatable"
                        @toggle-field="toggleCardField"
                        @move-field="moveFieldInList"
                        @update-config="updateFieldConfig"
                        @ensure-field="ensureCardField"
                    />
                </template>
            </TableToolbar>

            <!-- Board — single scroll container for both axes -->
            <div class="flex-1 min-h-0 overflow-auto">
                <!-- Unified render: swimlane groups when active, single group otherwise -->
                <div v-for="swim in displayGroups" :key="swim.key">
                    <!-- Swimlane header (only when Group is set) -->
                    <div
                        v-if="swimlaneGroups"
                        class="sticky left-0 flex items-center gap-2 px-4 h-8 bg-zinc-200/80 dark:bg-[#252525] border-b border-border-1 shrink-0"
                    >
                        <span
                            v-if="swim.value != null"
                            class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium truncate max-w-44"
                            :style="swim.pillStyle"
                            >{{ swim.label }}</span
                        >
                        <span
                            v-else
                            class="text-[12px] font-medium text-text-3 italic"
                            >No value</span
                        >
                        <span class="text-[11px] text-text-3 ml-1">{{
                            swim.total ?? swim.rows.length
                        }}</span>
                    </div>

                    <!-- Lane row for this group -->
                    <div
                        class="flex gap-3 items-start"
                        :class="swimlaneGroups ? 'p-3' : 'p-4'"
                    >
                        <div
                            v-for="lane in buildLanesForRows(swim.rows)"
                            :key="lane.key"
                            class="flex flex-col w-64 shrink-0"
                        >
                            <!-- Lane header -->
                            <div class="flex items-center gap-2 mb-2 px-1 h-7">
                                <span
                                    v-if="lane.value != null"
                                    class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium truncate max-w-44"
                                    :style="lane.pillStyle"
                                    >{{ lane.label }}</span
                                >
                                <span
                                    v-else
                                    class="text-[12px] font-medium text-text-3"
                                    >No value</span
                                >
                                <span
                                    class="text-[11px] text-text-3 ml-auto shrink-0"
                                    >{{ lane.rows.length }}</span
                                >
                                <!-- Lane transition: move all to another lane -->
                                <Flapout
                                    v-if="
                                        lane.rows.length > 0 &&
                                        laneChoices.length > 1
                                    "
                                    align="right"
                                >
                                    <template #trigger="{ toggle, setAnchor }">
                                        <button
                                            :ref="setAnchor"
                                            class="flex items-center justify-center w-5 h-5 rounded bg-transparent border-none text-text-3 cursor-pointer hover:text-text-1 hover:bg-zinc-200 dark:hover:bg-[#3c3c3c] transition-colors shrink-0"
                                            :disabled="
                                                laneTransition?.fromValue ===
                                                lane.value
                                            "
                                            title="Move all to…"
                                            @click.stop="toggle"
                                        >
                                            <RiMoreLine size="13" />
                                        </button>
                                    </template>
                                    <template #default="{ close }">
                                        <div
                                            class="bg-surface-1 border border-border-1 rounded-lg shadow-xl py-1 w-44"
                                        >
                                            <div
                                                class="px-3 py-1 text-[10px] font-semibold text-text-3 uppercase tracking-wide"
                                            >
                                                Move all to
                                            </div>
                                            <button
                                                v-for="choice in laneChoices.filter(
                                                    (c) => c !== lane.value,
                                                )"
                                                :key="choice"
                                                class="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-1 hover:bg-border-1 transition-colors cursor-pointer border-none bg-transparent"
                                                @click="
                                                    moveLane(
                                                        lane.value,
                                                        choice,
                                                    );
                                                    close();
                                                "
                                            >
                                                <RiArrowRightLine
                                                    size="13"
                                                    class="shrink-0 text-text-3"
                                                />
                                                {{ choice }}
                                            </button>
                                        </div>
                                    </template>
                                </Flapout>
                            </div>

                            <!-- Drop zone -->
                            <div
                                class="flex flex-col gap-2 min-h-16 rounded-lg p-1.5 transition-all"
                                :class="
                                    dragOverLane === dropKey(swim.key, lane.key)
                                        ? 'bg-brand-50 dark:bg-brand-900/20 ring-1 ring-inset ring-brand-300 dark:ring-brand-700/60'
                                        : 'bg-zinc-200/60 dark:bg-[#2a2a2a]'
                                "
                                @dragover.prevent="
                                    dragOverLane = dropKey(swim.key, lane.key)
                                "
                                @dragleave="
                                    onDragLeave($event, swim.key, lane.key)
                                "
                                @drop.prevent="
                                    onDrop($event, lane.value, swim.value)
                                "
                            >
                                <div
                                    v-for="row in lane.rows"
                                    :key="row.id"
                                    draggable="true"
                                    class="bg-surface-1 rounded-lg border border-border-1 cursor-pointer shadow-sm hover:shadow-md hover:border-zinc-300 dark:hover:border-[#555] transition-all overflow-hidden select-none p-2.5 shrink-0"
                                    :class="[
                                        draggingRowId === row.id
                                            ? 'opacity-30'
                                            : '',
                                    ]"
                                    @dragstart="onDragStart($event, row)"
                                    @dragend="onDragEnd"
                                    @click="$emit('open-row', { row })"
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
                            </div>

                            <!-- Add record to lane -->
                            <button
                                class="mt-1 flex items-center gap-1 px-2 py-1.5 text-[12px] text-text-3 hover:text-text-1 bg-transparent border-none cursor-pointer rounded-md hover:bg-zinc-200/70 dark:hover:bg-[#3c3c3c] transition-colors"
                                @click="addRecord(lane.value, swim.value)"
                            >
                                <RiAddLine size="13" />
                                Add record
                            </button>
                        </div>
                    </div>
                </div>

                <!-- Per-swimlane load more (grouped mode) -->
                <template v-if="localGroupByCol">
                    <div
                        v-for="swim in displayGroups.filter(s => s.hasMore)"
                        :key="'lm-' + swim.key"
                        class="flex justify-center px-4 py-2 border-t border-border-1"
                    >
                        <button
                            class="inline-flex items-center gap-1.5 px-4 py-1.5 text-[12px] font-medium text-text-2 bg-surface-1 border border-border-1 rounded-md cursor-pointer hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#555] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                            :disabled="swim.loadingMore"
                            @click="loadMoreGroup(swim.value)"
                        >
                            <span v-if="swim.loadingMore">Loading…</span>
                            <span v-else>Load more {{ swim.label }} ({{ swim.rows.length }} / {{ swim.total }})</span>
                        </button>
                    </div>
                </template>

                <!-- Global load more (flat / no-swimlane mode) -->
                <div
                    v-else-if="hasMore"
                    class="flex justify-center px-4 py-2 border-t border-border-1"
                >
                    <button
                        class="inline-flex items-center gap-1.5 px-4 py-1.5 text-[12px] font-medium text-text-2 bg-surface-1 border border-border-1 rounded-md cursor-pointer hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#555] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        :disabled="loadingMore"
                        @click="loadMore"
                    >
                        <span v-if="loadingMore">Loading…</span>
                        <span v-else>Load more ({{ rows.length }} / {{ total }})</span>
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
import TableToolbar from "./TableToolbar.vue";
import CardCellRenderer from "../records/CardCellRenderer.vue";
import BulkActionDialog from "./BulkActionDialog.vue";
import ImportCsvDialog from "./ImportCsvDialog.vue";
import {
    RiKanbanView,
    RiArrowRightLine,
    RiMoreLine,
    RiAddLine,
} from "@remixicon/vue";
import Flapout from "../../foundation/Flapout.vue";
import PanelCard from "../../foundation/PanelCard.vue";
import CardFieldsFlapout from "./CardFieldsFlapout.vue";
import EmptyState from "../../foundation/EmptyState.vue";
import { useViewRows } from "../records/useViewRows.js";
import { useGroupedRows } from "../records/useGroupedRows.js";
import { useSortedColumns } from "../records/useSortedColumns.js";
import { useFieldReordering } from "../records/useFieldReordering.js";
import { useCardFields } from "./useCardFields.js";
import { useNotifications } from "../../foundation/useNotifications.js";

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
    "group-by-change",
    "card-fields-change",
    "column-order-change",
    "hidden-kanban-columns-change",
]);
const { notify } = useNotifications();
const showBulkDialog = ref(false);
const showImportDialog = ref(false);
const laneTransition = ref(null);

const draggingRowId = ref(null);
const dragOverLane = ref(null); // composite key: "${swimKey}::${laneKey}"

// ── Columns ───────────────────────────────────────────────────────────────────

const sortedColumns = useSortedColumns(
    computed(() => props.table.columns ?? []),
    computed(() => props.viewConfig?.columnOrder),
);

const selectColumns = computed(() =>
    sortedColumns.value.filter((c) => c.type === "single-select"),
);

// Lane column (xCol) — determines vertical kanban columns
const localLaneCode = ref(
    props.viewConfig?.xCol ?? selectColumns.value[0]?.code ?? null,
);

watch(
    () => props.viewConfig?.xCol,
    (code) => {
        localLaneCode.value = code ?? selectColumns.value[0]?.code ?? null;
    },
);

watch(selectColumns, (cols) => {
    if (localLaneCode.value == null && cols.length > 0) {
        localLaneCode.value = cols[0].code;
    }
});

const laneCol = computed(
    () =>
        props.table.columns?.find((c) => c.code === localLaneCode.value) ??
        null,
);

const laneChoices = computed(() => {
    if (!laneCol.value) return [];
    const opts =
        typeof laneCol.value.options === "string"
            ? JSON.parse(laneCol.value.options)
            : laneCol.value.options;
    return opts?.choices ?? [];
});

// ── Hidden kanban columns ───────────────────────────────────────────────────

const hiddenColumnSet = ref(
    new Set(props.viewConfig?.hiddenKanbanColumns ?? []),
);

watch(
    () => props.viewConfig?.hiddenKanbanColumns,
    (v) => {
        hiddenColumnSet.value = new Set(v ?? []);
    },
);

function toggleColumnVisibility(choice) {
    const next = new Set(hiddenColumnSet.value);
    if (next.has(choice)) {
        next.delete(choice);
    } else {
        next.add(choice);
    }
    hiddenColumnSet.value = next;
    emit("hidden-kanban-columns-change", [...next].sort());
}

// ── Swimlane column (groupByCol) — must be declared before useViewRows/useGroupedRows
const localGroupByCol = ref(props.viewConfig?.groupByCol ?? "");

watch(
    () => props.viewConfig?.groupByCol,
    (code) => {
        localGroupByCol.value = code ?? "";
    },
);

// ── Row data (with hidden-column server-side filtering) ────────────────────

function hiddenColFilters() {
    if (!laneCol.value || hiddenColumnSet.value.size === 0) return [];
    return [...hiddenColumnSet.value].map((value) => ({
        col: laneCol.value.code,
        op: "is_not",
        value,
    }));
}

// Flat rows — used when no swimlane is active. Skipped when swimlane is set.
const { rows, total, hasMore, loadingMore, loadMore, searchDebounced } =
    useViewRows(
        () => props,
        () => hiddenColFilters(),
        () => !!localGroupByCol.value,
    );

// Grouped rows — used when swimlane (groupByCol) is active.
const { groups, groupHasMore, loadMoreGroup } = useGroupedRows(
    () => props.workspaceCode,
    () => props.tableCode,
    () => localGroupByCol.value || null,
    () => [...(props.viewConfig?.filters ?? []), ...hiddenColFilters()],
    () => props.viewConfig?.sort ?? [],
    () => searchDebounced.value,
);

// ── Swimlane column (continued)

const swimlaneCol = computed(() =>
    localGroupByCol.value
        ? (props.table.columns?.find((c) => c.code === localGroupByCol.value) ??
          null)
        : null,
);

const swimlaneChoices = computed(() => {
    if (!swimlaneCol.value) return [];
    const opts =
        typeof swimlaneCol.value.options === "string"
            ? JSON.parse(swimlaneCol.value.options)
            : swimlaneCol.value.options;
    return opts?.choices ?? [];
});

// Columns available as Group (swimlane) — excludes the current lane column
const groupableColumns = computed(() =>
    selectColumns.value.filter((c) => c.code !== localLaneCode.value),
);

// Columns shown in card fields flapout — excludes lane and swimlane columns
const nonLaneColumns = computed(() =>
    sortedColumns.value.filter(
        (c) =>
            c.code !== localLaneCode.value &&
            c.code !== (localGroupByCol.value || null),
    ),
);

// ── Card fields ───────────────────────────────────────────────────────────────

const {
    getFieldConfig,
    isCardField,
    ensureCardField,
    toggleCardField,
    updateFieldConfig,
    isTruncatable,
    visibleCardFields,
} = useCardFields(
    computed(() => props.viewConfig),
    nonLaneColumns,
    emit,
);

const { moveFieldInList } = useFieldReordering(
    sortedColumns,
    computed(
        () =>
            props.viewConfig?.columnOrder ??
            sortedColumns.value.map((c) => c.code),
    ),
    emit,
);

// ── Lane building ─────────────────────────────────────────────────────────────

function buildLanesForRows(rowSubset) {
    const col = laneCol.value;
    if (!col) return [];

    const colCode = col.code;
    const choices = laneChoices.value;
    const knownChoices = new Set(choices);

    const lanes = choices.map((choice) => {
        const bg = getChoiceColor(col, choice);
        return {
            key: choice,
            value: choice,
            label: choice,
            pillStyle: { backgroundColor: bg, color: textColorForBg(bg) },
            rows: rowSubset.filter((r) => r.data?.[colCode] === choice),
        };
    });

    const noValueRows = rowSubset.filter((r) => {
        const v = r.data?.[colCode];
        return v == null || v === "" || !knownChoices.has(v);
    });
    if (noValueRows.length > 0 || lanes.length === 0) {
        lanes.push({
            key: "__none__",
            value: null,
            label: "No value",
            pillStyle: {},
            rows: noValueRows,
        });
    }

    // Filter out hidden columns (always keep "No value")
    return lanes.filter(
        (l) => l.value == null || !hiddenColumnSet.value.has(l.value),
    );
}

// Unified display groups: API groups when swimlane active, single flat group otherwise.
const displayGroups = computed(() => {
    if (localGroupByCol.value && groups.value.length > 0) {
        const col = swimlaneCol.value;
        return groups.value.map((g) => {
            const bg = g.value != null && col ? getChoiceColor(col, g.value) : null;
            return {
                key: g.value ?? "__none__",
                value: g.value,
                label: g.value ?? "No value",
                pillStyle: bg ? { backgroundColor: bg, color: textColorForBg(bg) } : {},
                rows: g.rows,
                total: g.total,
                hasMore: groupHasMore(g),
                loadingMore: g.loadingMore,
            };
        });
    }
    return [{ key: "__flat__", value: null, label: null, pillStyle: {}, rows: rows.value, total: total.value }];
});

// ── Drag and drop ─────────────────────────────────────────────────────────────

function dropKey(swimKey, laneKey) {
    return `${swimKey}::${laneKey}`;
}

function onDragStart(event, row) {
    draggingRowId.value = row.id;
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", String(row.id));
}

function onDragEnd() {
    draggingRowId.value = null;
    dragOverLane.value = null;
}

function onDragLeave(event, swimKey, laneKey) {
    if (!event.currentTarget.contains(event.relatedTarget)) {
        if (dragOverLane.value === dropKey(swimKey, laneKey))
            dragOverLane.value = null;
    }
}

async function onDrop(event, newLaneValue, swimValue) {
    dragOverLane.value = null;
    const rowId = Number(event.dataTransfer.getData("text/plain"));
    if (!rowId) return;
    const row = rows.value.find((r) => r.id === rowId);
    if (!row) return;
    draggingRowId.value = null;

    const col = laneCol.value;
    if (!col) return;

    const colCode = col.code;
    const prevLaneVal = row.data?.[colCode] ?? null;
    const newData = { ...(row.data ?? {}), [colCode]: newLaneValue };

    // Also update swimlane column when dropped into a different swimlane
    if (swimlaneCol.value) {
        const swimCode = swimlaneCol.value.code;
        const prevSwimVal = row.data?.[swimCode] ?? null;
        if (swimValue !== prevSwimVal) newData[swimCode] = swimValue;
    }

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
        row.data = { ...(row.data ?? {}), [colCode]: prevLaneVal };
    }
}

// ── Add record ────────────────────────────────────────────────────────────────

async function addRecord(laneValue, swimValue) {
    const col = laneCol.value;
    const data = {};
    if (col && laneValue != null) data[col.code] = laneValue;
    if (swimlaneCol.value && swimValue != null)
        data[swimlaneCol.value.code] = swimValue;
    const row = await api.createRow(props.workspaceCode, props.tableCode, {
        data,
    });
    rows.value.push(row);
    emit("open-row", { row, isNew: true });
}

// ── Bulk actions ──────────────────────────────────────────────────────────────

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

async function moveLane(fromValue, toValue) {
    const col = laneCol.value;
    if (!col) return;
    laneTransition.value = { fromValue, loading: true };
    try {
        const filters =
            fromValue != null
                ? [
                      {
                          id: crypto.randomUUID(),
                          col: col.code,
                          op: "is",
                          value: fromValue,
                      },
                  ]
                : [
                      {
                          id: crypto.randomUUID(),
                          col: col.code,
                          op: "is_empty",
                          value: "",
                      },
                  ];
        const res = await api.bulkPatch(
            props.workspaceCode,
            props.tableCode,
            filters,
            { [col.code]: toValue },
        );
        notify(
            `${res.updated} record${res.updated !== 1 ? "s" : ""} moved to "${toValue}"`,
        );
        const colCode = col.code;
        for (const row of rows.value) {
            const val = row.data?.[colCode];
            const matches =
                fromValue != null
                    ? val === fromValue
                    : val == null || val === "";
            if (matches) row.data = { ...row.data, [colCode]: toValue };
        }
    } catch (e) {
        notify(e.message || "Bulk move failed");
    } finally {
        laneTransition.value = null;
    }
}

// ── Config changes ────────────────────────────────────────────────────────────

function onLaneChange() {
    emit("x-col-change", localLaneCode.value);
}

function onGroupByChange() {
    emit("group-by-change", localGroupByCol.value || null);
}
</script>
