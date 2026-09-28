import { describe, expect, it } from 'vitest'
import {
  MAX_RELEASE_EMOJIS,
  addEmoji,
  buildDevReleasePayload,
  buildMainReleasePayload,
  customEmojis,
  defaultMergeWhenPipelineSucceeds,
  defaultReleaseFormState,
  isValidEmojiName,
  missingConventionalBranches,
  normalizeCustomEmojiName,
  resolveDefaultBranches,
  toggleEmoji,
  type ReleaseFormState,
} from './release'

function makeForm(overrides: Partial<ReleaseFormState> = {}): ReleaseFormState {
  return {
    bump: 'minor',
    includeDev: false,
    sourceBranch: '',
    targetBranch: '',
    emojis: ['thumbsup', 'seedling'],
    removeSourceBranch: false,
    mergeWhenPipelineSucceeds: true,
    ...overrides,
  }
}

describe('defaultMergeWhenPipelineSucceeds', () => {
  it('defaults to true for the dev flow', () => {
    expect(defaultMergeWhenPipelineSucceeds('dev')).toBe(true)
  })

  it('defaults to false for the main flow', () => {
    expect(defaultMergeWhenPipelineSucceeds('main')).toBe(false)
  })
})

describe('defaultReleaseFormState', () => {
  it('returns minor/no-includeDev/default emojis with mergeWhenPipelineSucceeds=true for dev', () => {
    expect(defaultReleaseFormState('dev')).toEqual({
      bump: 'minor',
      includeDev: false,
      sourceBranch: '',
      targetBranch: '',
      emojis: ['thumbsup', 'seedling'],
      removeSourceBranch: false,
      mergeWhenPipelineSucceeds: true,
    })
  })

  it('defaults mergeWhenPipelineSucceeds=false for main', () => {
    expect(defaultReleaseFormState('main').mergeWhenPipelineSucceeds).toBe(false)
  })

  it('returns a fresh emojis array each call (no shared mutable default)', () => {
    const a = defaultReleaseFormState('dev')
    a.emojis.push('rocket')
    expect(defaultReleaseFormState('dev').emojis).toEqual(['thumbsup', 'seedling'])
  })
})

describe('toggleEmoji', () => {
  it('adds a name not already selected', () => {
    expect(toggleEmoji(['thumbsup'], 'rocket')).toEqual(['thumbsup', 'rocket'])
  })

  it('removes a name already selected', () => {
    expect(toggleEmoji(['thumbsup', 'rocket'], 'thumbsup')).toEqual(['rocket'])
  })

  it('never mutates the input array', () => {
    const selected = ['thumbsup']
    toggleEmoji(selected, 'rocket')
    expect(selected).toEqual(['thumbsup'])
  })

  it('refuses to select a new name past MAX_RELEASE_EMOJIS, returning the selection unchanged', () => {
    const atCap = Array.from({ length: MAX_RELEASE_EMOJIS }, (_, i) => `emoji_${i}`)
    const result = toggleEmoji(atCap, 'one_too_many')
    expect(result).toEqual(atCap)
    expect(result).toBe(atCap)
  })

  it('still deselects an already-selected name at the cap', () => {
    const atCap = Array.from({ length: MAX_RELEASE_EMOJIS }, (_, i) => `emoji_${i}`)
    expect(toggleEmoji(atCap, 'emoji_0')).toEqual(atCap.slice(1))
  })
})

describe('addEmoji', () => {
  it('adds a name not already present', () => {
    expect(addEmoji(['thumbsup'], 'custom_name')).toEqual(['thumbsup', 'custom_name'])
  })

  it('is idempotent for a name already present', () => {
    expect(addEmoji(['thumbsup', 'rocket'], 'rocket')).toEqual(['thumbsup', 'rocket'])
  })

  it('refuses to grow the selection past MAX_RELEASE_EMOJIS, returning the selection unchanged', () => {
    const atCap = Array.from({ length: MAX_RELEASE_EMOJIS }, (_, i) => `emoji_${i}`)
    const result = addEmoji(atCap, 'one_too_many')
    expect(result).toEqual(atCap)
    expect(result).toBe(atCap)
  })
})

