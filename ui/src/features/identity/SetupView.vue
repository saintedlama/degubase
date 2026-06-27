<template>
  <div class="min-h-screen flex items-center justify-center bg-zinc-50 dark:bg-[#1e1e1e] px-4">
    <div class="w-full max-w-sm">
      <div class="flex flex-col items-center gap-3 mb-8">
        <img src="/assets/logo.svg" class="w-12 h-12" alt="DeguBase" />
        <h1 class="text-xl font-bold text-text-1">Set up DeguBase</h1>
        <p class="text-sm text-text-2 text-center">Create your admin account to get started.</p>
      </div>

      <form
        class="bg-surface-1 border border-border-1 rounded-xl shadow-sm p-6 flex flex-col gap-4"
        @submit.prevent="submit"
      >
        <div class="flex flex-col gap-1.5">
          <label class="text-xs font-semibold text-text-2 tracking-wide">Name</label>
          <input
            v-model="name"
            type="text"
            autocomplete="name"
            placeholder="Your name"
            class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
          />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-xs font-semibold text-text-2 tracking-wide">Email</label>
          <input
            v-model="email"
            type="email"
            autocomplete="email"
            placeholder="admin@example.com"
            class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
            required
          />
        </div>

        <div class="flex flex-col gap-1.5">
          <label class="text-xs font-semibold text-text-2 tracking-wide">Password</label>
          <input
            v-model="password"
            type="password"
            autocomplete="new-password"
            placeholder="••••••••"
            class="bg-surface-2 border border-zinc-300 dark:border-[#3c3c3c] rounded-md text-text-1 px-3 py-2 text-[13px] outline-none focus:border-brand-500 focus:ring-3 focus:ring-brand-100 dark:focus:ring-brand-900/20 placeholder:text-zinc-400 dark:placeholder:text-[#6a6a6a] transition-all"
            required
            minlength="8"
          />
        </div>

        <p v-if="error" class="text-xs text-red-500">{{ error }}</p>

        <button
          type="submit"
          :disabled="loading"
          class="mt-1 px-4 py-2 rounded-md text-[13px] font-medium bg-brand-600 text-white border-none cursor-pointer hover:bg-brand-700 disabled:opacity-50 disabled:cursor-default transition-colors"
        >
          {{ loading ? 'Creating account…' : 'Create admin account' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../../api/client.js'
import { useAuth } from './useAuth.js'

const router = useRouter()
const { check } = useAuth()

const name = ref('')
const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await api.setup({ name: name.value, email: email.value, password: password.value })
    await check()
    router.push('/')
  } catch (e) {
    error.value = e.message || 'Setup failed. Please try again.'
  } finally {
    loading.value = false
  }
}
</script>
