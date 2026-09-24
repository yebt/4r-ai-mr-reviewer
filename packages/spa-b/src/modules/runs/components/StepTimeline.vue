<script setup lang="ts">
/**
 * Vertical timeline for a run's steps — replaces the flat step-card `<ol>`
 * that used to live inline in `pages/runs/[id].vue`. A connected line runs
 * down the left edge through a status dot per step (colored via
 * `stepStatusUi`'s `variant`, same lookup table `StepStatusChip` reads), with
 * the last step dropping its trailing segment. Keeps the existing
 * name/detail/`StepStatusChip` content — this only changes the shell around
 * it.
 */
import type { RoutineStep, RoutineStepStatus } from '../types'
import { Text } from '@shared/ui/design-system'
import StepStatusChip from './StepStatusChip.vue'

defineProps<{
  steps: RoutineStep[]
}>()

// Tailwind's scanner needs literal class strings, so map statuses to full
// classes rather than interpolating into a template literal — same pattern
// as `Badge.vue`'s `statusClassMap`. Dot color mirrors `stepStatusUi`'s
// `variant` (neutral/info/success/danger): pending is a hollow ring (not yet
// reached), running fills accent and pulses (respecting motion-reduce, same
// as the page's own "Live" indicator), done/failed fill solid, and skipped
// gets a filled-but-muted ring so it reads as "settled" rather than
// "pending".
const DOT_STATUS_CLASS: Record<RoutineStepStatus, string> = {
  pending: 'border-line-strong bg-bg-panel',
  running: 'border-info-solid bg-info-solid animate-pulse motion-reduce:animate-none',
  done: 'border-success-solid bg-success-solid',
  failed: 'border-danger-solid bg-danger-solid',
  skipped: 'border-line-strong bg-line-strong',
}
</script>

<template>
  <ol v-if="steps.length > 0" class="flex flex-col" data-testid="run-step-list">
    <li
      v-for="(step, index) in steps"
      :key="step.name"
      class="flex gap-3 pb-4 last:pb-0"
      data-testid="run-step-row"
    >
      <div class="flex flex-col items-center">
        <span
          class="z-10 size-3 shrink-0 rounded-full border-2"
          :class="DOT_STATUS_CLASS[step.status]"
          aria-hidden="true"
        />
        <span v-if="index < steps.length - 1" class="w-px flex-1 bg-line-subtle" />
      </div>
      <div class="flex min-w-0 flex-1 flex-col gap-1">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <Text class="truncate font-medium">{{ step.name }}</Text>
          <StepStatusChip :status="step.status" />
        </div>
        <Text muted size="sm" class="truncate">{{ step.detail }}</Text>
      </div>
    </li>
  </ol>
  <Text v-else muted size="sm">No steps recorded yet.</Text>
</template>
