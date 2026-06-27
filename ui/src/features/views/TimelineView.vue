<template>
    <div
        class="flex flex-col flex-1 min-h-0 overflow-hidden bg-zinc-100 dark:bg-[#1e1e1e]"
    >
        <!-- Toolbar -->
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
            <template #before>
                <template v-if="dateColumns.length > 0">
                    <RiCalendarLine size="12" class="text-text-3 shrink-0" />
                    <select
                        v-model="activeDateColCode"
                        class="tl-ctl"
                        @change="onDateColChange"
                    >
                        <option
                            v-for="col in dateColumns"
                            :key="col.code"
                            :value="col.code"
                        >
                            {{ col.name }}
                        </option>
                    </select>

                    <template v-if="selectColumns.length > 0">
                        <span class="text-[11px] text-text-2 shrink-0"
                            >Swimlanes</span
                        >
                        <select
                            v-model="localGroupByCol"
                            class="tl-ctl"
                            @change="onGroupByChange"
                        >
                            <option value="">None</option>
                            <option
                                v-for="col in selectColumns"
                                :key="col.code"
                                :value="col.code"
                            >
                                {{ col.name }}
                            </option>
                        </select>
                    </template>

                    <!-- Weekday visibility flapout -->
                    <Flapout align="left">
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
                                <RiCalendarLine size="12" />
                                Days
                                <span
                                    v-if="!allDaysVisible"
                                    class="text-[10px] font-bold"
                                    >{{ visibleWeekdaySet.size }}/7</span
                                >
                            </button>
                        </template>
                        <PanelCard class="p-3 w-52">
                            <div
                                class="text-[10px] font-semibold text-text-3 mb-2 uppercase tracking-wide"
                            >
                                Visible days
                            </div>
                            <div class="flex items-center gap-1">
                                <button
                                    v-for="(label, i) in DOW_SHORT"
                                    :key="i"
                                    class="flex-1 h-7 text-[11px] font-semibold rounded border transition-colors cursor-pointer"
                                    :class="
                                        visibleWeekdaySet.has(i)
                                            ? 'bg-brand-500 text-white border-brand-500 hover:bg-brand-600 hover:border-brand-600'
                                            : 'bg-surface-1 text-text-3 border-border-1 hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#555]'
                                    "
                                    :title="DOW_NAMES[i]"
                                    @click="toggleWeekday(i)"
                                >
                                    {{ label }}
                                </button>
                            </div>
                            <div class="flex gap-1.5 mt-2.5">
                                <button
                                    class="tl-preset-btn"
                                    @click="setPreset([0, 1, 2, 3, 4])"
                                >
                                    Workweek
                                </button>
                                <button
                                    class="tl-preset-btn"
                                    @click="setPreset([0, 1, 2, 3, 4, 5, 6])"
                                >
                                    All days
                                </button>
                            </div>
                        </PanelCard>
                    </Flapout>

                    <select
                        :value="zoom"
                        class="tl-ctl"
                        @change="setZoom($event.target.value)"
                    >
                        <option value="week">Week</option>
                        <option value="month">Month</option>
                        <option value="schedule">Schedule</option>
                        <option value="overview">Overview</option>
                    </select>

                    <div class="flex items-center gap-0.5">
                        <button
                            class="tl-nav-btn"
                            title="Previous"
                            @click="navigatePrev"
                        >
                            <RiArrowLeftSLine size="14" />
                        </button>
                        <button class="tl-today-btn" @click="navigateToday">
                            Today
                        </button>
                        <button
                            class="tl-nav-btn"
                            title="Next"
                            @click="navigateNext"
                        >
                            <RiArrowRightSLine size="14" />
                        </button>
                    </div>

                    <span
                        class="text-[12px] font-medium text-text-1 whitespace-nowrap min-w-20"
                        >{{ windowLabel }}</span
                    >
                </template>
                <span v-else class="text-[12px] text-text-3"
                    >No date columns</span
                >
            </template>

            <template #fields>
                <CardFieldsFlapout
                    :columns="sortedColumns"
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

        <!-- No date columns placeholder -->
        <div
            v-if="dateColumns.length === 0"
            class="flex flex-col items-center justify-center flex-1 gap-3 text-center p-8"
        >
            <RiCalendarLine
                size="40"
                class="text-zinc-300 dark:text-[#454545]"
            />
            <div>
                <p class="text-[14px] font-medium text-text-2">
                    No date columns
                </p>
                <p class="text-[12px] text-text-3 mt-1">
                    Add a date or datetime column to use the timeline view.
                </p>
            </div>
        </div>

        <template v-else>
            <!-- River -->
            <div
                ref="scrollContainer"
                class="flex-1 overflow-auto"
                @scroll="onScroll"
            >
                <!-- Overview: one column per workweek -->
                <template v-if="zoom === 'overview'">
                    <!-- Overview with swimlanes -->
                    <template v-if="swimlanes">
                        <div
                            :style="{
                                minWidth: totalWidth + LANE_LABEL_W + 'px',
                                minHeight: '100%',
                            }"
                        >
                            <!-- Sticky week header row -->
                            <div class="flex sticky top-0 z-20">
                                <div
                                    class="shrink-0 h-16 bg-zinc-100 dark:bg-[#1e1e1e] border-b border-r border-border-1"
                                    :style="{ width: LANE_LABEL_W + 'px' }"
                                />
                                <div
                                    v-for="week in visibleWeeks"
                                    :key="week.key"
                                    class="shrink-0 h-16 flex flex-col items-center justify-center bg-surface-1 border-b border-r border-border-1 last:border-r-0"
                                    :class="
                                        week.isCurrentWeek
                                            ? 'bg-brand-50/60 dark:bg-brand-900/20'
                                            : ''
                                    "
                                    :style="{
                                        width: weekColWidth(week) + 'px',
                                    }"
                                >
                                    <span
                                        v-if="week.showMonthLabel"
                                        class="text-[7px] font-bold uppercase tracking-wide leading-none"
                                        style="color: var(--color-brand-500)"
                                        >{{ week.monthName }}</span
                                    >
                                    <span
                                        v-else
                                        class="text-[7px] leading-none opacity-0"
                                        >X</span
                                    >
                                    <span
                                        class="font-semibold text-center leading-tight mt-0.5 px-0.5"
                                        :class="[
                                            week.hasCards
                                                ? 'text-[11px]'
                                                : 'text-[9px]',
                                            week.isCurrentWeek
                                                ? 'text-brand-500'
                                                : 'text-text-1',
                                        ]"
                                        >{{
                                            week.hasCards
                                                ? week.label
                                                : `${week.monday.getDate()}–${week.friday.getDate()}`
                                        }}</span
                                    >
                                </div>
                            </div>
                            <!-- Swimlane rows -->
                            <div
                                v-for="lane in swimlanes"
                                :key="lane.key"
                                class="flex border-b border-dashed border-border-1/60 last:border-b-0"
                            >
                                <!-- Lane label (sticky left) -->
                                <div
                                    class="shrink-0 sticky left-0 z-10 bg-zinc-100 dark:bg-[#1e1e1e] border-r border-border-1/50 flex items-start p-2"
                                    :style="{ width: LANE_LABEL_W + 'px' }"
                                >
                                    <span
                                        v-if="lane.value != null"
                                        class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium truncate max-w-full"
                                        :style="lane.pillStyle"
                                        >{{ lane.label }}</span
                                    >
                                    <span
                                        v-else
                                        class="text-[11px] font-medium text-text-3 italic"
                                        >No value</span
                                    >
                                </div>
                                <!-- Week cells -->
                                <div
                                    v-for="week in visibleWeeks"
                                    :key="week.key"
                                    class="shrink-0 flex flex-col min-h-12 transition-colors"
                                    :class="[
                                        week.isCurrentWeek
                                            ? 'bg-brand-50/20 dark:bg-brand-900/10'
                                            : '',
                                        dragOverKey ===
                                        lane.key + ':' + week.key
                                            ? 'ring-2 ring-inset ring-brand-400 dark:ring-brand-600 bg-brand-50/60 dark:bg-brand-900/20'
                                            : '',
                                    ]"
                                    :style="{
                                        width: weekColWidth(week) + 'px',
                                    }"
                                    @dragover.prevent="
                                        dragOverKey = lane.key + ':' + week.key
                                    "
                                    @dragleave="
                                        onCellDragLeave(
                                            $event,
                                            lane.key + ':' + week.key,
                                        )
                                    "
                                    @drop.prevent="onDropOnWeek($event, week)"
                                >
                                    <div
                                        v-if="
                                            (
                                                lane.rowsByWeek.get(week.key) ??
                                                []
                                            ).length > 0
                                        "
                                        class="flex flex-col gap-1 p-1.5"
                                    >
                                        <div
                                            v-for="row in lane.rowsByWeek.get(
                                                week.key,
                                            )"
                                            :key="row.id"
                                            draggable="true"
                                            class="bg-surface-1 border border-border-1 rounded p-1.5 cursor-grab active:cursor-grabbing hover:border-brand-300 dark:hover:border-brand-600 hover:shadow-sm transition-all"
                                            @dragstart="
                                                onDragStart($event, row)
                                            "
                                            @dragend="onDragEnd"
                                            @click="$emit('open-row', { row })"
                                        >
                                            <template
                                                v-for="field in visibleCardFields"
                                                :key="field.col.id"
                                            >
                                                <div
                                                    v-if="
                                                        hasValue(row, field.col)
                                                    "
                                                    class="mb-1 last:mb-0"
                                                >
                                                    <CardCellRenderer
                                                        :row="row"
                                                        :field="field"
                                                        :compact="true"
                                                        :workspace-code="
                                                            workspaceCode
                                                        "
                                                        :table-code="tableCode"
                                                    />
                                                </div>
                                            </template>
                                            <div
                                                v-if="
                                                    !visibleCardFields.some(
                                                        (f) =>
                                                            hasValue(
                                                                row,
                                                                f.col,
                                                            ),
                                                    )
                                                "
                                                class="text-[11px] text-zinc-300 dark:text-[#555] italic"
                                            >
                                                Empty
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </template>
                    <!-- Overview without swimlanes -->
                    <template v-else>
                        <div
                            class="flex items-start"
                            :style="{
                                minWidth: totalWidth + 'px',
                                minHeight: '100%',
                            }"
                        >
                            <div
                                v-for="week in visibleWeeks"
                                :key="week.key"
                                class="shrink-0 flex flex-col border-r border-border-1 last:border-r-0 transition-colors"
                                :class="[
                                    week.isCurrentWeek
                                        ? 'bg-brand-50/40 dark:bg-brand-900/10'
                                        : 'bg-surface-1',
                                    dragOverKey === week.key
                                        ? 'ring-2 ring-inset ring-brand-400 dark:ring-brand-600 bg-brand-50/60 dark:bg-brand-900/20'
                                        : '',
                                ]"
                                :style="{ width: weekColWidth(week) + 'px' }"
                                @dragover.prevent="dragOverKey = week.key"
                                @dragleave="onDayDragLeave($event, week)"
                                @drop.prevent="onDropOnWeek($event, week)"
                            >
                                <!-- Week header -->
                                <div
                                    class="h-16 flex flex-col items-center justify-center bg-surface-1 border-b border-border-1 shrink-0"
                                    :class="
                                        week.isCurrentWeek
                                            ? 'bg-brand-50/60 dark:bg-brand-900/20'
                                            : ''
                                    "
                                >
                                    <span
                                        v-if="week.showMonthLabel"
                                        class="text-[7px] font-bold uppercase tracking-wide leading-none"
                                        style="color: var(--color-brand-500)"
                                        >{{ week.monthName }}</span
                                    >
                                    <span
                                        v-else
                                        class="text-[7px] leading-none opacity-0"
                                        >X</span
                                    >
                                    <span
                                        class="font-semibold text-center leading-tight mt-0.5 px-0.5"
                                        :class="[
                                            week.hasCards
                                                ? 'text-[11px]'
                                                : 'text-[9px]',
                                            week.isCurrentWeek
                                                ? 'text-brand-500'
                                                : 'text-text-1',
                                        ]"
                                        >{{
                                            week.hasCards
                                                ? week.label
                                                : `${week.monday.getDate()}–${week.friday.getDate()}`
                                        }}</span
                                    >
                                </div>
                                <!-- Cards -->
                                <div
                                    v-if="week.hasCards"
                                    class="flex flex-col gap-1 p-1.5"
                                >
                                    <div
                                        v-for="row in week.rows"
                                        :key="row.id"
                                        draggable="true"
                                        class="bg-surface-1 border border-border-1 rounded p-1.5 cursor-grab active:cursor-grabbing hover:border-brand-300 dark:hover:border-brand-600 hover:shadow-sm transition-all"
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
                                                class="mb-1 last:mb-0"
                                            >
                                                <CardCellRenderer
                                                    :row="row"
                                                    :field="field"
                                                    :compact="true"
                                                    :workspace-code="
                                                        workspaceCode
                                                    "
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
                            </div>
                        </div>
                    </template>
                </template>

                <!-- No group-by: density river -->
                <template v-else-if="!swimlanes">
                    <div
                        class="flex items-start"
                        :style="{
                            minWidth: totalWidth + 'px',
                            minHeight: '100%',
                        }"
                    >
                        <div
                            v-for="day in visibleDays"
                            :key="day.key"
                            class="shrink-0 flex flex-col border-r border-border-1 last:border-r-0 transition-colors"
                            :class="[
                                day.isToday
                                    ? 'bg-brand-50/40 dark:bg-brand-900/10'
                                    : day.isWeekend
                                      ? 'bg-zinc-50 dark:bg-[#1c1c1c]'
                                      : 'bg-surface-1',
                                dragOverKey === day.key
                                    ? 'ring-2 ring-inset ring-brand-400 dark:ring-brand-600 bg-brand-50/60 dark:bg-brand-900/20'
                                    : '',
                            ]"
                            :style="{ width: dayColWidth(day) + 'px' }"
                            @dragover.prevent="dragOverKey = day.key"
                            @dragleave="onDayDragLeave($event, day)"
                            @drop.prevent="onDrop($event, day)"
                        >
                            <!-- Day header — fixed height so all columns align -->
                            <div
                                class="h-16 flex flex-col items-center justify-center bg-surface-1 border-b border-border-1 shrink-0"
                                :class="
                                    day.isToday
                                        ? 'bg-brand-50/60 dark:bg-brand-900/20'
                                        : ''
                                "
                            >
                                <span
                                    class="text-[9px] font-semibold uppercase tracking-wide leading-none"
                                    :class="
                                        day.isToday
                                            ? 'text-brand-400'
                                            : 'text-text-3'
                                    "
                                >
                                    {{
                                        day.hasCards
                                            ? day.dayName
                                            : day.dayName.slice(0, 1)
                                    }}
                                </span>
                                <span
                                    class="font-semibold flex items-center justify-center rounded-full mt-0.5"
                                    :class="[
                                        day.hasCards
                                            ? 'text-[15px] w-7 h-7'
                                            : 'text-[11px] w-5 h-5',
                                        day.isToday
                                            ? 'bg-brand-500 text-white'
                                            : 'text-text-1',
                                    ]"
                                    >{{ day.date.getDate() }}</span
                                >
                                <span
                                    v-if="day.showMonthLabel"
                                    class="font-bold leading-none mt-0.5"
                                    :class="
                                        day.hasCards
                                            ? 'text-[9px]'
                                            : 'text-[7px]'
                                    "
                                    style="color: var(--color-brand-500)"
                                    >{{
                                        day.hasCards
                                            ? day.monthName
                                            : day.monthName
                                                  .slice(0, 3)
                                                  .toUpperCase()
                                    }}</span
                                >
                                <span
                                    v-else
                                    class="text-[7px] leading-none mt-0.5 opacity-0"
                                    >X</span
                                >
                            </div>

                            <!-- Cards -->
                            <div
                                v-if="day.hasCards"
                                class="flex flex-col gap-1 p-1.5"
                            >
                                <div
                                    v-for="row in day.rows"
                                    :key="row.id"
                                    draggable="true"
                                    class="bg-surface-1 border border-border-1 rounded p-1.5 cursor-grab active:cursor-grabbing hover:border-brand-300 dark:hover:border-brand-600 hover:shadow-sm transition-all"
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
                                            class="mb-1 last:mb-0"
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
                        </div>
                    </div>
                </template>

                <!-- With group-by: swimlane layout -->
                <template v-else>
                    <div
                        :style="{ minWidth: totalWidth + LANE_LABEL_W + 'px' }"
                    >
                        <!-- Shared day header row (sticky top) -->
                        <div class="flex sticky top-0 z-20">
                            <div
                                class="shrink-0 h-16 bg-zinc-100 dark:bg-[#1e1e1e] border-b border-r border-border-1"
                                :style="{ width: LANE_LABEL_W + 'px' }"
                            />
                            <div
                                v-for="day in visibleDays"
                                :key="day.key"
                                class="shrink-0 h-16 flex flex-col items-center justify-center bg-surface-1 border-b border-r border-border-1 last:border-r-0"
                                :class="
                                    day.isToday
                                        ? 'bg-brand-50/60 dark:bg-brand-900/20'
                                        : ''
                                "
                                :style="{ width: dayColWidth(day) + 'px' }"
                            >
                                <span
                                    class="text-[9px] font-semibold uppercase tracking-wide leading-none"
                                    :class="
                                        day.isToday
                                            ? 'text-brand-400'
                                            : 'text-text-3'
                                    "
                                >
                                    {{
                                        day.hasCards
                                            ? day.dayName
                                            : day.dayName.slice(0, 1)
                                    }}
                                </span>
                                <span
                                    class="font-semibold flex items-center justify-center rounded-full mt-0.5"
                                    :class="[
                                        day.hasCards
                                            ? 'text-[15px] w-7 h-7'
                                            : 'text-[11px] w-5 h-5',
                                        day.isToday
                                            ? 'bg-brand-500 text-white'
                                            : 'text-text-1',
                                    ]"
                                    >{{ day.date.getDate() }}</span
                                >
                                <span
                                    v-if="day.showMonthLabel"
                                    class="font-bold leading-none mt-0.5"
                                    :class="
                                        day.hasCards
                                            ? 'text-[9px]'
                                            : 'text-[7px]'
                                    "
                                    style="color: var(--color-brand-500)"
                                    >{{
                                        day.hasCards
                                            ? day.monthName
                                            : day.monthName
                                                  .slice(0, 3)
                                                  .toUpperCase()
                                    }}</span
                                >
                                <span
                                    v-else
                                    class="text-[7px] leading-none mt-0.5 opacity-0"
                                    >X</span
                                >
                            </div>
                        </div>

                        <!-- Swimlane rows -->
                        <div
                            v-for="lane in swimlanes"
                            :key="lane.key"
                            class="flex border-b border-dashed border-border-1/60 last:border-b-0"
                        >
                            <!-- Lane label (sticky left) -->
                            <div
                                class="shrink-0 sticky left-0 z-10 bg-zinc-100 dark:bg-[#1e1e1e] border-r border-border-1/50 flex items-start p-2"
                                :style="{ width: LANE_LABEL_W + 'px' }"
                            >
                                <span
                                    v-if="lane.value != null"
                                    class="inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-medium truncate max-w-full"
                                    :style="lane.pillStyle"
                                    >{{ lane.label }}</span
                                >
                                <span
                                    v-else
                                    class="text-[11px] font-medium text-text-3 italic"
                                    >No value</span
                                >
                            </div>

                            <!-- Day cells -->
                            <div
                                v-for="day in visibleDays"
                                :key="day.key"
                                class="shrink-0 flex flex-col min-h-12 transition-colors"
                                :class="[
                                    day.isToday
                                        ? 'bg-brand-50/20 dark:bg-brand-900/10'
                                        : '',
                                    dragOverKey === lane.key + ':' + day.key
                                        ? 'ring-2 ring-inset ring-brand-400 dark:ring-brand-600 bg-brand-50/60 dark:bg-brand-900/20'
                                        : '',
                                ]"
                                :style="{ width: dayColWidth(day) + 'px' }"
                                @dragover.prevent="
                                    dragOverKey = lane.key + ':' + day.key
                                "
                                @dragleave="
                                    onCellDragLeave(
                                        $event,
                                        lane.key + ':' + day.key,
                                    )
                                "
                                @drop.prevent="onDrop($event, day)"
                            >
                                <div
                                    v-if="
                                        (lane.rowsByDate.get(day.key) ?? [])
                                            .length > 0
                                    "
                                    class="flex flex-col gap-1 p-1.5"
                                >
                                    <div
                                        v-for="row in lane.rowsByDate.get(
                                            day.key,
                                        )"
                                        :key="row.id"
                                        draggable="true"
                                        class="bg-surface-1 border border-border-1 rounded p-1.5 cursor-grab active:cursor-grabbing hover:border-brand-300 dark:hover:border-brand-600 hover:shadow-sm transition-all"
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
                                                class="mb-1 last:mb-0"
                                            >
                                                <CardCellRenderer
                                                    :row="row"
                                                    :field="field"
                                                    :compact="true"
                                                    :workspace-code="
                                                        workspaceCode
                                                    "
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
                            </div>
                        </div>
                    </div>
                </template>
            </div>

            <!-- Unscheduled drawer -->
            <div
                v-if="unscheduledRows.length > 0"
                class="shrink-0 border-t border-border-1 bg-surface-1"
            >
                <button
                    class="w-full flex items-center gap-1.5 px-4 py-1.5 text-left bg-transparent border-none cursor-pointer hover:bg-zinc-50 dark:hover:bg-[#252526] transition-colors"
                    @click="unscheduledOpen = !unscheduledOpen"
                >
                    <RiArrowRightSLine
                        size="12"
                        class="text-text-3 transition-transform duration-150 shrink-0"
                        :class="unscheduledOpen ? 'rotate-90' : ''"
                    />
                    <span
                        class="text-[10px] font-semibold text-text-3 uppercase tracking-wider"
                    >
                        Unscheduled ({{ unscheduledRows.length }})
                    </span>
                </button>
                <div
                    v-if="unscheduledOpen"
                    class="flex flex-wrap gap-1.5 px-4 pb-2"
                >
                    <div
                        v-for="row in unscheduledRows"
                        :key="row.id"
                        draggable="true"
                        class="bg-surface-1 border border-border-1 rounded p-1.5 cursor-grab active:cursor-grabbing hover:border-brand-300 dark:hover:border-brand-600 hover:shadow-sm transition-all"
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
                                class="mb-0.5 last:mb-0"
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
                            class="text-[10px] text-zinc-300 dark:text-[#555] italic"
                        >
                            Empty
                        </div>
                    </div>
                </div>
            </div>

            <!-- Load more -->
            <div
                v-if="hasMore"
                class="shrink-0 flex justify-center p-3 border-t border-border-1 bg-surface-1"
            >
                <button
                    class="inline-flex items-center gap-1.5 px-4 py-1.5 text-[12px] font-medium text-text-2 bg-surface-1 border border-border-1 rounded-md cursor-pointer hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#555] transition-colors disabled:opacity-50"
                    :disabled="loadingMore"
                    @click="loadMore"
                >
                    {{ loadingMore ? "Loading…" : "Load more" }}
                </button>
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
import { ref, computed, watch, onMounted, nextTick } from "vue";
import { api } from "../../api/client.js";
import { getCellValue, hasValue } from "../records/cellHelpers.js";
import { useViewRows } from "../records/useViewRows.js";
import { useSortedColumns } from "../records/useSortedColumns.js";
import { useFieldReordering } from "../records/useFieldReordering.js";
import { useCardFields } from "./useCardFields.js";
import { useNotifications } from "../../foundation/useNotifications.js";
import TableToolbar from "./TableToolbar.vue";
import CardFieldsFlapout from "./CardFieldsFlapout.vue";
import BulkActionDialog from "./BulkActionDialog.vue";
import ImportCsvDialog from "./ImportCsvDialog.vue";
import { getChoiceColor, textColorForBg } from "./palettes.js";
import CardCellRenderer from "../records/CardCellRenderer.vue";
import Flapout from "../../foundation/Flapout.vue";
import PanelCard from "../../foundation/PanelCard.vue";
import {
    RiCalendarLine,
    RiArrowLeftSLine,
    RiArrowRightSLine,
} from "@remixicon/vue";

