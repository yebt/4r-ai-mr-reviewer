import { describe, expect, it } from 'vitest'
import { ApiError } from '@shared/api/client'
import { errorMessage } from './errors'

describe('errorMessage', () => {
  it('maps 401 to an invalid-password message', () => {
    expect(errorMessage(new ApiError(401, 'Unauthorized'))).toBe('invalid password')
  })

  it('maps 429 to a rate-limit message', () => {
    expect(errorMessage(new ApiError(429, 'Too Many Requests'))).toBe(
      'too many attempts, wait a minute',
    )
  })
})
