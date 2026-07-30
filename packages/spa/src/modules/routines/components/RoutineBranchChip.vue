<script setup lang="ts">
import { computed } from 'vue'
import type { RoutineRun } from '@shared/api/types'
import { flowLabel } from '@modules/routines/format'

// A small, secondary indicator of the branch pair a release run moves between
// (e.g. development → main), plus an optional flow badge. Purely informational,
// so it renders nothing until both branches are known and never competes with
// the MR title for the row's primary slot.
const props = defineProps<{ run: RoutineRun }>()

const hasBranches = computed(
  () => !!props.run.sourceBranch && !!props.run.targetBranch,
)
const flow = computed(() => flowLabel(props.run.flow))
</script>

<template>
  <span v-if="hasBranches" class="flex min-w-0 items-center gap-1.5">
    <span
      v-if="flow"
      class="chip text-muted shrink-0"
      :title="`${props.run.flow} flow`"
    >{{ flow }}</span>
    <span
      class="text-muted min-w-0 truncate font-mono text-xs"
      :title="`${props.run.sourceBranch} → ${props.run.targetBranch}`"
    >
      {{ props.run.sourceBranch }}
      <span class="text-muted/60" aria-hidden="true">→</span>
      {{ props.run.targetBranch }}
    </span>
  </span>
</template>
