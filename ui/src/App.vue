<template>
  <div v-if="activeWorkspaceCode" class="flex h-screen overflow-hidden">
    <div
      v-if="sidebarOpen"
      class="fixed inset-0 bg-black/50 z-40 md:hidden"
      @click="sidebarOpen = false"
    />
    <AppSidebar
      ref="sidebar"
      :workspace-code="activeWorkspaceCode"
      :active-table-code="activeTableCode"
      :open="sidebarOpen"
      @close="sidebarOpen = false"
    />
    <div class="flex-1 min-w-0 flex flex-col overflow-hidden">
      <AppTopNav :show-burger="true" :show-brand="false" @burger="sidebarOpen = true" />
      <main class="flex-1 min-w-0 overflow-y-auto bg-zinc-100 dark:bg-[#1e1e1e] flex flex-col">
        <RouterView />
      </main>
    </div>
  </div>
  <RouterView v-else />
  <AppNotifications />
  <ProgressBar />
  <ConfirmDialog />
  <ReferencesBlockDialog />
</template>

<script setup>
import { ref, computed, watch, provide } from 'vue'
import { useRoute } from 'vue-router'
import AppSidebar from './foundation/AppSidebar.vue'
import AppTopNav from './foundation/AppTopNav.vue'
import AppNotifications from './foundation/AppNotifications.vue'
import ProgressBar from './foundation/ProgressBar.vue'
import ConfirmDialog from './foundation/ConfirmDialog.vue'
import ReferencesBlockDialog from './foundation/ReferencesBlockDialog.vue'
import { useTheme } from './foundation/useTheme.js'

useTheme()

const route = useRoute()
const sidebar = ref(null)
const sidebarOpen = ref(false)

const activeWorkspaceCode = computed(() => route.params.workspaceCode || null)
const activeTableCode = computed(() => route.params.tableCode || null)

watch(() => route.path, () => { sidebarOpen.value = false })

watch(activeTableCode, (newCode, oldCode) => {
  if (oldCode && !newCode) sidebar.value?.reload()
})

provide('reloadSidebar', () => sidebar.value?.reload())
</script>
