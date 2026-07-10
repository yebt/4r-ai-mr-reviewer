<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import type { Provider } from '@shared/api/types'
import { useProvidersStore } from '@modules/providers/store'

const emit = defineEmits<{ edit: [provider: Provider] }>()

const store = useProvidersStore()
const busyId = ref<string | null>(null)

async function run(id: string, fn: () => Promise<void>): Promise<boolean> {
  busyId.value = id
  try {
    await fn()
    return true
  } catch (e) {
    store.error = errorMessage(e)
    return false
  } finally {
    busyId.value = null
  }
}

async function setDefault(p: Provider) {
  if (await run(p.id, () => store.setDefault(p.id))) toast.success(`${p.name} is now the default`)
}

async function removeProvider(p: Provider) {
  const ok = await confirm({
    title: 'Delete provider',
    message: `Delete "${p.name}"?`,
    danger: true,
  })
  if (!ok) return
  if (await run(p.id, () => store.remove(p.id))) toast.success('Provider deleted')
}

onMounted(() => {
  if (store.items.length === 0) store.fetchAll()
})
</script>

<template>
  <div>
    <div class="label-mono mb-3">{{ store.items.length }} provider(s)</div>

    <p v-if="store.loading" class="text-muted py-3 text-sm">Loading…</p>
    <p v-else-if="store.error" class="text-danger py-3 text-sm">{{ store.error }}</p>
    <EmptyState
      v-else-if="store.items.length === 0"
      icon="i-lucide-cpu"
      title="No providers yet"
      hint="Add an AI provider to run reviews."
    />

    <ul v-else class="border-line/50 border-t">
      <li v-for="p in store.items" :key="p.id" class="row justify-between">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="text-ink truncate text-sm">{{ p.name }}</span>
            <span v-if="p.isDefault" class="chip text-accent">default</span>
          </div>
          <div class="text-muted truncate font-mono text-xs">
            {{ p.kind }}<template v-if="p.model"> · {{ p.model }}</template>
          </div>
        </div>
        <div class="flex shrink-0 items-center gap-1">
          <button
            v-if="!p.isDefault"
            class="btn-ghost text-xs"
            :disabled="busyId === p.id"
            @click="setDefault(p)"
          >
            Set default
          </button>
          <button
            class="btn-ghost hover:text-ink"
            :aria-label="`Edit ${p.name}`"
            @click="emit('edit', p)"
          >
            <span class="i-lucide-pencil text-sm" aria-hidden="true" />
          </button>
          <button
            class="btn-ghost hover:text-danger"
            :disabled="busyId === p.id"
            :aria-label="`Delete ${p.name}`"
            @click="removeProvider(p)"
          >
            <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
