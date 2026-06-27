<template>
    <div
        class="flex flex-col flex-1 min-h-0 overflow-hidden bg-zinc-100 dark:bg-[#1e1e1e]"
    >
        <!-- Sub-toolbar -->
        <TableToolbar
            :table="table"
            :view-config="viewConfig"
            @update:search="searchDebounced = $event"
            @sort-change="emit('sort-change', $event)"
            @filter-change="emit('filter-change', $event)"
            @add-record="addRecord"
            @bulk-action="showBulkDialog = true"
            @export-csv="exportCsv"
            @import-csv="showImportDialog = true"
        >
            <template v-if="selectColumns.length > 0" #before>
                <span class="text-[11px] text-text-2">Swimlanes</span>
                <select
                    v-model="localGroupByCol"
                    class="h-6 px-2 text-[11px] rounded border border-border-1 bg-surface-2 text-text-1 outline-none cursor-pointer"
                    @change="onGroupByChange"
                >
                    <option value="">None</option>
                    <option v-for="col in selectColumns" :key="col.code" :value="col.code">{{ col.name }}</option>
                </select>
            </template>
            <template #fields>
                <CardFieldsFlapout
                    :columns="selectableColumns"
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

        <!-- Empty state -->
        <EmptyState v-if="rows.length === 0" message="No records yet.">
            <template #icon
                ><RiLayoutGridLine
                    size="36"
                    class="text-zinc-300 dark:text-[#454545]"
            /></template>
            <template #action>
                <button
                    class="mt-1 inline-flex items-center gap-1 px-3 py-1.5 text-[12px] font-medium text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-700/50 rounded-md cursor-pointer hover:bg-brand-100 dark:hover:bg-brand-900/50 transition-colors"
                    @click="addRecord"
                >
                    <RiAddLine size="13" />
                    Add record
                </button>
            </template>
        </EmptyState>

        <!-- Card grid (with optional swimlane sections) -->
        <div v-else class="flex-1 overflow-y-auto p-4">
            <template v-for="swim in displayGroups" :key="swim.key">
                <!-- Swimlane header -->
                <div
                    v-if="localGroupByCol"
                    class="flex items-center gap-2 mb-3 mt-4 first:mt-0 pb-2 border-b border-border-1"
                >
                    <span
                        v-if="swim.value != null"
                        class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-semibold"
                        :style="swim.pillStyle"
                    >{{ swim.label }}</span>
                    <span v-else class="text-[12px] font-semibold text-text-3 italic">No value</span>
                    <span class="text-[11px] text-text-3">{{ swim.total ?? swim.rows.length }}</span>
                </div>

                <div
                    class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 items-start gap-3"
                    :class="localGroupByCol ? 'mb-2' : ''"
                >
                    <div
                        v-for="row in swim.rows"
                        :key="row.id"
                        class="bg-surface-1 rounded-lg border border-border-1 cursor-pointer shadow-sm hover:shadow-md hover:border-zinc-300 dark:hover:border-[#555] transition-all overflow-hidden p-3 relative"
                        @click="$emit('open-row', { row })"
                    >
                        <template
                            v-for="(field, idx) in visibleCardFields"
                            :key="field.col.id"
                        >
                            <div
                                v-if="hasValue(row, field.col)"
                                :class="
                                    field.col.type === 'image'
                                        ? 'mb-2 first:-mt-3 last:-mb-3'
                                        : 'mb-2 last:mb-0'
                                "
                            >
                                <CardCellRenderer
                                    :row="row"
                                    :field="field"
                                    :workspace-code="workspaceCode"
                                    :table-code="tableCode"
                                />
                            </div>
                            <!-- Floating copy-link over the first displayed field -->
                            <button
                                v-if="idx === 0"
                                class="absolute top-2 right-2 z-10 flex items-center justify-center w-6 h-6 rounded-full bg-white/75 dark:bg-black/45 backdrop-blur-sm border-none text-zinc-400 dark:text-zinc-300 cursor-pointer hover:text-brand-500 hover:bg-white dark:hover:bg-black/65 transition-colors"
                                title="Copy link"
                                @click.stop="copyRowLink(row)"
                            >
                                <RiLink size="13" />
                            </button>
                        </template>
                        <div
                            v-if="
                                !visibleCardFields.some((f) => hasValue(row, f.col))
                            "
                            class="text-[11px] text-zinc-300 dark:text-[#555] italic"
                        >
                            Empty
                        </div>
                    </div>
                </div>
            </template>

            <!-- Per-swimlane load more (grouped mode) -->
            <template v-if="localGroupByCol">
                <div
                    v-for="swim in displayGroups.filter(s => s.hasMore)"
                    :key="'lm-' + swim.key"
                    class="flex justify-center mt-2 mb-2"
                >
                    <button
                        class="inline-flex items-center gap-1.5 px-4 py-2 text-[12px] font-medium text-text-2 bg-surface-1 border border-border-1 rounded-md cursor-pointer hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#555] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                        :disabled="swim.loadingMore"
                        @click="loadMoreGroup(swim.value)"
                    >
                        <span v-if="swim.loadingMore">Loading…</span>
                        <span v-else>Load more {{ swim.label }} ({{ swim.rows.length }} / {{ swim.total }})</span>
                    </button>
                </div>
            </template>

            <!-- Global load more (flat / no-swimlane mode) -->
            <div v-else-if="hasMore" class="flex justify-center mt-4">
                <button
                    class="inline-flex items-center gap-1.5 px-4 py-2 text-[12px] font-medium text-text-2 bg-surface-1 border border-border-1 rounded-md cursor-pointer hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#555] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    :disabled="loadingMore"
                    @click="loadMore"
                >
                    <span v-if="loadingMore">Loading…</span>
                    <span v-else>Load more ({{ rows.length }} / {{ total }})</span>
                </button>
            </div>
        </div>

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
import { hasValue } from "../records/cellHelpers.js";
import { getChoiceColor, textColorForBg } from "./palettes.js";
import TableToolbar from "./TableToolbar.vue";
import BulkActionDialog from "./BulkActionDialog.vue";
import ImportCsvDialog from "./ImportCsvDialog.vue";
import CardCellRenderer from "../records/CardCellRenderer.vue";
import { RiAddLine, RiLayoutGridLine, RiLink } from "@remixicon/vue";
import EmptyState from "../../foundation/EmptyState.vue";
import CardFieldsFlapout from "./CardFieldsFlapout.vue";
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
    "group-by-change",
    "card-fields-change",
    "column-order-change",
]);

