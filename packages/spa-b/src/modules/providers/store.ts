import { computed } from 'vue'
import { defineStore } from 'pinia'
import { useMutation, useQuery } from '@pinia/colada'
import {
  createCrudResource,
  resolveErrorMessage,
  withoutItem,
  withPatchedItem,
} from '@shared/data/createCrudResource'
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
 *
 * `withOptimisticCreate` stays provider-specific (unlike accounts/telegram/
 * profiles, which just delegate to the generic helper): a new row created as
 * the default must unflip every other row's `isDefault` too.
 */
export function withOptimisticCreate(providers: Provider[], optimistic: Provider): Provider[] {
  const base = optimistic.isDefault
    ? providers.map((provider) => ({ ...provider, isDefault: false }))
    : providers
  return [...base, optimistic]
}

export function withoutProvider(providers: Provider[], id: string): Provider[] {
  return withoutItem(providers, id)
}

export function withPatchedProvider(
  providers: Provider[],
  id: string,
  patch: Partial<Omit<Provider, 'id'>>,
): Provider[] {
  return withPatchedItem(providers, id, patch)
}

export function withDefaultFlippedTo(providers: Provider[], id: string): Provider[] {
  return providers.map((provider) => ({ ...provider, isDefault: provider.id === id }))
}

/**
 * Default-first ordering for the list UI (`ProvidersSection`): `Array#sort`
 * is stable, so this only ever moves the default row to the top — every
 * other row keeps its relative order, which is what makes "Set default"
 * read as an obvious, single-item reorder rather than a full reshuffle.
 */
export function sortDefaultFirst(providers: Provider[]): Provider[] {
  return [...providers].sort((a, b) => Number(b.isDefault) - Number(a.isDefault))
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

export { resolveErrorMessage }

/**
 * Providers store, backed by @pinia/colada via `createCrudResource`. The
 * `['providers']` query is the single source of truth for the list — it owns
 * caching, request dedupe, and async loading state. The base mutations
 * (create/update/remove) patch that cache entry optimistically in `onMutate`
 * (snapshotting the previous value first), roll back to the snapshot in
 * `onError` (plus a `toast.error` for remove only), and reconcile with the
 * server in `onSettled` via `invalidateQueries` so the authoritative response
 * always wins.
 *
 * `setDefaultProvider` is a providers-specific extra layered on top of the
 * factory: it shares the resource's `queryCache`/`toast` but follows the same
 * optimistic-patch → rollback-on-error → invalidate shape by hand, since it
 * isn't one of the 3 base CRUD mutations.
 */
export const useProvidersStore = defineStore('providers', () => {
  const resource = createCrudResource<Provider, CreateProviderPayload, UpdateProviderPayload>({
    queryKey: PROVIDERS_QUERY_KEY,
    list: providersApi.listProviders,
    create: providersApi.createProvider,
    update: providersApi.updateProvider,
    remove: providersApi.deleteProvider,
    makeOptimistic: makeOptimisticProvider,
    toPatch: toProviderPatch,
    applyOptimisticCreate: withOptimisticCreate,
    messages: {
      createSuccess: 'Provider added',
      updateSuccess: 'Provider saved',
      removeSuccess: 'Provider deleted',
      removeErrorFallback: 'Failed to delete provider',
    },
  })

  const setDefaultMutation = useMutation({
    mutation: (id: string) => providersApi.setDefaultProvider(id),
    onMutate(id) {
      resource.queryCache.cancelQueries({ key: PROVIDERS_QUERY_KEY })
      const previous = resource.queryCache.getQueryData<Provider[]>(PROVIDERS_QUERY_KEY)
      resource.queryCache.setQueryData<Provider[]>(
        PROVIDERS_QUERY_KEY,
        withDefaultFlippedTo(previous ?? [], id),
      )
      return { previous }
    },
    onSuccess() {
      resource.toast.success('Set as default')
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        resource.queryCache.setQueryData(PROVIDERS_QUERY_KEY, context.previous)
      }
      resource.toast.error(resolveErrorMessage(err, 'Failed to set default provider'))
    },
    onSettled() {
      resource.queryCache.invalidateQueries({ key: PROVIDERS_QUERY_KEY })
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
    providers: resource.items,
    providersState: resource.state,
    asyncStatus: resource.asyncStatus,
    isLoading: resource.isLoading,
    error: resource.error,
    refetch: resource.refetch,

    // ['openrouter-models'] lazy query surface
    openRouterModels,
    openRouterLoading: openRouterModelsQuery.isLoading,
    openRouterError: openRouterModelsQuery.error,
    fetchOpenRouterModels,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows (e.g. ProviderForm's inline `formError`) on top of the
    // store-level optimistic update + rollback + toast handled above.
    createProvider: resource.create,
    isCreating: resource.isCreating,
    updateProvider: resource.update,
    isUpdating: resource.isUpdating,
    removeProvider: resource.remove,
    isRemoving: resource.isRemoving,
    setDefaultProvider: setDefaultMutation.mutateAsync,
    isSettingDefault: setDefaultMutation.isLoading,
    testConnection,
  }
})
