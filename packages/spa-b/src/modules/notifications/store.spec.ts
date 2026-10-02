import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import * as notificationsApi from './api'
import {
  isEventRouted,
  makeOptimisticRule,
  resolveErrorMessage,
  toEnabledPatch,
  unroutedEvents,
  useNotificationsStore,
  withoutRule,
} from './store'
import type { NotificationRule } from './types'

vi.mock('./api')

// `vi.mock` factories are hoisted above regular top-level statements, so the
// spies they close over must be created via `vi.hoisted` — a plain `const`
// declared below would still be in its temporal dead zone when the factory
// first runs.
const { mockToastSuccess, mockToastError } = vi.hoisted(() => ({
  mockToastSuccess: vi.fn(),
  mockToastError: vi.fn(),
}))

vi.mock('@shared/composables/useToast', () => ({
  useToast: () => ({
    success: mockToastSuccess,
    error: mockToastError,
    info: vi.fn(),
  }),
}))

const mockedListNotificationRules = vi.mocked(notificationsApi.listNotificationRules)
const mockedListNotificationEvents = vi.mocked(notificationsApi.listNotificationEvents)
const mockedCreateNotificationRule = vi.mocked(notificationsApi.createNotificationRule)
const mockedDeleteNotificationRule = vi.mocked(notificationsApi.deleteNotificationRule)
const mockedSetNotificationRuleEnabled = vi.mocked(notificationsApi.setNotificationRuleEnabled)

