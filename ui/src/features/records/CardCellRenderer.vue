<template>
    <template v-if="hasValue(row, field.col)">
        <div
            v-if="field.showLabel"
            class="truncate mb-0.5"
            :class="
                field.style === 'title'
                    ? compact
                        ? 'text-[11px] font-semibold text-zinc-800 dark:text-[#e0e0e0]'
                        : 'text-[12px] font-semibold text-zinc-800 dark:text-[#e0e0e0]'
                    : compact
                      ? 'text-[10px] text-text-3'
                      : 'text-[10px] font-medium text-text-3 uppercase tracking-wide'
            "
        >
            {{ field.col.name }}
        </div>

        <PillBadge
            v-if="field.col.type === 'single-select'"
            :col="field.col"
            :value="getCellValue(row, field.col)"
            :class="['px-1.5', field.style === 'title' ? 'text-[13px]' : '']"
        />

        <div
            v-else-if="field.col.type === 'multi-select'"
            class="flex flex-wrap gap-1"
        >
            <PillBadge
                v-for="v in [getCellValue(row, field.col)].flat()"
                :key="v"
                :col="field.col"
                :value="v"
                class="px-1.5"
            />
        </div>

        <span
            v-else-if="field.col.type === 'checkbox'"
            :class="[
                field.style === 'title'
                    ? compact
                        ? 'text-[13px] font-semibold'
                        : 'text-[14px] font-semibold'
                    : compact
                      ? 'text-[11px]'
                      : 'text-[12px]',
                getCellValue(row, field.col)
                    ? 'text-emerald-600'
                    : 'text-text-3',
            ]"
        >
            {{
                compact
                    ? getCellValue(row, field.col)
                        ? "✓"
                        : "✗"
                    : getCellValue(row, field.col)
                      ? "✓ Yes"
                      : "✗ No"
            }}
        </span>

        <div
            v-else-if="field.col.type === 'rating'"
            :class="
                field.style === 'title'
                    ? compact
                        ? 'text-[14px]'
                        : 'text-[15px]'
                    : 'text-[13px]'
            "
            class="flex gap-px"
        >
            <span
                v-for="s in 5"
                :key="s"
                :class="
                    s <= (getCellValue(row, field.col) || 0)
                        ? 'text-amber-400'
                        : 'text-zinc-200 dark:text-[#454545]'
                "
                >★</span
            >
        </div>

        <!-- Image column — full-bleed thumbnail, no filename -->
        <div
            v-else-if="
                field.col.type === 'image' &&
                fileMeta(row, field.col) &&
                isImageMime(fileMeta(row, field.col).mimeType)
            "
            :class="compact ? '-mx-2.5' : '-mx-3'"
        >
            <img
                :src="
                    thumbnailUrl(
                        workspaceCode,
                        tableCode,
                        row.id,
                        fileMeta(row, field.col).fileId,
                    )
                "
                :alt="fileMeta(row, field.col).filename"
                class="w-full object-contain block"
            />
        </div>

        <!-- File column -->
        <div
            v-else-if="field.col.type === 'file' && fileMeta(row, field.col)"
            class="flex items-center gap-1.5"
        >
            <span :class="compact ? 'text-[12px]' : 'text-[14px]'">📄</span>
            <a
                :href="
                    fileUrl(
                        workspaceCode,
                        tableCode,
                        row.id,
                        fileMeta(row, field.col).fileId,
                    )
                "
                :download="fileMeta(row, field.col).filename"
                :class="compact ? 'text-[11px]' : 'text-[12px]'"
                class="text-brand-600 dark:text-brand-400 truncate hover:underline"
                @click.stop
                >{{ fileMeta(row, field.col).filename }}</a
            >
        </div>

        <!-- Symbol / Emoji -->
        <div
            v-else-if="
                field.col.type === 'symbol' || field.col.type === 'emoji'
            "
        >
            <span
                :class="field.style === 'title' ? 'text-[18px]' : 'text-[14px]'"
                class="leading-none"
                :title="getEmoji(getCellValue(row, field.col))?.label"
                >{{ resolveEmoji(getCellValue(row, field.col)) }}</span
            >
        </div>

        <div
            v-else-if="
                field.col.type === 'checklist' &&
                getCellValue(row, field.col)?.length
            "
        >
            <div class="flex items-center gap-1 mb-1">
                <span class="text-[11px] text-text-2">
                    {{
                        getCellValue(row, field.col).filter((i) => i.checked)
                            .length
                    }}
                    / {{ getCellValue(row, field.col).length }}
                </span>
                <div
                    class="flex-1 h-1 bg-border-1 rounded-full overflow-hidden"
                >
                    <div
                        class="h-full bg-brand-500 rounded-full"
                        :style="{
                            width:
                                (getCellValue(row, field.col).filter(
                                    (i) => i.checked,
                                ).length /
                                    getCellValue(row, field.col).length) *
                                    100 +
                                '%',
                        }"
                    />
                </div>
            </div>
            <div
                v-for="item in getCellValue(row, field.col).slice(0, 3)"
                :key="item.text"
                class="flex items-center gap-1 text-[11px]"
            >
                <span
                    :class="
                        item.checked
                            ? 'text-brand-500'
                            : 'text-zinc-300 dark:text-[#555]'
                    "
                    >{{ item.checked ? "✓" : "○" }}</span
                >
                <span
                    :class="
                        item.checked
                            ? 'line-through text-text-3'
                            : 'text-text-1'
                    "
                    class="truncate"
                    >{{ item.text }}</span
                >
            </div>
            <div
                v-if="getCellValue(row, field.col).length > 3"
                class="text-[10px] text-text-3 mt-0.5"
            >
                +{{ getCellValue(row, field.col).length - 3 }} more
            </div>
        </div>

        <div
            v-else-if="
                field.col.type === 'row-link' && getCellValue(row, field.col)
            "
            class="flex items-center gap-1"
        >
            <RiNodeTree size="11" class="text-text-3 shrink-0" />
            <span
                :class="[
                    field.style === 'title'
                        ? compact
                            ? 'text-[13px] font-semibold text-zinc-900 dark:text-[#eaeaea]'
                            : 'text-[14px] font-semibold text-zinc-900 dark:text-[#eaeaea]'
                        : 'text-[12px] text-text-1',
                    'truncate flex-1',
                ]"
                >{{ getCellValue(row, field.col).label }}</span
            >
            <button
                class="shrink-0 flex items-center justify-center w-4 h-4 rounded text-text-3 hover:text-brand-600 bg-transparent border-none cursor-pointer transition-colors"
                title="Open linked record"
                @click.stop="
                    navigateToLinked(field.col, getCellValue(row, field.col).id)
                "
            >
                <RiArrowRightUpLine size="11" />
            </button>
        </div>

        <div
            v-else
            :class="[
                field.style === 'title'
                    ? compact
                        ? 'text-[14px] font-semibold text-zinc-900 dark:text-[#eaeaea]'
                        : 'text-[15px] font-semibold text-zinc-900 dark:text-[#eaeaea]'
                    : 'text-[12px] text-text-1',
                field.truncate !== false
                    ? compact
                        ? 'line-clamp-3'
                        : 'line-clamp-4'
                    : '',
                'wrap-break-word',
            ]"
        >
            {{ displayValue(row, field.col) }}
        </div>
    </template>
