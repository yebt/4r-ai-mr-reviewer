<route lang="json">
{
  "meta": { "public": true }
}
</route>

<script setup lang="ts">
/**
 * Catch-all 404 (unplugin-vue-router maps `[...path].vue` to
 * `/:path(.*)*`) — reached for any URL that doesn't match a real route.
 * `meta.public: true` lets the router guard (src/core/router/index.ts) let
 * an unauthenticated visitor straight through instead of bouncing them to
 * /login first; App.vue still renders it inside the normal AppShell (see
 * its isBare comment) since this isn't the login route.
 */
import { useRouter } from 'vue-router'
import { Button, Heading, Text } from '@shared/ui/design-system'

const router = useRouter()

function goHome() {
  router.push('/')
}
</script>

<template>
  <div class="flex min-h-[60dvh] flex-col items-center justify-center gap-3 text-center">
    <Text as="span" size="3xl" mono class="text-accent-text-strong">404</Text>
    <Heading :level="1" size="xl">Page not found</Heading>
    <Text muted class="max-w-sm">
      The page you're looking for doesn't exist or may have been moved.
    </Text>
    <Button class="mt-2" @click="goHome">Back home</Button>
  </div>
</template>
