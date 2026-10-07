import { describe, expect, it } from 'vitest'
import { ApiError } from './client'
import { resolveErrorMessage } from './resolveErrorMessage'

const FALLBACK = 'Failed to save provider'

describe('resolveErrorMessage', () => {
  it('uses the fallback for non-Error values', () => {
    expect(resolveErrorMessage('boom', FALLBACK)).toBe(FALLBACK)
    expect(resolveErrorMessage(undefined, FALLBACK)).toBe(FALLBACK)
  })

  it('keeps a clear 4xx domain message', () => {
    const err = new ApiError(409, 'A provider named "Anthropic" already exists')
    expect(resolveErrorMessage(err, FALLBACK)).toBe('A provider named "Anthropic" already exists')
    expect(resolveErrorMessage(new ApiError(422, 'Webhook secret is required'), FALLBACK)).toBe(
      'Webhook secret is required',
    )
  })

  it('replaces any 5xx message with a user-facing sentence built on the fallback', () => {
    const msg = resolveErrorMessage(new ApiError(500, 'boom'), FALLBACK)
    expect(msg).toMatch(/^Failed to save provider\./)
    expect(msg).toMatch(/try again/i)
    expect(msg).not.toContain('boom')
    expect(resolveErrorMessage(new ApiError(502, 'Bad Gateway'), FALLBACK)).not.toContain('Bad Gateway')
  })

  it.each([
    'runtime error: invalid memory address or nil pointer dereference',
    'panic: something at internal/app/reviews/service.go:142',
    'Error at handler (src/foo.ts:12:5)',
    'upstream connect error or disconnect/reset before headers',
    'pq: SQLSTATE 23505 duplicate key',
    'Unexpected non-JSON response from the API (check the dev proxy / VITE_API_TARGET)',
  ])('does not leak internal-looking text even on a 4xx: %s', (internal) => {
    const msg = resolveErrorMessage(new ApiError(400, internal), FALLBACK)
    expect(msg).not.toBe(internal)
    expect(msg).toMatch(/^Failed to save provider\./)
  })

  it('maps a network failure to a connectivity message', () => {
    const msg = resolveErrorMessage(new TypeError('Failed to fetch'), FALLBACK)
    expect(msg).toMatch(/reach the server/i)
  })

  it('keeps plain Error messages that look user-facing', () => {
    expect(resolveErrorMessage(new Error('Pick a provider first'), FALLBACK)).toBe('Pick a provider first')
  })

  it('does not double the full stop when the fallback already ends with one', () => {
    expect(resolveErrorMessage(new ApiError(500, 'x'), 'Failed to load.')).not.toContain('..')
  })
})
