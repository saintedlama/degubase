<template>
    <div
        class="flex flex-col flex-1 min-h-0 overflow-hidden bg-zinc-100 dark:bg-[#1e1e1e]"
    >
        <div v-if="loading" class="p-6 text-sm text-text-2">Loading…</div>
        <NotFoundView v-else-if="notFound" />
        <template v-else>
            <!-- Toolbar -->
            <div
                class="flex items-center gap-3 px-5 h-13 bg-surface-1 border-b border-border-1 shrink-0"
            >
                <h2 class="text-[15px] font-bold text-text-1 whitespace-nowrap">
                    {{ table.name }}
                </h2>

                <!-- View selector (right of table name) -->
                <Flapout align="left">
                    <template #trigger="{ toggle, setAnchor }">
                        <button
                            :ref="setAnchor"
                            data-testid="view-switcher-btn"
                            class="inline-flex items-center gap-1 h-7 px-2 text-[12px] font-medium text-text-1 bg-transparent border border-border-1 rounded cursor-pointer hover:bg-surface-2 transition-colors shrink-0"
                            @click.stop="toggle"
                        >
                            <RiTableView
                                v-if="activeView?.type === 'tabular'"
                                size="12"
                            />
                            <RiLayoutGridLine
                                v-else-if="activeView?.type === 'card'"
                                size="12"
                            />
                            <RiGridLine
                                v-else-if="activeView?.type === 'matrix'"
                                size="12"
                            />
                            <RiCalendarLine
                                v-else-if="activeView?.type === 'timeline'"
                                size="12"
                            />
                            <RiKanbanView v-else size="12" />
                            {{ activeView?.name ?? "View" }}
                            <RiArrowDownSLine size="13" class="text-text-3" />
                        </button>
                    </template>
                    <template #default="{ close }">
                        <div
                            class="bg-surface-1 border border-border-1 rounded-lg shadow-xl py-1.5 w-52"
                        >
                            <div
                                v-for="v in table.views"
                                :key="v.code"
                                class="flex items-center gap-0.5 px-1.5 group/view"
                            >
                                <!-- Inline rename input -->
                                <div
                                    v-if="renamingViewCode === v.code"
                                    class="flex-1 flex items-center gap-2 px-2 py-1"
                                >
                                    <RiTableView
                                        v-if="v.type === 'tabular'"
                                        size="13"
                                        class="shrink-0 text-text-3"
                                    />
                                    <RiLayoutGridLine
                                        v-else-if="v.type === 'card'"
                                        size="13"
                                        class="shrink-0 text-text-3"
                                    />
                                    <RiGridLine
                                        v-else-if="v.type === 'matrix'"
                                        size="13"
                                        class="shrink-0 text-text-3"
                                    />
                                    <RiCalendarLine
                                        v-else-if="v.type === 'timeline'"
                                        size="13"
                                        class="shrink-0 text-text-3"
                                    />
                                    <RiKanbanView
                                        v-else
                                        size="13"
                                        class="shrink-0 text-text-3"
                                    />
                                    <input
                                        :ref="
                                            (el) => {
                                                renameInputEl = el;
                                            }
                                        "
                                        v-model="renameValue"
                                        type="text"
                                        class="flex-1 min-w-0 text-[12px] bg-white dark:bg-[#2d2d30] border border-brand-400 dark:border-brand-600 rounded px-1 py-0 h-5 text-text-1 outline-none"
                                        @keydown.enter.prevent="commitRename(v)"
                                        @keydown.escape.stop="cancelRename()"
                                        @blur="commitRename(v)"
                                        @click.stop
                                    />
                                </div>
                                <!-- Normal view button -->
                                <button
                                    v-else
                                    class="flex-1 flex items-center gap-2 text-left text-[12px] px-2 py-1.5 rounded transition-colors cursor-pointer border-none bg-transparent"
                                    :class="
                                        v.code === activeViewCode
                                            ? 'text-brand-600 dark:text-brand-400 font-medium bg-brand-50 dark:bg-brand-900/20'
                                            : 'text-text-1 hover:bg-border-1'
                                    "
                                    @click="
                                        switchView(v);
                                        close();
                                    "
                                >
                                    <RiTableView
                                        v-if="v.type === 'tabular'"
                                        size="13"
                                        class="shrink-0"
                                    />
                                    <RiLayoutGridLine
                                        v-else-if="v.type === 'card'"
                                        size="13"
                                        class="shrink-0"
                                    />
                                    <RiGridLine
                                        v-else-if="v.type === 'matrix'"
                                        size="13"
                                        class="shrink-0"
                                    />
                                    <RiCalendarLine
                                        v-else-if="v.type === 'timeline'"
                                        size="13"
                                        class="shrink-0"
                                    />
                                    <RiKanbanView
                                        v-else
                                        size="13"
                                        class="shrink-0"
                                    />
                                    {{ v.name }}
                                    <RiStarFill
                                        v-if="v.id === table.default_view_id"
                                        size="11"
                                        class="shrink-0 text-brand-400 ml-auto"
                                        title="Default view"
                                    />
                                </button>
                                <!-- Action buttons (hidden while renaming) -->
                                <template v-if="renamingViewCode !== v.code">
                                    <button
                                        class="w-5 h-5 flex items-center justify-center text-zinc-300 dark:text-[#555] hover:text-zinc-500 dark:hover:text-[#9d9d9d] hover:bg-border-1 rounded opacity-0 group-hover/view:opacity-100 [@media(hover:none)]:opacity-100 transition-all border-none bg-transparent cursor-pointer"
                                        title="Rename view"
                                        @click.stop="startRename(v)"
                                    >
                                        <RiPencilLine size="11" />
                                    </button>
                                    <template
                                        v-if="v.id !== table.default_view_id"
                                    >
                                        <button
                                            class="w-5 h-5 flex items-center justify-center text-zinc-300 dark:text-[#555] hover:text-brand-400 hover:bg-brand-50 dark:hover:bg-brand-900/30 rounded opacity-0 group-hover/view:opacity-100 [@media(hover:none)]:opacity-100 transition-all border-none bg-transparent cursor-pointer"
                                            title="Set as default"
                                            @click.stop="setAsDefault(v)"
                                        >
                                            <RiStarLine size="12" />
                                        </button>
                                        <button
                                            class="w-5 h-5 flex items-center justify-center text-zinc-300 dark:text-[#555] hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/40 rounded opacity-0 group-hover/view:opacity-100 [@media(hover:none)]:opacity-100 transition-all border-none bg-transparent cursor-pointer"
                                            title="Delete view"
                                            @click.stop="deleteView(v)"
                                        >
                                            <RiDeleteBin6Line size="11" />
                                        </button>
                                    </template>
                                </template>
                            </div>
                            <div
                                class="border-t border-zinc-100 dark:border-[#3c3c3c] mt-1 pt-1 mx-1.5 px-1 pb-1"
                            >
                                <div
                                    class="text-[10px] font-semibold text-text-3 px-1.5 py-1 uppercase tracking-wide"
                                >
                                    New view
                                </div>
                                <button
                                    data-testid="new-view-btn-tabular"
                                    class="w-full flex items-center gap-2 px-2 py-1.5 rounded text-[12px] text-text-1 hover:bg-border-1 border-none bg-transparent cursor-pointer transition-colors"
                                    @click="openCreateDialog('tabular', close)"
                                >
                                    <RiTableView
                                        size="13"
                                        class="shrink-0 text-text-2"
                                    />Table
                                </button>
                                <button
                                    class="w-full flex items-center gap-2 px-2 py-1.5 rounded text-[12px] text-text-1 hover:bg-border-1 border-none bg-transparent cursor-pointer transition-colors"
                                    @click="openCreateDialog('card', close)"
                                >
                                    <RiLayoutGridLine
                                        size="13"
                                        class="shrink-0 text-text-2"
                                    />Cards
                                </button>
                                <button
                                    class="w-full flex items-center gap-2 px-2 py-1.5 rounded text-[12px] transition-colors border-none bg-transparent"
                                    :class="
                                        hasSelectCols
                                            ? 'cursor-pointer text-text-1 hover:bg-border-1'
                                            : 'opacity-40 cursor-not-allowed text-text-3'
                                    "
                                    :disabled="!hasSelectCols"
                                    @click="
                                        hasSelectCols &&
                                        openCreateDialog('kanban', close)
                                    "
                                >
                                    <RiKanbanView
                                        size="13"
                                        class="shrink-0 text-text-2"
                                    />
                                    Kanban
                                    <span
                                        v-if="!hasSelectCols"
                                        class="ml-auto text-[10px] text-zinc-300 dark:text-[#454545]"
                                        >needs select</span
                                    >
                                </button>
                                <button
                                    class="w-full flex items-center gap-2 px-2 py-1.5 rounded text-[12px] transition-colors border-none bg-transparent"
                                    :class="
                                        hasTwoSelectCols
                                            ? 'cursor-pointer text-text-1 hover:bg-border-1'
                                            : 'opacity-40 cursor-not-allowed text-text-3'
                                    "
                                    :disabled="!hasTwoSelectCols"
                                    @click="
                                        hasTwoSelectCols &&
                                        openCreateDialog('matrix', close)
                                    "
                                >
                                    <RiGridLine
                                        size="13"
                                        class="shrink-0 text-text-2"
                                    />
                                    Matrix
                                    <span
                                        v-if="!hasTwoSelectCols"
                                        class="ml-auto text-[10px] text-zinc-300 dark:text-[#454545]"
                                        >needs 2 selects</span
                                    >
                                </button>
                                <button
                                    class="w-full flex items-center gap-2 px-2 py-1.5 rounded text-[12px] text-text-1 hover:bg-border-1 border-none bg-transparent cursor-pointer transition-colors"
                                    @click="openCreateDialog('timeline', close)"
                                >
                                    <RiCalendarLine
                                        size="13"
                                        class="shrink-0 text-text-2"
                                    />Timeline
                                </button>
                            </div>
                        </div>
                    </template>
                </Flapout>

                <!-- Save view button (only shown when view config differs from saved state) -->
                <Flapout v-if="!showColumns && hasModifiedConfig" align="left">
                    <template #trigger="{ toggle, setAnchor }">
                        <button
                            :ref="setAnchor"
                            class="inline-flex items-center gap-1 h-6 px-2 text-[11px] font-medium text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-700/50 rounded cursor-pointer hover:bg-brand-100 dark:hover:bg-brand-900/50 transition-colors"
                            @click.stop="toggle"
                        >
                            Save view
                            <RiArrowDownSLine size="11" class="ml-0.5" />
                        </button>
                    </template>
                    <template #default="{ close }">
                        <div
                            class="bg-surface-1 border border-border-1 rounded-lg shadow-xl py-1 w-52"
                        >
                            <button
                                class="w-full text-left px-3 py-1.5 text-[12px] text-text-1 hover:bg-border-1 transition-colors cursor-pointer border-none bg-transparent"
                                @click="
                                    saveToCurrentView();
                                    close();
                                "
                            >
                                Update "{{ activeView?.name }}"
                            </button>
                            <button
                                class="w-full text-left px-3 py-1.5 text-[12px] text-text-1 hover:bg-border-1 transition-colors cursor-pointer border-none bg-transparent"
                                @click="
                                    showSaveAsDialog = true;
                                    close();
                                "
                            >
                                Save as new view…
                            </button>
                        </div>
                    </template>
                </Flapout>

                <span
                    v-if="table.context"
                    class="text-xs text-text-2 truncate max-w-75"
                    :title="table.context"
                    >{{ table.context }}</span
                >
                <div class="flex items-center gap-1 ml-auto">
                    <button
                        class="flex items-center justify-center w-7 h-7 bg-transparent border-none cursor-pointer rounded transition-colors"
                        :class="
                            showColumns
                                ? 'text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30'
                                : 'text-text-2 hover:text-text-1 hover:bg-border-1'
                        "
                        title="Edit columns"
                        @click="showColumns = !showColumns"
                    >
                        <RiLayoutColumnLine size="14" />
                    </button>

                    <button
                        class="flex items-center justify-center w-7 h-7 bg-transparent border-none text-text-2 cursor-pointer rounded hover:text-text-1 hover:bg-border-1 transition-colors"
                        title="Edit table"
                        @click="showEditDialog = true"
                    >
                        <RiPencilLine size="14" />
                    </button>
                    <button
                        class="flex items-center justify-center w-7 h-7 bg-transparent border-none text-text-2 cursor-pointer rounded hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/40 transition-colors"
                        title="Delete table"
                        @click="deleteTable"
                    >
                        <RiDeleteBin6Line size="14" />
                    </button>
                </div>
            </div>

            <!-- Columns panel -->
            <ColumnsView
                v-if="showColumns"
                :workspace-code="route.params.workspaceCode"
                :table-code="table.code"
                @columns-changed="load"
            />

            <!-- View content -->
            <div
                v-if="!showColumns"
                class="flex flex-col flex-1 min-h-0 overflow-hidden"
            >
                <TabularView
                    v-if="activeView?.type === 'tabular'"
                    :workspace-code="route.params.workspaceCode"
                    :table-code="table.code"
                    :table="table"
                    :view-config="viewConfig"
                    @reload-table="load"
                    @open-row="onOpenRow"
                    @sort-change="onSortChange"
                    @filter-change="onFilterChange"
                    @group-by-change="onGroupByChange"
                    @hidden-columns-change="onHiddenColumnsChange"
                    @column-order-change="onColumnOrderChange"
                    @column-updated="onColumnUpdated"
                />
                <KanbanView
                    v-else-if="activeView?.type === 'kanban'"
                    :workspace-code="route.params.workspaceCode"
                    :table-code="table.code"
                    :table="table"
                    :view-config="viewConfig"
                    @open-row="onOpenRow"
                    @reload-table="load"
                    @sort-change="onSortChange"
                    @filter-change="onFilterChange"
                    @x-col-change="onXColChange"
                    @group-by-change="onGroupByChange"
                    @card-fields-change="onCardFieldsChange"
                    @column-order-change="onColumnOrderChange"
                    @hidden-kanban-columns-change="onHiddenKanbanColumnsChange"
                />
                <CardView
                    v-else-if="activeView?.type === 'card'"
                    :workspace-code="route.params.workspaceCode"
                    :table-code="table.code"
                    :table="table"
                    :view-config="viewConfig"
                    @open-row="onOpenRow"
                    @reload-table="load"
                    @sort-change="onSortChange"
                    @filter-change="onFilterChange"
                    @group-by-change="onGroupByChange"
                    @card-fields-change="onCardFieldsChange"
                    @column-order-change="onColumnOrderChange"
                />
                <MatrixView
                    v-else-if="activeView?.type === 'matrix'"
                    :workspace-code="route.params.workspaceCode"
                    :table-code="table.code"
                    :table="table"
                    :view-config="viewConfig"
                    @open-row="onOpenRow"
                    @reload-table="load"
                    @sort-change="onSortChange"
                    @filter-change="onFilterChange"
                    @x-col-change="onXColChange"
                    @y-col-change="onYColChange"
                    @axis-labels-change="onAxisLabelsChange"
                    @card-fields-change="onCardFieldsChange"
                    @column-order-change="onColumnOrderChange"
                />
                <TimelineView
                    v-else-if="activeView?.type === 'timeline'"
                    :workspace-code="route.params.workspaceCode"
                    :table-code="table.code"
                    :table="table"
                    :view-config="viewConfig"
                    @open-row="onOpenRow"
                    @reload-table="load"
                    @sort-change="onSortChange"
                    @filter-change="onFilterChange"
                    @date-col-change="onDateColChange"
                    @timeline-zoom-change="onTimelineZoomChange"
                    @card-fields-change="onCardFieldsChange"
                    @column-order-change="onColumnOrderChange"
                    @weekdays-change="onWeekdaysChange"
                    @group-by-change="onGroupByChange"
                />
            </div>
        </template>

        <!-- Edit table dialog -->
        <ModalDialog
            v-if="showEditDialog"
            title="Edit Table"
            confirm-label="Save"
            :confirm-disabled="!editForm.name.trim()"
            @confirm="saveEdit"
            @cancel="showEditDialog = false"
        >
            <div class="flex flex-col gap-1.5">
                <label class="text-xs font-semibold text-text-2 tracking-wide"
                    >Name</label
                >
                <input
                    v-model="editForm.name"
                    type="text"
                    placeholder="Table name"
                    autofocus
                    class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
                    @keyup.enter="saveEdit"
                />
            </div>
            <div class="flex flex-col gap-1.5">
                <label class="text-xs font-semibold text-text-2 tracking-wide">
                    Context
                    <span class="font-normal text-text-3">(optional)</span>
                </label>
                <textarea
                    v-model="editForm.context"
                    placeholder="Describe this table…"
                    rows="3"
                    class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all resize-y"
                />
            </div>
            <div class="flex flex-col gap-1.5">
                <label class="text-xs font-semibold text-text-2 tracking-wide">
                    Icon <span class="font-normal text-text-3">(optional)</span>
                </label>
                <IconPicker v-model="editForm.icon" />
            </div>
        </ModalDialog>

        <!-- Create view dialog -->
        <ModalDialog
            v-if="showCreateDialog"
            :title="
                'New ' +
                {
                    tabular: 'table',
                    card: 'cards',
                    kanban: 'kanban',
                    matrix: 'matrix',
                    timeline: 'timeline',
                }[pendingCreateType] +
                ' view'
            "
            confirm-label="Create view"
            :confirm-disabled="!createName.trim()"
            @confirm="createViewWithName"
            @cancel="showCreateDialog = false"
        >
            <div class="flex flex-col gap-1.5">
                <label class="text-xs font-semibold text-text-2 tracking-wide"
                    >Name</label
                >
                <input
                    v-model="createName"
                    type="text"
                    placeholder="View name"
                    autofocus
                    class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
                    @keydown.enter.prevent="createViewWithName"
                />
            </div>
        </ModalDialog>

        <!-- Save as new view dialog -->
        <ModalDialog
            v-if="showSaveAsDialog"
            title="Save as new view"
            confirm-label="Save view"
            :confirm-disabled="!saveAsName.trim()"
            @confirm="saveAsNewView"
            @cancel="showSaveAsDialog = false"
        >
            <div class="flex flex-col gap-1.5">
                <label class="text-xs font-semibold text-text-2 tracking-wide"
                    >Name</label
                >
                <input
                    v-model="saveAsName"
                    type="text"
                    placeholder="View name"
                    autofocus
                    class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
                    @keydown.enter.prevent="saveAsNewView"
                />
            </div>
        </ModalDialog>
    </div>
