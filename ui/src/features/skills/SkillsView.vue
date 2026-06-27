<template>
    <div class="flex-1 flex flex-col overflow-hidden">
        <!-- Top bar -->
        <div
            class="flex items-center justify-between px-6 py-3 border-b border-border-1 bg-white dark:bg-[#1e1e1e] shrink-0"
        >
            <h1 class="text-[15px] font-bold text-text-1">Skills</h1>
            <button
                class="flex items-center gap-1.5 px-3 py-1.5 bg-brand-600 text-white text-[13px] font-medium rounded-md border-none cursor-pointer hover:bg-brand-700 transition-colors"
                @click="newSkill"
            >
                <RiAddLine size="14" />
                New skill
            </button>
        </div>

        <div class="flex-1 flex overflow-hidden">
            <!-- Skill list: full-width on mobile when nothing selected, fixed sidebar on desktop -->
            <div
                class="flex-col border-r border-border-1 bg-zinc-50 dark:bg-[#252526] overflow-y-auto"
                :class="
                    form
                        ? 'hidden md:flex md:w-64 md:shrink-0'
                        : 'flex flex-1 md:flex-none md:w-64 md:shrink-0'
                "
            >
                <div v-if="loading" class="px-4 py-3 text-xs text-text-2">
                    Loading…
                </div>
                <div
                    v-else-if="!skills.length"
                    class="px-4 py-3 text-xs text-text-2"
                >
                    No skills yet.
                </div>
                <button
                    v-for="sk in skills"
                    :key="sk.id"
                    class="flex flex-col items-start gap-0.5 px-4 py-3 text-left border-none cursor-pointer border-b border-border-1 transition-colors"
                    :class="
                        selected?.id === sk.id
                            ? 'bg-brand-50 dark:bg-brand-900/20'
                            : 'bg-transparent hover:bg-zinc-100 dark:hover:bg-[#2d2d30]'
                    "
                    @click="selectSkill(sk)"
                >
                    <div class="flex items-center gap-2 w-full min-w-0">
                        <RiCodeBoxLine
                            size="13"
                            class="text-brand-600 shrink-0"
                        />
                        <span
                            class="text-[13px] font-medium text-text-1 truncate flex-1"
                            >{{ sk.name }}</span
                        >
                    </div>
                    <span
                        class="text-[11px] text-text-2 pl-5 truncate w-full"
                        >{{ sk.description }}</span
                    >
                    <span
                        v-if="sk.table_ids?.length"
                        class="text-[11px] text-text-2 pl-5"
                    >
                        {{ sk.table_ids.length }}
                        {{ sk.table_ids.length === 1 ? "table" : "tables" }}
                    </span>
                </button>
            </div>

            <!-- Detail panel: hidden on mobile when nothing selected, full-width otherwise -->
            <div
                class="flex-col overflow-hidden"
                :class="form ? 'flex flex-1' : 'hidden md:flex md:flex-1'"
            >
                <div
                    v-if="!form"
                    class="flex-1 flex items-center justify-center text-sm text-text-2"
                >
                    Select a skill or create a new one.
                </div>

                <template v-else>
                    <!-- Back button (mobile only) -->
                    <button
                        class="md:hidden flex items-center gap-1.5 px-4 py-2.5 text-[13px] text-text-2 bg-zinc-50 dark:bg-[#252526] border-b border-border-1 shrink-0 hover:text-text-1 transition-colors"
                        @click="goBack"
                    >
                        <RiArrowLeftLine size="14" />
                        Skills
                    </button>

                    <!-- Settings bar -->
                    <div
                        class="flex flex-wrap items-end gap-3 px-5 py-3 border-b border-border-1 bg-white dark:bg-[#1e1e1e] shrink-0"
                    >
                        <div class="flex flex-col gap-1 min-w-36">
                            <label
                                class="text-[10px] font-semibold text-text-2 tracking-wide uppercase"
                                >Name</label
                            >
                            <input
                                v-if="!form.id || editing"
                                v-model="form.name"
                                class="form-input"
                                placeholder="e.g. Issues tracker"
                            />
                            <span v-else class="text-[13px] text-text-1 py-1">{{
                                form.name
                            }}</span>
                        </div>

                        <div class="flex flex-col gap-1 flex-1 min-w-48">
                            <label
                                class="text-[10px] font-semibold text-text-2 tracking-wide uppercase"
                                >Description</label
                            >
                            <input
                                v-if="!form.id || editing"
                                v-model="form.description"
                                class="form-input w-full"
                                placeholder="What does this skill help with?"
                            />
                            <span
                                v-else
                                class="text-[13px] text-text-2 py-1 truncate"
                                >{{ form.description }}</span
                            >
                        </div>

                        <!-- Table scope (new skill or editing) -->
                        <div
                            v-if="!form.id || editing"
                            class="flex flex-col gap-1 relative"
                            ref="tableScopeEl"
                        >
                            <label
                                class="text-[10px] font-semibold text-text-2 tracking-wide uppercase"
                                >Tables</label
                            >
                            <button
                                type="button"
                                class="form-input flex items-center gap-2 min-w-40 text-left"
                                @click="tableScopeOpen = !tableScopeOpen"
                            >
                                <span class="flex-1 truncate text-[13px]">{{
                                    tableScopeLabel
                                }}</span>
                                <RiArrowDownSLine
                                    size="13"
                                    class="text-zinc-400 shrink-0 transition-transform"
                                    :class="tableScopeOpen ? 'rotate-180' : ''"
                                />
                            </button>
                            <div
                                v-if="tableScopeOpen"
                                class="absolute top-full left-0 z-50 mt-1 w-56 bg-surface-1 border border-border-1 rounded-lg shadow-lg py-1 max-h-52 overflow-y-auto"
                            >
                                <label
                                    v-for="tbl in tables"
                                    :key="tbl.id"
                                    class="flex items-center gap-2 px-3 py-1.5 cursor-pointer hover:bg-surface-2"
                                >
                                    <input
                                        type="checkbox"
                                        class="accent-brand-600"
                                        :checked="
                                            form.table_ids.includes(tbl.id)
                                        "
                                        @change="toggleTable(tbl.id)"
                                    />
                                    <span
                                        class="text-[13px] text-text-1 truncate"
                                        >{{ tbl.name }}</span
                                    >
                                </label>
                                <div
                                    v-if="!tables.length"
                                    class="px-3 py-2 text-[12px] text-text-2"
                                >
                                    No tables.
                                </div>
                            </div>
                        </div>

                        <div class="flex items-center gap-2 ml-auto">
                            <!-- Existing skill, not editing -->
                            <template v-if="form.id && !editing">
                                <button
                                    class="px-3 py-1.5 text-[12px] font-medium text-red-500 bg-transparent border border-red-200 dark:border-red-900/40 rounded-md cursor-pointer hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                                    @click="confirmDelete"
                                >
                                    Delete
                                </button>
                                <button
                                    class="px-3 py-1.5 text-[12px] font-medium text-text-1 bg-surface-2 border border-border-1 rounded-md cursor-pointer hover:bg-border-1 transition-colors"
                                    @click="startEdit"
                                >
                                    Edit
                                </button>
                            </template>
                            <!-- Existing skill, editing -->
                            <template v-else-if="form.id && editing">
                                <button
                                    class="px-3 py-1.5 text-[12px] font-medium text-text-2 bg-surface-2 border border-border-1 rounded-md cursor-pointer hover:bg-border-1 transition-colors"
                                    @click="cancelEdit"
                                >
                                    Cancel
                                </button>
                                <SpinnerButton
                                    :loading="saving"
                                    :disabled="!form.name.trim()"
                                    idle="Save"
                                    busy="Saving…"
                                    class="px-3 py-1.5 text-[12px] font-medium bg-brand-600 text-white border-none rounded-md cursor-pointer hover:bg-brand-700 disabled:opacity-40 disabled:cursor-default transition-colors"
                                    @click="saveEdit"
                                />
                            </template>
                            <!-- New skill -->
                            <template v-else>
                                <button
                                    class="px-3 py-1.5 text-[12px] font-medium text-text-2 bg-surface-2 border border-border-1 rounded-md cursor-pointer hover:bg-border-1 transition-colors"
                                    @click="cancelNew"
                                >
                                    Cancel
                                </button>
                                <SpinnerButton
                                    :loading="saving"
                                    :disabled="!form.name.trim()"
                                    idle="Generate skill"
                                    busy="Generating…"
                                    class="px-3 py-1.5 text-[12px] font-medium bg-brand-600 text-white border-none rounded-md cursor-pointer hover:bg-brand-700 disabled:opacity-40 disabled:cursor-default transition-colors"
                                    @click="generateSkill"
                                />
                            </template>
                        </div>
                    </div>

                    <!-- Operations bar (new skill or editing) -->
                    <div
                        v-if="!form.id || editing"
                        class="flex items-center gap-2 px-5 py-2 border-b border-border-1 bg-zinc-50 dark:bg-[#252526] shrink-0 flex-wrap"
                    >
                        <span
                            class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d] mr-1"
                            >Operations</span
                        >
                        <button
                            v-for="op in ALL_OPS"
                            :key="op.key"
                            type="button"
                            class="px-2.5 py-0.5 rounded-full text-[11px] font-semibold border transition-colors cursor-pointer"
                            :class="
                                form.operations.includes(op.key)
                                    ? 'bg-brand-600 text-white border-brand-600'
                                    : 'bg-transparent text-zinc-400 dark:text-[#6d6d6d] border-zinc-300 dark:border-[#4a4a4a] hover:border-zinc-400 dark:hover:border-[#6d6d6d]'
                            "
                            @click="toggleOp(op.key)"
                        >
                            {{ op.label }}
                        </button>
                    </div>

                    <!-- Auth notice (new skill only) -->
                    <p
                        v-if="!form.id && !authDisabled"
                        class="px-5 py-2 text-[11px] text-zinc-400 dark:text-[#6d6d6d] border-b border-border-1 shrink-0"
                    >
                        Skills require a workspace API token for authentication.
                        <RouterLink
                            :to="`/workspaces/${workspaceCode}/settings/tokens`"
                            class="text-brand-500 hover:text-brand-600 no-underline"
                            >Manage tokens in Settings →</RouterLink
                        >
                    </p>

                    <p
                        v-if="saveError"
                        class="px-5 py-1.5 text-xs text-red-500 bg-red-50 dark:bg-red-900/10 shrink-0"
                    >
                        {{ saveError }}
                    </p>

                    <!-- Tab bar (existing skill only) -->
                    <div
                        v-if="form.id"
                        class="flex items-center gap-0 border-b border-border-1 bg-white dark:bg-[#1e1e1e] shrink-0 px-5"
                    >
                        <button
                            v-for="tab in ['install', 'preview']"
                            :key="tab"
                            class="px-3 py-2 text-[12px] font-medium border-b-2 -mb-px transition-colors border-none bg-transparent cursor-pointer capitalize"
                            :class="
                                activeTab === tab
                                    ? 'border-brand-500 text-brand-600 dark:text-brand-400'
                                    : 'border-transparent text-text-2 hover:text-text-1'
                            "
                            @click="navigateTab(tab)"
                        >
                            {{ tab }}
                        </button>
                    </div>

                    <!-- ── New skill form content ─────────────────────────────────── -->
                    <div
                        v-if="!form.id"
                        class="flex-1 flex flex-col overflow-hidden"
                    >
                        <!-- Table context cards (shown when tables are selected) -->
                        <div
                            v-if="form.table_ids.length"
                            class="shrink-0 max-h-44 overflow-y-auto border-b border-border-1 bg-white dark:bg-[#1e1e1e]"
                        >
                            <div class="px-5 py-2 flex flex-col gap-1.5">
                                <div
                                    v-for="id in form.table_ids"
                                    :key="id"
                                    class="flex items-start gap-3 rounded-lg px-3 py-2 bg-zinc-50 dark:bg-[#252526] border border-border-1"
                                >
                                    <div class="flex-1 min-w-0">
                                        <div class="flex items-center gap-2">
                                            <span
                                                class="text-[12px] font-semibold text-text-1"
                                                >{{ tableNameById(id) }}</span
                                            >
                                            <span
                                                v-if="
                                                    !tableDetails[id]?.context
                                                "
                                                class="flex items-center gap-1 text-[10px] font-medium text-amber-600 dark:text-amber-400"
                                            >
                                                <RiAlertLine size="11" /> No
                                                context
                                            </span>
                                        </div>
                                        <p
                                            v-if="tableDetails[id]?.context"
                                            class="text-[11px] text-text-2 mt-0.5 line-clamp-2"
                                        >
                                            {{ tableDetails[id].context }}
                                        </p>
                                        <p
                                            v-else
                                            class="text-[11px] text-zinc-400 dark:text-[#6d6d6d] mt-0.5"
                                        >
                                            Add a description to improve the
                                            generated skill quality.
                                        </p>
                                    </div>
                                    <button
                                        class="shrink-0 flex items-center gap-1 px-2 py-1 text-[11px] font-medium text-text-2 bg-transparent border border-border-1 rounded-md cursor-pointer hover:text-text-1 hover:border-zinc-300 dark:hover:border-[#5a5a5a] transition-colors"
                                        @click="openContextEdit(id)"
                                    >
                                        <RiEdit2Line size="11" />
                                        Edit
                                    </button>
                                </div>
                            </div>
                        </div>

                        <!-- Live preview -->
                        <div
                            class="flex items-center gap-3 px-5 py-2 border-b border-border-1 bg-white dark:bg-[#1e1e1e] shrink-0"
                        >
                            <span
                                class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]"
                                >Live preview</span
                            >
                            <span
                                v-if="fetchingPreview"
                                class="text-[11px] text-zinc-400 dark:text-[#6d6d6d] flex items-center gap-1"
                            >
                                <RiLoader4Line size="12" class="animate-spin" />
                                Loading preview…
                            </span>
                            <span
                                v-else-if="!form.table_ids.length"
                                class="text-[11px] text-zinc-300 dark:text-[#4a4a4a]"
                            >
                                — select tables to see the generated skill
                            </span>
                        </div>
                        <div class="flex-1 overflow-y-auto p-4 bg-[#282c34]">
                            <div
                                v-if="!form.table_ids.length"
                                class="h-full flex items-center justify-center"
                            >
                                <span
                                    class="text-[12px] text-zinc-500 dark:text-[#6d6d6d]"
                                    >Preview not available — please select some
                                    tables first.</span
                                >
                            </div>
                            <div
                                v-else-if="fetchingPreview"
                                class="h-full flex items-center justify-center"
                            >
                                <span
                                    class="text-[12px] text-zinc-400 dark:text-[#6d6d6d] flex items-center gap-2"
                                >
                                    <RiLoader4Line
                                        size="13"
                                        class="animate-spin"
                                    />
                                    Generating preview…
                                </span>
                            </div>
                            <pre
                                v-else
                                class="text-[12px] font-mono leading-relaxed whitespace-pre-wrap text-zinc-100"
                                >{{ previewMarkdown }}</pre
                            >
                        </div>
                    </div>

                    <!-- Router outlet for tab content (existing skill only) -->
                    <router-view v-else v-slot="{ Component }">
                        <keep-alive>
                            <component :is="Component" />
                        </keep-alive>
                    </router-view>
                </template>
            </div>
        </div>
    </div>

    <!-- Table context edit modal -->
    <ModalDialog
        v-if="editContextTableId !== null"
        :title="tableNameById(editContextTableId) + ' — Context'"
        confirm-label="Save"
        :confirm-disabled="savingContext"
        @confirm="saveTableContext"
        @cancel="editContextTableId = null"
    >
        <div class="flex flex-col gap-1.5">
            <label class="text-xs font-semibold text-text-2 tracking-wide"
                >Description</label
            >
            <textarea
                v-model="editContextValue"
                rows="5"
                placeholder="Describe what this table is for, what data it holds, and how it's used…"
                class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all resize-y"
            />
            <p class="text-[11px] text-zinc-400 dark:text-[#6d6d6d]">
                This context is embedded in the skill and helps the AI
                understand the table's purpose.
            </p>
        </div>
        <p v-if="contextSaveError" class="text-xs text-red-500">
            {{ contextSaveError }}
        </p>
    </ModalDialog>
