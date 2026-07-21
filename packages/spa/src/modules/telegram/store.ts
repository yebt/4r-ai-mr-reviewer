import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type { ResolvedChat, TelegramTarget, TelegramTargetInput } from '@shared/api/types'

export const useTelegramStore = defineStore('telegram', () => {
  const items = ref<TelegramTarget[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      items.value = await api.listTelegram()
    } catch (e) {
      error.value = errorMessage(e)
    } finally {
      loading.value = false
    }
  }

  async function add(input: TelegramTargetInput) {
    const created = await api.createTelegram(input)
    // The backend may have flipped the previous default; refetch to stay honest.
    await fetchAll()
    return created
  }

  async function setDefault(id: string) {
    await api.setDefaultTelegram(id)
    items.value = items.value.map((t) => ({ ...t, isDefault: t.id === id }))
  }

  async function test(id: string) {
    return await api.testTelegram(id)
  }

  async function remove(id: string) {
    await api.deleteTelegram(id)
    items.value = items.value.filter((t) => t.id !== id)
  }

  // Ask the backend which chats/threads the bot has recently seen. Errors are
  // let propagate so the page can toast them.
  async function resolve(botToken: string): Promise<ResolvedChat[]> {
    const res = await api.resolveTelegram(botToken)
    return res.chats
  }

  return { items, loading, error, fetchAll, add, setDefault, test, remove, resolve }
})
