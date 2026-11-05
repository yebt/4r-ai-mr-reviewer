import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import StepTimeline from './StepTimeline.vue'
import RunStatusChip from './RunStatusChip.vue'
import StepStatusChip from './StepStatusChip.vue'
import type { RoutineStep } from '../types'

const step = (status: RoutineStep['status']): RoutineStep =>
  ({ name: 'merge', status, detail: '', updatedAt: '' }) as RoutineStep

describe('running spinner', () => {
  it('draws a running step with the track-and-arc Spinner, not an open icon', () => {
    const wrapper = mount(StepTimeline, { props: { steps: [step('running')] } })

    expect(wrapper.find('[data-testid="spinner-track"]').exists()).toBe(true)
    expect(wrapper.find('.lucide-loader-circle').exists()).toBe(false)
  })

  it('gives the status chips the same spinner, hidden from assistive tech beside the label', () => {
    for (const chip of [
      mount(RunStatusChip, { props: { status: 'running' } }),
      mount(StepStatusChip, { props: { status: 'running' } }),
    ]) {
      const track = chip.find('[data-testid="spinner-track"]')
      expect(track.exists()).toBe(true)
      expect(track.element.closest('svg')?.getAttribute('aria-hidden')).toBe('true')
      expect(chip.text()).toContain('Running')
    }
  })

  it('keeps the plain icon for chips that are not spinning', () => {
    const chip = mount(RunStatusChip, { props: { status: 'done' } })

    expect(chip.find('[data-testid="spinner-track"]').exists()).toBe(false)
    expect(chip.find('svg').exists()).toBe(true)
  })
})