// Week: 80px empty cols (more breathing room, fewer days on screen)
// Month: 52px empty cols (tighter, more timeline visible)
const CARD_COL_W = 200;
const EMPTY_COL_W = { week: 80, month: 52, schedule: 52, overview: 48 };
const RENDER_DAYS = 90;
const RENDER_WEEKS = 26;
const LANE_LABEL_W = 120;

const props = defineProps({
    workspaceCode: { type: String, required: true },
    tableCode: { type: String, required: true },
    table: { type: Object, required: true },
    viewConfig: { type: Object, default: () => ({}) },
});

const emit = defineEmits([
    "open-row",
    "reload-table",
    "date-col-change",
    "timeline-zoom-change",
    "weekdays-change",
    "card-fields-change",
    "column-order-change",
    "sort-change",
    "filter-change",
    "group-by-change",
]);

// ── Date helpers ──────────────────────────────────────────────────────────

function addDays(date, n) {
    const d = new Date(date);
    d.setDate(d.getDate() + n);
    return d;
}

function sameDay(a, b) {
    return (
        a.getFullYear() === b.getFullYear() &&
        a.getMonth() === b.getMonth() &&
        a.getDate() === b.getDate()
    );
}

function formatDateValue(date) {
    const col = activeDateCol.value;
    if (!col || col.type === "created-at" || col.type === "updated-at")
        return null;
    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, "0");
    const d = String(date.getDate()).padStart(2, "0");
    if (col.type === "date") return `${y}-${m}-${d}`;
    return `${y}-${m}-${d}T00:00:00.000Z`;
}

