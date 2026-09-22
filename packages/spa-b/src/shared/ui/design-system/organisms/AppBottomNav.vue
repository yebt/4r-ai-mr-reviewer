<script setup lang="ts">
/**
 * Mobile bottom tab bar (docs/ui-patterns.md #1): primary destinations as
 * real tabs, plus a "More" tab that opens a Reka Drawer (bottom sheet, not
 * a Dialog — pattern #3) listing overflow nav + logout. Only mounted on
 * mobile — see AppShell.vue.
 */
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { DrawerClose, DrawerContent, DrawerHandle, DrawerOverlay, DrawerPortal, DrawerRoot, DrawerTitle, DrawerTrigger } from 'reka-ui'
import { useAuthStore } from '@modules/auth/store'
import { isNavItemActive, primaryNavItems, secondaryNavItems } from '@core/nav'
import Icon from '../atoms/Icon.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const drawerOpen = ref(false)

const showLogout = computed(() => auth.enabled && auth.authenticated)
const hasOverflow = computed(() => secondaryNavItems.length > 0 || showLogout.value)

function goTo(to: string) {
  drawerOpen.value = false
  router.push(to)
}

async function handleLogout() {
  drawerOpen.value = false
  await auth.logout()
  await router.push('/login')
}
</script>

<template>
  <nav
    aria-label="Primary"
    class="sticky bottom-0 z-20 flex items-stretch justify-around border-t border-line bg-bg-app/95 backdrop-blur-sm"
    style="padding-bottom: env(safe-area-inset-bottom)"
  >
    <RouterLink
      v-for="item in primaryNavItems"
      :key="item.to"
      :to="item.to"
      :aria-current="isNavItemActive(item, route.path) ? 'page' : undefined"
      class="flex min-h-11 flex-1 flex-col items-center justify-center gap-0.5 py-2 text-xs text-text-muted transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-focus-ring aria-[current=page]:text-accent-text"
    >
      <Icon :name="item.icon" size="md" />
      {{ item.label }}
    </RouterLink>

    <DrawerRoot v-if="hasOverflow" v-model:open="drawerOpen">
      <DrawerTrigger
        class="flex min-h-11 flex-1 flex-col items-center justify-center gap-0.5 py-2 text-xs text-text-muted transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-focus-ring"
      >
        <Icon name="ellipsis" size="md" />
        More
      </DrawerTrigger>
      <DrawerPortal>
        <DrawerOverlay class="overlay z-30" />
        <DrawerContent
          class="fixed inset-x-0 bottom-0 z-40 flex max-h-[80dvh] flex-col rounded-t-xl border-t border-line bg-bg-panel-raised pb-[env(safe-area-inset-bottom)] shadow-token-lg focus:outline-none"
        >
          <DrawerHandle class="mx-auto mt-2.5 h-1.5 w-10 shrink-0 rounded-full bg-line-strong" />
          <div class="flex items-center justify-between px-4 py-3">
            <DrawerTitle class="text-md font-semibold text-text">More</DrawerTitle>
            <DrawerClose
              class="flex size-9 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
              aria-label="Close"
            >
              <Icon name="x" size="sm" />
            </DrawerClose>
          </div>

          <div class="flex flex-col gap-1 overflow-y-auto px-3 pb-3">
            <button
              v-for="item in secondaryNavItems"
              :key="item.to"
              type="button"
              class="flex min-h-11 items-center gap-2.5 rounded-md px-3 text-sm text-text transition-colors hover:bg-bg-hover"
              @click="goTo(item.to)"
            >
              <Icon :name="item.icon" size="sm" />
              {{ item.label }}
            </button>

            <button
              v-if="showLogout"
              type="button"
              class="flex min-h-11 items-center gap-2.5 rounded-md px-3 text-sm text-danger-text transition-colors hover:bg-bg-hover"
              @click="handleLogout"
            >
              <Icon name="log-out" size="sm" />
              Log out
            </button>
          </div>
        </DrawerContent>
      </DrawerPortal>
    </DrawerRoot>
  </nav>
</template>