</template>

<script setup>
import { ref, computed, watch, onMounted, provide } from "vue";
import { onClickOutside } from "@vueuse/core";
import { useRoute, useRouter } from "vue-router";
import { api } from "../../api/client.js";
import { useAuth } from "../identity/useAuth.js";
import {
    RiAddLine,
    RiArrowDownSLine,
    RiArrowLeftLine,
    RiCodeBoxLine,
    RiLoader4Line,
    RiEdit2Line,
    RiAlertLine,
    RiDownloadLine,
} from "@remixicon/vue";
import SpinnerButton from "../../foundation/SpinnerButton.vue";
import ModalDialog from "../../foundation/ModalDialog.vue";
import { useConfirm } from "../../foundation/useConfirm.js";

const { confirm } = useConfirm();
const ALL_OPS = [
    { key: "read", label: "Read" },
    { key: "create", label: "Create" },
    { key: "update", label: "Update" },
    { key: "delete", label: "Delete" },
    { key: "views", label: "Views" },
    { key: "schema", label: "Schema" },
    { key: "scripting", label: "Scripting" },
];

const route = useRoute();
const router = useRouter();
const workspaceCode = route.params.workspaceCode;
const { authDisabled } = useAuth();

const skills = ref([]);
const tables = ref([]);
const tableDetails = ref({});
const fetchingPreview = ref(false);
const previewMarkdown = ref("");
const loading = ref(true);
const saving = ref(false);
const saveError = ref("");
const selected = ref(null);
const form = ref(null);
const tableScopeOpen = ref(false);
const tableScopeEl = ref(null);
const editing = ref(false);
const editSnapshot = ref(null);

