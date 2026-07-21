<script setup lang="ts">
import { useRouter } from 'vue-router'
import AppSidebar from '@shared/components/layout/AppSidebar.vue'
import AppBottomNav from '@shared/components/layout/AppBottomNav.vue'
import ConfirmDialog from '@shared/components/ui/ConfirmDialog.vue'
import ToastHost from '@shared/components/ui/ToastHost.vue'
import Breadcrumbs from '@shared/components/ui/Breadcrumbs.vue'
import { setBreadcrumbs, useBreadcrumbs } from '@shared/composables/useBreadcrumbs'

const { state: breadcrumbs, sticky, toggleSticky } = useBreadcrumbs()
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
        <!-- Permanent breadcrumb bar: reserved height keeps the layout stable.
             When a view opts in (pinnable) the user can pin it sticky; the sticky
             state spans the content column edge-to-edge with a backing so scrolled
             content doesn't show through. -->
        <div
          class="mb-6 flex min-h-[1.1rem] items-center justify-between gap-2"
          :class="
            breadcrumbs.pinnable && sticky
              ? 'bg-canvas border-line/50 sticky top-0 z-20 -mx-4 border-b px-4 py-2 md:-mx-8 md:px-8'
              : ''
          "
        >
          <Breadcrumbs :items="breadcrumbs.items" />
          <button
            v-if="breadcrumbs.pinnable"
            type="button"
            class="btn-ghost shrink-0 text-xs"
            :aria-pressed="sticky"
            :title="sticky ? 'Unpin breadcrumb' : 'Pin breadcrumb (sticky)'"
            @click="toggleSticky"
          >
            <span
              :class="sticky ? 'i-lucide-pin-off' : 'i-lucide-pin'"
              class="text-sm"
              aria-hidden="true"
            />
          </button>
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
