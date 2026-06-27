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
            @add-record="addRow"
            @bulk-action="showBulkDialog = true"
            @export-csv="exportCsv"
            @import-csv="showImportDialog = true"
        >
            <template #before>
                <span class="text-[11px] text-text-2">Group</span>
                <select
                    class="h-6 px-2 text-[11px] rounded border border-border-1 bg-surface-2 text-text-1 outline-none cursor-pointer"
                    :value="viewConfig?.groupByCol ?? ''"
                    @change="e => emit('group-by-change', e.target.value || null)"
                >
                    <option value="">None</option>
                    <option
                        v-for="col in groupableColumns"
                        :key="col.code"
                        :value="col.code"
                    >{{ col.name }}</option>
                </select>
            </template>
            <template #fields>
                <Flapout align="right">
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
                            <RiLayoutColumnLine size="12" />
                            Fields
                            <span
                                v-if="hiddenColumns.length > 0"
                                class="ml-0.5 inline-flex items-center justify-center min-w-4 h-4 px-1 rounded-full bg-brand-100 dark:bg-brand-900/50 text-brand-600 dark:text-brand-400 text-[10px] font-bold"
                                >{{ hiddenColumns.length }}</span
                            >
                        </button>
                    </template>
                    <PanelCard class="p-3 w-52 max-h-96 overflow-y-auto">
                        <div
                            class="text-[10px] font-semibold text-text-3 mb-2 uppercase tracking-wide"
                        >
                            Fields
                        </div>
                        <div
                            v-for="(col, idx) in sortedColumns"
                            :key="col.id"
                            class="flex items-center gap-1.5 py-0.5 group/field"
                        >
                            <input
                                type="checkbox"
                                :id="`fv-${col.id}`"
                                :checked="visibleColumns.includes(col)"
                                class="w-3 h-3 accent-brand-600 cursor-pointer shrink-0"
                                @change="toggleColumnVisibility(col)"
                            />
                            <label
                                :for="`fv-${col.id}`"
                                class="text-[12px] text-text-1 cursor-pointer truncate flex-1"
                                >{{ col.name }}</label
                            >
                            <div class="flex items-center gap-0.5">
                                <button
                                    :disabled="idx === 0"
                                    class="flex items-center justify-center w-4 h-4 rounded bg-transparent border-none text-text-3 hover:text-text-1 hover:bg-border-1 transition-colors disabled:opacity-30 disabled:cursor-default cursor-pointer"
                                    title="Move up"
                                    @click.stop="moveFieldInList(col, 'up')"
                                >
                                    <RiArrowUpSLine size="11" />
                                </button>
                                <button
                                    :disabled="idx === sortedColumns.length - 1"
                                    class="flex items-center justify-center w-4 h-4 rounded bg-transparent border-none text-text-3 hover:text-text-1 hover:bg-border-1 transition-colors disabled:opacity-30 disabled:cursor-default cursor-pointer"
                                    title="Move down"
                                    @click.stop="moveFieldInList(col, 'down')"
                                >
                                    <RiArrowDownSLine size="11" />
                                </button>
                            </div>
                        </div>
                        <button
                            v-if="hiddenColumns.length > 0"
                            class="mt-2 w-full text-[11px] text-brand-600 dark:text-brand-400 bg-transparent border-none cursor-pointer py-1 rounded hover:bg-brand-50 dark:hover:bg-brand-900/30 transition-colors"
                            @click="emit('hidden-columns-change', [])"
                        >
                            Show all fields
                        </button>
                    </PanelCard>
                </Flapout>
            </template>
        </TableToolbar>

        <div ref="scrollContainerRef" class="flex-1 overflow-auto">
            <table class="border-collapse text-[13px] text-text-1 min-w-full">
                <thead class="sticky top-0 z-3">
                    <tr>
                        <th
                            class="w-20 min-w-20 bg-[#f5f5f8] dark:bg-[#252526] border-r border-b-2 border-border-1"
                        ></th>

                        <th
                            v-for="col in visibleColumns"
                            :key="col.id"
                            class="bg-[#f5f5f8] dark:bg-[#252526] border-r border-b-2 border-border-1 p-0 min-w-40 max-w-75 group/header"
                        >
                            <div class="flex items-center gap-1.5 px-2.5 h-9">
                                <component
                                    :is="typeIcon(col.type)"
                                    size="12"
                                    class="text-text-2 shrink-0"
                                />
                                <span
                                    class="flex-1 flex items-center gap-1 text-xs font-semibold text-text-2 truncate tracking-wide cursor-pointer select-none hover:text-text-1 transition-colors"
                                    @click="handleColSort(col)"
                                >
                                    <span class="truncate">{{ col.name }}</span>
                                    <span
                                        v-if="colSortEntry(col)"
                                        class="text-[11px] text-brand-500 dark:text-brand-400 shrink-0"
                                    >
                                        {{
                                            colSortEntry(col).dir === "asc"
                                                ? "↑"
                                                : "↓"
                                        }}
                                    </span>
                                </span>
                                <!-- Column menu trigger: visible on header hover (desktop) or always on touch -->
                                <button
                                    class="flex items-center justify-center w-4.5 h-4.5 bg-transparent border-none cursor-pointer rounded shrink-0 transition-all opacity-0 group-hover/header:opacity-100 [@media(hover:none)]:opacity-100"
                                    :class="
                                        colMenu?.col.id === col.id
                                            ? 'opacity-100 text-brand-500 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30'
                                            : 'text-text-2 hover:text-text-1 hover:bg-border-1'
                                    "
                                    title="Column options"
                                    @click.stop="openColMenu($event, col)"
                                >
                                    <RiArrowDownSLine size="13" />
                                </button>
                            </div>
                        </th>

                        <th
                            class="bg-[#f5f5f8] dark:bg-[#252526] border-b-2 border-border-1 w-11 text-center"
                        >
                            <button
                                class="flex items-center justify-center w-7 h-7 mx-auto my-1 bg-transparent border border-dashed border-zinc-300 dark:border-[#454545] text-text-2 cursor-pointer rounded-md hover:text-brand-600 hover:border-brand-500 hover:bg-brand-50 dark:hover:bg-brand-900/30 transition-colors"
                                data-testid="add-column-btn"
                                title="Add column"
                                @click="startAddCol(null)"
                            >
                                <RiAddLine size="15" />
                            </button>
                        </th>
                    </tr>
                </thead>

                <tbody>
                    <tr v-if="virtualPaddingTop > 0">
                        <td
                            :colspan="visibleColumns.length + 2"
                            :style="{
                                height: virtualPaddingTop + 'px',
                                padding: '0',
                                border: 'none',
                            }"
                        />
                    </tr>

                    <tr
                        v-for="{ vr, item, rowIdx } in virtualRows"
                        :key="
                            item.type === 'header'
                                ? 'h-' + item.key
                                : item.row.id
                        "
                        :class="item.type !== 'header' && 'group'"
                    >
                        <!-- Group header -->
                        <template v-if="item.type === 'header'">
                            <td
                                :colspan="visibleColumns.length + 2"
                                class="h-8 border-b border-border-1 bg-[#f0f0f4] dark:bg-[#2a2a2d] px-1"
                            >
                                <button
                                    class="flex items-center gap-2 w-full h-full bg-transparent border-none cursor-pointer text-left px-1"
                                    @click="toggleGroup(item.key)"
                                >
                                    <RiArrowRightSLine
                                        size="12"
                                        class="text-text-3 shrink-0 transition-transform"
                                        :class="
                                            collapsedGroups.has(item.key)
                                                ? ''
                                                : 'rotate-90'
                                        "
                                    />
                                    <span
                                        class="text-[12px] font-semibold text-text-1 truncate flex-1"
                                        >{{ item.label }}</span
                                    >
                                    <span
                                        class="text-[11px] text-text-3 shrink-0 mr-1"
                                        >{{ item.count }}</span
                                    >
                                </button>
                            </td>
                        </template>

                        <!-- Data row -->
                        <template v-else>
                            <!-- Row actions: id + three-dots menu -->
                            <td
                                class="w-20 min-w-20 align-middle border-r border-b border-border-1 bg-[#f9f9fb] dark:bg-[#222222] group-hover:bg-zinc-100 dark:group-hover:bg-[#2a2a2a] transition-colors px-1.5"
                            >
                                <div
                                    class="flex items-center justify-between gap-1"
                                >
                                    <Flapout align="right">
                                        <template
                                            #trigger="{ toggle, setAnchor }"
                                        >
                                            <button
                                                :ref="setAnchor"
                                                class="flex items-center justify-center w-4 h-4 bg-transparent border-none text-text-3 cursor-pointer rounded hover:text-text-1 hover:bg-border-1 transition-colors"
                                                title="Row actions"
                                                @click.stop="toggle"
                                            >
                                                <RiMoreLine size="11" />
                                            </button>
                                        </template>
                                        <template #default="{ close }">
                                            <div
                                                class="bg-surface-1 border border-border-1 rounded-lg shadow-xl py-1 w-40"
                                            >
                                                <button
                                                    class="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-1 hover:bg-border-1 transition-colors cursor-pointer border-none bg-transparent"
                                                    data-testid="open-row-btn"
                                                    @click.stop="
                                                        $emit('open-row', {
                                                            row: item.row,
                                                            index: vr.index,
                                                        });
                                                        close();
                                                    "
                                                >
                                                    <RiArrowRightUpLine
                                                        size="13"
                                                        class="shrink-0 text-text-2"
                                                    />
                                                    Open record
                                                </button>
                                                <button
                                                    class="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-1 hover:bg-border-1 transition-colors cursor-pointer border-none bg-transparent"
                                                    @click.stop="
                                                        duplicateRow(item.row);
                                                        close();
                                                    "
                                                >
                                                    <RiFileCopyLine
                                                        size="13"
                                                        class="shrink-0 text-text-2"
                                                    />
                                                    Duplicate
                                                </button>
                                                <button
                                                    class="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-text-1 hover:bg-border-1 transition-colors cursor-pointer border-none bg-transparent"
                                                    @click.stop="
                                                        copyRowLink(item.row);
                                                        close();
                                                    "
                                                >
                                                    <RiLink
                                                        size="13"
                                                        class="shrink-0 text-text-2"
                                                    />
                                                    Copy link
                                                </button>
                                                <div
                                                    class="h-px bg-border-1 mx-2 my-1"
                                                />
                                                <button
                                                    class="w-full flex items-center gap-2 px-3 py-1.5 text-[12px] text-red-500 hover:bg-red-50 dark:hover:bg-red-950/40 transition-colors cursor-pointer border-none bg-transparent"
                                                    data-testid="delete-row-btn"
                                                    title="Delete record"
                                                    @click.stop="
                                                        confirmDeleteRow(
                                                            item.row,
                                                        );
                                                        close();
                                                    "
                                                >
                                                    <RiDeleteBin6Line
                                                        size="13"
                                                        class="shrink-0"
                                                    />
                                                    Delete
                                                </button>
                                            </div>
                                        </template>
                                    </Flapout>
                                    <span
                                        class="text-[10px] text-text-3 font-mono flex-1 text-right select-none pr-0.5"
                                        >{{ item.row.id }}</span
                                    >
                                </div>
                            </td>

                            <!-- Data cells -->
                            <td
                                v-for="(col, colIdx) in visibleColumns"
                                :key="col.id"
                                :data-row-id="item.row.id"
                                :data-col-id="col.id"
                                class="border-r border-b border-border-1 p-0 h-8.5 min-w-40 max-w-75 bg-surface-1 align-middle group-hover:bg-zinc-50 dark:group-hover:bg-[#2d2d30] transition-colors"
                                :class="{
                                    'outline-2 outline-brand-500 -outline-offset-1 z-1 bg-white! dark:bg-[#252526]!':
                                        isEditing(item.row.id, col.id) ||
                                        isFlapout(item.row.id, col.id) ||
                                        isFileFlapout(item.row.id, col.id),
                                    'cursor-text':
                                        !isReadOnly(col.type) &&
                                        col.type !== 'checkbox' &&
                                        col.type !== 'rating' &&
                                        col.type !== 'file' &&
                                        col.type !== 'image' &&
                                        col.type !== 'checklist' &&
                                        col.type !== 'row-link',
                                    'cursor-pointer':
                                        col.type === 'file' ||
                                        col.type === 'image' ||
                                        col.type === 'checklist' ||
                                        col.type === 'row-link',
                                    'cursor-default':
                                        col.type === 'created-at' ||
                                        col.type === 'updated-at' ||
                                        col.type === 'checkbox' ||
                                        col.type === 'rating',
                                }"
                                @click="startEdit(item.row, col, $event)"
                            >
                                <!-- Tree indent + cell (first column, hierarchy mode, not editing) -->
                                <template
                                    v-if="
                                        parentCol &&
                                        colIdx === 0 &&
                                        !isEditing(item.row.id, col.id)
                                    "
                                >
                                    <div class="flex items-center h-full">
                                        <span
                                            :style="{
                                                width: item.depth * 16 + 'px',
                                            }"
                                            class="shrink-0"
                                        />
                                        <button
                                            v-if="hasChildren.has(item.row.id)"
                                            data-testid="tree-toggle"
                                            class="flex items-center justify-center w-4 h-4 shrink-0 bg-transparent border-none text-text-3 cursor-pointer hover:text-text-1 transition-colors"
                                            @click.stop="
                                                toggleParent(item.row.id)
                                            "
                                        >
                                            <RiArrowDownSLine
                                                v-if="
                                                    !collapsedParents.has(
                                                        item.row.id,
                                                    )
                                                "
                                                size="11"
                                            />
                                            <RiArrowRightSLine
                                                v-else
                                                size="11"
                                            />
                                        </button>
                                        <span v-else class="w-4 shrink-0" />
                                        <CellRenderer
                                            :col="col"
                                            :row="item.row"
                                            :workspace-code="workspaceCode"
                                            :table-code="tableCode"
                                            @toggle="
                                                (checked) =>
                                                    toggleCheckbox(
                                                        item.row,
                                                        col,
                                                        checked,
                                                    )
                                            "
                                            @rate="
                                                (value) =>
                                                    setRating(
                                                        item.row,
                                                        col,
                                                        value,
                                                    )
                                            "
                                            @upload="
                                                onCellFileUpload(
                                                    col,
                                                    item.row,
                                                    $event,
                                                )
                                            "
                                        />
                                    </div>
                                </template>

                                <!-- Editing state -->
                                <FieldEditor
                                    v-else-if="isEditing(item.row.id, col.id)"
                                    v-model="editVal"
                                    :col="col"
                                    :workspace-code="workspaceCode"
                                    :table-code="tableCode"
                                    @commit="commitEdit"
                                    @cancel="cancelEdit"
                                    @tab="
                                        onTabKey(
                                            $event,
                                            item.row.id,
                                            col.id,
                                            col.code,
                                        )
                                    "
                                    @column-updated="
                                        emit('column-updated', $event)
                                    "
                                />

                                <!-- Display state -->
                                <CellRenderer
                                    v-else
                                    :col="col"
                                    :row="item.row"
                                    :workspace-code="workspaceCode"
                                    :table-code="tableCode"
                                    @toggle="
                                        (checked) =>
                                            toggleCheckbox(
                                                item.row,
                                                col,
                                                checked,
                                            )
                                    "
                                    @rate="
                                        (value) =>
                                            setRating(item.row, col, value)
                                    "
                                    @upload="
                                        onCellFileUpload(col, item.row, $event)
                                    "
                                />
                            </td>

                            <td
                                class="border-b border-border-1 bg-surface-1 group-hover:bg-zinc-50 dark:group-hover:bg-[#2d2d30] transition-colors"
                            ></td>
                        </template>
                    </tr>

                    <tr v-if="virtualPaddingBottom > 0">
                        <td
                            :colspan="visibleColumns.length + 2"
                            :style="{
                                height: virtualPaddingBottom + 'px',
                                padding: '0',
                                border: 'none',
                            }"
                        />
                    </tr>

                    <!-- Add row -->
                    <tr>
                        <td
                            :colspan="sortedColumns.length + 2"
                            class="border-none bg-zinc-100 dark:bg-[#1e1e1e] px-2 py-1"
                        >
                            <button
                                class="inline-flex items-center gap-1.5 bg-transparent border-none text-text-2 text-xs cursor-pointer px-2.5 py-1.5 rounded-md hover:text-text-1 hover:bg-border-1 transition-colors"
                                data-testid="add-row-btn"
                                @click="addRow"
                            >
                                <RiAddLine size="13" />
                                Add record
                            </button>
                        </td>
                    </tr>

                    <!-- Load more -->
                    <tr v-if="hasMore">
                        <td
                            :colspan="sortedColumns.length + 2"
                            class="border-none bg-zinc-100 dark:bg-[#1e1e1e] px-2 py-1 text-center"
                        >
                            <button
                                class="inline-flex items-center gap-1.5 bg-transparent border-none text-text-2 text-xs cursor-pointer px-3 py-1.5 rounded-md hover:text-text-1 hover:bg-border-1 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                :disabled="loadingMore"
                                @click="loadMore"
                            >
                                <span v-if="loadingMore">Loading…</span>
                                <span v-else
                                    >Load more ({{ rows.length }} /
                                    {{ total }})</span
                                >
                            </button>
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>

        <AddColumnDialog
            v-if="showAddCol"
            :workspace-code="workspaceCode"
            :table-code="tableCode"
            @confirm="onAddColumn"
            @cancel="cancelAddCol"
        />

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

        <CellFlapout
            v-if="flapoutCell"
            :model-value="flapoutCell.value"
            :col-type="flapoutCell.colType"
            :col-name="flapoutCell.colName"
            :column="flapoutCell.column ?? {}"
            :anchor-rect="flapoutCell.rect"
            :workspace-code="workspaceCode"
            @close="closeFlapout"
            @navigate="onFlapoutNavigate"
        />

        <FileFlapout
            v-if="fileFlapout"
            :meta="fileFlapout.meta"
            :col-type="fileFlapout.colType"
            :col-name="fileFlapout.colName"
            :workspace-code="workspaceCode"
            :table-code="tableCode"
            :row-id="fileFlapout.rowId"
            :col-id="fileFlapout.colId"
            :col-code="fileFlapout.colCode"
            :anchor-rect="fileFlapout.rect"
            @close="closeFileFlapout()"
            @row-updated="closeFileFlapout($event)"
        />

        <!-- Column action menu -->
        <Teleport to="body">
            <template v-if="colMenu">
                <!-- Backdrop to catch outside clicks -->
                <div class="fixed inset-0 z-199" @click="closeMenu" />

                <!-- Menu panel -->
                <div
                    class="fixed z-200 col-popup"
                    :style="menuStyle"
                    @click.stop
                >
                    <!-- ── Edit field form ─────────────────────────────── -->
                    <ColumnEditForm
                        v-if="colEditMode"
                        :column="colMenu.col"
                        :workspace-code="workspaceCode"
                        :table-code="tableCode"
                        @saved="onColumnSaved"
                        @cancel="colEditMode = false"
                    />

                    <!-- ── Action menu ─────────────────────────────────── -->
                    <template v-else>
                        <template v-if="!isVirtual(colMenu.col)">
                            <button class="menu-item" @click="enterEditMode">
                                <RiPencilLine size="14" class="shrink-0" />
                                Edit field
                            </button>

                            <div class="menu-sep" />

                            <button
                                class="menu-item"
                                @click="insertCol('left')"
                            >
                                <RiArrowLeftSLine size="14" class="shrink-0" />
                                Insert left
                            </button>
                            <button
                                class="menu-item"
                                @click="insertCol('right')"
                            >
                                <RiArrowRightSLine size="14" class="shrink-0" />
                                Insert right
                            </button>
                            <button
                                class="menu-item"
                                :class="{ 'menu-item--disabled': !canMoveLeft }"
                                @click="canMoveLeft && moveCol('left')"
                            >
                                <RiArrowLeftSLine size="14" class="shrink-0" />
                                Move left
                            </button>
                            <button
                                class="menu-item"
                                :class="{
                                    'menu-item--disabled': !canMoveRight,
                                }"
                                @click="canMoveRight && moveCol('right')"
                            >
                                <RiArrowRightSLine size="14" class="shrink-0" />
                                Move right
                            </button>

                            <div class="menu-sep" />
                        </template>

                        <button class="menu-item" @click="menuSort('asc')">
                            <RiArrowUpLine size="14" class="shrink-0" />
                            Sort A → Z
                        </button>
                        <button class="menu-item" @click="menuSort('desc')">
                            <RiArrowDownLine size="14" class="shrink-0" />
                            Sort Z → A
                        </button>

                        <div class="menu-sep" />

                        <button class="menu-item" @click="menuAddFilter">
                            <RiFilterLine size="14" class="shrink-0" />
                            Create filter
                        </button>

                        <button class="menu-item" @click="menuHideColumn">
                            <RiEyeOffLine size="14" class="shrink-0" />
                            Hide field
                        </button>

                        <template v-if="!isVirtual(colMenu.col)">
                            <div class="menu-sep" />
                            <button
                                class="menu-item menu-item--danger"
                                @click="menuDeleteCol"
                            >
                                <RiDeleteBin6Line size="14" class="shrink-0" />
                                Delete field
                            </button>
                        </template>
                    </template>
                </div>
            </template>
        </Teleport>
    </div>
