<script setup lang="ts">
import { useRouter } from 'vue-router'
import AppSidebar from '@shared/components/layout/AppSidebar.vue'
import AppBottomNav from '@shared/components/layout/AppBottomNav.vue'
import ConfirmDialog from '@shared/components/ui/ConfirmDialog.vue'
import ToastHost from '@shared/components/ui/ToastHost.vue'
import Breadcrumbs from '@shared/components/ui/Breadcrumbs.vue'
import { setBreadcrumbs, useBreadcrumbs } from '@shared/composables/useBreadcrumbs'

const breadcrumbs = useBreadcrumbs()
const router = useRouter()

// Reset the breadcrumb trail before each navigation; pages declare their own.
router.beforeEach(() => {
  setBreadcrumbs([])
  return true
})
</script>

<template>
  <div class="flex h-screen w-screen overflow-hidden">
    <!-- Desktop: static sidebar -->
    <div class="hidden shrink-0 md:block">
      <AppSidebar class="h-full" />
    </div>

    <main class="flex-1 overflow-y-auto">
      <div class="mx-auto max-w-5xl px-4 py-6 pb-24 md:px-8 md:py-8 md:pb-8">
        <!-- Permanent breadcrumb bar: reserved height keeps the layout stable. -->
        <div class="mb-6 min-h-[1.1rem]" v-if="breadcrumbs.items.length">
          <Breadcrumbs :items="breadcrumbs.items" />
        </div>
        <RouterView />
      </div>
    </main>

    <!-- Mobile: bottom navigation -->
    <AppBottomNav />
  </div>

  <ConfirmDialog />
  <ToastHost />
</template>