</template>

<script setup>
import { ref, reactive, watch, computed, nextTick, inject } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "../../api/client.js";
import Flapout from "../../foundation/Flapout.vue";
import TabularView from "../views/TabularView.vue";
import KanbanView from "../views/KanbanView.vue";
import CardView from "../views/CardView.vue";
import MatrixView from "../views/MatrixView.vue";
import TimelineView from "../views/TimelineView.vue";
import ModalDialog from "../../foundation/ModalDialog.vue";
import IconPicker from "../../foundation/IconPicker.vue";
import ColumnsView from "../schema/ColumnsView.vue";
import {
    RiPencilLine,
    RiDeleteBin6Line,
    RiLayoutColumnLine,
    RiTableView,
    RiKanbanView,
    RiLayoutGridLine,
    RiGridLine,
    RiCalendarLine,
    RiArrowDownSLine,
    RiStarLine,
    RiStarFill,
} from "@remixicon/vue";
import { useConfirm } from "../../foundation/useConfirm.js";
import NotFoundView from "../workspace/NotFoundView.vue";

const route = useRoute();
const router = useRouter();
const reloadSidebar = inject("reloadSidebar", () => {});
const { confirm } = useConfirm();

// ── Table state ───────────────────────────────────────────────────────────────

const table = ref(null);
const loading = ref(false);
const notFound = ref(false);
const showEditDialog = ref(false);
const editForm = reactive({ name: "", context: "", icon: "" });