const VALID_TABS = ["install", "preview"];

const activeTab = computed(() => {
    const segments = route.path.split("/");
    const last = segments[segments.length - 1];
    return VALID_TABS.includes(last) ? last : "install";
});

// Table context edit
const editContextTableId = ref(null);
const editContextValue = ref("");
const savingContext = ref(false);
const contextSaveError = ref("");

// ── Provide shared state for child tab components ──
provide("workspaceCode", workspaceCode);
provide("skillForm", form);
provide("tables", tables);
provide("editing", editing);
provide("previewMarkdown", previewMarkdown);
provide("fetchingPreview", fetchingPreview);
provide("downloadSkill", downloadSkill);
provide("tableNameById", tableNameById);

// Fetch table context for the context cards shown when tables are selected
watch(
    () => form.value?.table_ids,
    async (ids) => {
        if (!ids?.length) return;
        await Promise.all(
            ids
                .filter((id) => !tableDetails.value[id])
                .map(async (id) => {
                    const tbl = tables.value.find((t) => t.id === id);
                    if (!tbl) return;
                    const full = await api.getTable(workspaceCode, tbl.code);
                    tableDetails.value[id] = full;
                }),
        );
    },
    { deep: true },
);

// Debounced live preview via the backend /preview endpoint.
let previewTimer = null;
watch(
    () => {
        if (!form.value) return null;
        if (!form.value.id || editing.value)
            return [
                editing.value,
                form.value.table_ids?.slice(),
                form.value.operations?.slice(),
                form.value.name,
                form.value.description,
            ];
        return null;
    },
    async () => {
        clearTimeout(previewTimer);
        if (!form.value || !form.value.table_ids?.length) {
            previewMarkdown.value = "";
            fetchingPreview.value = false;
            return;
        }
        fetchingPreview.value = true;
        previewTimer = setTimeout(async () => {
            try {
                previewMarkdown.value = await api.previewSkill(workspaceCode, {
                    name: form.value.name,
                    description: form.value.description,
                    table_ids: form.value.table_ids,
                    operations: form.value.operations,
                });
            } catch {
                previewMarkdown.value = "(failed to load preview)";
            } finally {
                fetchingPreview.value = false;
            }
        }, 400);
    },
    { deep: true },
);

