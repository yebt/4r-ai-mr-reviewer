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
 *
 * AppShell owns the desktop layout, so it's the single source of truth for
 * the sidebar's rendered width: it derives `--sidebar-w` from the shared
 * useSidebar() collapse state and exposes it as a CSS var on the desktop
 * wrapper; AppSidebar.vue only ever consumes `var(--sidebar-w)`, never
 * duplicates the width literals.
 */
import { computed } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import { useCommandPalette } from '@shared/composables/useCommandPalette'
import { useColorScheme } from '@shared/composables/useColorScheme'
import { useSidebar } from '@shared/composables/useSidebar'
import AppSidebar from '../organisms/AppSidebar.vue'
import AppBottomNav from '../organisms/AppBottomNav.vue'
import Icon from '../atoms/Icon.vue'

const { open: openPalette } = useCommandPalette()
const { colorMode } = useColorScheme()
const { collapsed: sidebarCollapsed } = useSidebar()

const themeToggleIcon = computed(() => (colorMode.value === 'dark' ? 'sun' : 'moon'))
const sidebarWidthStyle = computed(() => ({ '--sidebar-w': sidebarCollapsed.value ? '3.5rem' : '15rem' }))

function toggleTheme() {
  colorMode.value = colorMode.value === 'dark' ? 'light' : 'dark'
}

// Single primary breakpoint (768px) is enough for this app's two-line
// strategy — see docs/ui-patterns.md #8.
const isDesktop = useMediaQuery('(min-width: 768px)')
</script>

<template>
  <div class="min-h-dvh bg-bg-app text-text">
    <!-- Desktop: persistent sidebar + main column -->
    <div v-if="isDesktop" class="flex min-h-dvh" :style="sidebarWidthStyle">
      <AppSidebar />
      <div class="flex min-w-0 flex-1 flex-col">
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
        <div class="flex items-center gap-2">
          <button
            type="button"
            class="flex size-9 items-center justify-center rounded-md border border-line bg-bg-panel text-text-muted transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            aria-label="Toggle theme"
            @click="toggleTheme"
          >
            <Icon :name="themeToggleIcon" size="sm" />
          </button>
          <button
            type="button"
            class="flex size-9 items-center justify-center rounded-md border border-line bg-bg-panel text-text-muted transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
            aria-label="Search"
            @click="openPalette()"
          >
            <Icon name="search" size="sm" />
          </button>
        </div>
      </header>

      <main class="flex-1 overflow-y-auto px-4 py-4">
        <slot />
      </main>

      <AppBottomNav />
    </div>
  </div>
</template>
