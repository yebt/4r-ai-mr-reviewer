<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import type { TelegramTarget } from '@shared/api/types'
import { useTelegramStore } from '@modules/telegram/store'

const store = useTelegramStore()
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

async function setDefault(t: TelegramTarget) {
  if (await run(t.id, () => store.setDefault(t.id))) toast.success(`${t.name} is now the default`)
}

async function sendTest(t: TelegramTarget) {
  busyId.value = t.id
  try {
    const res = await store.test(t.id)
    if (res.status === 'sent') toast.success(`Test message sent to ${t.name}`)
    else toast.error(`Test failed: ${res.status}`)
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    busyId.value = null
  }
}

async function removeTarget(t: TelegramTarget) {
  const ok = await confirm({
    title: 'Delete Telegram target',
    message: `Delete "${t.name}"?`,
    danger: true,
  })
  if (!ok) return
  if (await run(t.id, () => store.remove(t.id))) toast.success('Telegram target deleted')
}

onMounted(() => {
  if (store.items.length === 0) store.fetchAll()
})
</script>

<template>
  <div>
    <div class="label-mono mb-3">{{ store.items.length }} target(s)</div>

    <p v-if="store.loading" class="text-muted py-3 text-sm">Loading…</p>
    <p v-else-if="store.error" class="text-danger py-3 text-sm">{{ store.error }}</p>
    <EmptyState
      v-else-if="store.items.length === 0"
      icon="i-lucide-send"
      title="No Telegram targets yet"
      hint="Add a target to receive review notifications on Telegram."
    />

    <ul v-else class="border-line/50 border-t">
      <li v-for="t in store.items" :key="t.id" class="row justify-between">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="text-ink truncate text-sm">{{ t.name }}</span>
            <span v-if="t.isDefault" class="chip text-accent">default</span>
          </div>
          <div class="text-muted truncate font-mono text-xs">
            {{ t.chatId }}<template v-if="t.threadId"> · thread {{ t.threadId }}</template>
          </div>
        </div>
        <div class="flex shrink-0 items-center gap-1">
          <button class="btn-ghost text-xs" :disabled="busyId === t.id" @click="sendTest(t)">
            Send test
          </button>
          <button
            v-if="!t.isDefault"
            class="btn-ghost text-xs"
            :disabled="busyId === t.id"
            @click="setDefault(t)"
          >
            Set default
          </button>
          <button
            class="btn-ghost hover:text-danger"
            :disabled="busyId === t.id"
            :aria-label="`Delete ${t.name}`"
            @click="removeTarget(t)"
          >
            <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
