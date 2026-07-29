<script setup lang="ts">
import type { RoutineRunStatus } from '@shared/api/types'
import { runStatusUi } from '@modules/routines/format'

const props = defineProps<{ status: RoutineRunStatus }>()
</script>

<template>
  <span class="chip" :class="runStatusUi[props.status].class">
    <span
      :class="[runStatusUi[props.status].icon, runStatusUi[props.status].spin ? 'animate-spin' : '']"
      class="text-sm"
      aria-hidden="true"
    />
    {{ runStatusUi[props.status].label }}
  </span>
</template>
