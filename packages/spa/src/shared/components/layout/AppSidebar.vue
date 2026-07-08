<script setup lang="ts">
const links = [
  { to: '/', label: 'Quick access', icon: 'i-lucide-zap', exact: true },
  { to: '/repos', label: 'Repositories', icon: 'i-lucide-folder-git-2' },
  { to: '/accounts', label: 'Accounts', icon: 'i-lucide-users' },
  { to: '/providers', label: 'AI providers', icon: 'i-lucide-cpu' },
]
</script>

<template>
  <aside class="flex w-56 shrink-0 flex-col border-r border-line bg-surface">
    <div class="border-b border-line px-5 py-4">
      <div class="font-mono text-sm font-semibold tracking-tight text-ink">ai-reviewer</div>
      <div class="label-mono mt-1">4R quality gate</div>
    </div>
    <nav class="flex flex-col gap-1 p-3">
      <RouterLink
        v-for="link in links"
        :key="link.to"
        :to="link.to"
        class="flex items-center gap-3 rounded-md px-3 py-2 text-sm text-muted transition-colors hover:(bg-surface-2 text-ink)"
        active-class="bg-surface-2! text-ink!"
        :exact-active-class="link.exact ? 'bg-surface-2! text-ink!' : ''"
      >
        <span :class="link.icon" class="text-base" aria-hidden="true" />
        {{ link.label }}
      </RouterLink>
    </nav>
  </aside>
</template>
