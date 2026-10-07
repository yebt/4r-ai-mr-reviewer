import { describe, expect, it } from 'vitest'
import { ApiError } from './client'
import { isNotFound } from './isNotFound'

describe('isNotFound', () => {
  it('is true only for a 404 ApiError', () => {
    expect(isNotFound(new ApiError(404, 'review not found'))).toBe(true)
    expect(isNotFound(new ApiError(500, 'boom'))).toBe(false)
    expect(isNotFound(new Error('404'))).toBe(false)
    expect(isNotFound(null)).toBe(false)
  })
})
