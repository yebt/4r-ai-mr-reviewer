<script setup lang="ts">
/**
 * Root component: branches between a bare render (the /login route only)
 * and the full centralized shell (everything else, including the public
 * 404 catch-all) so the shell — and its globally-mounted CommandPalette/
 * ToastHost — never mounts on the one route the user isn't authenticated
 * into yet. Checks the login route by PATH, not `meta.public` or name:
 * other public routes (e.g. the 404 catch-all, src/pages/[...path].vue) are
 * public *and* still render inside the shell; and /login is registered
 * imperatively so its name isn't in the typed-router's RouteNamedMap union.
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppShell from '@shared/ui/design-system/templates/AppShell.vue'
import CommandPalette from '@shared/ui/design-system/organisms/CommandPalette.vue'
import ToastHost from '@shared/ui/design-system/organisms/ToastHost.vue'

const route = useRoute()
const isBare = computed(() => route.path === '/login')
</script>

<template>
  <RouterView v-if="isBare" />
  <template v-else>
    <AppShell>
      <RouterView />
    </AppShell>
    <CommandPalette />
    <ToastHost />
  </template>
</template>

<style scoped></style>
