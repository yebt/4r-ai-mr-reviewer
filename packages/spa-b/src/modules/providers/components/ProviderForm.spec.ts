import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { Select } from '@shared/ui/design-system'
import ProviderForm from './ProviderForm.vue'
import type { Provider } from '../types'

// `vi.mock` factories are hoisted above regular top-level statements, so the
// spies they close over must be created via `vi.hoisted` — mirrors the
// pattern used by store.spec.ts / LoginPage.spec.ts for mocked stores.
const {
  createProviderMock,
  updateProviderMock,
  testConnectionMock,
  fetchOpenRouterModelsMock,
  openRouterModels,
} = vi.hoisted(() => ({
  createProviderMock: vi.fn(),
  updateProviderMock: vi.fn(),
  testConnectionMock: vi.fn(),
  fetchOpenRouterModelsMock: vi.fn(),
  openRouterModels: [
    { id: 'openai/gpt-4o', name: 'GPT-4o', contextLength: 128000 },
    { id: 'anthropic/claude-3.5-sonnet', name: 'Claude 3.5 Sonnet', contextLength: 200000 },
    { id: 'google/gemini-1.5-pro', name: 'Gemini 1.5 Pro', contextLength: 1000000 },
  ],
}))

vi.mock('../store', () => ({
  useProvidersStore: () => ({
    openRouterModels,
    openRouterLoading: false,
    openRouterError: null,
    fetchOpenRouterModels: fetchOpenRouterModelsMock,
    createProvider: createProviderMock,
    updateProvider: updateProviderMock,
    testConnection: testConnectionMock,
  }),
}))

function makeProvider(overrides: Partial<Provider> = {}): Provider {
  return {
    id: 'p1',
    name: 'My provider',
    kind: 'openai-compat',
    baseUrl: 'https://api.openai.com/v1',
    model: 'gpt-4o',
    isDefault: false,
    temperature: null,
    models: [],
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

async function setSelectKind(wrapper: VueWrapper, kind: string) {
  // The Kind field renders the design-system Select (Reka SelectRoot), which
  // exposes `kind` via a plain `v-model` binding on the component instance
  // rather than a native <select> — drive it directly through the emitted
  // `update:model-value` event to avoid coupling the test to Reka's portal.
  const selectComponent = wrapper.findComponent(Select)
  selectComponent.vm.$emit('update:model-value', kind)
  await wrapper.vm!.$nextTick()
}

describe('ProviderForm', () => {
  beforeEach(() => {
    createProviderMock.mockReset().mockResolvedValue(makeProvider())
    updateProviderMock.mockReset().mockResolvedValue(makeProvider())
    testConnectionMock.mockReset()
    fetchOpenRouterModelsMock.mockReset()
  })

  describe('Base URL validation', () => {
    it('blocks submit with a required-field error when openai-compat has a blank Base URL', async () => {
      const wrapper = mount(ProviderForm)

      await fieldControl(wrapper, 'Name').setValue('My provider')
      await fieldControl(wrapper, 'Model').setValue('gpt-4o')
      // Base URL left blank; kind defaults to openai-compat.

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.text()).toContain('Base URL is required')
      expect(createProviderMock).not.toHaveBeenCalled()
    })

    it('submits fine with a blank Base URL for openrouter (backend uses its default endpoint)', async () => {
      const wrapper = mount(ProviderForm)

      await setSelectKind(wrapper, 'openrouter')
      await fieldControl(wrapper, 'Name').setValue('My OpenRouter provider')
      // Base URL intentionally left blank.

      // Add a model so the Model select has something to pick, then select it.
      const search = wrapper.get('input[placeholder="Search OpenRouter models…"]')
      await search.setValue('gpt-4o')
      await wrapper.get('[data-testid="openrouter-result"]').trigger('click')

      const modelSelect = fieldControl(wrapper, 'Model')
      await modelSelect.setValue('openai/gpt-4o')

      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.text()).not.toContain('Base URL is required')
      expect(createProviderMock).toHaveBeenCalledTimes(1)
      expect(createProviderMock).toHaveBeenCalledWith(
        expect.objectContaining({ baseUrl: '', kind: 'openrouter' }),
      )
    })
  })

  describe('OpenRouter Model select + model search', () => {
    it('populates the Model select from form.models when editing an existing openrouter provider', async () => {
      const provider = makeProvider({
        kind: 'openrouter',
        baseUrl: '',
        model: 'openai/gpt-4o',
        models: ['openai/gpt-4o', 'anthropic/claude-3.5-sonnet'],
      })
      const wrapper = mount(ProviderForm, { props: { provider } })

      const modelSelect = fieldControl(wrapper, 'Model')
      const optionValues = modelSelect.findAll('option').map((option) => option.attributes('value'))

      expect(optionValues).toEqual(expect.arrayContaining(['openai/gpt-4o', 'anthropic/claude-3.5-sonnet']))
    })

    it('appends a searched model to form.models (reflected in the Model select) and dedupes repeat clicks', async () => {
      const provider = makeProvider({ kind: 'openrouter', baseUrl: '', models: [] })
      const wrapper = mount(ProviderForm, { props: { provider } })

      const search = wrapper.get('input[placeholder="Search OpenRouter models…"]')
      await search.setValue('gpt-4o')

      const resultButton = wrapper.get('[data-testid="openrouter-result"]')
      await resultButton.trigger('click')
      await resultButton.trigger('click') // clicking the same result again must not duplicate it

      const modelSelect = fieldControl(wrapper, 'Model')
      const optionValues = modelSelect.findAll('option').map((option) => option.attributes('value'))

      expect(optionValues.filter((value) => value === 'openai/gpt-4o')).toHaveLength(1)
    })

    it('renders no results while the search query is empty, to avoid rendering the whole catalog', async () => {
      const provider = makeProvider({ kind: 'openrouter', baseUrl: '', models: [] })
      const wrapper = mount(ProviderForm, { props: { provider } })

      expect(wrapper.findAll('[data-testid="openrouter-result"]')).toHaveLength(0)
      expect(wrapper.text()).toContain('Type to search models…')
    })
  })
})