</template>

<script setup>
import { ref, computed, nextTick, watch } from "vue";
import { useVirtualizer } from "@tanstack/vue-virtual";
import { api, fileUrl, thumbnailUrl } from "../../api/client.js";
import AddColumnDialog from "../schema/AddColumnDialog.vue";
import BulkActionDialog from "../views/BulkActionDialog.vue";
import ImportCsvDialog from "../views/ImportCsvDialog.vue";
import CellFlapout from "../records/CellFlapout.vue";
import CellRenderer from "../records/CellRenderer.vue";
import FieldEditor from "../records/FieldEditor.vue";
import FileFlapout from "../records/FileFlapout.vue";
import ColumnEditForm from "../schema/ColumnEditForm.vue";
import Flapout from "../../foundation/Flapout.vue";
import TableToolbar from "../views/TableToolbar.vue";
import PanelCard from "../../foundation/PanelCard.vue";
import { TYPE_FAMILIES, typeIcon } from "../schema/columnTypes.js";
import { getCellValue } from "../records/cellHelpers.js";
import { getColOps } from "../views/filterHelpers.js";
import { useViewRows } from "../records/useViewRows.js";
import { useSortedColumns } from "../records/useSortedColumns.js";
import { useFieldReordering } from "../records/useFieldReordering.js";
import { useNotifications } from "../../foundation/useNotifications.js";
import { useConfirm } from "../../foundation/useConfirm.js";
import { useReferencesBlock } from "../../foundation/useReferencesBlock.js";
import {
    RiMoreLine,
    RiAddLine,
    RiArrowRightUpLine,
    RiFileCopyLine,
    RiArrowDownSLine,
    RiArrowLeftSLine,
    RiArrowRightSLine,
    RiArrowUpSLine,
    RiArrowUpLine,
    RiArrowDownLine,
    RiPencilLine,
    RiDeleteBin6Line,
    RiFilterLine,
    RiEyeOffLine,
    RiLayoutColumnLine,
    RiLink,
} from "@remixicon/vue";

