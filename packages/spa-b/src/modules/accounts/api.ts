import { request } from '@shared/api/client'
import type { Account, CreateAccountPayload, UpdateAccountPayload } from './types'

export function listAccounts(): Promise<Account[]> {
  return request<Account[]>('GET', '/accounts')
}

export function createAccount(payload: CreateAccountPayload): Promise<Account> {
  return request<Account>('POST', '/accounts', payload)
}

export function updateAccount(id: string, payload: UpdateAccountPayload): Promise<Account> {
  return request<Account>('PATCH', `/accounts/${id}`, payload)
}

export function deleteAccount(id: string): Promise<void> {
  return request<void>('DELETE', `/accounts/${id}`)
}
