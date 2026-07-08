import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type { Provider, ProviderKind } from '@shared/api/types'

export interface AddProviderInput {
  name: string
  kind: ProviderKind
  baseUrl: string
  model: string
  apiKey: string
  makeDefault: boolean
}

export const useProvidersStore = defineStore('providers', () => {
  const items = ref<Provider[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      items.value = await api.listProviders()
    } catch (e) {
      error.value = errorMessage(e)
    } finally {
      loading.value = false
    }
  }

  async function add(input: AddProviderInput) {
    const created = await api.createProvider(input)
    // The backend may have flipped the previous default; refetch to stay honest.
    await fetchAll()
    return created
  }

  async function setDefault(id: string) {
    await api.setDefaultProvider(id)
    items.value = items.value.map((p) => ({ ...p, isDefault: p.id === id }))
  }

  async function remove(id: string) {
    await api.deleteProvider(id)
    items.value = items.value.filter((p) => p.id !== id)
  }

  return { items, loading, error, fetchAll, add, setDefault, remove }
})
