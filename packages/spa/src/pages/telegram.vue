<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import type { TelegramTarget } from '@shared/api/types'
import { useTelegramStore } from '@modules/telegram/store'

const store = useTelegramStore()

const blank = () => ({
  name: '',
  botToken: '',
  chatId: '',
  threadId: '',
  isDefault: false,
})
const form = reactive(blank())
const submitting = ref(false)
const formError = ref<string | null>(null)
const busyId = ref<string | null>(null)

const valid = computed(
  () => form.name.trim() && form.botToken.trim() && form.chatId.trim(),
)

async function submit() {
  if (!valid.value || submitting.value) return
  submitting.value = true
  formError.value = null
  try {
    await store.add({
      name: form.name.trim(),
      botToken: form.botToken.trim(),
      chatId: form.chatId.trim(),
      threadId: form.threadId.trim(),
      isDefault: form.isDefault,
    })
    toast.success('Telegram target added')
    Object.assign(form, blank())
  } catch (e) {
    formError.value = errorMessage(e)
  } finally {
    submitting.value = false
  }
}

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
    <PageHeader title="Telegram" label="Notifications" />

    <form class="mb-10 flex flex-col gap-5" @submit.prevent="submit">
      <div class="section-title">New target</div>

      <div>
        <label class="field-label" for="tg-name">Name</label>
        <input
          id="tg-name"
          v-model="form.name"
          class="field-underline"
          placeholder="team-channel"
          autocomplete="off"
        />
      </div>

      <div>
        <label class="field-label" for="tg-token">Bot token</label>
        <input
          id="tg-token"
          v-model="form.botToken"
          type="password"
          class="field-underline"
          placeholder="123456:ABC-…"
          autocomplete="off"
        />
      </div>

      <div>
        <label class="field-label" for="tg-chat">Chat ID</label>
        <input
          id="tg-chat"
          v-model="form.chatId"
          class="field-underline"
          placeholder="-1001234567890"
          autocomplete="off"
        />
      </div>

      <div>
        <label class="field-label" for="tg-thread">
          Thread ID <span class="text-muted/60 normal-case">— optional</span>
        </label>
        <input
          id="tg-thread"
          v-model="form.threadId"
          class="field-underline"
          placeholder="topic thread id"
          autocomplete="off"
        />
      </div>

      <label class="text-muted flex cursor-pointer items-center gap-2 text-sm select-none">
        <input v-model="form.isDefault" type="checkbox" class="accent-accent" />
        Set as default
      </label>

      <p v-if="formError" class="text-danger text-sm">{{ formError }}</p>

      <div>
        <button type="submit" class="btn-accent" :disabled="!valid || submitting">
          <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
          {{ submitting ? 'Saving' : 'Add target' }}
        </button>
      </div>
    </form>

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
