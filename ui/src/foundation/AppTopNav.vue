<template>
    <header
        class="flex items-center justify-between px-4 md:px-6 h-14 bg-surface-1 border-b border-border-1 shrink-0"
    >
        <div class="flex items-center gap-2.5">
            <button
                v-if="showBurger"
                class="md:hidden flex items-center justify-center w-8 h-8 bg-transparent border-none text-zinc-500 cursor-pointer hover:text-zinc-700 dark:hover:text-zinc-300 hover:bg-border-1 rounded-md transition-colors"
                @click="$emit('burger')"
            >
                <RiMenuLine size="18" />
            </button>

            <RouterLink
                v-if="showBrand"
                to="/"
                class="flex items-center gap-2.5 no-underline"
            >
                <img src="/assets/logo.svg" class="w-7 h-7" alt="DeguBase" />
                <span
                    class="text-[15px] font-bold text-text-1 tracking-tight hidden sm:block"
                    >DeguBase</span
                >
            </RouterLink>
        </div>

        <div class="flex items-center gap-2">
            <slot />

            <div ref="userMenuRef" class="relative">
                <button
                    class="flex items-center justify-center w-8 h-8 bg-transparent border border-border-1 text-text-2 rounded-full cursor-pointer hover:text-zinc-700 dark:hover:text-zinc-300 hover:bg-border-1 transition-colors"
                    :title="currentUser?.name || currentUser?.username"
                    @click="dropdownOpen = !dropdownOpen"
                >
                    <RiUserLine size="16" />
                </button>

                <div
                    v-if="dropdownOpen"
                    class="absolute right-0 top-full mt-1.5 w-52 bg-surface-1 border border-border-1 rounded-xl shadow-lg z-50 py-1 overflow-hidden"
                >
                    <div
                        class="px-3 py-2 border-b border-zinc-100 dark:border-[#3c3c3c] mb-1"
                    >
                        <div class="text-xs font-semibold text-text-1 truncate">
                            {{ currentUser?.name || currentUser?.username }}
                        </div>
                    </div>

                    <button
                        class="flex items-center gap-2.5 w-full px-3 py-1.75 text-[13px] text-text-2 bg-transparent border-none cursor-pointer hover:bg-zinc-50 dark:hover:bg-[#2d2d2d] hover:text-text-1 transition-colors text-left"
                        @click="toggle()"
                    >
                        <RiSunLine v-if="isDark" size="15" />
                        <RiMoonLine v-else size="15" />
                        {{ isDark ? "Light mode" : "Dark mode" }}
                    </button>

                    <a
                        href="/api/docs/index.html"
                        target="_blank"
                        rel="noopener"
                        class="flex items-center gap-2.5 w-full px-3 py-1.75 text-[13px] text-text-2 bg-transparent border-none cursor-pointer hover:bg-zinc-50 dark:hover:bg-[#2d2d2d] hover:text-text-1 transition-colors no-underline"
                    >
                        <RiCodeSSlashLine size="15" />
                        API docs
                    </a>

                    <div
                        v-if="authDisabled || currentUser?.is_admin"
                        class="border-t border-zinc-100 dark:border-[#3c3c3c] mt-1 pt-1"
                    >
                        <RouterLink
                            to="/admin/snapshots"
                            class="flex items-center gap-2.5 w-full px-3 py-1.75 text-[13px] text-text-2 bg-transparent border-none cursor-pointer hover:bg-zinc-50 dark:hover:bg-[#2d2d2d] hover:text-text-1 transition-colors no-underline"
                            @click="dropdownOpen = false"
                        >
                            <RiDatabase2Line size="15" />
                            Snapshots
                        </RouterLink>
                        <RouterLink
                            to="/admin/jobs"
                            class="flex items-center gap-2.5 w-full px-3 py-1.75 text-[13px] text-text-2 bg-transparent border-none cursor-pointer hover:bg-zinc-50 dark:hover:bg-[#2d2d2d] hover:text-text-1 transition-colors no-underline"
                            @click="dropdownOpen = false"
                        >
                            <RiHistoryLine size="15" />
                            Jobs
                        </RouterLink>
                    </div>

                    <div
                        v-if="!authDisabled"
                        class="border-t border-zinc-100 dark:border-[#3c3c3c] mt-1 pt-1"
                    >
                        <button
                            class="flex items-center gap-2.5 w-full px-3 py-1.75 text-[13px] text-text-2 bg-transparent border-none cursor-pointer hover:bg-zinc-50 dark:hover:bg-[#2d2d2d] hover:text-text-1 transition-colors text-left"
                            @click="doLogout"
                        >
                            <RiLogoutBoxRLine size="15" />
                            Sign out
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </header>
</template>

<script setup>
import { ref } from "vue";
import { onClickOutside } from "@vueuse/core";
import { useRouter } from "vue-router";
import { useTheme } from "./useTheme.js";
import { useAuth } from "../features/identity/useAuth.js";
import {
    RiMenuLine,
    RiUserLine,
    RiSunLine,
    RiMoonLine,
    RiLogoutBoxRLine,
    RiCodeSSlashLine,
    RiDatabase2Line,
    RiHistoryLine,
} from "@remixicon/vue";

defineProps({
    showBurger: { type: Boolean, default: false },
    showBrand: { type: Boolean, default: true },
});
defineEmits(["burger"]);

const router = useRouter();
const { isDark, toggle } = useTheme();
const { user: currentUser, logout, authDisabled } = useAuth();

const dropdownOpen = ref(false);
const userMenuRef = ref(null);

onClickOutside(userMenuRef, () => {
    dropdownOpen.value = false;
});

async function doLogout() {
    dropdownOpen.value = false;
    await logout();
    router.push("/login");
}
</script>
