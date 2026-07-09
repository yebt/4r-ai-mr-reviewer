<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import AppSidebar from '@shared/components/layout/AppSidebar.vue'
import AppBottomNav from '@shared/components/layout/AppBottomNav.vue'
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

    <!-- Mobile: off-canvas drawer (opened from the bottom nav's "More") -->
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
      <div class="mx-auto max-w-5xl px-4 py-6 pb-24 md:px-8 md:py-8 md:pb-8">
        <!-- Permanent breadcrumb bar: reserved height keeps the layout stable. -->
        <div class="mb-6 min-h-[1.1rem]">
          <Breadcrumbs :items="breadcrumbs.items" />
        </div>
        <RouterView />
      </div>
    </main>

    <!-- Mobile: bottom navigation -->
    <AppBottomNav @more="sidebarOpen = true" />
  </div>

  <ConfirmDialog />
  <ToastHost />
</template>
