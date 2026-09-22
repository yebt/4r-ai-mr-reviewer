import { computed } from 'vue'
import { defineStore } from 'pinia'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { useToast } from '@shared/composables/useToast'
import * as providersApi from './api'
import type {
  CreateProviderPayload,
  OpenRouterModel,
  Provider,
  TestProviderPayload,
  TestProviderResult,
  UpdateProviderPayload,
} from './types'

export const PROVIDERS_QUERY_KEY = ['providers'] as const
export const OPENROUTER_MODELS_QUERY_KEY = ['openrouter-models'] as const

/**
 * Pure cache-patch helpers used by the mutations' `onMutate` hooks below.
 * Exported so the optimistic-update logic can be unit-tested directly,
 * without spinning up a full Pinia Colada (Vue app + plugin) context.
 */
export function withOptimisticCreate(providers: Provider[], optimistic: Provider): Provider[] {
  const base = optimistic.isDefault
    ? providers.map((provider) => ({ ...provider, isDefault: false }))
    : providers
  return [...base, optimistic]
}

export function withoutProvider(providers: Provider[], id: string): Provider[] {
  return providers.filter((provider) => provider.id !== id)
}

export function withPatchedProvider(
  providers: Provider[],
  id: string,
  patch: Partial<Omit<Provider, 'id'>>,
): Provider[] {
  return providers.map((provider) => (provider.id === id ? { ...provider, ...patch } : provider))
}

export function withDefaultFlippedTo(providers: Provider[], id: string): Provider[] {
  return providers.map((provider) => ({ ...provider, isDefault: provider.id === id }))
}

/**
 * `apiKey` is write-only and never lives on `Provider` — never spread the
 * raw update payload onto the cache, or it leaks a stray `apiKey` field.
 */
export function toProviderPatch(payload: UpdateProviderPayload): Partial<Omit<Provider, 'id'>> {
  return {
    name: payload.name,
    kind: payload.kind,
    baseUrl: payload.baseUrl,
    model: payload.model,
    temperature: payload.temperature,
    models: payload.models,
  }
}