watch(showEditDialog, (open) => {
    if (open) {
        editForm.name = table.value?.name ?? "";
        editForm.context = table.value?.context ?? "";
        editForm.icon = table.value?.icon ?? "";
    }
});
const showColumns = ref(false);

const activeViewCode = computed(() => route.params.viewCode);
const activeView = computed(
    () =>
        table.value?.views?.find((v) => v.code === activeViewCode.value) ??
        null,
);

// ── View config (sort + filter) ───────────────────────────────────────────────

const viewConfig = ref({
    sort: [],
    filters: [],
    groupByCol: null,
    cardFields: null,
    hiddenCols: null,
    columnOrder: null,
    dateCol: null,
    timelineZoom: null,
    visibleWeekdays: null,
    hiddenKanbanColumns: null,
});
const showSaveAsDialog = ref(false);

const saveAsName = ref("");

const showCreateDialog = ref(false);
const pendingCreateType = ref("tabular");
const createName = ref("");

const renamingViewCode = ref(null);
const renameValue = ref("");
const renameInputEl = ref(null);

const sortedTableColumns = computed(() =>
    [...(table.value?.columns ?? [])].sort((a, b) => a.position - b.position),
);

const selectColumns = computed(() =>
    sortedTableColumns.value.filter((c) => c.type === "single-select"),
);

