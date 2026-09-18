<template>
    <aside
        class="fixed md:relative inset-y-0 left-0 z-50 md:z-auto w-59 min-w-59 bg-[#1c1d2e] dark:bg-[#141520] flex flex-col overflow-y-auto border-r border-white/6 shrink-0 transition-transform duration-300 ease-in-out md:translate-x-0"
        :class="open ? 'translate-x-0' : '-translate-x-full'"
    >
        <!-- Brand -->
        <RouterLink
            to="/"
            class="flex items-center gap-2.5 px-3.5 py-4 border-b border-white/6 no-underline"
        >
            <img
                src="/assets/logo.svg"
                class="w-7 h-7 shrink-0"
                alt="DeguBase"
            />
            <span class="text-sm font-semibold text-white tracking-tight flex-1"
                >DeguBase</span
            >
            <button
                class="md:hidden flex items-center justify-center w-6 h-6 text-slate-500 hover:text-slate-300 hover:bg-white/6 rounded transition-colors"
                @click.prevent="$emit('close')"
            >
                <RiCloseLine size="16" />
            </button>
        </RouterLink>

        <!-- Workspace header -->
        <div
            class="flex items-center gap-2 px-3.5 py-3 border-b border-white/6"
        >
            <RouterLink
                to="/"
                title="All workspaces"
                class="flex items-center justify-center w-5.5 h-5.5 text-slate-500 no-underline rounded shrink-0 hover:text-slate-300 hover:bg-white/6 transition-colors"
            >
                <RiArrowLeftSLine size="16" />
            </RouterLink>
            <span
                class="text-[13px] font-semibold text-slate-300 truncate flex-1"
                >{{ workspaceName }}</span
            >
        </div>

        <!-- States -->
        <div v-if="loading" class="px-3.5 py-3 text-xs text-slate-500">
            Loading…
        </div>
        <div v-else-if="error" class="px-3.5 py-3 text-xs text-red-400">
            {{ error }}
        </div>

        <!-- Table list -->
        <nav v-else class="flex-1 py-2">
            <div
                class="px-3.5 pt-2 pb-1 text-[10px] font-bold tracking-widest uppercase text-slate-500"
            >
                Tables
            </div>

            <RouterLink
                v-for="tbl in tables"
                :key="tbl.code"
                :to="tableHref(tbl)"
                class="flex items-center gap-2 px-3.5 py-1.5 rounded-md mx-1.5 my-px min-h-8 transition-colors hover:bg-white/6 no-underline"
                :class="activeTableCode === tbl.code ? 'bg-brand-600/14' : ''"
            >
                <span
                    class="shrink-0 flex"
                    :class="
                        activeTableCode === tbl.code
                            ? 'text-brand-400'
                            : 'text-slate-500'
                    "
                >
                    <i
                        v-if="tbl.icon"
                        :class="[
                            resolveTableIcon(tbl.icon),
                            'text-sm leading-none',
                        ]"
                    />
                    <RiTableView v-else size="13" />
                </span>
                <span
                    class="flex-1 text-[13px] truncate"
                    :class="
                        activeTableCode === tbl.code
                            ? 'text-brand-400'
                            : 'text-slate-300'
                    "
                    >{{ tbl.name }}</span
                >
            </RouterLink>

            <button
                data-testid="new-table-btn"
                :disabled="creating"
                class="flex items-center gap-1.5 w-[calc(100%-12px)] mx-1.5 mt-1 px-2 py-1.5 bg-transparent border-none text-slate-500 text-xs cursor-pointer rounded-md hover:text-slate-300 hover:bg-white/6 transition-colors text-left disabled:opacity-50 disabled:cursor-default"
                @click="showCreateDialog = true"
            >
                <RiLoader4Line v-if="creating" size="14" class="animate-spin" />
                <RiAddLine v-else size="14" />
                {{ creating ? "Creating…" : "New table" }}
            </button>
        </nav>

        <!-- Bottom nav -->
        <div class="border-t border-white/6 px-1.5 py-2 flex flex-col gap-0.5">
            <RouterLink
                :to="`/workspaces/${props.workspaceCode}/settings/mcp`"
                class="flex items-center gap-2 px-2 py-1.5 no-underline text-xs rounded-md transition-colors"
                :class="
                    isMCP
                        ? 'text-brand-400 bg-brand-600/14'
                        : 'text-slate-500 hover:text-slate-300 hover:bg-white/6'
                "
            >
                <RiRobot2Line size="13" />
                MCP
            </RouterLink>
            <RouterLink
                :to="`/workspaces/${props.workspaceCode}/automations/scripts`"
                class="flex items-center gap-2 px-2 py-1.5 no-underline text-xs rounded-md transition-colors"
                :class="
                    isScripts
                        ? 'text-brand-400 bg-brand-600/14'
                        : 'text-slate-500 hover:text-slate-300 hover:bg-white/6'
                "
            >
                <RiCodeSSlashLine size="13" />
                Scripts
            </RouterLink>
            <RouterLink
                :to="`/workspaces/${props.workspaceCode}/automations/skills`"
                class="flex items-center gap-2 px-2 py-1.5 no-underline text-xs rounded-md transition-colors"
                :class="
                    isSkills
                        ? 'text-brand-400 bg-brand-600/14'
                        : 'text-slate-500 hover:text-slate-300 hover:bg-white/6'
                "
            >
                <RiCodeBoxLine size="13" />
                Skills
            </RouterLink>
            <RouterLink
                v-if="!authDisabled"
                :to="`/workspaces/${props.workspaceCode}/settings/tokens`"
                class="flex items-center gap-2 px-2 py-1.5 no-underline text-xs rounded-md transition-colors"
                :class="
                    isTokens
                        ? 'text-brand-400 bg-brand-600/14'
                        : 'text-slate-500 hover:text-slate-300 hover:bg-white/6'
                "
            >
                <RiKey2Line size="13" />
                API Tokens
            </RouterLink>
            <RouterLink
                v-if="!authDisabled && user?.is_admin"
                :to="`/workspaces/${props.workspaceCode}/settings/users`"
                class="flex items-center gap-2 px-2 py-1.5 no-underline text-xs rounded-md transition-colors"
                :class="
                    isUsers
                        ? 'text-brand-400 bg-brand-600/14'
                        : 'text-slate-500 hover:text-slate-300 hover:bg-white/6'
                "
            >
                <RiGroupLine size="13" />
                Users
            </RouterLink>
        </div>
    </aside>

    <NewTableDialog
        v-if="showCreateDialog"
        @confirm="createTable"
        @cancel="showCreateDialog = false"
    />