const MONTHS = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec",
];
const DOW_NAMES = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
const DOW_SHORT = ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"];
const ALL_WEEKDAYS = [0, 1, 2, 3, 4, 5, 6];

// ── Date columns ──────────────────────────────────────────────────────────

const dateColumns = computed(() =>
    (props.table.columns ?? []).filter((c) =>
        ["date", "datetime", "created-at", "updated-at"].includes(c.type),
    ),
);

const activeDateColCode = ref(props.viewConfig?.dateCol ?? null);

watch(
    dateColumns,
    (cols) => {
        if (!activeDateColCode.value && cols.length > 0)
            activeDateColCode.value = cols[0].code;
    },
    { immediate: true },
);

watch(
    () => props.viewConfig?.dateCol,
    (v) => {
        if (v) activeDateColCode.value = v;
    },
);

const activeDateCol = computed(
    () =>
        (props.table.columns ?? []).find(
            (c) => c.code === activeDateColCode.value,
        ) ?? null,
);

function onDateColChange() {
    emit("date-col-change", activeDateColCode.value);
}

// ── Group by ──────────────────────────────────────────────────────────────

const selectColumns = computed(() =>
    (props.table.columns ?? []).filter((c) => c.type === "single-select"),
);

const localGroupByCol = ref(props.viewConfig?.groupByCol ?? null);

