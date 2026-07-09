<script setup lang="ts">
const emit = defineEmits<{ navigate: [] }>()

const links = [
  { to: '/', label: 'Quick access', icon: 'i-lucide-zap', exact: true },
  { to: '/repos', label: 'Repositories', icon: 'i-lucide-folder-git-2' },
  { to: '/reviews', label: 'Reviews', icon: 'i-lucide-list-checks' },
  { to: '/accounts', label: 'Accounts', icon: 'i-lucide-users' },
  { to: '/providers', label: 'AI providers', icon: 'i-lucide-cpu' },
  { to: '/skills', label: 'Skills', icon: 'i-lucide-book-open' },
]
</script>

<template>
  <aside class="flex w-52 shrink-0 flex-col border-r border-line/60 bg-canvas">
    <div class="px-5 py-5">
      <div class="font-mono text-sm font-semibold tracking-tight text-ink">ai&#8209;reviewer</div>
      <div class="label-mono mt-1">4R quality gate</div>
    </div>
    <nav class="flex flex-col">
      <RouterLink
        v-for="link in links"
        :key="link.to"
        :to="link.to"
        class="flex items-center gap-3 border-l-2 border-transparent px-5 py-2.5 text-sm text-muted transition-colors hover:text-ink"
        active-class="border-accent! text-ink!"
        :exact-active-class="link.exact ? 'border-accent! text-ink!' : ''"
        @click="emit('navigate')"
      >
        <span :class="link.icon" class="text-[0.95rem] opacity-70" aria-hidden="true" />
        {{ link.label }}
      </RouterLink>
    </nav>
  </aside>
</template>
