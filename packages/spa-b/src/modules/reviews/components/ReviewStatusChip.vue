<script setup lang="ts">
/**
 * Status chip for a single `Review` — maps the DTO's `status` to a `Badge`
 * status/label pair. Pure presentation: no store access.
 */
import { computed } from 'vue'
import { Badge, type BadgeStatus } from '@shared/ui/design-system'
import type { ReviewStatus } from '../types'

const props = defineProps<{
  status: ReviewStatus
}>()

const badgeStatusMap: Record<ReviewStatus, BadgeStatus> = {
  awaiting_approval: 'warning',
  running: 'info',
  done: 'success',
  error: 'danger',
  pending: 'neutral',
  cancelled: 'neutral',
}

const labelMap: Record<ReviewStatus, string> = {
  awaiting_approval: 'Awaiting approval',
  running: 'Running',
  done: 'Done',
  error: 'Error',
  pending: 'Pending',
  cancelled: 'Cancelled',
}

const badgeStatus = computed(() => badgeStatusMap[props.status])
const label = computed(() => labelMap[props.status])
</script>

<template>
  <Badge :status="badgeStatus">{{ label }}</Badge>
</template>
