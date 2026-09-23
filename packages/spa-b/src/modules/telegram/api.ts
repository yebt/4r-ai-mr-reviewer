import { request } from '@shared/api/client'
import type {
  CreateTelegramTargetPayload,
  TelegramTarget,
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

/**
 * Sends a test message to the target. Resolves on success (the backend returns
 * 200 {status:'sent'}); throws ApiError on failure (404 unknown target, 502
 * delivery failure).
 */
export function testTelegramTarget(id: string): Promise<void> {
  return request<void>('POST', `/telegram/${id}/test`)
}

export function deleteTelegramTarget(id: string): Promise<void> {
  return request<void>('DELETE', `/telegram/${id}`)
}