const hasSelectCols = computed(() => selectColumns.value.length > 0);
const hasTwoSelectCols = computed(() => selectColumns.value.length >= 2);

function parseConfig(cfg, viewType = null) {
    if (!cfg)
        return {
            sort: [],
            filters: [],
            groupByCol: null,
            xCol: null,
            yCol: null,
            cardFields: null,
            hiddenCols: null,
            columnOrder: null,
            dateCol: null,
            timelineZoom: null,
            visibleWeekdays: null,
            hiddenKanbanColumns: null,
        };
    // Migrate old single-sort object to array
    let sort = cfg.sort ?? [];
    if (sort && !Array.isArray(sort)) sort = [sort];
    // Migrate legacy cardFields with colId → col
    let cardFields = cfg.cardFields ?? null;
    if (Array.isArray(cardFields)) {
        cardFields = cardFields.map((f) =>
            typeof f === "string"
                ? { col: f, showLabel: true, truncate: true, style: "default" }
                : f.colId && !f.col
                  ? { ...f, col: f.colId }
                  : f,
        );
    }
    // Migrate old kanban: groupByCol was used for lane columns, now xCol
    let groupByCol = cfg.groupByCol ?? null;
    let xCol = cfg.xCol ?? null;
    if (viewType === "kanban" && xCol == null && groupByCol != null) {
        xCol = groupByCol;
        groupByCol = null;
    }
    return {
        sort,
        filters: cfg.filters ?? [],
        groupByCol,
        xCol,
        yCol: cfg.yCol ?? null,
        hideAxisLabels: cfg.hideAxisLabels ?? false,
        cardFields,
        hiddenCols: cfg.hiddenCols ?? null,
        columnOrder: cfg.columnOrder ?? null,
        dateCol: cfg.dateCol ?? null,
        timelineZoom: cfg.timelineZoom ?? cfg.calendarZoom ?? null,
        visibleWeekdays: cfg.visibleWeekdays ?? null,
        hiddenKanbanColumns: cfg.hiddenKanbanColumns ?? null,
    };
}

