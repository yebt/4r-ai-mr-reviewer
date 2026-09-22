<script setup lang="ts">
/**
 * Desktop persistent left sidebar (docs/ui-patterns.md #1): brand, a
 * search/⌘K trigger with a visible Kbd hint, the central nav model, and a
 * user menu with Logout. Only mounted on desktop — see AppShell.vue, which
 * swaps this out for AppBottomNav via a JS breakpoint check rather than
 * hiding it with CSS.
 */
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@modules/auth/store'
import { useCommandPalette } from '@shared/composables/useCommandPalette'
import { useColorScheme } from '@shared/composables/useColorScheme'
import { isNavItemActive, navItems } from '@core/nav'
import Icon from '../atoms/Icon.vue'
import Kbd from '../atoms/Kbd.vue'
import Text from '../atoms/Text.vue'
import Button from '../atoms/Button.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { open: openPalette } = useCommandPalette()
const { colorMode } = useColorScheme()

const showLogout = computed(() => auth.enabled && auth.authenticated)
const themeToggleIcon = computed(() => (colorMode.value === 'dark' ? 'sun' : 'moon'))

function toggleTheme() {
  colorMode.value = colorMode.value === 'dark' ? 'light' : 'dark'
}

async function handleLogout() {
  await auth.logout()
  await router.push('/login')
}
</script>

<template>
  <aside class="flex w-60 shrink-0 flex-col gap-1 border-r border-line bg-bg-panel p-3">
    <div class="flex items-center gap-2 px-2 py-2">
      <div
        class="grid size-7 place-items-center rounded-md bg-accent text-text-on-accent"
        aria-hidden="true"
      >
        <Icon name="layers" size="sm" />
      </div>
      <Text as="span" size="md" class="font-semibold tracking-tight">4R</Text>
    </div>

    <button
      type="button"
      class="flex min-h-9 items-center justify-between gap-2 rounded-md border border-line bg-bg-panel-raised px-2.5 text-sm text-text-muted transition-colors hover:bg-bg-hover hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
      @click="openPalette()"
    >
      <span class="flex items-center gap-2">
        <Icon name="search" size="sm" />
        Search
      </span>
      <Kbd>⌘K</Kbd>
    </button>

    <nav aria-label="Primary" class="mt-2 flex flex-1 flex-col gap-0.5">
      <RouterLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        :aria-current="isNavItemActive(item, route.path) ? 'page' : undefined"
        class="flex min-h-11 items-center gap-2.5 rounded-md px-2.5 text-sm text-text-muted transition-colors hover:bg-bg-hover hover:text-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring aria-[current=page]:bg-accent-subtle-bg aria-[current=page]:font-medium aria-[current=page]:text-accent-text-strong"
      >
        <Icon :name="item.icon" size="sm" />
        {{ item.label }}
      </RouterLink>
    </nav>

    <div class="flex items-center gap-1">
      <Button
        variant="ghost"
        size="sm"
        aria-label="Toggle theme"
        @click="toggleTheme"
      >
        <Icon :name="themeToggleIcon" size="sm" />
      </Button>
      <Button
        v-if="showLogout"
        variant="ghost"
        size="sm"
        class="flex-1 justify-start"
        @click="handleLogout"
      >
        <template #leading>
          <Icon name="log-out" size="sm" />
        </template>
        Log out
      </Button>
    </div>
  </aside>
</template>