function makeRule(overrides: Partial<NotificationRule> = {}): NotificationRule {
  return {
    id: 'r1',
    event: 'review.finished',
    notifierKind: 'telegram',
    notifierId: 't1',
    repoId: '',
    enabled: true,
    createdAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

/** A promise plus its resolve/reject, so a test can assert the optimistic
 * cache state before the underlying request settles. */
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

/**
 * Mounts a throwaway component that just instantiates `useNotificationsStore()`
 * under a real Pinia + Pinia Colada app context — required because
 * `useQuery`/`useMutation`/`useQueryCache` need an active injection context,
 * exactly like any other Vue composable.
 */
function mountStore() {
  let store!: ReturnType<typeof useNotificationsStore>
  const Harness = defineComponent({
    setup() {
      store = useNotificationsStore()
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia(), PiniaColada] },
  })
  return { wrapper, store }
}

describe('pure helpers', () => {
  it('isEventRouted is true only when an ENABLED rule targets that event', () => {
    const rules = [
      makeRule({ event: 'review.finished', enabled: true }),
      makeRule({ id: 'r2', event: 'release.finished', enabled: false }),
    ]
    expect(isEventRouted('review.finished', rules)).toBe(true)
    expect(isEventRouted('release.finished', rules)).toBe(false)
    expect(isEventRouted('unknown.event', rules)).toBe(false)
  })

  it('unroutedEvents returns events with no enabled rule routing them', () => {
    const events = ['review.finished', 'release.finished', 'deploy.finished']
    const rules = [
      makeRule({ event: 'review.finished', enabled: true }),
      makeRule({ id: 'r2', event: 'release.finished', enabled: false }),
    ]
    expect(unroutedEvents(events, rules)).toEqual(['release.finished', 'deploy.finished'])
  })

  it('unroutedEvents is empty once every event has an enabled rule', () => {
    const events = ['review.finished']
    const rules = [makeRule({ event: 'review.finished', enabled: true })]
    expect(unroutedEvents(events, rules)).toEqual([])
  })

  it('withoutRule drops the matching row only', () => {
    const existing = [makeRule({ id: 'r1' }), makeRule({ id: 'r2' })]
    expect(withoutRule(existing, 'r1')).toEqual([existing[1]])
  })

  it('toEnabledPatch narrows to just the enabled field', () => {
    expect(toEnabledPatch({ enabled: false })).toEqual({ enabled: false })
  })

  it('makeOptimisticRule builds a temp-id row carrying the create payload, always enabled', () => {
    const optimistic = makeOptimisticRule({
      event: 'review.finished',
      notifierId: 't1',
      repoId: 'repo-1',
    })
    expect(optimistic.id).toMatch(/^temp-/)
    expect(optimistic.event).toBe('review.finished')
    expect(optimistic.notifierId).toBe('t1')
    expect(optimistic.notifierKind).toBe('telegram')
    expect(optimistic.repoId).toBe('repo-1')
    expect(optimistic.enabled).toBe(true)
  })

  it('resolveErrorMessage prefers the Error message and falls back otherwise', () => {
    expect(resolveErrorMessage(new Error('boom'), 'fallback')).toBe('boom')
    expect(resolveErrorMessage('not an error', 'fallback')).toBe('fallback')
  })
})

describe('useNotificationsStore (@pinia/colada)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedListNotificationEvents.mockResolvedValue({ events: [] })
  })

  it('the rules query maps the mocked listNotificationRules() result into store.rules', async () => {
    const rules = [makeRule()]
    mockedListNotificationRules.mockResolvedValueOnce(rules)

    const { store } = mountStore()
    await flushPromises()

    expect(mockedListNotificationRules).toHaveBeenCalledTimes(1)
    expect(store.rules).toEqual(rules)
    expect(store.isLoading).toBe(false)
    expect(store.rulesState.status).toBe('success')
  })

  it('the events query maps the mocked listNotificationEvents() { events } into store.events', async () => {
    mockedListNotificationRules.mockResolvedValueOnce([])
    mockedListNotificationEvents.mockResolvedValue({ events: ['review.finished', 'release.finished'] })

    const { store } = mountStore()
    await flushPromises()

    expect(store.events).toEqual(['review.finished', 'release.finished'])
  })

  it('createRule appends an optimistic temp row, then reconciles with the server id on success', async () => {
    mockedListNotificationRules.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    const created = makeRule({ id: 'server-1' })
    const { promise, resolve } = deferred<NotificationRule>()
    mockedCreateNotificationRule.mockReturnValueOnce(promise)
    mockedListNotificationRules.mockResolvedValueOnce([created])

    const call = store.createRule({
      event: created.event,
      notifierId: created.notifierId,
      repoId: created.repoId,
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.rules).toHaveLength(1)
    expect(store.rules[0]!.id).toMatch(/^temp-/)

    resolve(created)
    await call
    await flushPromises()

    expect(store.rules.some((rule) => rule.id === 'server-1')).toBe(true)
    expect(mockToastSuccess).toHaveBeenCalledWith('Notification rule added')
  })

  it('removeRule optimistically drops the row and rolls back + toasts on error', async () => {
    mockedListNotificationRules.mockResolvedValueOnce([makeRule({ id: 'r1' }), makeRule({ id: 'r2' })])
    const { store } = mountStore()
    await flushPromises()

    const { promise, reject } = deferred<void>()
    mockedDeleteNotificationRule.mockReturnValueOnce(promise)
    mockedListNotificationRules.mockResolvedValueOnce([makeRule({ id: 'r1' }), makeRule({ id: 'r2' })])

    const call = store.removeRule('r1').catch(() => undefined)
    await Promise.resolve()
    await Promise.resolve()

    expect(store.rules.map((rule) => rule.id)).toEqual(['r2'])

    reject(new Error('Delete failed'))
    await call
    await flushPromises()

    expect(store.rules.map((rule) => rule.id).sort()).toEqual(['r1', 'r2'])
    expect(mockToastError).toHaveBeenCalledWith('Delete failed')
  })

  it('setEnabled optimistically patches the enabled flag before the request resolves, then reconciles', async () => {
    mockedListNotificationRules.mockResolvedValueOnce([makeRule({ id: 'r1', enabled: true })])
    const { store } = mountStore()
    await flushPromises()

    const updated = makeRule({ id: 'r1', enabled: false })
    const { promise, resolve } = deferred<NotificationRule>()
    mockedSetNotificationRuleEnabled.mockReturnValueOnce(promise)
    mockedListNotificationRules.mockResolvedValueOnce([updated])

    const call = store.setEnabled('r1', false)
    await Promise.resolve()
    await Promise.resolve()

    // Optimistic: flipped locally before the request settles.
    expect(store.rules[0]!.enabled).toBe(false)
    expect(mockedSetNotificationRuleEnabled).toHaveBeenCalledWith('r1', false)

    resolve(updated)
    await call
    await flushPromises()

    expect(store.rules[0]!.enabled).toBe(false)
  })

  it('setEnabled rolls back on error', async () => {
    mockedListNotificationRules.mockResolvedValueOnce([makeRule({ id: 'r1', enabled: true })])
    const { store } = mountStore()
    await flushPromises()

    const { promise, reject } = deferred<NotificationRule>()
    mockedSetNotificationRuleEnabled.mockReturnValueOnce(promise)
    mockedListNotificationRules.mockResolvedValueOnce([makeRule({ id: 'r1', enabled: true })])

    const call = store.setEnabled('r1', false).catch(() => undefined)
    await Promise.resolve()
    await Promise.resolve()

    expect(store.rules[0]!.enabled).toBe(false)

    reject(new Error('Update failed'))
    await call
    await flushPromises()

    expect(store.rules[0]!.enabled).toBe(true)
  })
})
