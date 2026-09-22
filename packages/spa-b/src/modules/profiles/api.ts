import { request } from '@shared/api/client'
import type { CreateProfilePayload, Profile, UpdateProfilePayload } from './types'

export function listProfiles(): Promise<Profile[]> {
  return request<Profile[]>('GET', '/profiles')
}

export function createProfile(payload: CreateProfilePayload): Promise<Profile> {
  return request<Profile>('POST', '/profiles', payload)
}

export function updateProfile(id: string, payload: UpdateProfilePayload): Promise<Profile> {
  return request<Profile>('PATCH', `/profiles/${id}`, payload)
}

export function deleteProfile(id: string): Promise<void> {
  return request<void>('DELETE', `/profiles/${id}`)
}

/** Re-runs the style-guide distillation for an existing profile. */
export function redistillProfile(id: string): Promise<Profile> {
  return request<Profile>('POST', `/profiles/${id}/redistill`)
}