watch(
    activeView,
    (view) => {
        if (!view) return;
        viewConfig.value = parseConfig(view.config, view.type);
    },
    { immediate: true },
);

// ── Modified config detection ─────────────────────────────────────────────────

const hasModifiedConfig = computed(() => {
    if (!activeView.value) return false;
    const saved = parseConfig(activeView.value.config, activeView.value.type);
    const current = viewConfig.value;
    return (
        JSON.stringify({
            sort: saved.sort,
            filters: saved.filters,
            groupByCol: saved.groupByCol,
            xCol: saved.xCol,
            yCol: saved.yCol,
            hideAxisLabels: saved.hideAxisLabels,
            cardFields: saved.cardFields,
            hiddenCols: saved.hiddenCols,
            columnOrder: saved.columnOrder,
            dateCol: saved.dateCol,
            timelineZoom: saved.timelineZoom,
            visibleWeekdays: saved.visibleWeekdays,
            hiddenKanbanColumns: saved.hiddenKanbanColumns,
        }) !==
        JSON.stringify({
            sort: current.sort,
            filters: current.filters,
            groupByCol: current.groupByCol,
            xCol: current.xCol,
            yCol: current.yCol,
            hideAxisLabels: current.hideAxisLabels,
            cardFields: current.cardFields,
            hiddenCols: current.hiddenCols,
            columnOrder: current.columnOrder,
            dateCol: current.dateCol,
            timelineZoom: current.timelineZoom,
            visibleWeekdays: current.visibleWeekdays,
            hiddenKanbanColumns: current.hiddenKanbanColumns,
        })
    );
});

