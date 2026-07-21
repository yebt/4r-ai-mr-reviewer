import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listNotificationEvents: vi.fn(),
    listNotificationRules: vi.fn(),
    createNotificationRule: vi.fn(),
    setNotificationRuleEnabled: vi.fn(),
    deleteNotificationRule: vi.fn(),
  },
}))

import { api } from '@shared/api/client'
import { useNotificationsStore } from '@modules/notifications/store'

const mocked = api as unknown as {
  listNotificationEvents: ReturnType<typeof vi.fn>
  listNotificationRules: ReturnType<typeof vi.fn>
  createNotificationRule: ReturnType<typeof vi.fn>
  setNotificationRuleEnabled: ReturnType<typeof vi.fn>
  deleteNotificationRule: ReturnType<typeof vi.fn>
}

const rule = (id: string, enabled = true) => ({
  id,
  event: 'review.finished',
  notifierKind: 'telegram',
  notifierId: `tg-${id}`,
  enabled,
  createdAt: '',
})

describe('notifications store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchAll loads events and rules from the api', async () => {
    mocked.listNotificationEvents.mockResolvedValue({ events: ['review.finished'] })
    mocked.listNotificationRules.mockResolvedValue([rule('1'), rule('2')])
    const store = useNotificationsStore()
    await store.fetchAll()
    expect(mocked.listNotificationEvents).toHaveBeenCalledOnce()
    expect(mocked.listNotificationRules).toHaveBeenCalledOnce()
    expect(store.events).toEqual(['review.finished'])
    expect(store.rules.map((r) => r.id)).toEqual(['1', '2'])
    expect(store.error).toBeNull()
  })

  it('fetchAll sets error on failure', async () => {
    mocked.listNotificationEvents.mockRejectedValue(new Error('boom'))
    mocked.listNotificationRules.mockResolvedValue([])
    const store = useNotificationsStore()
    await store.fetchAll()
    expect(store.error).toBe('boom')
    expect(store.loading).toBe(false)
  })

  it('add creates a rule and appends it', async () => {
    const created = rule('3')
    mocked.createNotificationRule.mockResolvedValue(created)
    const store = useNotificationsStore()
    store.rules = [rule('1')]
    await store.add({ event: 'review.finished', notifierId: 'tg-3' })
    expect(mocked.createNotificationRule).toHaveBeenCalledWith({
      event: 'review.finished',
      notifierId: 'tg-3',
    })
    expect(store.rules.map((r) => r.id)).toEqual(['1', '3'])
  })

  it('setEnabled patches and replaces the rule from the response', async () => {
    mocked.setNotificationRuleEnabled.mockResolvedValue(rule('1', false))
    const store = useNotificationsStore()
    store.rules = [rule('1', true), rule('2', true)]
    await store.setEnabled('1', false)
    expect(mocked.setNotificationRuleEnabled).toHaveBeenCalledWith('1', false)
    expect(store.rules.find((r) => r.id === '1')?.enabled).toBe(false)
    expect(store.rules.find((r) => r.id === '2')?.enabled).toBe(true)
  })

  it('remove deletes the rule', async () => {
    mocked.deleteNotificationRule.mockResolvedValue(undefined)
    const store = useNotificationsStore()
    store.rules = [rule('1'), rule('2')]
    await store.remove('1')
    expect(mocked.deleteNotificationRule).toHaveBeenCalledWith('1')
    expect(store.rules.map((r) => r.id)).toEqual(['2'])
  })
})
