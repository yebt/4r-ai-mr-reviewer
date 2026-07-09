<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import AppSidebar from '@shared/components/layout/AppSidebar.vue'
import ConfirmDialog from '@shared/components/ui/ConfirmDialog.vue'
import ToastHost from '@shared/components/ui/ToastHost.vue'
import Breadcrumbs from '@shared/components/ui/Breadcrumbs.vue'
import { setBreadcrumbs, useBreadcrumbs } from '@shared/composables/useBreadcrumbs'

const breadcrumbs = useBreadcrumbs()
const router = useRouter()
const sidebarOpen = ref(false)

// Reset the trail and close the mobile drawer before each navigation.
router.beforeEach(() => {
  setBreadcrumbs([])
  sidebarOpen.value = false
  return true
})
</script>

<template>
  <div class="flex h-screen w-screen overflow-hidden">
    <!-- Desktop: static sidebar -->
    <div class="hidden shrink-0 md:block">
      <AppSidebar class="h-full" />
    </div>

    <!-- Mobile: off-canvas drawer -->
    <Transition name="fade">
      <div
        v-if="sidebarOpen"
        class="fixed inset-0 z-40 bg-canvas/70 md:hidden"
        @click="sidebarOpen = false"
      />
    </Transition>
    <Transition name="drawer">
      <div v-if="sidebarOpen" class="fixed inset-y-0 left-0 z-50 md:hidden">
        <AppSidebar class="h-full" @navigate="sidebarOpen = false" />
      </div>
    </Transition>

    <main class="flex-1 overflow-y-auto">
      <!-- Mobile top bar -->
      <div class="flex items-center gap-3 border-b border-line/60 px-4 py-3 md:hidden">
        <button type="button" class="btn-ghost" aria-label="Open menu" @click="sidebarOpen = true">
          <span class="i-lucide-menu text-lg" aria-hidden="true" />
        </button>
        <span class="font-mono text-sm font-semibold tracking-tight text-ink">ai&#8209;reviewer</span>
      </div>

      <div class="mx-auto max-w-5xl px-4 py-6 md:px-8 md:py-8">
        <!-- Permanent breadcrumb bar: reserved height keeps the layout stable. -->
        <div class="mb-6 min-h-[1.1rem]">
          <Breadcrumbs :items="breadcrumbs.items" />
        </div>
        <RouterView />
      </div>
    </main>
  </div>

  <ConfirmDialog />
  <ToastHost />
</template>
