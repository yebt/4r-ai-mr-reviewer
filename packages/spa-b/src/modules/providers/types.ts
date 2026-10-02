/**
 * Providers feature — DTO and payload types, verified against the real
 * backend contract (see module README/task notes). `apiKey` is write-only:
 * the server never returns it on `Provider`.
 */

export type ProviderKind = 'openai-compat' | 'anthropic' | 'gemini' | 'openrouter'

export interface Provider {
  id: string
  name: string
  kind: ProviderKind
  baseUrl: string
  model: string
  isDefault: boolean
  temperature: number | null
  models: string[]
  createdAt: string
}

export interface OpenRouterModel {
  id: string
  name: string
  contextLength: number
}

/** `POST /providers` body. */
export interface CreateProviderPayload {
  name: string
  kind: ProviderKind
  baseUrl: string
  model: string
  apiKey: string
  makeDefault: boolean
  temperature: number | null
  models: string[]
}

/** `PATCH /providers/{id}` body — no `makeDefault`; blank `apiKey` keeps the stored key. */
export interface UpdateProviderPayload {
  name: string
  kind: ProviderKind
  baseUrl: string
  model: string
  apiKey: string
  temperature: number | null
  models: string[]
}

/** `POST /providers/test` body. `id` targets an existing saved provider. */
export interface TestProviderPayload {
  id?: string
  kind: ProviderKind
  baseUrl: string
  model: string
  apiKey: string
}

/** Never throws for a reachable-but-rejecting provider — always `{ ok, error? }`. */
export interface TestProviderResult {
  ok: boolean
  error?: string
}
