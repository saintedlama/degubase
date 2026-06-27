<template>
  <div class="min-h-screen flex items-center justify-center bg-zinc-50 dark:bg-[#1e1e1e] px-4">
    <div class="w-full max-w-sm">
      <!-- Brand -->
      <div class="flex flex-col items-center gap-3 mb-8">
        <img src="/assets/logo.svg" class="w-12 h-12" alt="DeguBase" />
        <h1 class="text-xl font-bold text-text-1">
          {{ isFirstRun ? 'Welcome to DeguBase' : 'Sign in to DeguBase' }}
        </h1>
      </div>

      <!-- Setup mode callout -->
      <div v-if="isFirstRun" class="mb-4 rounded-xl border border-brand-200 dark:border-brand-800/60 bg-brand-50 dark:bg-brand-900/20 px-4 py-3.5 flex flex-col gap-2">
        <div class="flex items-center gap-2">
          <RiShieldUserLine size="15" class="text-brand-600 dark:text-brand-400 shrink-0" />
          <span class="text-[13px] font-semibold text-brand-700 dark:text-brand-300">Setup mode</span>
        </div>
        <ul class="flex flex-col gap-1 text-[12px] text-brand-700 dark:text-brand-300 pl-1 list-none">
          <li class="flex items-start gap-1.5"><span class="mt-px shrink-0">·</span>No users exist yet.</li>
          <li class="flex items-start gap-1.5"><span class="mt-px shrink-0">·</span>The first login automatically becomes the admin account.</li>
          <li class="flex items-start gap-1.5"><span class="mt-px shrink-0">·</span>After that, only the admin can create new users — no self-registration.</li>
        </ul>
      </div>

      <form
        novalidate
        class="bg-surface-1 border border-border-1 rounded-xl shadow-sm p-6 flex flex-col gap-4"
        @submit.prevent="submit"
      >
        <div class="flex flex-col gap-1.5">
          <label class="text-xs font-semibold text-text-2 tracking-wide">Username</label>
          <input
            ref="usernameInput"
            v-model="username"
            type="text"
            autocomplete="username"
            placeholder="username"
            class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
            required
          />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-xs font-semibold text-text-2 tracking-wide">Password</label>
          <input
            v-model="password"
            type="password"
            autocomplete="current-password"
            placeholder="••••••••"
            class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
            required
            :minlength="isFirstRun ? 8 : undefined"
          />
        </div>

        <p v-if="error" class="text-xs text-red-500">{{ error }}</p>

        <button
          type="submit"
          :disabled="loading"
          class="mt-1 px-4 py-2 rounded-md text-[13px] font-medium bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 disabled:opacity-50 disabled:cursor-default transition-colors"
        >
          {{ loading ? '…' : isFirstRun ? 'Create admin account' : 'Sign in' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../../api/client.js'
import { useAuth } from './useAuth.js'
import { RiShieldUserLine } from '@remixicon/vue'

const router = useRouter()
const { login } = useAuth()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const isFirstRun = ref(false)
const usernameInput = ref(null)

onMounted(async () => {
  try {
    const status = await api.getSetupStatus()
    if (status.auth_disabled) {
      router.push('/')
      return
    }
    isFirstRun.value = status.setup_required
  } catch {
    // backend unreachable — proceed as normal login
  }
  nextTick(() => usernameInput.value?.focus())
})

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await login(username.value, password.value)
    router.push('/')
  } catch (e) {
    error.value = e.status === 401 ? 'Invalid username or password.' : (e.message || 'Login failed.')
  } finally {
    loading.value = false
  }
}
</script>