</template>

<script setup>
import { ref, watch, computed } from "vue";
import { useRouter, useRoute } from "vue-router";
import { api } from "../api/client.js";
import { useAuth } from "../features/identity/useAuth.js";
import NewTableDialog from "../features/workspace/NewTableDialog.vue";
import { resolveTableIcon } from "../features/workspace/tableIcons.js";
import { instantiateTemplate } from "../features/workspace/templates.js";
import {
    RiArrowLeftSLine,
    RiTableView,
    RiAddLine,
    RiLoader4Line,
    RiCloseLine,
    RiKey2Line,
    RiGroupLine,
    RiCodeSSlashLine,
    RiCodeBoxLine,
    RiRobot2Line,
} from "@remixicon/vue";

const props = defineProps({
    workspaceCode: String,
    activeTableCode: String,
    open: { type: Boolean, default: false },
});

const emit = defineEmits(["close"]);

const router = useRouter();
const route = useRoute();
const { authDisabled, user } = useAuth();
const isTokens = computed(
    () => route.path === `/workspaces/${props.workspaceCode}/settings/tokens`,
);
const isUsers = computed(
    () => route.path === `/workspaces/${props.workspaceCode}/settings/users`,
);
const isMCP = computed(
    () => route.path === `/workspaces/${props.workspaceCode}/settings/mcp`,
);
const isScripts = computed(() =>
    route.path.startsWith(
        `/workspaces/${props.workspaceCode}/automations/scripts`,
    ),
);
const isSkills = computed(() =>
    route.path.startsWith(
        `/workspaces/${props.workspaceCode}/automations/skills`,
    ),
);
const workspaceName = ref("");
const tables = ref([]);
const loading = ref(true);
const error = ref(null);
const creating = ref(false);
const showCreateDialog = ref(false);

async function load(workspaceCode) {
    loading.value = true;
    error.value = null;
    try {
        const [ws, tbls] = await Promise.all([
            api.getWorkspace(workspaceCode),
            api.listTables(workspaceCode),
        ]);
        workspaceName.value = ws?.name ?? "";
        tables.value = tbls ?? [];
    } catch (e) {
        error.value = e.message;
    } finally {
        loading.value = false;
    }
}

watch(
    () => props.workspaceCode,
    (code) => code && load(code),
    { immediate: true },
);

async function createTable({ name, context, icon, code, template }) {
    showCreateDialog.value = false;
    creating.value = true;
    try {
        const newTable = await api.createTable(props.workspaceCode, {
            name,
            context,
            icon,
            code,
        });
        let targetView = newTable.views?.find(
            (v) => v.id === newTable.default_view_id,
        ) ?? null;
        if (template) {
            targetView = (await instantiateTemplate(api, props.workspaceCode, newTable, template)) ?? targetView;
        }
        await load(props.workspaceCode);
        if (targetView) {
            router.push(
                `/workspaces/${props.workspaceCode}/tables/${newTable.code}/views/${targetView.code}`,
            );
        }
    } finally {
        creating.value = false;
    }
}

function tableHref(tbl) {
    const views = tbl.views ?? [];
    const view = views.find((v) => v.id === tbl.default_view_id) ?? views[0];
    return view
        ? `/workspaces/${props.workspaceCode}/tables/${tbl.code}/views/${view.code}`
        : `/workspaces/${props.workspaceCode}`;
}

defineExpose({
    async reload() {
        if (props.workspaceCode) await load(props.workspaceCode);
    },
});
</script>