watch(
    () => props.viewConfig?.groupByCol,
    (v) => {
        localGroupByCol.value = v ?? null;
    },
);

const groupByCol = computed(
    () =>
        (props.table.columns ?? []).find(
            (c) => c.code === localGroupByCol.value,
        ) ?? null,
);

const groupByChoices = computed(() => {
    if (!groupByCol.value) return [];
    const opts =
        typeof groupByCol.value.options === "string"
            ? JSON.parse(groupByCol.value.options)
            : groupByCol.value.options;
    return opts?.choices ?? [];
});

function onGroupByChange() {
    emit("group-by-change", localGroupByCol.value || null);
}

// ── Field selection ───────────────────────────────────────────────────────

const sortedColumns = useSortedColumns(
    computed(() => props.table.columns ?? []),
    computed(() => props.viewConfig?.columnOrder),
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
    sortedColumns,
    emit,
);

// ── Zoom ──────────────────────────────────────────────────────────────────

const zoom = ref(props.viewConfig?.timelineZoom ?? "week");

watch(
    () => props.viewConfig?.timelineZoom,
    (v) => {
        if (v) zoom.value = v;
    },
);

const emptyColWidth = computed(() => EMPTY_COL_W[zoom.value]);

function dayColWidth(day) {
    return day.hasCards ? CARD_COL_W : emptyColWidth.value;
}

