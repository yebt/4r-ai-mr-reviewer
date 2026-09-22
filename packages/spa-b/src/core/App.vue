<script setup lang="ts">
/**
 * Root component: branches between a bare render (the public /login route)
 * and the full centralized shell (everything else) so the shell — and its
 * globally-mounted CommandPalette — never mounts on a route the user isn't
 * authenticated into yet.
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppShell from '@shared/ui/design-system/templates/AppShell.vue'
import CommandPalette from '@shared/ui/design-system/organisms/CommandPalette.vue'

const route = useRoute()
const isBare = computed(() => route.meta.public === true)
</script>

<template>
  <RouterView v-if="isBare" />
  <template v-else>
    <AppShell>
      <RouterView />
    </AppShell>
    <CommandPalette />
  </template>
</template>

<style scoped></style>
