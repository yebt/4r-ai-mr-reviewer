import { describe, expect, it } from 'vitest'
import type { Finding, FindingHumanized } from '@shared/api/types'
import { buildFindingBody, buildFindingMarkdown } from '@modules/reviews/humanize-overrides'

const finding = (over: Partial<Finding> = {}): Finding => ({
  index: 0,
  dimension: 'risk',
  severity: 'high',
  file: 'a.ts',
  line: 3,
  issue: 'orig issue',
  why: 'orig why',
  fix: 'orig fix',
  blocking: false,
  published: false,
  ...over,
})

const humanized = (over: Partial<FindingHumanized> = {}): FindingHumanized => ({
  issue: 'kind issue',
  why: 'kind why',
  fix: 'kind fix',
  ...over,
})

describe('buildFindingBody', () => {
  it('restores the dimension/severity tag and includes why + fix', () => {
    expect(buildFindingBody(finding(), humanized())).toBe(
      '**[R1 Risk · HIGH]** kind issue\n\n**Why:** kind why\n\n**Suggested fix:** kind fix\n',
    )
  })

  it('omits the Why block when the humanized why is empty', () => {
    expect(buildFindingBody(finding(), humanized({ why: '' }))).toBe(
      '**[R1 Risk · HIGH]** kind issue\n\n**Suggested fix:** kind fix\n',
    )
  })

  it('omits the Suggested fix block when the humanized fix is empty', () => {
    expect(buildFindingBody(finding(), humanized({ fix: '' }))).toBe(
      '**[R1 Risk · HIGH]** kind issue\n\n**Why:** kind why\n\n',
    )
  })

  it('emits only the header when why and fix are both empty', () => {
    expect(buildFindingBody(finding(), humanized({ why: '', fix: '' }))).toBe(
      '**[R1 Risk · HIGH]** kind issue\n\n',
    )
  })

  it('appends the blocking footer when the finding is blocking', () => {
    expect(buildFindingBody(finding({ blocking: true }), humanized())).toBe(
      '**[R1 Risk · HIGH]** kind issue\n\n**Why:** kind why\n\n**Suggested fix:** kind fix\n\n_Blocking._',
    )
  })

  it('labels each dimension and uppercases the severity', () => {
    const body = buildFindingBody(
      finding({ dimension: 'reliability', severity: 'medium' }),
      humanized({ why: '', fix: '' }),
    )
    expect(body).toBe('**[R3 Reliability · MEDIUM]** kind issue\n\n')
  })
})

describe('buildFindingMarkdown', () => {
  it('includes the header, file:line, why and fix', () => {
    expect(buildFindingMarkdown(finding(), humanized())).toBe(
      '**[R1 Risk · HIGH]** kind issue\n\n`a.ts:3`\n\n**Why:** kind why\n\n**Suggested fix:** kind fix\n',
    )
  })

  it('omits the location when the finding has no file', () => {
    expect(buildFindingMarkdown(finding({ file: '' }), humanized({ why: '', fix: '' }))).toBe(
      '**[R1 Risk · HIGH]** kind issue\n',
    )
  })

  it('drops the line suffix when line is 0', () => {
    expect(buildFindingMarkdown(finding({ line: 0 }), humanized({ why: '', fix: '' }))).toBe(
      '**[R1 Risk · HIGH]** kind issue\n\n`a.ts`\n',
    )
  })

  it('appends the blocking footer', () => {
    expect(
      buildFindingMarkdown(finding({ blocking: true }), humanized({ why: '', fix: '' })),
    ).toBe('**[R1 Risk · HIGH]** kind issue\n\n`a.ts:3`\n\n_Blocking._\n')
  })
})