const tableScopeLabel = computed(() => {
    if (!form.value?.table_ids?.length) return "Select tables";
    if (form.value.table_ids.length === 1)
        return tableNameById(form.value.table_ids[0]);
    return `${form.value.table_ids.length} tables`;
});

function tableNameById(id) {
    return tables.value.find((t) => t.id === id)?.name ?? `Table #${id}`;
}

function toggleTable(id) {
    const ids = form.value.table_ids;
    const idx = ids.indexOf(id);
    if (idx === -1) ids.push(id);
    else ids.splice(idx, 1);
}

function toggleOp(key) {
    const ops = form.value.operations;
    const idx = ops.indexOf(key);
    if (idx === -1) ops.push(key);
    else ops.splice(idx, 1);
}

function openContextEdit(tableId) {
    editContextTableId.value = tableId;
    editContextValue.value = tableDetails.value[tableId]?.context ?? "";
    contextSaveError.value = "";
}

async function saveTableContext() {
    const id = editContextTableId.value;
    const tbl = tableDetails.value[id];
    if (!tbl) return;
    savingContext.value = true;
    contextSaveError.value = "";
    try {
        const updated = await api.updateTable(workspaceCode, tbl.code, {
            name: tbl.name,
            context: editContextValue.value,
        });
        tableDetails.value[id] = { ...tbl, ...updated };
        editContextTableId.value = null;
    } catch (e) {
        contextSaveError.value = e.message || "Failed to save.";
    } finally {
        savingContext.value = false;
    }
}

