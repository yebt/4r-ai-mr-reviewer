import { request } from '@shared/api/client'
import type {
  CreateProviderPayload,
  OpenRouterModel,
  Provider,
  TestProviderPayload,
  TestProviderResult,
  UpdateProviderPayload,
} from './types'

export function listProviders(): Promise<Provider[]> {
  return request<Provider[]>('GET', '/providers')
}

export function createProvider(payload: CreateProviderPayload): Promise<Provider> {
  return request<Provider>('POST', '/providers', payload)
}

export function updateProvider(id: string, payload: UpdateProviderPayload): Promise<Provider> {
  return request<Provider>('PATCH', `/providers/${id}`, payload)
}

export function setDefaultProvider(id: string): Promise<void> {
  return request<void>('POST', `/providers/${id}/default`)
}

/** Never throws for a reachable-but-rejecting provider — the backend always resolves `{ ok, error? }`. */
export function testProvider(payload: TestProviderPayload): Promise<TestProviderResult> {
  return request<TestProviderResult>('POST', '/providers/test', payload)
}

export function deleteProvider(id: string): Promise<void> {
  return request<void>('DELETE', `/providers/${id}`)
}

export function fetchOpenRouterModels(): Promise<OpenRouterModel[]> {
  return request<OpenRouterModel[]>('GET', '/openrouter/models')
}