// ── Visible weekdays ──────────────────────────────────────────────────────

// Mon-first indices: 0=Mon 1=Tue 2=Wed 3=Thu 4=Fri 5=Sat 6=Sun
const visibleWeekdaySet = ref(
    new Set(props.viewConfig?.visibleWeekdays ?? ALL_WEEKDAYS),
);

watch(
    () => props.viewConfig?.visibleWeekdays,
    (v) => {
        visibleWeekdaySet.value = new Set(v ?? ALL_WEEKDAYS);
    },
);

const allDaysVisible = computed(() => visibleWeekdaySet.value.size === 7);

function toggleWeekday(idx) {
    const next = new Set(visibleWeekdaySet.value);
    if (next.has(idx)) {
        if (next.size <= 1) return; // always keep at least one day
        next.delete(idx);
    } else {
        next.add(idx);
    }
    visibleWeekdaySet.value = next;
    emit(
        "weekdays-change",
        [...next].sort((a, b) => a - b),
    );
}

function setPreset(days) {
    visibleWeekdaySet.value = new Set(days);
    emit("weekdays-change", [...days]);
}

function setZoom(z) {
    zoom.value = z;
    emit("timeline-zoom-change", z);
    nextTick(() => scrollToToday());
}

// ── Window ────────────────────────────────────────────────────────────────