onClickOutside(tableScopeEl, () => {
    tableScopeOpen.value = false;
});

// ── Load skills list & select from route ──
async function load() {
    loading.value = true;
    try {
        const [sk, tbls] = await Promise.all([
            api.listSkills(workspaceCode),
            api.listTables(workspaceCode),
        ]);
        skills.value = sk;
        tables.value = tbls;

        const idParam = route.params.skillId
            ? Number(route.params.skillId)
            : null;
        if (idParam) {
            const match = sk.find((s) => s.id === idParam);
            if (match) setForm(match);
        }
    } finally {
        loading.value = false;
    }
}

function setForm(sk) {
    selected.value = sk;
    form.value = {
        ...sk,
        table_ids: [...(sk.table_ids ?? [])],
        operations: [...(sk.operations ?? [])],
    };
    saveError.value = "";
    editing.value = false;
    tableScopeOpen.value = false;
}

function clearForm() {
    form.value = null;
    selected.value = null;
    saveError.value = "";
}

watch(
    () => route.params.skillId,
    (id) => {
        // Only handle navigation changes after initial load — onMounted(load) covers the first mount.
        if (!skills.value.length) return;
        if (id) {
            const match = skills.value.find((s) => s.id === Number(id));
            if (match) setForm(match);
        } else {
            clearForm();
        }
    },
);

