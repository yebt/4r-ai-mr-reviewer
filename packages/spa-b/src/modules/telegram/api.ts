import { request } from '@shared/api/client'
import type {
  CreateTelegramTargetPayload,
  TelegramTarget,
  TestTelegramTargetResult,
  UpdateTelegramTargetPayload,
} from './types'

export function listTelegramTargets(): Promise<TelegramTarget[]> {
  return request<TelegramTarget[]>('GET', '/telegram')
}

export function createTelegramTarget(payload: CreateTelegramTargetPayload): Promise<TelegramTarget> {
  return request<TelegramTarget>('POST', '/telegram', payload)
}

export function updateTelegramTarget(
  id: string,
  payload: UpdateTelegramTargetPayload,
): Promise<TelegramTarget> {
  return request<TelegramTarget>('PUT', `/telegram/${id}`, payload)
}

export function setDefaultTelegramTarget(id: string): Promise<void> {
  return request<void>('POST', `/telegram/${id}/default`)
}

/** Never throws for a reachable-but-rejecting target — the backend always resolves `{ ok, error? }`. */
export function testTelegramTarget(id: string): Promise<TestTelegramTargetResult> {
  return request<TestTelegramTargetResult>('POST', `/telegram/${id}/test`)
}

export function deleteTelegramTarget(id: string): Promise<void> {
  return request<void>('DELETE', `/telegram/${id}`)
}
