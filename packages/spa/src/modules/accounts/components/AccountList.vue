<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import { useAccountsStore } from '@modules/accounts/store'

const emit = defineEmits<{ add: [] }>()

const store = useAccountsStore()
const removingId = ref<string | null>(null)

onMounted(() => {
  if (store.items.length === 0) store.fetchAll()
})

async function remove(id: string) {
  const acc = store.items.find((a) => a.id === id)
  const ok = await confirm({
    title: 'Delete account',
    message: `Delete "${acc?.name}"? Its tracked repositories will be removed too.`,
    danger: true,
  })
  if (!ok) return

  removingId.value = id
  try {
    await store.remove(id)
    toast.success('Account deleted')
  } catch (e) {
    store.error = errorMessage(e)
  } finally {
    removingId.value = null
  }
}
</script>

<template>
  <div>
    <div class="label-mono mb-3">
      {{ store.items.length }} {{ store.items.length === 1 ? 'account' : 'accounts' }}
    </div>

    <p v-if="store.loading" class="text-muted py-3 text-sm">Loading…</p>
    <p v-else-if="store.error" class="text-danger py-3 text-sm">{{ store.error }}</p>
    <EmptyState
      v-else-if="store.items.length === 0"
      icon="i-lucide-users"
      title="No accounts yet"
      hint="A GitLab account is a base URL + a personal access token (api scope) 4R uses to read MRs and post reviews."
    >
      <template #action>
        <button class="btn-accent text-xs" @click="emit('add')">
          <span class="i-lucide-plus text-sm" aria-hidden="true" />
          Add your first account
        </button>
      </template>
    </EmptyState>

    <ul v-else class="border-line/50 border-t">
      <li v-for="acc in store.items" :key="acc.id" class="row justify-between">
        <div class="min-w-0">
          <div class="text-ink truncate text-sm">{{ acc.name }}</div>
          <div class="text-muted truncate font-mono text-xs">{{ acc.baseUrl }}</div>
        </div>
        <button
          class="btn-ghost hover:text-danger shrink-0"
          :disabled="removingId === acc.id"
          :aria-label="`Delete ${acc.name}`"
          @click="remove(acc.id)"
        >
          <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
        </button>
      </li>
    </ul>
  </div>
</template>