const props = defineProps({
    workspaceCode: { type: String, required: true },
    tableCode: { type: String, required: true },
    table: { type: Object, required: true },
    viewConfig: { type: Object, default: () => ({ sort: null, filters: [] }) },
});
const emit = defineEmits([
    "reload-table",
    "open-row",
    "sort-change",
    "filter-change",
    "hidden-columns-change",
    "column-order-change",
    "column-updated",
    "group-by-change",
]);

const { notify } = useNotifications();
const { confirm } = useConfirm();
const { showBlock } = useReferencesBlock();
const {
    rows,
    total,
    hasMore,
    loadingMore,
    loadMore,
    loadRows,
    searchDebounced,
} = useViewRows(() => props);
const showAddCol = ref(false);
const addColContext = ref(null);
const showBulkDialog = ref(false);
const showImportDialog = ref(false);
const editing = ref(null);
const editVal = ref("");
const flapoutCell = ref(null); // { rowId, colId, colType, colName, value, rect }
const fileFlapout = ref(null); // { rowId, colId, colType, colName, meta, rect }
const rowRevisions = {}; // rowId → revisionId for bundling rapid saves

// ── Column menu ───────────────────────────────────────────────────────────────

const colMenu = ref(null);
const colEditMode = ref(false);

const colEditFamily = computed(() =>
    colMenu.value ? (TYPE_FAMILIES[colMenu.value.col.type] ?? "text") : "text",
);