describe('customEmojis', () => {
  it('returns only names outside the curated EMOJI_CHOICES list', () => {
    expect(customEmojis(['thumbsup', 'my_custom_name', 'rocket'])).toEqual(['my_custom_name'])
  })

  it('returns an empty array when every selected name is curated', () => {
    expect(customEmojis(['thumbsup', 'seedling'])).toEqual([])
  })
})

describe('isValidEmojiName', () => {
  it.each(['thumbsup', 'white_check_mark', 'my-emoji', 'a1+b2', 'x'])('accepts %s', (name) => {
    expect(isValidEmojiName(name)).toBe(true)
  })

  it.each(['Thumbsup', 'has space', 'emoji!', ':rocket:', ''])('rejects %s', (name) => {
    expect(isValidEmojiName(name)).toBe(false)
  })
})

describe('normalizeCustomEmojiName', () => {
  it('trims surrounding whitespace', () => {
    expect(normalizeCustomEmojiName('  rocket  ')).toBe('rocket')
  })

  it('strips a leading and trailing colon (pasted :emoji: form)', () => {
    expect(normalizeCustomEmojiName(':rocket:')).toBe('rocket')
  })

  it('leaves an already-plain name untouched', () => {
    expect(normalizeCustomEmojiName('rocket')).toBe('rocket')
  })
})

describe('missingConventionalBranches', () => {
  it('returns both when neither exists', () => {
    expect(missingConventionalBranches(['feature-x'])).toEqual(['development', 'main'])
  })

  it('returns only the missing one', () => {
    expect(missingConventionalBranches(['development', 'feature-x'])).toEqual(['main'])
  })

  it('returns an empty array when both exist', () => {
    expect(missingConventionalBranches(['development', 'main', 'feature-x'])).toEqual([])
  })
})

describe('resolveDefaultBranches', () => {
  it('picks development/main when both present', () => {
    expect(resolveDefaultBranches(['development', 'main', 'feature-x'])).toEqual({
      source: 'development',
      target: 'main',
    })
  })

  it('leaves a missing conventional branch blank rather than guessing', () => {
    expect(resolveDefaultBranches(['feature-x'])).toEqual({ source: '', target: '' })
  })
})

describe('buildDevReleasePayload', () => {
  it('includes emojis when the selection is non-empty', () => {
    const payload = buildDevReleasePayload(makeForm({ emojis: ['thumbsup'] }), 42)
    expect(payload).toEqual({
      mrIid: 42,
      bump: 'minor',
      emojis: ['thumbsup'],
      removeSourceBranch: false,
      mergeWhenPipelineSucceeds: true,
    })
  })

  it('omits emojis when the selection is empty, so the backend default applies', () => {
    const payload = buildDevReleasePayload(makeForm({ emojis: [] }), 42)
    expect(payload).toEqual({
      mrIid: 42,
      bump: 'minor',
      removeSourceBranch: false,
      mergeWhenPipelineSucceeds: true,
    })
    expect(payload).not.toHaveProperty('emojis')
  })
})

describe('buildMainReleasePayload', () => {
  it('trims branch names and includes emojis when non-empty', () => {
    const payload = buildMainReleasePayload(
      makeForm({ sourceBranch: ' development ', targetBranch: ' main ', includeDev: true }),
    )
    expect(payload).toEqual({
      bump: 'minor',
      includeDev: true,
      sourceBranch: 'development',
      targetBranch: 'main',
      emojis: ['thumbsup', 'seedling'],
      removeSourceBranch: false,
      mergeWhenPipelineSucceeds: true,
    })
  })

  it('omits emojis when the selection is empty', () => {
    const payload = buildMainReleasePayload(makeForm({ emojis: [] }))
    expect(payload).not.toHaveProperty('emojis')
  })

  it('sends blank branch names as empty strings so the backend default applies', () => {
    const payload = buildMainReleasePayload(makeForm({ sourceBranch: '', targetBranch: '' }))
    expect(payload.sourceBranch).toBe('')
    expect(payload.targetBranch).toBe('')
  })
})