function initialWindowStart() {
    const d = new Date();
    d.setHours(0, 0, 0, 0);
    d.setDate(d.getDate() - 7);
    return d;
}

const windowStart = ref(initialWindowStart());

function navigatePrev() {
    const step =
        zoom.value === "week"
            ? -7
            : zoom.value === "overview"
              ? -(13 * 7)
              : -14;
    windowStart.value = addDays(windowStart.value, step);
    nextTick(() => {
        if (scrollContainer.value) scrollContainer.value.scrollLeft = 0;
    });
}

function navigateNext() {
    const step =
        zoom.value === "week" ? 7 : zoom.value === "overview" ? 13 * 7 : 14;
    windowStart.value = addDays(windowStart.value, step);
    nextTick(() => {
        if (scrollContainer.value) scrollContainer.value.scrollLeft = 0;
    });
}

async function navigateToday() {
    windowStart.value = initialWindowStart();
    await nextTick();
    scrollToToday();
}

// ── Scroll ────────────────────────────────────────────────────────────────

const scrollContainer = ref(null);
const scrollLeftPx = ref(0);

function onScroll(e) {
    scrollLeftPx.value = e.target.scrollLeft;
}

function scrollToToday() {
    const el = scrollContainer.value;
    if (!el) return;
    if (zoom.value === "overview") {
        let left = 0;
        for (const week of visibleWeeks.value) {
            if (week.isCurrentWeek) {
                el.scrollLeft = Math.max(0, left - 16);
                return;
            }
            left += weekColWidth(week);
        }
        return;
    }
    let left = 0;
    for (const day of visibleDays.value) {
        if (sameDay(day.date, today)) {
            el.scrollLeft = Math.max(0, left - 16);
            return;
        }
        left += dayColWidth(day);
    }
}

onMounted(() => nextTick(() => scrollToToday()));

// ── Row data ──────────────────────────────────────────────────────────────

const today = (() => {
    const d = new Date();
    d.setHours(0, 0, 0, 0);
    return d;
})();

const { notify } = useNotifications();
const showBulkDialog = ref(false);
const showImportDialog = ref(false);

const { rows, hasMore, loadingMore, loadMore, searchDebounced } = useViewRows(
    () => props,
);

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

