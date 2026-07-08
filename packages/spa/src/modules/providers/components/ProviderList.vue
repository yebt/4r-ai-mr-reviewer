<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { useProvidersStore } from '@modules/providers/store'

const store = useProvidersStore()
const busyId = ref<string | null>(null)

onMounted(() => {
  if (store.items.length === 0) store.fetchAll()
})

async function run(id: string, fn: () => Promise<void>) {
  busyId.value = id
  try {
    await fn()
  } catch (e) {
    store.error = errorMessage(e)
  } finally {
    busyId.value = null
  }
}
</script>

<template>
  <div>
    <div class="label-mono mb-3">{{ store.items.length }} provider(s)</div>

    <p v-if="store.loading" class="py-3 text-sm text-muted">Loading…</p>
    <p v-else-if="store.error" class="py-3 text-sm text-danger">{{ store.error }}</p>
    <p v-else-if="store.items.length === 0" class="py-3 text-sm text-muted">
      No providers yet. Add one to run reviews.
    </p>

    <ul v-else class="border-t border-line/50">
      <li v-for="p in store.items" :key="p.id" class="row justify-between">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="truncate text-sm text-ink">{{ p.name }}</span>
            <span v-if="p.isDefault" class="chip text-accent">default</span>
          </div>
          <div class="truncate font-mono text-xs text-muted">
            {{ p.kind }}<template v-if="p.model"> · {{ p.model }}</template>
          </div>
        </div>
        <div class="flex shrink-0 items-center gap-1">
          <button
            v-if="!p.isDefault"
            class="btn-ghost text-xs"
            :disabled="busyId === p.id"
            @click="run(p.id, () => store.setDefault(p.id))"
          >
            Set default
          </button>
          <button
            class="btn-ghost hover:text-danger"
            :disabled="busyId === p.id"
            :aria-label="`Delete ${p.name}`"
            @click="run(p.id, () => store.remove(p.id))"
          >
            <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