</template>

<script setup>
import { useRoute, useRouter } from "vue-router";
import { fileUrl, thumbnailUrl } from "../../api/client.js";
import { getCellValue, hasValue, displayValue } from "./cellHelpers.js";
import { getEmoji, resolveEmoji } from "../views/emojis.js";
import { RiNodeTree, RiArrowRightUpLine } from "@remixicon/vue";
import PillBadge from "../../foundation/PillBadge.vue";

function fileMeta(row, col) {
    const v = getCellValue(row, col);
    return v && typeof v === "object" && v.fileId ? v : null;
}
function isImageMime(mime) {
    return mime && mime.startsWith("image/");
}

const props = defineProps({
    row: { type: Object, required: true },
    field: { type: Object, required: true },
    compact: { type: Boolean, default: false },
    workspaceCode: { type: String, required: true },
    tableCode: { type: String, required: true },
});

const route = useRoute();
const router = useRouter();

function rowLinkTargetCode(col) {
    const opts =
        typeof col.options === "string" ? JSON.parse(col.options) : col.options;
    return opts?.targetTableCode ?? props.tableCode;
}

function navigateToLinked(col, rowId) {
    router.push(
        `/workspaces/${props.workspaceCode}/tables/${rowLinkTargetCode(col)}/rows/${rowId}?returnTo=${encodeURIComponent(route.fullPath)}`,
    );
}
</script>
