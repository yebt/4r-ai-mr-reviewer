<script setup lang="ts">
import { computed } from 'vue'
import type { ConfirmDecision, RoutineRun } from '@shared/api/types'
import { stepLabel, stepStatusUi } from '@modules/routines/format'
import RoutineStatusChip from '@modules/routines/components/RoutineStatusChip.vue'

const props = defineProps<{
  run: RoutineRun
  resuming?: boolean
  // The confirmation decision currently in flight (so only the clicked button
  // spins), or null when no confirm is pending.
  confirming?: ConfirmDecision | null
}>()
const emit = defineEmits<{
  resume: [id: string]
  confirm: [id: string, decision: ConfirmDecision]
}>()

// The computed release summary is worth showing once the tag has been resolved.
const hasSummary = computed(
  () => props.run.state.nextTag != null || props.run.state.featCount != null,
)
const lastTag = computed(() => props.run.state.lastTag || 'no previous tag')
const featCount = computed(() => props.run.state.featCount ?? 0)
const fixCount = computed(() => props.run.state.fixCount ?? 0)

const awaiting = computed(() => props.run.status === 'awaiting_confirmation')
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

    <!-- Computed release summary: previous/next tag and the conventional-commit
         counts the tag bump was derived from. -->
    <dl v-if="hasSummary" class="border-line/40 bg-surface mb-3 grid gap-2 border px-3 py-3 text-sm">
      <div class="flex items-center justify-between gap-3">
        <dt class="label-mono">Previous tag</dt>
        <dd class="text-ink font-mono">{{ lastTag }}</dd>
      </div>
      <div class="flex items-center justify-between gap-3">
        <dt class="label-mono">Next tag</dt>
        <dd class="text-accent font-mono">{{ props.run.state.nextTag ?? '—' }}</dd>
      </div>
      <div class="flex items-center justify-between gap-3">
        <dt class="label-mono">Commits</dt>
        <dd class="text-muted font-mono text-xs">feat {{ featCount }}, fix {{ fixCount }}</dd>
      </div>
    </dl>

    <!-- lastError is surfaced prominently while the run is blocked so the reason
         it stopped is obvious before resuming. -->
    <p
      v-if="props.run.status === 'blocked' && props.run.lastError"
      class="text-warn border-warn/40 bg-warn/10 mb-3 border px-3 py-2 text-sm"
      role="alert"
    >
      {{ props.run.lastError }}
    </p>

    <!-- Interactive confirmation gate. The run is paused waiting on the user; the
         two actions are mutually exclusive and both call the confirm endpoint. -->
    <div
      v-if="awaiting"
      class="border-warn/50 bg-warn/10 mb-4 border px-4 py-4"
      role="alertdialog"
      aria-labelledby="confirm-heading"
    >
      <div class="mb-2 flex items-center gap-2">
        <span class="i-lucide-circle-pause text-warn text-base" aria-hidden="true" />
        <h3 id="confirm-heading" class="section-title text-warn">Confirmation required</h3>
      </div>
      <p class="text-ink mb-1 text-sm">
        Ready to release <span class="text-accent font-mono">{{ props.run.state.nextTag ?? '—' }}</span>
        <span class="text-muted"> · feat {{ featCount }}, fix {{ fixCount }}</span>
      </p>
      <p class="text-muted mb-3 text-xs">
        Merging now merges the merge request into its protected branch
        (development or main) and pushes the new tag. This cannot be undone.
      </p>
      <div class="flex flex-col gap-2 sm:flex-row">
        <button
          type="button"
          class="btn-danger-solid text-xs"
          :disabled="props.confirming != null"
          @click="emit('confirm', props.run.id, 'merge')"
        >
          <span
            :class="props.confirming === 'merge' ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-git-merge'"
            class="text-sm"
            aria-hidden="true"
          />
          {{ props.confirming === 'merge' ? 'Merging…' : 'Merge now' }}
        </button>
        <button
          type="button"
          class="btn-line text-xs"
          :disabled="props.confirming != null"
          @click="emit('confirm', props.run.id, 'wait')"
        >
          <span
            :class="props.confirming === 'wait' ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-hand'"
            class="text-sm"
            aria-hidden="true"
          />
          {{ props.confirming === 'wait' ? 'Saving…' : "I'll merge manually — just wait" }}
        </button>
      </div>
    </div>

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