// ── Sort / filter / group / fields handlers ───────────────────────────────────

function onSortChange(sorts) {
    viewConfig.value = { ...viewConfig.value, sort: sorts };
}

function onFilterChange(filters) {
    viewConfig.value = { ...viewConfig.value, filters };
}

function onGroupByChange(colCode) {
    viewConfig.value = { ...viewConfig.value, groupByCol: colCode };
}

function onXColChange(colCode) {
    viewConfig.value = { ...viewConfig.value, xCol: colCode };
}

function onYColChange(colCode) {
    viewConfig.value = { ...viewConfig.value, yCol: colCode };
}

function onAxisLabelsChange(hide) {
    viewConfig.value = { ...viewConfig.value, hideAxisLabels: hide };
}

function onCardFieldsChange(fields) {
    viewConfig.value = { ...viewConfig.value, cardFields: fields };
}

function onHiddenColumnsChange(codes) {
    viewConfig.value = { ...viewConfig.value, hiddenCols: codes };
}

function onColumnOrderChange(order) {
    viewConfig.value = { ...viewConfig.value, columnOrder: order };
}

function onDateColChange(colCode) {
    viewConfig.value = { ...viewConfig.value, dateCol: colCode };
}

function onTimelineZoomChange(zoom) {
    viewConfig.value = { ...viewConfig.value, timelineZoom: zoom };
}