export function makeOptimisticProvider(payload: CreateProviderPayload): Provider {
  return {
    id: `temp-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    name: payload.name,
    kind: payload.kind,
    baseUrl: payload.baseUrl,
    model: payload.model,
    isDefault: payload.makeDefault,
    temperature: payload.temperature,
    models: payload.models,
    createdAt: new Date().toISOString(),
  }
}

export function resolveErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback
}

/**
 * Providers store, backed by @pinia/colada. The `['providers']` query is the
 * single source of truth for the list — it owns caching, request dedupe, and
 * async loading state. Every mutation (create/update/remove/setDefault)
 * patches that cache entry optimistically in `onMutate` (snapshotting the
 * previous value first), rolls back to the snapshot in `onError` (plus a
 * `toast.error`), and reconciles with the server in `onSettled` via
 * `invalidateQueries` so the authoritative response always wins.
 */
export const useProvidersStore = defineStore('providers', () => {
  const queryCache = useQueryCache()
  const toast = useToast()

  const providersQuery = useQuery({
    key: PROVIDERS_QUERY_KEY,
    query: providersApi.listProviders,
  })

  const providers = computed(() => providersQuery.data.value ?? [])

  const createMutation = useMutation({
    mutation: (payload: CreateProviderPayload) => providersApi.createProvider(payload),
    onMutate(payload) {
      queryCache.cancelQueries({ key: PROVIDERS_QUERY_KEY })
      const previous = queryCache.getQueryData<Provider[]>(PROVIDERS_QUERY_KEY)
      const optimistic = makeOptimisticProvider(payload)
      queryCache.setQueryData<Provider[]>(
        PROVIDERS_QUERY_KEY,
        withOptimisticCreate(previous ?? [], optimistic),
      )
      return { previous, tempId: optimistic.id }
    },
    onSuccess(created, _payload, { tempId }) {
      const current = queryCache.getQueryData<Provider[]>(PROVIDERS_QUERY_KEY) ?? []
      queryCache.setQueryData<Provider[]>(
        PROVIDERS_QUERY_KEY,
        current.map((provider) => (provider.id === tempId ? created : provider)),
      )
      toast.success('Provider added')
    },
    onError(err, _payload, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(PROVIDERS_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to add provider'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: PROVIDERS_QUERY_KEY })
    },
  })

  const updateMutation = useMutation({
    mutation: (vars: { id: string; payload: UpdateProviderPayload }) =>
      providersApi.updateProvider(vars.id, vars.payload),
    onMutate(vars) {
      queryCache.cancelQueries({ key: PROVIDERS_QUERY_KEY })
      const previous = queryCache.getQueryData<Provider[]>(PROVIDERS_QUERY_KEY)
      queryCache.setQueryData<Provider[]>(
        PROVIDERS_QUERY_KEY,
        withPatchedProvider(previous ?? [], vars.id, toProviderPatch(vars.payload)),
      )
      return { previous }
    },
    onSuccess(updated) {
      const current = queryCache.getQueryData<Provider[]>(PROVIDERS_QUERY_KEY) ?? []
      queryCache.setQueryData<Provider[]>(
        PROVIDERS_QUERY_KEY,
        withPatchedProvider(current, updated.id, updated),
      )
      toast.success('Saved')
    },
    onError(err, _vars, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(PROVIDERS_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to save provider'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: PROVIDERS_QUERY_KEY })
    },
  })

  const removeMutation = useMutation({
    mutation: (id: string) => providersApi.deleteProvider(id),
    onMutate(id) {
      queryCache.cancelQueries({ key: PROVIDERS_QUERY_KEY })
      const previous = queryCache.getQueryData<Provider[]>(PROVIDERS_QUERY_KEY)
      queryCache.setQueryData<Provider[]>(PROVIDERS_QUERY_KEY, withoutProvider(previous ?? [], id))
      return { previous }
    },
    onSuccess() {
      toast.success('Provider deleted')
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(PROVIDERS_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to delete provider'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: PROVIDERS_QUERY_KEY })
    },
  })

  const setDefaultMutation = useMutation({
    mutation: (id: string) => providersApi.setDefaultProvider(id),
    onMutate(id) {
      queryCache.cancelQueries({ key: PROVIDERS_QUERY_KEY })
      const previous = queryCache.getQueryData<Provider[]>(PROVIDERS_QUERY_KEY)
      queryCache.setQueryData<Provider[]>(
        PROVIDERS_QUERY_KEY,
        withDefaultFlippedTo(previous ?? [], id),
      )
      return { previous }
    },
    onSuccess() {
      toast.success('Set as default')
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(PROVIDERS_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to set default provider'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: PROVIDERS_QUERY_KEY })
    },
  })

  // Lazy — only fetched on demand (the OpenRouter model browser), never on
  // module load. `enabled: false` skips the automatic mount-time fetch;
  // `refetch()` still forces the request on demand regardless of `enabled`.
  const openRouterModelsQuery = useQuery({
    key: OPENROUTER_MODELS_QUERY_KEY,
    query: providersApi.fetchOpenRouterModels,
    enabled: false,
  })
  const openRouterModels = computed<OpenRouterModel[]>(() => openRouterModelsQuery.data.value ?? [])

  function fetchOpenRouterModels() {
    return openRouterModelsQuery.refetch()
  }

  /** Never throws for a reachable-but-rejecting provider — resolves `{ ok, error? }`. */
  function testConnection(payload: TestProviderPayload): Promise<TestProviderResult> {
    return providersApi.testProvider(payload)
  }

  return {
    // ['providers'] query surface
    providers,
    providersState: providersQuery.state,
    asyncStatus: providersQuery.asyncStatus,
    isLoading: providersQuery.isLoading,
    error: providersQuery.error,
    refetch: providersQuery.refetch,

    // ['openrouter-models'] lazy query surface
    openRouterModels,
    openRouterLoading: openRouterModelsQuery.isLoading,
    openRouterError: openRouterModelsQuery.error,
    fetchOpenRouterModels,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows (e.g. ProviderForm's inline `formError`) on top of the
    // store-level optimistic update + rollback + toast handled above.
    createProvider: createMutation.mutateAsync,
    isCreating: createMutation.isLoading,
    updateProvider: (id: string, payload: UpdateProviderPayload) =>
      updateMutation.mutateAsync({ id, payload }),
    isUpdating: updateMutation.isLoading,
    removeProvider: removeMutation.mutateAsync,
    isRemoving: removeMutation.isLoading,
    setDefaultProvider: setDefaultMutation.mutateAsync,
    isSettingDefault: setDefaultMutation.isLoading,
    testConnection,
  }
})
