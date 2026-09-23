import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { Select } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import RepoForm from './RepoForm.vue'
import type { Repo } from '../types'

// `vi.mock` factories are hoisted above regular top-level statements, so the
// data they close over must be created via `vi.hoisted` — mirrors the
// pattern used by ProviderForm.spec.ts.
const { createRepoMock, assignRepoMock, accounts, profiles, providers } = vi.hoisted(() => ({
  createRepoMock: vi.fn(),
  assignRepoMock: vi.fn(),
  accounts: [{ id: 'acc1', name: 'Acc 1', baseUrl: '', createdAt: '2024-01-01T00:00:00.000Z' }],
  profiles: [] as { id: string; name: string }[],
  providers: [
    {
      id: 'prov-openrouter',
      name: 'My OpenRouter',
      kind: 'openrouter',
      baseUrl: '',
      model: 'openai/gpt-4o',
      isDefault: false,
      temperature: null,
      models: ['openai/gpt-4o', 'anthropic/claude-3.5-sonnet'],
      createdAt: '2024-01-01T00:00:00.000Z',
    },
    {
      id: 'prov-plain',
      name: 'My OpenAI',
      kind: 'openai-compat',
      baseUrl: 'https://api.openai.com/v1',
      model: 'gpt-4o',
      isDefault: false,
      temperature: null,
      models: [] as string[],
      createdAt: '2024-01-01T00:00:00.000Z',
    },
  ],
}))

// Read-only reuse of accounts/providers/profiles, deep-imported the same way
// RepoForm.vue does — mocked here since this module never edits them.
vi.mock('@modules/accounts/store', () => ({
  useAccountsStore: () => ({ accounts }),
}))
vi.mock('@modules/profiles/store', () => ({
  useProfilesStore: () => ({ profiles }),
}))
vi.mock('@modules/providers/store', () => ({
  useProvidersStore: () => ({ providers }),
}))
vi.mock('../store', () => ({
  useReposStore: () => ({ createRepo: createRepoMock, assignRepo: assignRepoMock }),
}))

function makeRepo(overrides: Partial<Repo> = {}): Repo {
  return {
    id: 'r1',
    name: 'My repo',
    url: 'https://gitlab.com/group/project',
    accountId: 'acc1',
    providerId: '',
    model: '',
    defaultProfileId: '',
    webhookEnabled: false,
    webhookRequireConfirmation: false,
    webhookSecret: '',
    webhookPath: '',
    createdAt: '2024-01-01T00:00:00.000Z',
    ...overrides,
  }
}

/** Resolves the field's control element via its `<label for>` association. */
function fieldControl(wrapper: VueWrapper, labelText: string) {
  const label = wrapper.findAll('label').find((candidate) => candidate.text().trim().startsWith(labelText))
  if (!label) throw new Error(`Field "${labelText}" not found`)
  const forId = label.attributes('for')
  return wrapper.get(`#${forId}`)
}

/**
 * Resolves the design-system `Select` component instance backing a labeled
 * field, or `undefined` when that field isn't currently rendered as a Select
 * (e.g. the Model field falls back to a free-text Input). Several Selects can
 * be mounted at once (Account/Provider/Model/Profile), so this matches on
 * the field's `<label for>` id rather than assuming the first `Select` found.
 */
function selectByLabel(wrapper: VueWrapper, labelText: string) {
  const trigger = fieldControl(wrapper, labelText)
  return wrapper.findAllComponents(Select).find((component) => component.element.contains(trigger.element))
}

async function setSelect(wrapper: VueWrapper, labelText: string, value: string) {
  const component = selectByLabel(wrapper, labelText)
  if (!component) throw new Error(`Select for "${labelText}" not found`)
  component.vm.$emit('update:model-value', value)
  await wrapper.vm!.$nextTick()
}

describe('RepoForm', () => {
  beforeEach(() => {
    createRepoMock.mockReset().mockResolvedValue(makeRepo())
    assignRepoMock.mockReset().mockResolvedValue(makeRepo())
  })

  describe('Model field', () => {
    it('renders as free text when no provider is selected', () => {
      const wrapper = mount(RepoForm, { props: { repo: makeRepo({ providerId: '' }) } })

      expect(selectByLabel(wrapper, 'Model')).toBeUndefined()
      const control = fieldControl(wrapper, 'Model')
      expect(control.element.tagName).toBe('INPUT')
      expect(control.attributes('placeholder')).toBe('gpt-4o')
    })

    it('renders as free text when the selected provider has no models list', () => {
      const wrapper = mount(RepoForm, { props: { repo: makeRepo({ providerId: 'prov-plain' }) } })

      expect(selectByLabel(wrapper, 'Model')).toBeUndefined()
    })

    it('renders as a Select of the provider models when the selected provider declares a models list', () => {
      const wrapper = mount(RepoForm, {
        props: { repo: makeRepo({ providerId: 'prov-openrouter', model: 'openai/gpt-4o' }) },
      })

      const select = selectByLabel(wrapper, 'Model')
      expect(select).toBeDefined()
      const items = select!.props('items') as SelectItemOption[]
      expect(items.map((item) => item.value)).toEqual([
        '__use_default_model__',
        'openai/gpt-4o',
        'anthropic/claude-3.5-sonnet',
      ])
    })

    it('switches from free text to a Select when the provider selection changes to one with models', async () => {
      const wrapper = mount(RepoForm, { props: { repo: makeRepo({ providerId: '' }) } })

      expect(selectByLabel(wrapper, 'Model')).toBeUndefined()

      await setSelect(wrapper, 'Provider', 'prov-openrouter')

      expect(selectByLabel(wrapper, 'Model')).toBeDefined()
    })

    it('preserves an existing model value through the Select and maps "use default" back to an empty string', async () => {
      const wrapper = mount(RepoForm, {
        props: { repo: makeRepo({ providerId: 'prov-openrouter', model: 'anthropic/claude-3.5-sonnet' }) },
      })

      await setSelect(wrapper, 'Model', '__use_default_model__')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(assignRepoMock).toHaveBeenCalledWith('r1', expect.objectContaining({ model: '' }))
    })
  })
})
