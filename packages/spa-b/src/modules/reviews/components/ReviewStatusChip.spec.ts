import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ReviewStatusChip from './ReviewStatusChip.vue'
import type { ReviewStatus } from '../types'

describe('ReviewStatusChip', () => {
  it.each([
    ['awaiting_approval', 'warning', 'Awaiting approval'],
    ['running', 'info', 'Running'],
    ['done', 'success', 'Done'],
    ['error', 'danger', 'Error'],
    ['pending', 'neutral', 'Pending'],
    ['cancelled', 'neutral', 'Cancelled'],
  ] as [ReviewStatus, string, string][])(
    'maps status "%s" to Badge status "%s" with label "%s"',
    (status, badgeStatus, label) => {
      const wrapper = mount(ReviewStatusChip, { props: { status } })
      expect(wrapper.attributes('data-status')).toBe(badgeStatus)
      expect(wrapper.text()).toBe(label)
    },
  )
})
