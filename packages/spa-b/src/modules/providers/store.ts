import { ref } from 'vue'
import { defineStore } from 'pinia'
import * as providersApi from './api'
import type {
  CreateProviderPayload,
  OpenRouterModel,
  Provider,
  TestProviderPayload,
  TestProviderResult,
  UpdateProviderPayload,
} from './types'

/**
 * Providers store. Every mutation reconciles local state from the
 * authoritative server response instead of refetching the whole list —
 * `create`/`update`/`setDefault` all return (or imply) enough to patch
 * `providers` in place.
 */
export const useProvidersStore = defineStore('providers', () => {
  const providers = ref<Provider[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const openRouterModels = ref<OpenRouterModel[]>([])
  const openRouterLoading = ref(false)
  const openRouterError = ref<string | null>(null)

  async function fetchProviders(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      providers.value = await providersApi.listProviders()
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load providers'
    } finally {
      loading.value = false
    }
  }

  async function createProvider(payload: CreateProviderPayload): Promise<Provider> {
    const created = await providersApi.createProvider(payload)
    if (created.isDefault) {
      for (const provider of providers.value) provider.isDefault = false
    }
    providers.value.push(created)
    return created
  }

  async function updateProvider(id: string, payload: UpdateProviderPayload): Promise<Provider> {
    const updated = await providersApi.updateProvider(id, payload)
    const index = providers.value.findIndex((provider) => provider.id === id)
    if (index === -1) providers.value.push(updated)
    else providers.value.splice(index, 1, updated)
    return updated
  }

  async function removeProvider(id: string): Promise<void> {
    await providersApi.deleteProvider(id)
    providers.value = providers.value.filter((provider) => provider.id !== id)
  }

  async function setDefaultProvider(id: string): Promise<void> {
    await providersApi.setDefaultProvider(id)
    for (const provider of providers.value) provider.isDefault = provider.id === id
  }

  /** Never throws for a reachable-but-rejecting provider — resolves `{ ok, error? }`. */
  function testConnection(payload: TestProviderPayload): Promise<TestProviderResult> {
    return providersApi.testProvider(payload)
  }

  async function fetchOpenRouterModels(): Promise<void> {
    openRouterLoading.value = true
    openRouterError.value = null
    try {
      openRouterModels.value = await providersApi.fetchOpenRouterModels()
    } catch (err) {
      openRouterError.value =
        err instanceof Error ? err.message : 'Failed to load OpenRouter models'
    } finally {
      openRouterLoading.value = false
    }
  }

  return {
    providers,
    loading,
    error,
    openRouterModels,
    openRouterLoading,
    openRouterError,
    fetchProviders,
    createProvider,
    updateProvider,
    removeProvider,
    setDefaultProvider,
    testConnection,
    fetchOpenRouterModels,
  }
})
