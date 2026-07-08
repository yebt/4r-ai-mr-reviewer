<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { useAccountsStore } from '@modules/accounts/store'

const store = useAccountsStore()
const removingId = ref<string | null>(null)

onMounted(() => {
  if (store.items.length === 0) store.fetchAll()
})

async function remove(id: string) {
  removingId.value = id
  try {
    await store.remove(id)
  } catch (e) {
    store.error = errorMessage(e)
  } finally {
    removingId.value = null
  }
}
</script>

<template>
  <div>
    <div class="label-mono mb-3">{{ store.items.length }} account(s)</div>

    <p v-if="store.loading" class="py-3 text-sm text-muted">Loading…</p>
    <p v-else-if="store.error" class="py-3 text-sm text-danger">{{ store.error }}</p>
    <p v-else-if="store.items.length === 0" class="py-3 text-sm text-muted">
      No accounts yet. Add one to start.
    </p>

    <ul v-else class="border-t border-line/50">
      <li v-for="acc in store.items" :key="acc.id" class="row justify-between">
        <div class="min-w-0">
          <div class="truncate text-sm text-ink">{{ acc.name }}</div>
          <div class="truncate font-mono text-xs text-muted">{{ acc.baseUrl }}</div>
        </div>
        <button
          class="btn-ghost shrink-0 hover:text-danger"
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