function onWeekdaysChange(days) {
    viewConfig.value = { ...viewConfig.value, visibleWeekdays: days };
}

function onHiddenKanbanColumnsChange(columns) {
    viewConfig.value = { ...viewConfig.value, hiddenKanbanColumns: columns };
}

function onColumnUpdated(updatedCol) {
    if (!table.value) return;
    table.value = {
        ...table.value,
        columns: table.value.columns.map((c) =>
            c.id === updatedCol.id ? updatedCol : c,
        ),
    };
}

// ── View CRUD ─────────────────────────────────────────────────────────────────

async function saveToCurrentView() {
    if (!activeView.value) return;
    const { workspaceCode, tableCode } = route.params;
    await api.updateView(workspaceCode, tableCode, activeView.value.code, {
        name: activeView.value.name,
        config: viewConfig.value,
    });
    await load();
}

async function saveAsNewView() {
    const name = saveAsName.value.trim();
    if (!name) return;
    const { workspaceCode, tableCode } = route.params;
    const type = activeView.value?.type ?? "tabular";
    const newView = await api.createView(workspaceCode, tableCode, {
        name,
        type,
        config: viewConfig.value,
    });
    showSaveAsDialog.value = false;
    saveAsName.value = "";
    await load();
    router.push(
        `/workspaces/${workspaceCode}/tables/${tableCode}/views/${newView.code}`,
    );
}

async function createNewView(type, name) {
    if (type === "kanban" && !hasSelectCols.value) return;
    if (type === "matrix" && !hasTwoSelectCols.value) return;
    const { workspaceCode, tableCode } = route.params;
    const typeNames = {
        tabular: "Table",
        card: "Cards",
        kanban: "Kanban",
        matrix: "Matrix",
        timeline: "Timeline",
    };
    const payload = { name: name ?? typeNames[type], type };
    if (type === "kanban") {
        const xCol = selectColumns.value[0]?.code ?? null;
        if (xCol != null) payload.config = { xCol };
    }
    if (type === "matrix") {
        const xCol = selectColumns.value[0]?.code ?? null;
        const yCol = selectColumns.value[1]?.code ?? null;
        if (xCol != null && yCol != null) payload.config = { xCol, yCol };
    }
    const newView = await api.createView(workspaceCode, tableCode, payload);
    await load();
    router.push(
        `/workspaces/${workspaceCode}/tables/${tableCode}/views/${newView.code}`,
    );
}

function openCreateDialog(type, closeFn) {
    const typeNames = {
        tabular: "Table",
        card: "Cards",
        kanban: "Kanban",
        matrix: "Matrix",
        timeline: "Timeline",
    };
    pendingCreateType.value = type;
    createName.value = typeNames[type];
    showCreateDialog.value = true;
    closeFn();
}

async function createViewWithName() {
    const name = createName.value.trim();
    if (!name) return;
    showCreateDialog.value = false;
    await createNewView(pendingCreateType.value, name);
}

function startRename(view) {
    renamingViewCode.value = view.code;
    renameValue.value = view.name;
}

async function commitRename(view) {
    if (renamingViewCode.value !== view.code) return;
    const name = renameValue.value.trim();
    renamingViewCode.value = null;
    if (!name || name === view.name) return;
    const { workspaceCode, tableCode } = route.params;
    await api.updateView(workspaceCode, tableCode, view.code, {
        name,
        config: view.config,
    });
    await load();
}

