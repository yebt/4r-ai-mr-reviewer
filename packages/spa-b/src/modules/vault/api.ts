import { request } from '@shared/api/client'
import type { ChangeVaultPasswordPayload, ChangeVaultPasswordResult, VaultStatus } from './types'

export function fetchVaultStatus(): Promise<VaultStatus> {
  return request<VaultStatus>('GET', '/vault/status')
}

/** 401 wrong current password, 409 not initialized, 501 unavailable. */
export function changeVaultPassword(payload: ChangeVaultPasswordPayload): Promise<ChangeVaultPasswordResult> {
  return request<ChangeVaultPasswordResult>('POST', '/vault/password', payload)
}
