<script setup lang="ts">
/**
 * Desktop persistent left sidebar (docs/ui-patterns.md #1): brand, a
 * search/⌘K trigger with a visible Kbd hint, the central nav model, and a
 * user menu with Logout. Only mounted on desktop — see AppShell.vue, which
 * swaps this out for AppBottomNav via a JS breakpoint check rather than
 * hiding it with CSS.
 *
 * Collapsible via useSidebar (persisted, shared with AppShell.vue — see
 * that file for the single-source-of-truth width). Collapsed renders an
 * icon-only rail: labels stay in the DOM as `sr-only` text (real accessible
 * names, not just `aria-label`) plus a `title` tooltip, so every control
 * keeps its accessible name either way.
 *
 * The Icon atom's registry has no `chevron-left`/`chevron-right` (this file
 * doesn't own that atom) — `chevron-down` rotated ±90° stands in for both.
 */
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@modules/auth/store'
import { useCommandPalette } from '@shared/composables/useCommandPalette'
import { useColorScheme } from '@shared/composables/useColorScheme'
import { useSidebar } from '@shared/composables/useSidebar'
import { isNavItemActive, sidebarNavItems } from '@core/nav'
import Icon from '../atoms/Icon.vue'
import Kbd from '../atoms/Kbd.vue'
import Text from '../atoms/Text.vue'
import Button from '../atoms/Button.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { open: openPalette } = useCommandPalette()
const { colorMode } = useColorScheme()
const { collapsed, toggle: toggleCollapsed } = useSidebar()

const showLogout = computed(() => auth.enabled && auth.authenticated)
const themeToggleIcon = computed(() => (colorMode.value === 'dark' ? 'sun' : 'moon'))
const toggleLabel = computed(() => (collapsed.value ? 'Expand sidebar' : 'Collapse sidebar'))

function toggleTheme() {
  colorMode.value = colorMode.value === 'dark' ? 'light' : 'dark'
}

async function handleLogout() {
  await auth.logout()
  await router.push('/login')
}
</script>

<template>
  <aside
    class="flex w-[var(--sidebar-w,15rem)] shrink-0 flex-col gap-1 overflow-hidden border-r border-line bg-bg-panel p-3 transition-[width] duration-[var(--duration-base)] ease-[var(--ease-standard)]"
  >
    <div class="flex items-center gap-2 px-2 py-2" :class="collapsed ? 'justify-center' : 'justify-between'">
      <div class="flex min-w-0 items-center gap-2">
        <div
          class="grid size-7 shrink-0 place-items-center rounded-md bg-accent text-text-on-accent"
          aria-hidden="true"
        >
          <Icon name="layers" size="sm" />
        </div>
        <Text v-if="!collapsed" as="span" size="md" class="truncate font-semibold tracking-tight">4R</Text>
        <span v-else class="sr-only">4R</span>
      </div>
      <button
        v-if="!collapsed"
        type="button"
        class="flex size-7 shrink-0 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
        :aria-expanded="!collapsed"
        :aria-label="toggleLabel"
        :title="toggleLabel"
        @click="toggleCollapsed"
      >
        <Icon name="chevron-down" size="sm" class="rotate-90" />
      </button>
    </div>

    <button
      v-if="collapsed"
      type="button"
      class="flex min-h-8 shrink-0 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      :aria-expanded="!collapsed"
      :aria-label="toggleLabel"
      :title="toggleLabel"
      @click="toggleCollapsed"
    >
      <Icon name="chevron-down" size="sm" class="-rotate-90" />
    </button>

    <button
      type="button"
      class="flex min-h-9 items-center gap-2 rounded-md border border-line bg-bg-panel-raised px-2.5 text-sm text-text-muted transition-colors hover:bg-bg-hover hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      :class="collapsed ? 'justify-center' : 'justify-between'"
      :title="collapsed ? 'Search' : undefined"
      @click="openPalette()"
    >
      <span class="flex min-w-0 items-center gap-2">
        <Icon name="search" size="sm" class="shrink-0" />
        <span :class="{ 'sr-only': collapsed }">Search</span>
      </span>
      <Kbd v-if="!collapsed">⌘K</Kbd>
    </button>

    <nav aria-label="Primary" class="mt-2 flex flex-1 flex-col gap-0.5">
      <RouterLink
        v-for="item in sidebarNavItems"
        :key="item.to"
        :to="item.to"
        :aria-current="isNavItemActive(item, route.path) ? 'page' : undefined"
        :title="collapsed ? item.label : undefined"
        class="flex min-h-11 items-center gap-2.5 rounded-md px-2.5 text-sm text-text-muted transition-colors hover:bg-bg-hover hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring aria-[current=page]:bg-accent-subtle-bg aria-[current=page]:font-medium aria-[current=page]:text-accent-text-strong"
        :class="collapsed ? 'justify-center px-0' : undefined"
      >
        <Icon :name="item.icon" size="sm" class="shrink-0" />
        <span :class="{ 'sr-only': collapsed }">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <div class="flex items-center gap-1" :class="collapsed ? 'flex-col' : ''">
      <Button
        variant="ghost"
        size="sm"
        aria-label="Toggle theme"
        title="Toggle theme"
        @click="toggleTheme"
      >
        <Icon :name="themeToggleIcon" size="sm" />
      </Button>
      <Button
        v-if="showLogout"
        variant="ghost"
        size="sm"
        :class="collapsed ? undefined : 'flex-1 justify-start'"
        :title="collapsed ? 'Log out' : undefined"
        @click="handleLogout"
      >
        <template #leading>
          <Icon name="log-out" size="sm" />
        </template>
        <span :class="{ 'sr-only': collapsed }">Log out</span>
      </Button>
    </div>
  </aside>
</template>