const menuStyle = computed(() => {
    if (!colMenu.value) return {};
    const wide = colEditMode.value && colEditFamily.value === "select";
    const w = wide ? 264 : 200;
    const left = Math.min(colMenu.value.rawLeft, window.innerWidth - w - 8);
    return {
        top: colMenu.value.top + "px",
        left: left + "px",
        width: w + "px",
    };
});

function openColMenu(event, col) {
    const rect = event.currentTarget.getBoundingClientRect();
    colMenu.value = { col, top: rect.bottom + 4, rawLeft: rect.left };
    colEditMode.value = false;
}

function closeMenu() {
    colMenu.value = null;
    colEditMode.value = false;
}

function enterEditMode() {
    colEditMode.value = true;
}

function onColumnSaved() {
    closeMenu();
    emit("reload-table");
}

function menuSort(dir) {
    const col = colMenu.value.col;
    const sorts = props.viewConfig?.sort ?? [];
    const existing = sorts.find((s) => s.col === col.code);
    let next;
    if (existing && existing.dir === dir) {
        next = sorts.filter((s) => s.col !== col.code);
    } else if (existing) {
        next = sorts.map((s) => (s.col === col.code ? { ...s, dir } : s));
    } else {
        next = [...sorts, { col: col.code, dir }];
    }
    emit("sort-change", next);
    closeMenu();
}