const { notify } = useNotifications();
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

// ── Swimlane column (groupByCol) — must be declared before useViewRows/useGroupedRows
const localGroupByCol = ref(props.viewConfig?.groupByCol ?? "");

watch(() => props.viewConfig?.groupByCol, (code) => {
    localGroupByCol.value = code ?? "";
});

// Flat rows — used when no swimlane is active. Skipped when swimlane is set.
const { rows, total, hasMore, loadingMore, loadMore, searchDebounced } =
    useViewRows(() => props, () => [], () => !!localGroupByCol.value);

// Grouped rows — used when swimlane (groupByCol) is active.
const { groups, groupHasMore, loadMoreGroup } = useGroupedRows(
    () => props.workspaceCode,
    () => props.tableCode,
    () => localGroupByCol.value || null,
    () => props.viewConfig?.filters ?? [],
    () => props.viewConfig?.sort ?? [],
    () => searchDebounced.value,
);

const sortedColumns = useSortedColumns(
    computed(() => props.table.columns ?? []),
    computed(() => props.viewConfig?.columnOrder),
);

const selectableColumns = computed(() => sortedColumns.value);

const selectColumns = computed(() =>
    sortedColumns.value.filter(c => c.type === "single-select")
);

// ── Swimlane (Group) ──────────────────────────────────────────────────────────

const swimlaneCol = computed(() =>
    localGroupByCol.value
        ? (props.table.columns?.find(c => c.code === localGroupByCol.value) ?? null)
        : null
);

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

function onGroupByChange() {
    emit("group-by-change", localGroupByCol.value || null);
}

// ── Card fields ───────────────────────────────────────────────────────────────

const { moveFieldInList } = useFieldReordering(
    selectableColumns,
    computed(
        () =>
            props.viewConfig?.columnOrder ??
            selectableColumns.value.map((c) => c.code),
    ),
    emit,
);

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
    selectableColumns,
    emit,
);

function copyRowLink(row) {
    const url = `${window.location.origin}/workspaces/${props.workspaceCode}/tables/${props.tableCode}/rows/${row.id}`;
    navigator.clipboard.writeText(url).then(() => notify("Link copied"));
}

async function addRecord() {
    const row = await api.createRow(props.workspaceCode, props.tableCode, {
        data: {},
    });
    rows.value.push(row);
    emit("open-row", { row, isNew: true });
}
</script>
