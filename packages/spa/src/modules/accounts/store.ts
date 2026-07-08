import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type { Account } from '@shared/api/types'

export const useAccountsStore = defineStore('accounts', () => {
  const items = ref<Account[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      items.value = await api.listAccounts()
    } catch (e) {
      error.value = errorMessage(e)
    } finally {
      loading.value = false
    }
  }

  async function add(input: { name: string; baseUrl: string; token: string }) {
    const created = await api.createAccount(input)
    items.value = [...items.value, created]
    return created
  }

  async function remove(id: string) {
    await api.deleteAccount(id)
    items.value = items.value.filter((a) => a.id !== id)
  }

  return { items, loading, error, fetchAll, add, remove }
})
