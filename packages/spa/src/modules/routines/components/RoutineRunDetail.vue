<script setup lang="ts">
import { computed } from 'vue'
import type { ConfirmDecision, RoutineRun } from '@shared/api/types'
import { isRunCancelable, runTitle, stepLabel, stepStatusUi } from '@modules/routines/format'
import { confirm } from '@shared/composables/useConfirm'
import RoutineStatusChip from '@modules/routines/components/RoutineStatusChip.vue'
import RoutineBranchChip from '@modules/routines/components/RoutineBranchChip.vue'

const props = defineProps<{
  run: RoutineRun
  resuming?: boolean
  // True while a skip request for this run's failed step is in flight.
  skipping?: boolean
  // The confirmation decision currently in flight (so only the clicked button
  // spins), or null when no confirm is pending.
  confirming?: ConfirmDecision | null
  // True while a delete request for this run is in flight (disables the button).
  deleting?: boolean
  // True while a cancel request for this run is in flight (disables the button).
  cancelling?: boolean
}>()
const emit = defineEmits<{
  resume: [id: string]
  skip: [id: string]
  confirm: [id: string, decision: ConfirmDecision]
  delete: [id: string]
  cancel: [id: string]
}>()

// A run can be cancelled until it reaches a terminal state; the button hides
// once done/cancelled and Delete takes over.
const cancelable = computed(() => isRunCancelable(props.run.status))

// The computed release summary is worth showing once the tag has been resolved.
const hasSummary = computed(
  () => props.run.state.nextTag != null || props.run.state.featCount != null,
)
const lastTag = computed(() => props.run.state.lastTag || 'no previous tag')
const featCount = computed(() => props.run.state.featCount ?? 0)
const fixCount = computed(() => props.run.state.fixCount ?? 0)

const awaiting = computed(() => props.run.status === 'awaiting_confirmation')

// The merge/tag is the single most irreversible action in the product — it
// mutates a real protected branch. Gate it behind the same danger confirm as
// Delete/Cancel, and name the concrete target so the user authorizes what they
// can actually see (not a generic "development or main").
async function requestMerge() {
  const target = props.run.targetBranch || 'its protected branch'
  const tag = props.run.state.nextTag ?? 'the new tag'
  const ok = await confirm({
    title: 'Merge and release',
    message: `Merge this merge request into ${target} and push ${tag}? This merges a protected branch on your real GitLab project and cannot be undone.`,
    danger: true,
    confirmText: 'Merge and release',
  })
  if (ok) emit('confirm', props.run.id, 'merge')
}

// Steps that can be skipped client-side; the server has the final say (409 for
// essential steps). Skip is only offered on a blocked run whose failed step is
// one of these non-critical steps.
const SKIPPABLE_STEPS = new Set(['react', 'comment', 'approve', 'notify'])
const skippableFailedStep = computed(() => {
  if (props.run.status !== 'blocked') return null
  const failed = props.run.steps.find((s) => s.status === 'failed')
  return failed && SKIPPABLE_STEPS.has(failed.name) ? failed : null
})
</script>

<template>
  <div class="border-line/50 mt-3 border-t pt-4">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-3">
        <!-- Primary: the MR title when captured, else the !{mrIid} fallback. -->
        <span class="text-ink min-w-0 truncate text-sm font-medium">{{ runTitle(props.run) }}</span>
        <span
          v-if="props.run.state.mrTitle && props.run.mrIid"
          class="text-muted shrink-0 font-mono text-xs"
        >!{{ props.run.mrIid }}</span>
        <RoutineStatusChip :status="props.run.status" />
      </div>
      <div class="flex shrink-0 items-center gap-3">
        <span v-if="props.run.state.nextTag" class="chip text-ink">
          <span class="i-lucide-tag text-sm" aria-hidden="true" />
          next tag: {{ props.run.state.nextTag }}
        </span>
        <!-- Cancel is offered while the run is still cancelable (not terminal);
             it aborts a running/pending run or a blocked/awaiting pause. -->
        <button
          v-if="cancelable"
          type="button"
          class="btn-line text-xs hover:border-danger hover:text-danger"
          :disabled="props.cancelling"
          :aria-label="`Cancel ${runTitle(props.run)}`"
          @click="emit('cancel', props.run.id)"
        >
          <span
            :class="props.cancelling ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-ban'"
            class="text-sm"
            aria-hidden="true"
          />
          Cancel
        </button>
        <!-- Delete is offered only for a run that is not running; the backend 409
             is a backstop. -->
        <button
          v-if="props.run.status !== 'running'"
          type="button"
          class="btn-ghost text-xs hover:text-danger"
          :disabled="props.deleting"
          :aria-label="`Delete ${runTitle(props.run)}`"
          @click="emit('delete', props.run.id)"
        >
          <span
            :class="props.deleting ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-trash-2'"
            class="text-sm"
            aria-hidden="true"
          />
          Delete
        </button>
      </div>
    </div>

    <!-- Secondary branch/flow indicator; renders only when both branches are
         known (absent on legacy runs). -->
    <div v-if="props.run.sourceBranch && props.run.targetBranch" class="mb-3">
      <RoutineBranchChip :run="props.run" />
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
        <dd class="text-ink font-mono">{{ props.run.state.nextTag ?? '—' }}</dd>
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
        Merging now merges this merge request into
        <span class="text-ink font-mono">{{ props.run.targetBranch || 'its protected branch' }}</span>
        and pushes <span class="text-ink font-mono">{{ props.run.state.nextTag ?? 'the new tag' }}</span>
        on your real GitLab project. This cannot be undone.
      </p>
      <div class="flex flex-col gap-2 sm:flex-row">
        <button
          type="button"
          class="btn-danger-solid text-xs"
          :disabled="props.confirming != null"
          @click="requestMerge"
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
      <li v-for="(step, i) in props.run.steps" :key="step.name" class="row items-start">
        <span class="label-mono mt-1 w-4 shrink-0 text-right" aria-hidden="true">{{ i + 1 }}</span>
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

    <div v-if="props.run.status === 'blocked'" class="mt-4 flex flex-wrap items-center gap-3">
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
      <!-- Skip is offered only when the blocked run's failed step is a
           non-critical, client-side-skippable step; the backend 409 is a
           backstop for essential steps. -->
      <button
        v-if="skippableFailedStep"
        type="button"
        class="btn-line text-xs"
        :disabled="props.skipping"
        @click="emit('skip', props.run.id)"
      >
        <span
          :class="props.skipping ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-skip-forward'"
          class="text-sm"
          aria-hidden="true"
        />
        {{ props.skipping ? 'Skipping…' : 'Skip step' }}
      </button>
    </div>
  </div>
</template>