function menuAddFilter() {
    const col = colMenu.value.col;
    const ops = getColOps(col);
    emit("filter-change", [
        ...(props.viewConfig?.filters ?? []),
        {
            id: crypto.randomUUID(),
            col: col.code,
            op: ops[0]?.value ?? "contains",
            value: "",
        },
    ]);
    closeMenu();
}

async function menuDeleteCol() {
    const col = colMenu.value.col;
    closeMenu();
    if (
        !(await confirm({
            title: "Delete column?",
            message: `"${col.name}" and all its data will be permanently deleted.`,
            confirmLabel: "Delete",
        }))
    )
        return;
    await api.deleteColumn(props.workspaceCode, props.tableCode, col.code);
    emit("reload-table");
}

function insertCol(dir) {
    addColContext.value = { refColCode: colMenu.value.col.code, dir };
    closeMenu();
    showAddCol.value = true;
}

const colMenuIdx = computed(() => {
    if (!colMenu.value || isVirtual(colMenu.value.col)) return -1;
    return realSortedColumns.value.findIndex(
        (c) => c.id === colMenu.value.col.id,
    );
});
const canMoveLeft = computed(() => colMenuIdx.value > 0);
const canMoveRight = computed(
    () =>
        colMenuIdx.value !== -1 &&
        colMenuIdx.value < realSortedColumns.value.length - 1,
);

function menuHideColumn() {
    const col = colMenu.value.col;
    closeMenu();
    toggleColumnVisibility(col);
}

async function moveCol(dir) {
    const idx = colMenuIdx.value;
    if (idx === -1) return;
    const newOrder = realSortedColumns.value.map((c) => c.code);
    if (dir === "left") {
        if (idx === 0) return;
        [newOrder[idx - 1], newOrder[idx]] = [newOrder[idx], newOrder[idx - 1]];
    } else {
        if (idx === newOrder.length - 1) return;
        [newOrder[idx], newOrder[idx + 1]] = [newOrder[idx + 1], newOrder[idx]];
    }
    closeMenu();
    await api.reorderColumns(props.workspaceCode, props.tableCode, newOrder);
    emit("reload-table");
}

const sortedColumns = useSortedColumns(
    computed(() => props.table.columns ?? []),
    computed(() => props.viewConfig?.columnOrder),
);

const groupableColumns = computed(() =>
    sortedColumns.value.filter(c => !['created-at', 'updated-at'].includes(c.type))
)

const { moveFieldInList } = useFieldReordering(
    sortedColumns,
    computed(
        () =>
            props.viewConfig?.columnOrder ??
            sortedColumns.value.map((c) => c.code),
    ),
    emit,
);

// Real (non-virtual) columns only — used for reorder operations.
const realSortedColumns = computed(() =>
    sortedColumns.value.filter((c) => c.id > 0),
);

// null hiddenCols → hide system timestamp columns by default.
const visibleColumns = computed(() => {
    const hidden = props.viewConfig?.hiddenCols;
    if (hidden === null || hidden === undefined)
        return sortedColumns.value.filter(
            (c) => !["created-at", "updated-at"].includes(c.type),
        );
    return sortedColumns.value.filter((c) => !hidden.includes(c.code));
});

const hiddenColumns = computed(() =>
    sortedColumns.value.filter((c) => !visibleColumns.value.includes(c)),
);

function isVirtual(col) {
    return col.id < 0;
}

function currentHiddenSlugs() {
    const h = props.viewConfig?.hiddenCols;
    if (h != null) return h;
    return sortedColumns.value
        .filter((c) => ["created-at", "updated-at"].includes(c.type))
        .map((c) => c.code);
}

function toggleColumnVisibility(col) {
    const slugs = currentHiddenSlugs();
    const next = slugs.includes(col.code)
        ? slugs.filter((s) => s !== col.code)
        : [...slugs, col.code];
    emit("hidden-columns-change", next);
}

// ── Sort helpers ──────────────────────────────────────────────────────────────

function colSortEntry(col) {
    const sorts = props.viewConfig?.sort ?? [];
    return sorts.find((s) => s.col === col.code) ?? null;
}

function handleColSort(col) {
    const sorts = props.viewConfig?.sort ?? [];
    const existing = sorts.find((s) => s.col === col.code);
    let next;
    if (!existing) {
        next = [...sorts, { col: col.code, dir: "asc" }];
    } else if (existing.dir === "asc") {
        next = sorts.map((s) =>
            s.col === col.code ? { ...s, dir: "desc" } : s,
        );
    } else {
        next = sorts.filter((s) => s.col !== col.code);
    }
    emit("sort-change", next);
}

// ── Computed rows ─────────────────────────────────────────────────────────────

const displayedRows = computed(() => rows.value);

const scrollContainerRef = ref(null);

// Grouping must be declared before rowVirtualizer (which reads flatItems).
const collapsedGroups = ref(new Set());

const groupByCol = computed(() => {
    const slug = props.viewConfig?.groupByCol;
    if (!slug) return null;
    return sortedColumns.value.find((c) => c.code === slug) ?? null;
});

function groupKey(val) {
    if (val == null || val === "") return "__empty__";
    if (Array.isArray(val)) return val.join(" ");
    if (typeof val === "object") return String(val.id ?? JSON.stringify(val));
    return String(val);
}

function groupLabel(val) {
    if (val == null || val === "") return "No value";
    if (Array.isArray(val)) return val.join(", ");
    if (typeof val === "object" && val.label) return val.label;
    return String(val);
}

const parentCol = computed(() => {
    return (
        sortedColumns.value.find((col) => {
            if (col.type !== "row-link" || !col.options) return false;
            const opts =
                typeof col.options === "string"
                    ? JSON.parse(col.options)
                    : col.options;
            return opts?.targetTableCode === props.tableCode;
        }) ?? null
    );
});

const collapsedParents = ref(new Set());

function toggleParent(rowId) {
    const s = new Set(collapsedParents.value);
    s.has(rowId) ? s.delete(rowId) : s.add(rowId);
    collapsedParents.value = s;
}

const hasChildren = computed(() => {
    const col = parentCol.value;
    if (!col) return new Set();
    const allRows = displayedRows.value;
    const rowIds = new Set(allRows.map((r) => r.id));
    const s = new Set();
    for (const row of allRows) {
        const val = getCellValue(row, col);
        if (val?.id && rowIds.has(val.id)) s.add(val.id);
    }
    return s;
});