function parseRowDate(row) {
    if (!activeDateCol.value) return null;
    const val = getCellValue(row, activeDateCol.value);
    if (!val) return null;
    if (activeDateCol.value.type === "date") {
        const parts = String(val).split("-");
        if (parts.length === 3)
            return new Date(+parts[0], +parts[1] - 1, +parts[2]);
    }
    const d = new Date(val);
    if (isNaN(d.getTime())) return null;
    return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

const scheduledRows = computed(() =>
    rows.value.filter((r) => parseRowDate(r) !== null),
);
const unscheduledRows = computed(() =>
    rows.value.filter((r) => parseRowDate(r) === null),
);

const rowsByDate = computed(() => {
    const map = new Map();
    for (const row of scheduledRows.value) {
        const date = parseRowDate(row);
        if (!date) continue;
        const key = `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
        if (!map.has(key)) map.set(key, []);
        map.get(key).push(row);
    }
    return map;
});

// ── Overview (workweek) helpers ───────────────────────────────────────────

function getMondayOfWeek(date) {
    const d = new Date(date);
    d.setHours(0, 0, 0, 0);
    const day = d.getDay();
    d.setDate(d.getDate() + (day === 0 ? -6 : 1 - day));
    return d;
}

function isSameWeek(monday, date) {
    return sameDay(monday, getMondayOfWeek(date));
}

function formatWeekLabel(monday, friday) {
    const sm = MONTHS[monday.getMonth()];
    const em = MONTHS[friday.getMonth()];
    if (sm === em) return `${sm} ${monday.getDate()}–${friday.getDate()}`;
    return `${sm} ${monday.getDate()} – ${em} ${friday.getDate()}`;
}

const rowsByWeek = computed(() => {
    const map = new Map();
    for (const row of scheduledRows.value) {
        const date = parseRowDate(row);
        if (!date) continue;
        const monday = getMondayOfWeek(date);
        const key = `${monday.getFullYear()}-${monday.getMonth()}-${monday.getDate()}`;
        if (!map.has(key)) map.set(key, []);
        map.get(key).push(row);
    }
    return map;
});

function weekColWidth(week) {
    return week.hasCards ? CARD_COL_W : EMPTY_COL_W.overview;
}

const visibleWeeks = computed(() => {
    if (zoom.value !== "overview") return [];
    const result = [];
    const startMonday = getMondayOfWeek(windowStart.value);
    for (let i = 0; i < RENDER_WEEKS; i++) {
        const monday = addDays(startMonday, i * 7);
        const friday = addDays(monday, 4);
        const key = `${monday.getFullYear()}-${monday.getMonth()}-${monday.getDate()}`;
        const rows = rowsByWeek.value.get(key) ?? [];
        const prev = result[result.length - 1];
        const showMonthLabel =
            i === 0 || (prev && prev.monday.getMonth() !== monday.getMonth());
        result.push({
            key,
            monday,
            friday,
            label: formatWeekLabel(monday, friday),
            showMonthLabel,
            monthName: MONTHS[monday.getMonth()],
            isCurrentWeek: isSameWeek(monday, today),
            rows,
            hasCards: rows.length > 0,
        });
    }
    return result;
});

const swimlanes = computed(() => {
    if (!groupByCol.value) return null;
    const col = groupByCol.value;
    const colCode = col.code;
    const choices = groupByChoices.value;
    const knownChoices = new Set(choices);

    function buildMaps(matchFn) {
        const byDate = new Map();
        const byWeek = new Map();
        for (const row of scheduledRows.value) {
            if (!matchFn(row)) continue;
            const date = parseRowDate(row);
            if (!date) continue;
            const dateKey = `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
            if (!byDate.has(dateKey)) byDate.set(dateKey, []);
            byDate.get(dateKey).push(row);
            const monday = getMondayOfWeek(date);
            const weekKey = `${monday.getFullYear()}-${monday.getMonth()}-${monday.getDate()}`;
            if (!byWeek.has(weekKey)) byWeek.set(weekKey, []);
            byWeek.get(weekKey).push(row);
        }
        return { byDate, byWeek };
    }

    const lanes = choices.map((choice) => {
        const bg = getChoiceColor(col, choice);
        const { byDate, byWeek } = buildMaps(
            (r) => r.data?.[colCode] === choice,
        );
        return {
            key: choice,
            value: choice,
            label: choice,
            pillStyle: { backgroundColor: bg, color: textColorForBg(bg) },
            rowsByDate: byDate,
            rowsByWeek: byWeek,
        };
    });

    let hasNoValue = false;
    for (const row of scheduledRows.value) {
        const v = row.data?.[colCode];
        if (v == null || v === "" || !knownChoices.has(v)) {
            hasNoValue = true;
            break;
        }
    }
    if (hasNoValue || lanes.length === 0) {
        const { byDate, byWeek } = buildMaps((r) => {
            const v = r.data?.[colCode];
            return v == null || v === "" || !knownChoices.has(v);
        });
        lanes.push({
            key: "__none__",
            value: null,
            label: "No value",
            pillStyle: {},
            rowsByDate: byDate,
            rowsByWeek: byWeek,
        });
    }

    return lanes;
});

