import { describe, expect, it } from 'vitest'
import { SCENARIOS, answersToSamples } from '@modules/profiles/guided'

describe('answersToSamples', () => {
  it('maps answers to samples in scenario order', () => {
    const answers = {
      opener: 'Hey,',
      'minor-critique': 'this name is a bit unclear',
      praise: 'nice refactor',
    }
    expect(answersToSamples(answers)).toEqual([
      'this name is a bit unclear',
      'nice refactor',
      'Hey,',
    ])
  })

  it('trims surrounding whitespace on each answer', () => {
    const answers = { 'minor-critique': '  unclear name  \n' }
    expect(answersToSamples(answers)).toEqual(['unclear name'])
  })

  it('skips empty and whitespace-only answers', () => {
    const answers = {
      'minor-critique': 'real answer',
      'blocking-critique': '',
      praise: '   ',
      request: '\n\t',
    }
    expect(answersToSamples(answers)).toEqual(['real answer'])
  })

  it('returns an empty array when every answer is empty', () => {
    const answers = Object.fromEntries(SCENARIOS.map((s) => [s.key, '  ']))
    expect(answersToSamples(answers)).toEqual([])
  })

  it('returns an empty array for an empty record', () => {
    expect(answersToSamples({})).toEqual([])
  })

  it('ignores keys that are not known scenarios', () => {
    const answers = { unknown: 'ignored', praise: 'kept' }
    expect(answersToSamples(answers)).toEqual(['kept'])
  })

  it('does not mutate the input record', () => {
    const answers = { praise: '  kept  ' }
    answersToSamples(answers)
    expect(answers).toEqual({ praise: '  kept  ' })
  })
})