const flatItems = computed(() => {
    const col = groupByCol.value;
    const pCol = parentCol.value;
    const allRows = displayedRows.value;

    if (pCol) {
        const collapsed = collapsedParents.value; // read at top level so Vue tracks the dep
        const rowById = new Map(allRows.map((r) => [r.id, r]));
        const childrenOf = new Map();
        const roots = [];
        for (const row of allRows) {
            const val = getCellValue(row, pCol);
            const parentId = val?.id ?? null;
            if (parentId && rowById.has(parentId)) {
                if (!childrenOf.has(parentId)) childrenOf.set(parentId, []);
                childrenOf.get(parentId).push(row);
            } else {
                roots.push(row);
            }
        }
        const items = [];
        const visited = new Set();
        const stack = roots.map((r) => ({ row: r, depth: 0 }));
        while (stack.length) {
            const { row, depth } = stack.shift();
            if (visited.has(row.id)) continue;
            visited.add(row.id);
            items.push({ type: "row", row, depth });
            if (!collapsed.has(row.id)) {
                const children = childrenOf.get(row.id) ?? [];
                for (let i = children.length - 1; i >= 0; i--) {
                    stack.unshift({ row: children[i], depth: depth + 1 });
                }
            }
        }
        for (const row of allRows) {
            if (!visited.has(row.id)) {
                const val = getCellValue(row, pCol);
                const parentId = val?.id ?? null;
                // Only promote as root when parent truly absent from view (not just collapsed)
                if (!parentId || !rowById.has(parentId)) {
                    items.push({ type: "row", row, depth: 0 });
                }
            }
        }
        return items;
    }

    if (!col) return allRows.map((row) => ({ type: "row", row }));

    const groups = new Map();
    for (const row of allRows) {
        const cellVal = getCellValue(row, col);
        const key = groupKey(cellVal);
        const label = groupLabel(cellVal);
        if (!groups.has(key)) groups.set(key, { key, label, rows: [] });
        groups.get(key).rows.push(row);
    }

    const items = [];
    for (const { key, label, rows } of groups.values()) {
        items.push({ type: "header", key, label, count: rows.length });
        if (!collapsedGroups.value.has(key)) {
            for (const row of rows) items.push({ type: "row", row });
        }
    }
    return items;
});

function toggleGroup(key) {
    const s = new Set(collapsedGroups.value);
    s.has(key) ? s.delete(key) : s.add(key);
    collapsedGroups.value = s;
}

const rowVirtualizer = useVirtualizer(
    computed(() => ({
        count: flatItems.value.length,
        getScrollElement: () => scrollContainerRef.value,
        estimateSize: (idx) =>
            flatItems.value[idx]?.type === "header" ? 32 : 34,
        overscan: 10,
    })),
);

const virtualRows = computed(() => {
    let dataIdx = 0;
    return rowVirtualizer.value.getVirtualItems().map((vr) => {
        const item = flatItems.value[vr.index];
        const rowIdx = item?.type === "row" ? dataIdx++ : -1;
        return { vr, item, rowIdx };
    });
});

const virtualPaddingTop = computed(() => virtualRows.value[0]?.vr.start ?? 0);
const virtualPaddingBottom = computed(() => {
    const items = virtualRows.value;
    if (!items.length) return 0;
    return rowVirtualizer.value.getTotalSize() - items[items.length - 1].vr.end;
});

// ── Cell helpers ──────────────────────────────────────────────────────────────

function isLongText(type) {
    return type === "long-text" || type === "markdown";
}
function isReadOnly(type) {
    return (
        type === "created-at" ||
        type === "updated-at" ||
        type === "file" ||
        type === "image"
    );
}
function isEditing(rowId, colId) {
    return editing.value?.rowId === rowId && editing.value?.colId === colId;
}
function isFlapout(rowId, colId) {
    return (
        flapoutCell.value?.rowId === rowId && flapoutCell.value?.colId === colId
    );
}
function isFileFlapout(rowId, colId) {
    return (
        fileFlapout.value?.rowId === rowId && fileFlapout.value?.colId === colId
    );
}

function parseFileMeta(cellVal) {
    if (!cellVal || typeof cellVal !== "object") return null;
    return cellVal;
}

const uploadingCol = ref(null);

async function onCellFileUpload(col, row, event) {
    const file = event.target.files?.[0];
    if (!file) return;
    uploadingCol.value = col.id;
    try {
        const updated = await api.uploadFile(
            props.workspaceCode,
            props.tableCode,
            row.id,
            col.code,
            file,
        );
        row.data = updated.data;
        row.updated_at = updated.updated_at;
    } catch (e) {
        notify(e.message || "File upload failed");
    } finally {
        uploadingCol.value = null;
        event.target.value = "";
    }
}

function startEdit(row, col, event = null) {
    if (col.type === "file" || col.type === "image") {
        const meta = parseFileMeta(getCellValue(row, col));
        if (meta) {
            if (isFileFlapout(row.id, col.id)) return;
            const rect =
                event?.currentTarget?.getBoundingClientRect?.() ??
                document
                    .querySelector(
                        `td[data-row-id="${row.id}"][data-col-id="${col.id}"]`,
                    )
                    ?.getBoundingClientRect() ??
                null;
            fileFlapout.value = {
                rowId: row.id,
                colId: col.id,
                colCode: col.code,
                colType: col.type,
                colName: col.name,
                meta,
                rect,
            };
        }
        return; // no content: label handles upload via @click.stop
    }
    if (
        isReadOnly(col.type) ||
        col.type === "checkbox" ||
        col.type === "rating"
    )
        return;
    if (
        isLongText(col.type) ||
        col.type === "multi-select" ||
        col.type === "checklist" ||
        col.type === "row-link"
    ) {
        if (isFlapout(row.id, col.id)) return;
        const rect =
            event?.currentTarget?.getBoundingClientRect?.() ??
            document
                .querySelector(
                    `td[data-row-id="${row.id}"][data-col-id="${col.id}"]`,
                )
                ?.getBoundingClientRect() ??
            null;
        const rawVal = getCellValue(row, col);
        const value =
            col.type === "multi-select"
                ? Array.isArray(rawVal)
                    ? rawVal
                    : rawVal
                      ? [rawVal]
                      : []
                : col.type === "checklist"
                  ? Array.isArray(rawVal)
                      ? rawVal
                      : []
                  : col.type === "row-link"
                    ? (rawVal ?? null)
                    : (rawVal ?? "");
        flapoutCell.value = {
            rowId: row.id,
            colId: col.id,
            colCode: col.code,
            colType: col.type,
            colName: col.name,
            value,
            column: col,
            rect,
        };
        return;
    }
    if (isEditing(row.id, col.id)) return;
    editing.value = { rowId: row.id, colId: col.id, colCode: col.code };
    const v = getCellValue(row, col);
    editVal.value =
        col.type === "multi-select"
            ? Array.isArray(v)
                ? v
                : v
                  ? [v]
                  : []
            : (v ?? "");
    nextTick(() => {
        document.querySelector(".cell-editor")?.focus();
    });
}