const visibleDays = computed(() => {
    const result = [];
    for (let i = 0; i < RENDER_DAYS; i++) {
        const date = addDays(windowStart.value, i);
        const dow = date.getDay(); // 0=Sun … 6=Sat
        const monFirstDow = dow === 0 ? 6 : dow - 1; // 0=Mon … 6=Sun
        if (!visibleWeekdaySet.value.has(monFirstDow)) continue;
        const key = `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
        const rows = rowsByDate.value.get(key) ?? [];
        if (zoom.value === "schedule" && rows.length === 0) continue;
        const prev = result[result.length - 1];
        const showMonthLabel =
            date.getDate() === 1 ||
            result.length === 0 ||
            (prev && prev.date.getMonth() !== date.getMonth());
        result.push({
            key,
            date,
            dayName: DOW_NAMES[monFirstDow],
            monthName: MONTHS[date.getMonth()],
            isToday: sameDay(date, today),
            isWeekend: dow === 0 || dow === 6,
            showMonthLabel,
            rows,
            hasCards: rows.length > 0,
        });
    }
    return result;
});

const totalWidth = computed(() => {
    if (zoom.value === "overview")
        return visibleWeeks.value.reduce((sum, w) => sum + weekColWidth(w), 0);
    return visibleDays.value.reduce((sum, day) => sum + dayColWidth(day), 0);
});

const windowLabel = computed(() => {
    if (zoom.value === "overview") {
        let left = 0;
        for (const week of visibleWeeks.value) {
            if (left + weekColWidth(week) > scrollLeftPx.value)
                return `${week.monthName} ${week.monday.getFullYear()}`;
            left += weekColWidth(week);
        }
        return "";
    }
    let left = 0;
    for (const day of visibleDays.value) {
        if (left + dayColWidth(day) > scrollLeftPx.value)
            return `${day.monthName} ${day.date.getFullYear()}`;
        left += dayColWidth(day);
    }
    return "";
});

// ── Unscheduled drawer ────────────────────────────────────────────────────

const unscheduledOpen = ref(false);

// ── Drag and drop ─────────────────────────────────────────────────────────

const dragRow = ref(null);
const dragOverKey = ref(null);

function onDragStart(event, row) {
    dragRow.value = row;
    event.dataTransfer.effectAllowed = "move";
    // Transparent drag image so the card itself shows
    event.dataTransfer.setData("text/plain", String(row.id));
}

function onDragEnd() {
    dragRow.value = null;
    dragOverKey.value = null;
}

function onDayDragLeave(event, day) {
    // Only clear if leaving the column entirely (not moving to a child element)
    if (!event.currentTarget.contains(event.relatedTarget)) {
        if (dragOverKey.value === day.key) dragOverKey.value = null;
    }
}

function onCellDragLeave(event, cellKey) {
    if (!event.currentTarget.contains(event.relatedTarget)) {
        if (dragOverKey.value === cellKey) dragOverKey.value = null;
    }
}

async function onDrop(event, day) {
    dragOverKey.value = null;
    const row = dragRow.value;
    dragRow.value = null;
    if (!row) return;

    const newVal = formatDateValue(day.date);
    if (!newVal) return;

    // Skip if same day
    const currentDate = parseRowDate(row);
    if (currentDate && sameDay(currentDate, day.date)) return;

    // Optimistic update
    const colCode = activeDateColCode.value;
    row.data = { ...row.data, [colCode]: newVal };

    await api.patchRow(props.workspaceCode, props.tableCode, row.id, {
        data: { [colCode]: newVal },
    });
}

async function onDropOnWeek(event, week) {
    dragOverKey.value = null;
    const row = dragRow.value;
    dragRow.value = null;
    if (!row) return;
    const newVal = formatDateValue(week.monday);
    if (!newVal) return;
    const currentDate = parseRowDate(row);
    if (currentDate && isSameWeek(week.monday, currentDate)) return;
    const colCode = activeDateColCode.value;
    row.data = { ...row.data, [colCode]: newVal };
    await api.patchRow(props.workspaceCode, props.tableCode, row.id, {
        data: { [colCode]: newVal },
    });
}

// ── Add record ────────────────────────────────────────────────────────────

async function addRecord() {
    const row = await api.createRow(props.workspaceCode, props.tableCode, {
        data: {},
    });
    rows.value.push(row);
    emit("open-row", { row, isNew: true });
}
</script>

<style scoped>
.tl-ctl {
    height: 24px;
    padding: 0 6px;
    font-size: 11px;
    border-radius: 5px;
    border: 1px solid var(--border-1);
    background: var(--surface-2);
    color: var(--text-1);
    outline: none;
    cursor: pointer;
}
.tl-ctl:focus {
    border-color: var(--color-brand-600);
}

.tl-nav-btn {
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    color: var(--text-2);
    cursor: pointer;
    border-radius: 4px;
    transition:
        background 0.15s,
        color 0.15s;
}
.tl-nav-btn:hover {
    background: var(--border-1);
    color: var(--text-1);
}

.tl-today-btn {
    height: 24px;
    padding: 0 8px;
    font-size: 11px;
    font-weight: 500;
    background: transparent;
    border: 1px solid var(--border-1);
    color: var(--text-2);
    cursor: pointer;
    border-radius: 4px;
    transition:
        background 0.15s,
        color 0.15s;
}
.tl-today-btn:hover {
    background: var(--border-1);
    color: var(--text-1);
}

.tl-preset-btn {
    flex: 1;
    height: 24px;
    font-size: 11px;
    font-weight: 500;
    background: var(--surface-2);
    border: 1px solid var(--border-1);
    color: var(--text-2);
    border-radius: 4px;
    cursor: pointer;
    transition:
        background 0.15s,
        color 0.15s;
}
.tl-preset-btn:hover {
    background: var(--border-1);
    color: var(--text-1);
}
</style>
