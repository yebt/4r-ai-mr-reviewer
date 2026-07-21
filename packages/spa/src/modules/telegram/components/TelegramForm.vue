<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { ResolvedChat, ResolvedThread } from '@shared/api/types'
import { useTelegramStore } from '@modules/telegram/store'

const emit = defineEmits<{ done: [] }>()

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
const error = ref<string | null>(null)

// Resolve feature: discovered chats/threads the bot has recently seen. null means
// the picker is closed; an (empty) array means we resolved and have a result set.
const resolved = ref<ResolvedChat[] | null>(null)
const resolving = ref(false)

async function resolveChats() {
  const token = form.botToken.trim()
  if (!token || resolving.value) return
  resolving.value = true
  resolved.value = []
  try {
    resolved.value = await store.resolve(token)
  } catch (e) {
    resolved.value = null
    toast.error(errorMessage(e))
  } finally {
    resolving.value = false
  }
}

function useChannel(chat: ResolvedChat) {
  form.chatId = chat.chatId
  form.threadId = ''
  resolved.value = null
  toast.success(`Selected ${chat.title}`)
}

function useThread(chat: ResolvedChat, thread: ResolvedThread) {
  form.chatId = chat.chatId
  form.threadId = thread.threadId
  resolved.value = null
  toast.success(`Selected ${chat.title} · ${thread.name || `thread ${thread.threadId}`}`)
}

const valid = computed(() => form.name.trim() && form.botToken.trim() && form.chatId.trim())

async function submit() {
  if (!valid.value || submitting.value) return
  submitting.value = true
  error.value = null
  try {
    await store.add({
      name: form.name.trim(),
      botToken: form.botToken.trim(),
      chatId: form.chatId.trim(),
      threadId: form.threadId.trim(),
      isDefault: form.isDefault,
    })
    toast.success('Telegram target added')
    emit('done')
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <form class="flex flex-col gap-5" @submit.prevent="submit">
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
      <div class="flex flex-wrap items-end gap-2 sm:flex-nowrap">
        <div class="min-w-0 flex-1">
          <input
            id="tg-token"
            v-model="form.botToken"
            type="password"
            class="field-underline"
            placeholder="123456:ABC-…"
            autocomplete="off"
          />
        </div>
        <button
          type="button"
          class="btn-line shrink-0 text-xs"
          :disabled="resolving || !form.botToken.trim()"
          @click="resolveChats"
        >
          <span v-if="resolving" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
          Resolve
        </button>
      </div>

      <div v-if="resolved !== null" class="border-line/50 mt-3 border-t pt-3">
        <p v-if="resolving" class="text-muted text-sm">Resolving…</p>
        <p v-else-if="resolved.length === 0" class="text-muted text-sm">
          No recent chats. Send a message in the group/topic so the bot can see it, then resolve
          again.
        </p>
        <ul v-else class="flex flex-col gap-3">
          <li v-for="chat in resolved" :key="chat.chatId">
            <div class="flex items-center justify-between gap-2">
              <div class="min-w-0">
                <div class="text-ink truncate text-sm">{{ chat.title }}</div>
                <div class="text-muted truncate font-mono text-xs">
                  {{ chat.chatId }} · {{ chat.type }}
                </div>
              </div>
              <button type="button" class="btn-ghost shrink-0 text-xs" @click="useChannel(chat)">
                Use channel
              </button>
            </div>
            <ul v-if="chat.threads.length" class="mt-1 flex flex-col gap-1 pl-4">
              <li
                v-for="thread in chat.threads"
                :key="thread.threadId"
                class="flex items-center justify-between gap-2"
              >
                <span class="text-muted truncate text-xs">
                  {{ thread.name || `thread ${thread.threadId}` }}
                </span>
                <button
                  type="button"
                  class="btn-ghost shrink-0 text-xs"
                  @click="useThread(chat, thread)"
                >
                  Use thread
                </button>
              </li>
            </ul>
          </li>
        </ul>
      </div>
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

    <p v-if="error" class="text-danger text-sm">{{ error }}</p>

    <div>
      <button type="submit" class="btn-accent" :disabled="!valid || submitting">
        <span v-if="submitting" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
        {{ submitting ? 'Saving' : 'Add target' }}
      </button>
    </div>
  </form>
</template>
