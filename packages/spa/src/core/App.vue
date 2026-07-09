<script setup lang="ts">
import { useRouter } from 'vue-router'
import AppSidebar from '@shared/components/layout/AppSidebar.vue'
import ConfirmDialog from '@shared/components/ui/ConfirmDialog.vue'
import Breadcrumbs from '@shared/components/ui/Breadcrumbs.vue'
import { setBreadcrumbs, useBreadcrumbs } from '@shared/composables/useBreadcrumbs'

const breadcrumbs = useBreadcrumbs()
const router = useRouter()

// Reset the trail before each navigation; the destination page declares its own.
router.beforeEach(() => {
  setBreadcrumbs([])
  return true
})
</script>

<template>
  <div class="flex h-screen w-screen overflow-hidden">
    <AppSidebar />
    <main class="flex-1 overflow-y-auto">
      <div class="mx-auto max-w-5xl px-8 py-8">
        <!-- Permanent bar: reserved height keeps the layout stable. -->
        <div class="mb-6 min-h-[1.1rem]">
          <Breadcrumbs :items="breadcrumbs.items" />
        </div>
        <RouterView />
      </div>
    </main>
  </div>
  <ConfirmDialog />
</template>