async function closeFlapout(newVal) {
    if (!flapoutCell.value) return;
    const { rowId, colId, colCode, value: origVal } = flapoutCell.value;
    flapoutCell.value = null;
    if (newVal === origVal) return;
    const row = rows.value.find((r) => r.id === rowId);
    if (!row) return;
    const newData = { ...(row.data ?? {}), [colCode]: newVal };
    row.data = newData;
    try {
        const { row: updated, revisionId } = await api.updateRow(
            props.workspaceCode,
            props.tableCode,
            rowId,
            { data: newData },
            rowRevisions[rowId],
        );
        rowRevisions[rowId] = revisionId;
        row.updated_at = updated.updated_at;
    } catch (e) {
        notify(e.message || "Save failed");
    }
}

function closeFileFlapout(updatedRow) {
    if (updatedRow) {
        const row = rows.value.find((r) => r.id === updatedRow.id);
        if (row) {
            row.data = updatedRow.data;
            row.updated_at = updatedRow.updated_at;
        }
    }
    fileFlapout.value = null;
}

async function onFlapoutNavigate({ dir, value }) {
    if (!flapoutCell.value) return;
    const { rowId, colId, colCode, value: origVal } = flapoutCell.value;
    flapoutCell.value = null;

    if (value !== origVal) {
        const row = rows.value.find((r) => r.id === rowId);
        if (row) {
            const newData = { ...(row.data ?? {}), [colCode]: value };
            row.data = newData;
            api.updateRow(props.workspaceCode, props.tableCode, rowId, {
                data: newData,
            })
                .then((updated) => {
                    row.updated_at = updated.updated_at;
                })
                .catch((e) => notify(e.message || "Save failed"));
        }
    }

    await nextTick();

    const editableCols = visibleColumns.value.filter(
        (c) =>
            !isReadOnly(c.type) && c.type !== "checkbox" && c.type !== "rating",
    );
    const colIdx = editableCols.findIndex((c) => c.id === colId);

    if (dir === "next") {
        if (colIdx < editableCols.length - 1) {
            const row = rows.value.find((r) => r.id === rowId);
            if (row) startEdit(row, editableCols[colIdx + 1]);
        } else {
            const rowIdx = displayedRows.value.findIndex((r) => r.id === rowId);
            if (rowIdx < displayedRows.value.length - 1) {
                const nextRow = displayedRows.value[rowIdx + 1];
                if (editableCols.length) startEdit(nextRow, editableCols[0]);
            } else {
                const newRow = await api.createRow(
                    props.workspaceCode,
                    props.tableCode,
                    { data: {} },
                );
                rows.value.push(newRow);
                await nextTick();
                if (editableCols.length) startEdit(newRow, editableCols[0]);
            }
        }
    } else {
        if (colIdx > 0) {
            const row = rows.value.find((r) => r.id === rowId);
            if (row) startEdit(row, editableCols[colIdx - 1]);
        } else {
            const rowIdx = displayedRows.value.findIndex((r) => r.id === rowId);
            if (rowIdx > 0) {
                const prevRow = displayedRows.value[rowIdx - 1];
                if (editableCols.length)
                    startEdit(prevRow, editableCols[editableCols.length - 1]);
            }
        }
    }
}

let tabbingFlag = false;

async function commitEdit() {
    if (tabbingFlag) return; // tab navigation handles the save
    if (!editing.value) return;
    const { rowId, colId, colCode } = editing.value;
    const val = editVal.value;
    editing.value = null;
    const row = rows.value.find((r) => r.id === rowId);
    if (!row) return;
    try {
        const { row: updated, revisionId } = await api.patchRow(
            props.workspaceCode,
            props.tableCode,
            rowId,
            { data: { [colCode]: val } },
            rowRevisions[rowId],
        );
        rowRevisions[rowId] = revisionId;
        row.data = updated.data;
        row.updated_at = updated.updated_at;
    } catch (e) {
        notify(e.message || "Save failed");
    }
}

function onTabKey(event, rowId, colId, colCode) {
    event.shiftKey
        ? onCellShiftTab(rowId, colId, colCode)
        : onCellTab(rowId, colId, colCode);
}

async function onCellShiftTab(rowId, colId, colCode) {
    if (tabbingFlag) return;
    tabbingFlag = true;

    const val = editVal.value;
    editing.value = null;
    await nextTick();

    const row = rows.value.find((r) => r.id === rowId);
    if (row) {
        api.patchRow(
            props.workspaceCode,
            props.tableCode,
            rowId,
            { data: { [colCode]: val } },
            rowRevisions[rowId],
        )
            .then(({ row: updated, revisionId }) => {
                row.updated_at = updated.updated_at;
                row.data = updated.data;
                rowRevisions[rowId] = revisionId;
            })
            .catch((e) => notify(e.message || "Save failed"));
    }

    tabbingFlag = false;

    const editableCols = visibleColumns.value.filter(
        (c) =>
            !isReadOnly(c.type) && c.type !== "checkbox" && c.type !== "rating",
    );
    const colIdx = editableCols.findIndex((c) => c.id === colId);

    if (colIdx > 0) {
        const sameRow = rows.value.find((r) => r.id === rowId);
        if (sameRow) startEdit(sameRow, editableCols[colIdx - 1]);
    } else {
        const rowIdx = displayedRows.value.findIndex((r) => r.id === rowId);
        if (rowIdx > 0) {
            const prevRow = displayedRows.value[rowIdx - 1];
            if (editableCols.length)
                startEdit(prevRow, editableCols[editableCols.length - 1]);
        }
    }
}

