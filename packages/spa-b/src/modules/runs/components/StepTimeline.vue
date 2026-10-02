<script setup lang="ts">
/**
 * Vertical progress timeline for a run's steps.
 *
 * Each step is a 24px status node on one continuous rail (the segment between
 * two reached steps turns success-colored, so progress reads at a glance),
 * followed by a readable label (`stepLabel`), the backend's detail line and a
 * right-aligned meta: completion time for done steps, a word for everything
 * else. Status lives in the node + meta, not in a chip on every row.
 *
 * `attentionStep` names the step that is waiting on the user (the confirm
 * gate, or the step a blocked run stopped at). It renders as "Waiting for
 * you" and gets the `attention` slot beneath it, so the page can put the
 * decision (Merge / Wait, Resume / Skip) exactly where the run is paused.
 */
import { computed } from 'vue'
import { Icon, Text } from '@shared/ui/design-system'
import { formatTime, stepLabel } from '../format'
import type { RoutineStep } from '../types'

const props = defineProps<{
  steps: RoutineStep[]
  attentionStep?: string
}>()

type NodeState = 'done' | 'running' | 'attention' | 'failed' | 'skipped' | 'pending'

// Literal class strings for Tailwind's scanner (same approach as Badge.vue).
const NODE_CLASS: Record<NodeState, string> = {
  done: 'bg-success-bg text-success-text',
  running: 'bg-info-bg text-info-text',
  attention: 'bg-warning-bg text-warning-text ring-1 ring-warning-solid',
  failed: 'bg-danger-bg text-danger-text',
  skipped: 'bg-bg-hover text-text-muted',
  pending: 'border border-line bg-bg-panel',
}

const NODE_ICON: Partial<Record<NodeState, string>> = {
  done: 'check',
  running: 'loader-circle',
  attention: 'circle-pause',
  failed: 'x',
  skipped: 'minus',
}

const META: Partial<Record<NodeState, { text: string; class: string }>> = {
  running: { text: 'Running', class: 'text-info-text' },
  attention: { text: 'Waiting for you', class: 'font-medium text-warning-text' },
  failed: { text: 'Failed', class: 'text-danger-text' },
  skipped: { text: 'Skipped', class: 'text-text-muted' },
}

const rows = computed(() =>
  props.steps.map((step, index) => {
    const state: NodeState = step.name === props.attentionStep ? 'attention' : step.status
    const next = props.steps[index + 1]
    return {
      step,
      state,
      label: stepLabel(step.name),
      // The rail below a step is "reached" once the next step has started.
      railReached: step.status === 'done' && next !== undefined && next.status !== 'pending',
      isLast: index === props.steps.length - 1,
      isCurrent: state === 'attention' || state === 'running',
    }
  }),
)
</script>

<template>
  <ol v-if="steps.length > 0" class="flex flex-col" data-testid="run-step-list">
    <li
      v-for="row in rows"
      :key="row.step.name"
      class="relative flex gap-3 pb-5 last:pb-0"
      :aria-current="row.isCurrent ? 'step' : undefined"
      data-testid="run-step-row"
    >
      <span
        v-if="!row.isLast"
        class="absolute top-6 bottom-0 left-3 w-px -translate-x-1/2"
        :class="row.railReached ? 'bg-success-solid' : 'bg-line'"
        aria-hidden="true"
      />

      <span
        class="relative grid size-6 shrink-0 place-items-center rounded-full"
        :class="NODE_CLASS[row.state]"
        aria-hidden="true"
      >
        <Icon
          v-if="NODE_ICON[row.state]"
          :name="NODE_ICON[row.state]!"
          size="xs"
          :class="row.state === 'running' ? 'motion-safe:animate-spin' : ''"
        />
      </span>

      <div class="flex min-w-0 flex-1 flex-col gap-0.5 pt-0.5">
        <div class="flex items-baseline justify-between gap-3">
          <Text
            :class="
              row.state === 'pending'
                ? 'text-text-muted'
                : row.isCurrent || row.state === 'failed'
                  ? 'font-semibold'
                  : 'font-medium'
            "
          >
            {{ row.label }}
          </Text>
          <span
            v-if="META[row.state]"
            class="shrink-0 text-xs"
            :class="META[row.state]!.class"
          >
            {{ META[row.state]!.text }}
          </span>
          <time
            v-else-if="row.state === 'done' && row.step.updatedAt"
            :datetime="row.step.updatedAt"
            class="shrink-0 text-xs text-text-muted tabular-nums"
          >
            {{ formatTime(row.step.updatedAt) }}
          </time>
          <span v-if="row.state === 'done'" class="sr-only">Done</span>
          <span v-else-if="row.state === 'pending'" class="sr-only">Pending</span>
        </div>

        <Text
          v-if="row.step.detail && row.state !== 'pending'"
          muted
          size="sm"
          class="break-words [overflow-wrap:anywhere]"
        >
          {{ row.step.detail }}
        </Text>

        <div v-if="row.state === 'attention' && $slots.attention" class="mt-2">
          <slot name="attention" :step="row.step" />
        </div>
      </div>
    </li>
  </ol>
  <Text v-else muted size="sm">No steps recorded yet.</Text>
</template>
