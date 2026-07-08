import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type { Repo } from '@shared/api/types'

export const useReposStore = defineStore('repos', () => {
  const items = ref<Repo[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      items.value = await api.listRepos()
    } catch (e) {
      error.value = errorMessage(e)
    } finally {
      loading.value = false
    }
  }

  async function add(input: {
    name: string
    url: string
    accountId: string
    providerId: string
    model: string
  }) {
    const created = await api.createRepo(input)
    items.value = [...items.value, created]
    return created
  }

  async function assign(id: string, input: { providerId: string; model: string }) {
    const updated = await api.assignRepo(id, input)
    items.value = items.value.map((r) => (r.id === id ? updated : r))
    return updated
  }

  async function remove(id: string) {
    await api.deleteRepo(id)
    items.value = items.value.filter((r) => r.id !== id)
  }

  return { items, loading, error, fetchAll, add, assign, remove }
})
