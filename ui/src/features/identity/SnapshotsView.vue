<template>
    <div class="min-h-screen flex flex-col bg-zinc-100 dark:bg-[#1e1e1e]">
        <AppTopNav />

        <!-- Content -->
        <div class="flex-1 p-10 max-w-240 w-full mx-auto">
            <div class="flex items-start justify-between mb-7">
                <div>
                    <h1 class="text-[22px] font-bold text-text-1 mb-1">
                        Snapshots
                    </h1>
                    <p class="text-[13px] text-text-2">
                        Snapshots are point-in-time copies of the database.
                        Restoring a snapshot will replace all current data and
                        restart the server.
                    </p>
                </div>
                <button
                    class="inline-flex items-center gap-1.5 px-3.5 py-1.75 bg-brand-600 text-white rounded-md text-[13px] font-medium border-none cursor-pointer hover:bg-brand-700 transition-colors shrink-0 disabled:opacity-50 disabled:cursor-not-allowed"
                    :disabled="creating"
                    @click="createSnapshot"
                >
                    <RiLoader4Line
                        v-if="creating"
                        size="15"
                        class="animate-spin"
                    />
                    <RiSaveLine v-else size="15" />
                    {{ creating ? "Creating…" : "Create snapshot" }}
                </button>
            </div>

            <div v-if="loading" class="text-sm text-text-2">Loading…</div>
            <div v-else-if="!snapshots.length" class="text-sm text-text-2">
                No snapshots yet.
            </div>
            <div
                v-else
                class="flex flex-col divide-y divide-zinc-200 dark:divide-[#3c3c3c] border border-border-1 rounded-xl overflow-hidden"
            >
                <div
                    v-for="snap in snapshots"
                    :key="snap.id"
                    class="flex items-center gap-3 px-4 py-3 bg-surface-1"
                >
                    <RiDatabase2Line size="16" class="text-text-3 shrink-0" />
                    <div class="flex-1 min-w-0">
                        <div class="text-[13px] font-medium text-text-1">
                            {{ formatDate(snap.created_at) }}
                        </div>
                        <div class="text-[11px] text-text-2">
                            {{ formatSize(snap.size_bytes) }}
                        </div>
                    </div>
                    <button
                        class="flex items-center gap-1 px-2.5 py-1 text-[12px] font-medium text-text-2 bg-transparent border border-border-1 rounded-md cursor-pointer hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#555] transition-colors"
                        @click="restore(snap)"
                    >
                        <RiRestartLine size="12" />
                        Restore
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import AppTopNav from "../../foundation/AppTopNav.vue";
import { api } from "../../api/client.js";
import { useConfirm } from "../../foundation/useConfirm.js";
import { useNotifications } from "../../foundation/useNotifications.js";
import {
    RiSaveLine,
    RiLoader4Line,
    RiDatabase2Line,
    RiRestartLine,
} from "@remixicon/vue";

const snapshots = ref([]);
const loading = ref(true);
const creating = ref(false);
const { confirm } = useConfirm();
const { notify } = useNotifications();

onMounted(load);

async function load() {
    loading.value = true;
    try {
        snapshots.value = await api.listSnapshots();
    } finally {
        loading.value = false;
    }
}

async function createSnapshot() {
    creating.value = true;
    try {
        await api.createSnapshot();
        await load();
        notify("Snapshot created");
    } finally {
        creating.value = false;
    }
}

async function restore(snap) {
    const ok = await confirm({
        title: "Restore snapshot?",
        message: `This will replace all current data with the snapshot from ${formatDate(snap.created_at)} and restart the server.`,
        confirmLabel: "Restore",
    });
    if (!ok) return;
    await api.restoreSnapshot(snap.id);
}

function formatDate(iso) {
    return new Date(iso).toLocaleString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
    });
}

function formatSize(bytes) {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
</script>
