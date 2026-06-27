<template>
    <div class="flex-1 overflow-y-auto px-6 py-6">
        <div class="max-w-2xl flex flex-col gap-6">
            <section>
                <div class="flex items-center gap-3 mb-3">
                    <span
                        class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]"
                        >Download</span
                    >
                    <div class="flex-1 h-px bg-border-1" />
                </div>
                <p class="text-[12px] text-text-2 mb-3">
                    Download the current skill file to install manually or
                    commit to your project.
                </p>
                <button
                    class="flex items-center gap-2 px-3 py-1.5 text-[12px] font-medium bg-zinc-100 dark:bg-[#2d2d30] text-text-1 border border-border-1 rounded-md cursor-pointer hover:bg-border-1 transition-colors"
                    @click="downloadSkill"
                >
                    <RiDownloadLine size="13" />
                    {{ kebabName }}.md
                </button>
            </section>

            <section>
                <div class="flex items-center gap-3 mb-3">
                    <span
                        class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]"
                        >Agent prompt</span
                    >
                    <div class="flex-1 h-px bg-border-1" />
                </div>
                <p class="text-[12px] text-text-2 mb-3">
                    Paste into a chat with your agent to have it fetch the skill
                    definition and install or update it automatically.
                </p>
                <div
                    class="flex items-start gap-2 bg-zinc-900 dark:bg-[#141414] rounded-lg px-4 py-3"
                >
                    <code
                        class="flex-1 text-[12px] font-mono text-zinc-100 break-all select-all whitespace-pre-wrap"
                        >{{ agentPrompt }}</code
                    >
                    <button
                        class="shrink-0 flex items-center justify-center w-7 h-7 text-zinc-500 hover:text-zinc-200 bg-transparent border-none cursor-pointer rounded transition-colors"
                        :title="copied ? 'Copied!' : 'Copy'"
                        @click="copy"
                    >
                        <RiCheckLine
                            v-if="copied"
                            size="14"
                            class="text-emerald-400"
                        />
                        <RiFileCopyLine v-else size="14" />
                    </button>
                </div>
            </section>

            <section v-if="!authDisabled">
                <div class="flex items-center gap-3 mb-3">
                    <span
                        class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]"
                        >Authentication</span
                    >
                    <div class="flex-1 h-px bg-border-1" />
                </div>
                <p class="text-[12px] text-text-2">
                    The skill uses
                    <code
                        class="font-mono text-[11px] bg-zinc-100 dark:bg-[#2d2d30] px-1 rounded"
                        >DEGUBASE_TOKEN</code
                    >
                    for authentication. Set it to any workspace API token from
                    <RouterLink
                        :to="`/workspaces/${workspaceCode}/settings/tokens`"
                        class="text-brand-500 hover:text-brand-600 no-underline font-medium"
                        >Settings → API Tokens</RouterLink
                    >.
                </p>
            </section>

            <section>
                <div class="flex items-center gap-3 mb-3">
                    <span
                        class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]"
                        >Tables covered</span
                    >
                    <div class="flex-1 h-px bg-border-1" />
                </div>
                <div
                    v-if="!form.table_ids?.length"
                    class="text-[12px] text-text-2"
                >
                    No tables selected.
                </div>
                <div v-else class="flex flex-wrap gap-2">
                    <span
                        v-for="id in form.table_ids"
                        :key="id"
                        class="px-2 py-0.5 rounded-md text-[12px] font-medium bg-brand-50 dark:bg-brand-900/20 text-brand-700 dark:text-brand-400 border border-brand-200 dark:border-brand-800/40"
                        >{{ tableNameById(id) }}</span
                    >
                </div>
            </section>

            <section v-if="form.operations?.length">
                <div class="flex items-center gap-3 mb-3">
                    <span
                        class="text-[10px] font-bold tracking-widest uppercase text-zinc-400 dark:text-[#6d6d6d]"
                        >Permissions</span
                    >
                    <div class="flex-1 h-px bg-border-1" />
                </div>
                <div class="flex flex-wrap gap-2">
                    <span
                        v-for="op in form.operations"
                        :key="op"
                        class="px-2 py-0.5 rounded-full text-[11px] font-semibold bg-zinc-100 dark:bg-[#2d2d30] text-text-2 border border-border-1 capitalize"
                        >{{ op }}</span
                    >
                </div>
            </section>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, inject } from "vue";
import { useRouter } from "vue-router";
import { useAuth } from "../identity/useAuth.js";
import { RiCheckLine, RiFileCopyLine, RiDownloadLine } from "@remixicon/vue";

const router = useRouter();
const { authDisabled } = useAuth();

const workspaceCode = inject("workspaceCode");
const form = inject("skillForm");
const tables = inject("tables");
const downloadSkill = inject("downloadSkill");
const tableNameById = inject("tableNameById");

const copied = ref(false);

const kebabName = computed(() => toKebabCase(form.value?.name ?? "skill"));
const skillUrl = computed(
    () =>
        `${window.location.origin}/api/workspaces/${workspaceCode}/skills/${form.value?.id}`,
);

const agentPrompt = computed(() => {
    const name = form.value?.name ?? "this skill";
    let prompt = `Install or update the skill '${name}' using 'curl -s ${skillUrl.value}/context'\n`;
    prompt +=
        "Inspect the skill and prompt me where to install the skill and for instructions when and how to use the skill.";

    if (!authDisabled.value) {
        prompt += `\n\nThis skill requires a DEGUBASE_TOKEN for authentication. Ask me for a token.`;
    }
    return prompt;
});

function toKebabCase(str) {
    return str
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-|-$/g, "");
}

async function copy() {
    try {
        await navigator.clipboard.writeText(agentPrompt.value);
        copied.value = true;
        setTimeout(() => {
            copied.value = false;
        }, 2000);
    } catch {
        /* ignore */
    }
}
</script>