async function onCellTab(rowId, colId, colCode) {
    if (tabbingFlag) return;
    tabbingFlag = true;

    // Snapshot value before the editor is removed from DOM
    const val = editVal.value;
    editing.value = null;
    await nextTick(); // Vue removes the editor; blur fires here but commitEdit is suppressed

    // Persist the cell value
    const row = rows.value.find((r) => r.id === rowId);
    if (row) {
        api.patchRow(
            props.workspaceCode,
            props.tableCode,
            rowId,
            { data: { [colCode]: val } },
            rowRevisions[rowId],
        )
            .then(({ row: updated, revisionId }) => {
                row.updated_at = updated.updated_at;
                row.data = updated.data;
                rowRevisions[rowId] = revisionId;
            })
            .catch((e) => notify(e.message || "Save failed"));
    }

    tabbingFlag = false;

    // Find next editable cell among visible columns (skip hidden, read-only, checkbox, rating)
    const editableCols = visibleColumns.value.filter(
        (c) =>
            !isReadOnly(c.type) && c.type !== "checkbox" && c.type !== "rating",
    );
    const colIdx = editableCols.findIndex((c) => c.id === colId);

    if (colIdx < editableCols.length - 1) {
        // Next column, same row
        const sameRow = rows.value.find((r) => r.id === rowId);
        if (sameRow) startEdit(sameRow, editableCols[colIdx + 1]);
    } else {
        // End of row — advance to next displayed row or create one
        const rowIdx = displayedRows.value.findIndex((r) => r.id === rowId);
        if (rowIdx < displayedRows.value.length - 1) {
            const nextRow = displayedRows.value[rowIdx + 1];
            if (editableCols.length) startEdit(nextRow, editableCols[0]);
        } else {
            const newRow = await api.createRow(
                props.workspaceCode,
                props.tableCode,
                { data: {} },
            );
            rows.value.push(newRow);
            await nextTick();
            if (editableCols.length) startEdit(newRow, editableCols[0]);
        }
    }
}

function cancelEdit() {
    editing.value = null;
}

async function setRating(row, col, value) {
    const newData = { ...(row.data ?? {}), [col.code]: value || null };
    row.data = newData;
    try {
        const { row: updated, revisionId } = await api.updateRow(
            props.workspaceCode,
            props.tableCode,
            row.id,
            { data: newData },
            rowRevisions[row.id],
        );
        rowRevisions[row.id] = revisionId;
        row.updated_at = updated.updated_at;
    } catch (e) {
        notify(e.message || "Save failed");
    }
}

async function toggleCheckbox(row, col, newVal) {
    const newData = { ...(row.data ?? {}), [col.code]: newVal };
    row.data = newData;
    try {
        const { row: updated, revisionId } = await api.updateRow(
            props.workspaceCode,
            props.tableCode,
            row.id,
            { data: newData },
            rowRevisions[row.id],
        );
        rowRevisions[row.id] = revisionId;
        row.updated_at = updated.updated_at;
    } catch (e) {
        row.data = { ...(row.data ?? {}), [col.code]: !newVal };
    }
}

async function addRow() {
    const row = await api.createRow(props.workspaceCode, props.tableCode, {
        data: {},
    });
    rows.value.push(row);
    await nextTick();
    document
        .querySelector(`td[data-row-id="${row.id}"]`)
        ?.scrollIntoView({ behavior: "smooth", block: "nearest" });
    const firstCol = visibleColumns.value.find(
        (c) =>
            !isReadOnly(c.type) && c.type !== "checkbox" && c.type !== "rating",
    );
    if (firstCol) startEdit(row, firstCol);
}

function onBulkDone(count) {
    showBulkDialog.value = false;
    notify(`${count} record${count !== 1 ? "s" : ""} updated`);
    loadRows();
}

function exportCsv() {
    const viewConfig = props.viewConfig;
    const params = {};
    if (viewConfig?.filters?.length)
        params.filters = JSON.stringify(viewConfig.filters);
    if (viewConfig?.sort?.length) params.sort = JSON.stringify(viewConfig.sort);
    const url = api.exportCsvUrl(props.workspaceCode, props.tableCode, params);
    const a = document.createElement("a");
    a.href = url;
    a.download = "";
    a.click();
}

function onImportDone(count) {
    showImportDialog.value = false;
    notify(`${count} record${count !== 1 ? "s" : ""} imported`);
    loadRows();
}

function copyRowLink(row) {
    const url = `${window.location.origin}/workspaces/${props.workspaceCode}/tables/${props.tableCode}/rows/${row.id}`;
    navigator.clipboard.writeText(url).then(() => notify("Link copied"));
}

async function confirmDeleteRow(row) {
    if (
        !(await confirm({
            title: "Delete record?",
            message:
                "This record will be permanently deleted. This cannot be undone.",
            confirmLabel: "Delete",
        }))
    )
        return;
    try {
        await api.deleteRow(props.workspaceCode, props.tableCode, row.id);
        rows.value = rows.value.filter((r) => r.id !== row.id);
    } catch (e) {
        if (e.status === 409 && e.code === "referenced") {
            showBlock({
                workspaceCode: props.workspaceCode,
                references: e.body.references,
            });
        } else {
            notify(e.message || "Delete failed");
        }
    }
}

async function duplicateRow(row) {
    const newRow = await api.createRow(props.workspaceCode, props.tableCode, {
        data: { ...row.data },
    });
    const idx = rows.value.findIndex((r) => r.id === row.id);
    rows.value.splice(idx + 1, 0, newRow);
}

// ── Column add (with optional insert context) ─────────────────────────────────

function startAddCol(context) {
    addColContext.value = context;
    showAddCol.value = true;
}

async function onAddColumn(newCol) {
    if (addColContext.value) {
        const { refColCode, dir } = addColContext.value;
        const colCodes = sortedColumns.value.map((c) => c.code);
        const refIdx = colCodes.indexOf(refColCode);
        if (refIdx !== -1) {
            const insertIdx = dir === "left" ? refIdx : refIdx + 1;
            const newOrder = [...colCodes];
            newOrder.splice(insertIdx, 0, newCol.code);
            await api.reorderColumns(
                props.workspaceCode,
                props.tableCode,
                newOrder,
            );
        }
    }

    showAddCol.value = false;
    addColContext.value = null;
    emit("reload-table");
}

function cancelAddCol() {
    showAddCol.value = false;
    addColContext.value = null;
}
</script>

<style scoped>
/* ── Column action menu ─────────────────────────────────────── */
.col-popup {
    background: var(--surface-1);
    border: 1px solid var(--border-1);
    border-radius: 8px;
    box-shadow:
        0 8px 24px rgba(0, 0, 0, 0.14),
        0 2px 6px rgba(0, 0, 0, 0.08);
    overflow: hidden;
}
.menu-item {
    display: flex;
    align-items: center;
    gap: 9px;
    width: 100%;
    padding: 7px 12px;
    font-size: 13px;
    color: var(--text-1);
    background: transparent;
    border: none;
    cursor: pointer;
    text-align: left;
    transition: background 0.1s;
}
.menu-item:hover {
    background: var(--toolbar-hover-bg);
}
.menu-item--danger {
    color: #ef4444;
}
.menu-item--danger:hover {
    background: color-mix(in srgb, #ef4444 8%, transparent);
}
.menu-item--disabled {
    opacity: 0.35;
    cursor: default;
}
.menu-item--disabled:hover {
    background: transparent;
}
.menu-sep {
    height: 1px;
    background: var(--border-1);
    margin: 2px 0;
}
</style>