onMounted(load);

// ── Navigation ──
const BASE = computed(() => `/workspaces/${workspaceCode}/automations/skills`);

function navigateTab(tab) {
    if (!form.value?.id) return;
    router.replace(`${BASE.value}/${form.value.id}/${tab}`);
}

function selectSkill(sk) {
    router.push(`${BASE.value}/${sk.id}/install`);
}

function newSkill() {
    selected.value = null;
    form.value = {
        id: null,
        name: "",
        description: "",
        table_ids: [],
        operations: ["read", "create", "update", "delete"],
    };
    saveError.value = "";
    tableScopeOpen.value = false;
    router.replace(BASE.value);
}

function cancelNew() {
    clearForm();
    router.replace(BASE.value);
}

function goBack() {
    clearForm();
    router.replace(BASE.value);
}

function startEdit() {
    editSnapshot.value = {
        ...form.value,
        table_ids: [...form.value.table_ids],
        operations: [...form.value.operations],
    };
    editing.value = true;
}

function cancelEdit() {
    form.value = editSnapshot.value;
    editSnapshot.value = null;
    editing.value = false;
    saveError.value = "";
}

async function saveEdit() {
    if (!form.value.name.trim()) return;
    saving.value = true;
    saveError.value = "";
    try {
        const updated = await api.updateSkill(workspaceCode, form.value.id, {
            name: form.value.name.trim(),
            description: form.value.description.trim(),
            table_ids: form.value.table_ids,
            operations: form.value.operations,
        });
        const idx = skills.value.findIndex((s) => s.id === updated.id);
        if (idx !== -1) skills.value[idx] = updated;
        setForm(updated);
    } catch (e) {
        saveError.value = e.message || "Failed to save.";
    } finally {
        saving.value = false;
    }
}

