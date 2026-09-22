/**
 * Profiles (writing-voice / humanization) feature — DTO and payload types,
 * verified against the real backend contract via `curl localhost:8082/profiles`.
 */

export type StyleGuideStatus = '' | 'pending' | 'ready' | 'error'

export interface Profile {
  id: string
  name: string
  language: string
  formality: string
  emojis: boolean
  samples: string[]
  styleGuide: string
  styleGuideStatus: StyleGuideStatus
  styleGuideError: string
  createdAt: string
  updatedAt: string
}

/** `POST /profiles` body. */
export interface CreateProfilePayload {
  name: string
  language: string
  formality: string
  emojis: boolean
  samples: string[]
}

/** `PATCH /profiles/{id}` body. */
export interface UpdateProfilePayload {
  name: string
  language: string
  formality: string
  emojis: boolean
  samples: string[]
}