function cancelRename() {
    renamingViewCode.value = null;
}

async function setAsDefault(view) {
    const { workspaceCode, tableCode } = route.params;
    const updated = await api.updateTable(workspaceCode, tableCode, {
        name: table.value.name,
        context: table.value.context ?? "",
        default_view_id: view.id,
    });
    table.value = updated;
    router.push(
        `/workspaces/${workspaceCode}/tables/${tableCode}/views/${view.code}`,
    );
}

async function deleteView(view) {
    if (
        !(await confirm({
            title: "Delete view?",
            message: `"${view.name}" will be permanently deleted.`,
            confirmLabel: "Delete",
        }))
    )
        return;
    const { workspaceCode, tableCode } = route.params;
    await api.deleteView(workspaceCode, tableCode, view.code);
    await load();
    const defaultView = table.value?.views?.find(
        (v) => v.id === table.value?.default_view_id,
    );
    if (defaultView)
        router.push(
            `/workspaces/${workspaceCode}/tables/${tableCode}/views/${defaultView.code}`,
        );
}

watch(showSaveAsDialog, (open) => {
    if (!open) saveAsName.value = "";
});
watch(showCreateDialog, (open) => {
    if (!open) createName.value = "";
});

watch(renamingViewCode, async (code) => {
    if (code) {
        await nextTick();
        renameInputEl.value?.focus();
        renameInputEl.value?.select();
    }
});

// ── Table CRUD ────────────────────────────────────────────────────────────────

function onOpenRow({ row, isNew }) {
    const { workspaceCode, tableCode } = route.params;
    const newParam = isNew ? "&new=1" : "";
    router.push(
        `/workspaces/${workspaceCode}/tables/${tableCode}/rows/${row.id}?view=${activeViewCode.value}${newParam}`,
    );
}

async function load() {
    const { workspaceCode, tableCode } = route.params;
    loading.value = true;
    notFound.value = false;
    try {
        table.value = await api.getTable(workspaceCode, tableCode);
    } catch (e) {
        if (e.status === 404) notFound.value = true;
        else throw e;
    } finally {
        loading.value = false;
    }
}

watch(
    () => route.params.tableCode,
    () => {
        showColumns.value = false;
        load();
    },
    { immediate: true },
);

watch(
    table,
    (t) => {
        document.title = t ? `${t.name} — degubase` : "degubase";
    },
    { immediate: true },
);

function switchView(v) {
    const { workspaceCode, tableCode } = route.params;
    router.push(
        `/workspaces/${workspaceCode}/tables/${tableCode}/views/${v.code}`,
    );
}

async function saveEdit() {
    if (!editForm.name.trim()) return;
    showEditDialog.value = false;
    const { workspaceCode } = route.params;
    const updated = await api.updateTable(workspaceCode, table.value.code, {
        name: editForm.name.trim(),
        context: editForm.context.trim(),
        icon: editForm.icon,
    });
    table.value = updated;
    const defaultView = updated.views?.find(
        (v) => v.id === updated.default_view_id,
    );
    if (updated.code !== route.params.tableCode && defaultView) {
        router.replace(
            `/workspaces/${workspaceCode}/tables/${updated.code}/views/${defaultView.code}`,
        );
    }
    reloadSidebar();
}

async function deleteTable() {
    const ok = await confirm({
        title: "Delete table",
        message: `Delete "${table.value.name}" and all its data? This cannot be undone.`,
        confirmLabel: "Delete",
        danger: true,
    });
    if (!ok) return;
    const { workspaceCode } = route.params;
    await api.deleteTable(workspaceCode, table.value.code);
    router.push(`/workspaces/${workspaceCode}`);
}
</script>

<style scoped>
.filter-ctl {
    height: 26px;
    padding: 0 6px;
    font-size: 11px;
    border-radius: 5px;
    border: 1px solid var(--border-1);
    background: var(--surface-2);
    color: var(--text-1);
    outline: none;
    cursor: pointer;
    max-width: 160px;
}
.filter-val {
    height: 26px;
    padding: 0 8px;
    font-size: 11px;
    border-radius: 5px;
    border: 1px solid var(--border-1);
    background: var(--surface-2);
    color: var(--text-1);
    outline: none;
    width: 130px;
}
.filter-ctl:focus,
.filter-val:focus {
    border-color: var(--color-brand-600);
}
</style>
