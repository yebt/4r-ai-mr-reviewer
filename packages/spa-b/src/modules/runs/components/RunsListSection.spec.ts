import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import RunsListSection from './RunsListSection.vue'
import * as runsApi from '../api'
import * as reposApi from '@modules/repos/api'
import type { RoutineRun } from '../types'
import type { Repo } from '@modules/repos/types'

vi.mock('../api')
vi.mock('@modules/repos/api')

vi.mock('@shared/composables/useToast', () => ({
  useToast: () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn() }),
}))

const mockedListRecentRoutines = vi.mocked(runsApi.listRecentRoutines)
const mockedListRepos = vi.mocked(reposApi.listRepos)

function makeRun(overrides: Partial<RoutineRun> = {}): RoutineRun {
  return {
    id: 'run1',
    kind: 'release',
    repoId: 'repo1',
    mrIid: 42,
    status: 'running',
    steps: [],
    state: { mrTitle: 'Add API docs' },
    lastError: '',
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    archived: false,
    repoName: 'my-repo',
    ...overrides,
  }
}

function makeRepo(overrides: Partial<Repo> = {}): Repo {
  return {
    id: 'repo1',
    name: 'my-repo',
    url: 'https://example.com/repo.git',
    accountId: 'acc1',
    providerId: '',
    model: '',
    defaultProfileId: '',
    webhookEnabled: false,
    webhookRequireConfirmation: false,
    webhookSecret: '',
    webhookPath: '',
    createdAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

async function mountSection() {
  const wrapper = mount(RunsListSection, {
    global: {
      plugins: [createPinia(), PiniaColada],
      stubs: {
        RouterLink: { template: '<a :href="to"><slot /></a>', props: ['to'] },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('RunsListSection', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockedListRepos.mockResolvedValue([makeRepo()])
  })

  it('renders the MR title as the bold primary line and the repo name as the muted secondary line', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce([makeRun()])
    const wrapper = await mountSection()

    const row = wrapper.find('[data-testid="run-row"]')
    expect(row.find('.font-semibold').text()).toBe('Add API docs')
    // Primary (title) renders before secondary (repo name) in DOM order.
    expect(row.text().indexOf('Add API docs')).toBeLessThan(row.text().indexOf('my-repo'))
  })

  it('wraps the row body in a link to /runs/{id}', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce([makeRun({ id: 'run42' })])
    const wrapper = await mountSection()

    const link = wrapper.find('[data-testid="run-row"] a')
    expect(link.attributes('href')).toBe('/runs/run42')
  })

  it('renders an "Archived" badge and dims the row for an archived run', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce([makeRun({ archived: true })])
    const wrapper = await mountSection()

    const row = wrapper.find('[data-testid="run-row"]')
    expect(row.classes()).toContain('opacity-60')
    const badge = wrapper.find('[data-testid="run-archived-badge"]')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('Archived')
  })

  it('does not show the archived badge or dimming for a non-archived run', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce([makeRun({ archived: false })])
    const wrapper = await mountSection()

    const row = wrapper.find('[data-testid="run-row"]')
    expect(row.classes()).not.toContain('opacity-60')
    expect(wrapper.find('[data-testid="run-archived-badge"]').exists()).toBe(false)
  })
})
