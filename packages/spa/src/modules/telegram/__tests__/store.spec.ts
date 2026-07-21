import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listTelegram: vi.fn(),
    createTelegram: vi.fn(),
    setDefaultTelegram: vi.fn(),
    testTelegram: vi.fn(),
    deleteTelegram: vi.fn(),
    resolveTelegram: vi.fn(),
  },
}))

import { api } from '@shared/api/client'
import { useTelegramStore } from '@modules/telegram/store'

const mocked = api as unknown as {
  listTelegram: ReturnType<typeof vi.fn>
  createTelegram: ReturnType<typeof vi.fn>
  setDefaultTelegram: ReturnType<typeof vi.fn>
  testTelegram: ReturnType<typeof vi.fn>
  deleteTelegram: ReturnType<typeof vi.fn>
  resolveTelegram: ReturnType<typeof vi.fn>
}

const target = (id: string, isDefault = false) => ({
  id,
  name: `t-${id}`,
  chatId: `chat-${id}`,
  threadId: '',
  isDefault,
  createdAt: '',
})

describe('telegram store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchAll loads items from the api', async () => {
    mocked.listTelegram.mockResolvedValue([target('1'), target('2')])
    const store = useTelegramStore()
    await store.fetchAll()
    expect(mocked.listTelegram).toHaveBeenCalledOnce()
    expect(store.items.map((t) => t.id)).toEqual(['1', '2'])
  })

  it('add creates then refetches to reflect default changes', async () => {
    mocked.createTelegram.mockResolvedValue(target('2', true))
    mocked.listTelegram.mockResolvedValue([target('1', false), target('2', true)])
    const store = useTelegramStore()
    await store.add({ name: 't-2', botToken: 'k', chatId: 'chat-2', isDefault: true })
    expect(mocked.createTelegram).toHaveBeenCalledOnce()
    expect(mocked.listTelegram).toHaveBeenCalledOnce()
    expect(store.items).toHaveLength(2)
  })

  it('setDefault flips isDefault to a single target', async () => {
    mocked.setDefaultTelegram.mockResolvedValue({ status: 'ok' })
    const store = useTelegramStore()
    store.items = [target('1', true), target('2', false)]
    await store.setDefault('2')
    expect(mocked.setDefaultTelegram).toHaveBeenCalledWith('2')
    expect(store.items.find((t) => t.id === '2')?.isDefault).toBe(true)
    expect(store.items.find((t) => t.id === '1')?.isDefault).toBe(false)
  })

  it('test calls the test endpoint and returns its status', async () => {
    mocked.testTelegram.mockResolvedValue({ status: 'sent' })
    const store = useTelegramStore()
    const res = await store.test('1')
    expect(mocked.testTelegram).toHaveBeenCalledWith('1')
    expect(res).toEqual({ status: 'sent' })
  })

  it('resolve returns the chats from the api', async () => {
    const chats = [{ chatId: '-100', title: 'Team', type: 'supergroup', threads: [] }]
    mocked.resolveTelegram.mockResolvedValue({ chats })
    const store = useTelegramStore()
    const res = await store.resolve('bot-token')
    expect(mocked.resolveTelegram).toHaveBeenCalledWith('bot-token')
    expect(res).toEqual(chats)
  })

  it('resolve propagates api errors', async () => {
    mocked.resolveTelegram.mockRejectedValue(new Error('bad token'))
    const store = useTelegramStore()
    await expect(store.resolve('bad')).rejects.toThrow('bad token')
  })

  it('remove drops the target', async () => {
    mocked.deleteTelegram.mockResolvedValue(undefined)
    const store = useTelegramStore()
    store.items = [target('1'), target('2')]
    await store.remove('1')
    expect(mocked.deleteTelegram).toHaveBeenCalledWith('1')
    expect(store.items.map((t) => t.id)).toEqual(['2'])
  })
})
