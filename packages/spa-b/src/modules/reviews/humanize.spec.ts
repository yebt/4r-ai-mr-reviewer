import { describe, expect, it } from 'vitest'
import { buildFindingBody, DIMENSION_LABELS, ORIGINAL } from './humanize'
import type { Finding } from './types'

function makeFinding(overrides: Partial<Finding> = {}): Finding {
  return {
    index: 0,
    dimension: 'risk',
    severity: 'high',
    file: 'src/app.ts',
    line: 12,
    issue: 'Original issue',
    why: 'Original why',
    fix: 'Original fix',
    blocking: false,
    published: false,
    ...overrides,
  }
}

describe('ORIGINAL', () => {
  it('is a sentinel distinct from any real tab index', () => {
    expect(ORIGINAL).toBe(-1)
  })
})

describe('buildFindingBody', () => {
  it('builds a blocking risk/high finding with why and fix — exact string', () => {
    const finding = makeFinding({ dimension: 'risk', severity: 'high', blocking: true })
    const body = buildFindingBody(finding, { issue: 'Issue text', why: 'Why text', fix: 'Fix text' })

    expect(body).toBe(
      '**[R1 Risk · HIGH]** Issue text\n\n' + '**Why:** Why text\n\n' + '**Suggested fix:** Fix text\n' + '\n_Blocking._',
    )
  })

  it('omits the Why paragraph when why is empty', () => {
    const finding = makeFinding({ blocking: false })
    const body = buildFindingBody(finding, { issue: 'Issue text', why: '', fix: 'Fix text' })

    expect(body).toBe('**[R1 Risk · HIGH]** Issue text\n\n' + '**Suggested fix:** Fix text\n')
  })

  it('omits the Suggested fix paragraph when fix is empty', () => {
    const finding = makeFinding({ blocking: false })
    const body = buildFindingBody(finding, { issue: 'Issue text', why: 'Why text', fix: '' })

    expect(body).toBe('**[R1 Risk · HIGH]** Issue text\n\n' + '**Why:** Why text\n\n')
  })

  it('omits the Blocking suffix when the finding is not blocking', () => {
    const finding = makeFinding({ blocking: false })
    const body = buildFindingBody(finding, { issue: 'Issue text', why: '', fix: '' })

    expect(body).toBe('**[R1 Risk · HIGH]** Issue text\n\n')
  })

  it.each([
    ['risk', 'R1 Risk'],
    ['readability', 'R2 Readability'],
    ['reliability', 'R3 Reliability'],
    ['resilience', 'R4 Resilience'],
  ] as const)('maps dimension %s to header label %s', (dimension, label) => {
    expect(DIMENSION_LABELS[dimension]).toBe(label)
    const finding = makeFinding({ dimension })
    const body = buildFindingBody(finding, { issue: 'Issue', why: '', fix: '' })
    expect(body.startsWith(`**[${label} · HIGH]**`)).toBe(true)
  })

  it('uppercases severity in the header', () => {
    const finding = makeFinding({ severity: 'medium' })
    const body = buildFindingBody(finding, { issue: 'Issue', why: '', fix: '' })
    expect(body.startsWith('**[R1 Risk · MEDIUM]**')).toBe(true)
  })
})
