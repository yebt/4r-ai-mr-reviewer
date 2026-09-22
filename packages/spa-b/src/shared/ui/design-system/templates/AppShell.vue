<script setup lang="ts">
/**
 * Centralized responsive app shell — the single entry point every
 * authenticated route renders inside (see App.vue). Two structural
 * layouts, not a CSS reflow, split at the 768px breakpoint per
 * docs/ui-patterns.md #1 and #8: desktop gets a persistent left sidebar,
 * mobile gets a top bar + bottom tab bar. The swap runs on `useMediaQuery`
 * (a real JS branch) rather than `hidden md:flex`, so only the active
 * tree is ever mounted — matching the playbook's "two actual render
 * branches, not just reflow" guidance for structural nav changes.
 */
import { computed } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import { useRoute } from 'vue-router'
import { navItems } from '@core/nav'
import { useCommandPalette } from '@shared/composables/useCommandPalette'
import AppSidebar from '../organisms/AppSidebar.vue'
import AppBottomNav from '../organisms/AppBottomNav.vue'
import AppHeader from '../organisms/AppHeader.vue'
import Icon from '../atoms/Icon.vue'

const route = useRoute()
const { open: openPalette } = useCommandPalette()

// Single primary breakpoint (768px) is enough for this app's two-line
// strategy — see docs/ui-patterns.md #8.
const isDesktop = useMediaQuery('(min-width: 768px)')

const pageTitle = computed(() => {
  const match = navItems.find((item) => item.to === route.path)
  return match?.label ?? route.meta.title ?? 'Overview'
})
</script>

<template>
  <div class="min-h-dvh bg-bg-app text-text">
    <!-- Desktop: persistent sidebar + main column -->
    <div v-if="isDesktop" class="flex min-h-dvh">
      <AppSidebar />
      <div class="flex min-w-0 flex-1 flex-col">
        <AppHeader :title="pageTitle">
          <template #actions>
            <slot name="header-actions" />
          </template>
        </AppHeader>
        <main class="flex-1 overflow-y-auto px-6 py-6">
          <slot />
        </main>
      </div>
    </div>

    <!-- Mobile: top bar + scrollable content + bottom tab bar -->
    <div v-else class="flex min-h-dvh flex-col">
      <header
        class="sticky top-0 z-20 flex items-center justify-between gap-3 border-b border-line bg-bg-app/95 px-4 py-3 backdrop-blur-sm"
        style="padding-top: max(0.75rem, env(safe-area-inset-top))"
      >
        <div class="flex items-center gap-2">
          <div
            class="grid size-6 place-items-center rounded-md bg-accent text-text-on-accent"
            aria-hidden="true"
          >
            <Icon name="layers" size="xs" />
          </div>
          <span class="text-md font-semibold tracking-tight">4R</span>
        </div>
        <button
          type="button"
          class="flex size-9 items-center justify-center rounded-md border border-line bg-bg-panel text-text-muted transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
          aria-label="Search"
          @click="openPalette()"
        >
          <Icon name="search" size="sm" />
        </button>
      </header>

      <main class="flex-1 overflow-y-auto px-4 py-4">
        <slot />
      </main>

      <AppBottomNav />
    </div>
  </div>
</template>
