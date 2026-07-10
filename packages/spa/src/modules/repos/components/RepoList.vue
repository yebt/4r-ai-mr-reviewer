<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import type { Repo } from '@shared/api/types'
import { useReposStore } from '@modules/repos/store'
import { useAccountsStore } from '@modules/accounts/store'
import { useProvidersStore } from '@modules/providers/store'

const emit = defineEmits<{ edit: [repo: Repo] }>()

const repos = useReposStore()
const accounts = useAccountsStore()
const providers = useProvidersStore()
const busyId = ref<string | null>(null)

onMounted(() => {
  if (repos.items.length === 0) repos.fetchAll()
  if (accounts.items.length === 0) accounts.fetchAll()
  if (providers.items.length === 0) providers.fetchAll()
})

const accountName = computed(
  () => (id: string) => accounts.items.find((a) => a.id === id)?.name ?? '—',
)
const providerLabel = computed(() => (repo: Repo) => {
  if (!repo.providerId) return 'default provider'
  const p = providers.items.find((x) => x.id === repo.providerId)
  return p ? p.name : 'unknown provider'
})

async function remove(id: string) {
  const r = repos.items.find((x) => x.id === id)
  const ok = await confirm({
    title: 'Delete repository',
    message: `Delete "${r?.name}"? Its reviews will be removed too.`,
    danger: true,
  })
  if (!ok) return

  busyId.value = id
  try {
    await repos.remove(id)
    toast.success('Repository deleted')
  } catch (e) {
    repos.error = errorMessage(e)
  } finally {
    busyId.value = null
  }
}
</script>

<template>
  <div>
    <div class="label-mono mb-3">{{ repos.items.length }} repository(ies)</div>

    <p v-if="repos.loading" class="text-muted py-3 text-sm">Loading…</p>
    <p v-else-if="repos.error" class="text-danger py-3 text-sm">{{ repos.error }}</p>
    <EmptyState
      v-else-if="repos.items.length === 0"
      icon="i-lucide-folder-git-2"
      title="No repositories yet"
      hint="Track a repository to start reviewing its merge requests."
    />

    <ul v-else class="border-line/50 border-t">
      <li v-for="r in repos.items" :key="r.id" class="row flex-wrap justify-between gap-y-2">
        <div class="min-w-0 flex-1">
          <RouterLink :to="`/repos/${r.id}`" class="text-ink hover:text-accent text-sm">
            {{ r.name }}
          </RouterLink>
          <div class="text-muted truncate font-mono text-xs">{{ r.url }}</div>
          <div class="label-mono mt-0.5">
            {{ accountName(r.accountId) }} · {{ providerLabel(r)
            }}<template v-if="r.model"> · {{ r.model }}</template>
          </div>
        </div>
        <div class="flex w-full items-center justify-end gap-1 sm:w-auto">
          <RouterLink :to="`/repos/${r.id}`" class="btn-ghost text-xs">Open</RouterLink>
          <button
            class="btn-ghost hover:text-ink"
            :aria-label="`Reassign ${r.name}`"
            @click="emit('edit', r)"
          >
            <span class="i-lucide-pencil text-sm" aria-hidden="true" />
          </button>
          <button
            class="btn-ghost hover:text-danger"
            :disabled="busyId === r.id"
            :aria-label="`Delete ${r.name}`"
            @click="remove(r.id)"
          >
            <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