async function generateSkill() {
    if (!form.value.name.trim()) return;
    saving.value = true;
    saveError.value = "";
    try {
        const payload = {
            name: form.value.name.trim(),
            description: form.value.description.trim(),
            table_ids: form.value.table_ids,
            operations: form.value.operations,
        };
        const result = await api.createSkill(workspaceCode, payload);
        skills.value.push(result.skill);
        setForm(result.skill);
        router.replace(`${BASE.value}/${result.skill.id}/install`);
    } catch (e) {
        saveError.value = e.message || "Failed to generate skill.";
    } finally {
        saving.value = false;
    }
}

async function confirmDelete() {
    if (
        !(await confirm({
            title: "Delete skill?",
            message: `"${form.value.name}" will be permanently deleted. This cannot be undone.`,
            confirmLabel: "Delete",
        }))
    )
        return;
    await api.deleteSkill(workspaceCode, form.value.id);
    skills.value = skills.value.filter((s) => s.id !== form.value.id);
    clearForm();
    router.replace(BASE.value);
}

async function downloadSkill() {
    const url =
        window.location.origin + api.skillMdUrl(workspaceCode, form.value.id);
    const resp = await fetch(url);
    const text = await resp.text();
    const blob = new Blob([text], { type: "text/markdown" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = toKebabCase(form.value.name) + ".md";
    a.click();
    URL.revokeObjectURL(a.href);
}

function toKebabCase(s) {
    return s
        .toLowerCase()
        .replace(/\s+/g, "-")
        .replace(/[^a-z0-9-]/g, "");
}
</script>

<style scoped>
.form-input {
    background: var(--input-bg);
    border: 1px solid var(--input-border);
    border-radius: 6px;
    color: var(--text-1);
    font-size: 13px;
    padding: 5px 9px;
    outline: none;
    box-sizing: border-box;
}
.form-input:focus {
    border-color: var(--color-brand-600);
    box-shadow: 0 0 0 3px rgba(210, 105, 30, 0.12);
}
</style>
