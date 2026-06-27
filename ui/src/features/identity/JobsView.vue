<template>
    <div class="min-h-screen flex flex-col bg-zinc-100 dark:bg-[#1e1e1e]">
        <AppTopNav />

        <div class="flex-1 p-10 max-w-[960px] w-full mx-auto">
            <div class="flex items-start justify-between mb-7">
                <div>
                    <h1 class="text-[22px] font-bold text-text-1 mb-1">Jobs</h1>
                    <p class="text-[13px] text-text-2">
                        Background job runs and their outcomes.
                    </p>
                </div>
            </div>

            <div v-if="loading" class="text-sm text-text-2">Loading…</div>
            <div v-else-if="!runs.length" class="text-sm text-text-2">
                No job runs yet.
            </div>
            <div
                v-else
                class="flex flex-col divide-y divide-zinc-200 dark:divide-[#3c3c3c] border border-border-1 rounded-xl overflow-hidden"
            >
                <div
                    v-for="run in runs"
                    :key="run.id"
                    class="flex items-center gap-3 px-4 py-3 bg-surface-1"
                >
                    <div class="shrink-0">
                        <RiCheckboxCircleFill
                            v-if="run.status === 'completed'"
                            size="16"
                            class="text-green-500"
                        />
                        <RiCloseCircleFill
                            v-else-if="run.status === 'failed'"
                            size="16"
                            class="text-red-500"
                        />
                        <RiLoader4Line
                            v-else
                            size="16"
                            class="text-text-3 animate-spin"
                        />
                    </div>
                    <div class="flex-1 min-w-0">
                        <div class="text-[13px] font-medium text-text-1">
                            {{ run.job_name }}
                        </div>
                        <div class="text-[11px] text-text-2">
                            <span
                                :class="
                                    run.status === 'failed'
                                        ? 'text-red-500'
                                        : ''
                                "
                                >{{ run.outcome || run.status }}</span
                            >
                        </div>
                    </div>
                    <div class="text-[11px] text-text-2 text-right shrink-0">
                        <div>{{ formatDate(run.started_at) }}</div>
                        <div v-if="run.finished_at" class="text-text-3">
                            {{
                                formatDuration(run.started_at, run.finished_at)
                            }}
                        </div>
                    </div>
                </div>
            </div>

            <div
                v-if="totalPages > 1"
                class="flex items-center justify-center gap-2 mt-6"
            >
                <button
                    class="px-3 py-1.5 text-[13px] font-medium text-text-2 bg-surface-1 border border-border-1 rounded-md cursor-pointer hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#555] transition-colors disabled:opacity-40 disabled:cursor-default"
                    :disabled="page <= 1"
                    @click="goToPage(page - 1)"
                >
                    Previous
                </button>
                <span class="text-[13px] text-text-2 px-2"
                    >{{ page }} / {{ totalPages }}</span
                >
                <button
                    class="px-3 py-1.5 text-[13px] font-medium text-text-2 bg-surface-1 border border-border-1 rounded-md cursor-pointer hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#555] transition-colors disabled:opacity-40 disabled:cursor-default"
                    :disabled="page >= totalPages"
                    @click="goToPage(page + 1)"
                >
                    Next
                </button>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import AppTopNav from "../../foundation/AppTopNav.vue";
import { api } from "../../api/client.js";
import {
    RiCheckboxCircleFill,
    RiCloseCircleFill,
    RiLoader4Line,
} from "@remixicon/vue";

const runs = ref([]);
const loading = ref(true);
const page = ref(1);
const totalPages = ref(1);

onMounted(() => load());

async function load() {
    loading.value = true;
    try {
        const result = await api.listJobRuns({
            page: page.value,
            pageSize: 25,
        });
        runs.value = result.data ?? [];
        page.value = result.page;
        totalPages.value = result.total_pages;
    } finally {
        loading.value = false;
    }
}

function goToPage(p) {
    page.value = p;
    load();
}

function formatDate(iso) {
    return new Date(iso).toLocaleString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
    });
}

function formatDuration(start, end) {
    const ms = new Date(end) - new Date(start);
    const s = Math.floor(ms / 1000);
    if (s < 60) return `${s}s`;
    const m = Math.floor(s / 60);
    if (m < 60) return `${m}m ${s % 60}s`;
    return `${Math.floor(m / 60)}h ${m % 60}m`;
}
</script>
