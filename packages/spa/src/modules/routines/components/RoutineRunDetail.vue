<script setup lang="ts">
import type { RoutineRun } from '@shared/api/types'
import { stepLabel, stepStatusUi } from '@modules/routines/format'
import RoutineStatusChip from '@modules/routines/components/RoutineStatusChip.vue'

const props = defineProps<{ run: RoutineRun; resuming?: boolean }>()
const emit = defineEmits<{ resume: [id: string] }>()
</script>

<template>
  <div class="border-line/50 mt-3 border-t pt-4">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <span class="text-ink font-mono text-sm">!{{ props.run.mrIid }}</span>
        <RoutineStatusChip :status="props.run.status" />
      </div>
      <span v-if="props.run.state.nextTag" class="chip text-accent">
        <span class="i-lucide-tag text-sm" aria-hidden="true" />
        next tag: {{ props.run.state.nextTag }}
      </span>
    </div>

    <!-- lastError is surfaced prominently while the run is blocked so the reason
         it stopped is obvious before resuming. -->
    <p
      v-if="props.run.status === 'blocked' && props.run.lastError"
      class="text-warn border-warn/40 bg-warn/10 mb-3 border px-3 py-2 text-sm"
      role="alert"
    >
      {{ props.run.lastError }}
    </p>

    <!-- Step ledger: one row per step, its state shown by icon + label. -->
    <ul>
      <li v-for="step in props.run.steps" :key="step.name" class="row items-start">
        <span
          :class="[
            stepStatusUi[step.status].icon,
            stepStatusUi[step.status].class,
            stepStatusUi[step.status].spin ? 'animate-spin' : '',
          ]"
          class="mt-0.5 shrink-0 text-base"
          aria-hidden="true"
        />
        <div class="min-w-0 flex-1">
          <div class="text-ink flex items-center gap-2 text-sm">
            <span>{{ stepLabel[step.name] }}</span>
            <span class="label-mono">
              <span class="sr-only">Status: </span>{{ stepStatusUi[step.status].label }}
            </span>
          </div>
          <div v-if="step.detail" class="text-muted mt-0.5 text-xs">{{ step.detail }}</div>
        </div>
      </li>
    </ul>

    <div v-if="props.run.status === 'blocked'" class="mt-4">
      <button
        type="button"
        class="btn-line text-xs"
        :disabled="props.resuming"
        @click="emit('resume', props.run.id)"
      >
        <span
          :class="props.resuming ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-play'"
          class="text-sm"
          aria-hidden="true"
        />
        {{ props.resuming ? 'Resuming…' : 'Resume' }}
      </button>
    </div>
  </div>
</template>
