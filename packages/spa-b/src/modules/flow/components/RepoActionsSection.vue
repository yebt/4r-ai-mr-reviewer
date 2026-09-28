<script setup lang="ts">
/**
 * Actions tab — organism. A repo's routine runs (`GET /repos/{id}/routines`),
 * newest first, each row linking to the run detail page
 * (`src/pages/runs/[id].vue`). Live-polls every 2.5s while any loaded run is
 * non-terminal (`isRunActive` — pending/running/blocked/
 * awaiting_confirmation), gated on document visibility too — mirrors
 * `useRunsStore`/`useRunDetail`'s exact leak-safe pattern (an
 * `{ immediate: true }` watch drives `resume`/`pause`, and `onUnmounted`
 * stops the interval since navigating away is otherwise invisible to it).
 */
import { computed, onUnmounted, watch } from 'vue'
import { useQuery } from '@pinia/colada'
import { useDocumentVisibility, useIntervalFn } from '@vueuse/core'
import { Alert, Button, Icon, Skeleton, Text } from '@shared/ui/design-system'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { listRepoRoutines, repoRoutinesQueryKey } from '@modules/runs/api'
import { flowLabel, formatDateTime, isRunActive, routineKindLabel, runTitle, RunStatusChip } from '@modules/runs'

const props = defineProps<{ repoId: string }>()

const POLL_INTERVAL_MS = 2500

const { data, isLoading, error, refetch } = useQuery({
  key: () => repoRoutinesQueryKey(props.repoId),
  query: () => listRepoRoutines(props.repoId),
})

const runs = computed(() => data.value ?? [])
const hasActiveRun = computed(() => runs.value.some((run) => isRunActive(run.status)))

const documentVisibility = useDocumentVisibility()
const canPoll = computed(() => hasActiveRun.value && documentVisibility.value === 'visible')

const { pause, resume } = useIntervalFn(() => refetch(), POLL_INTERVAL_MS, { immediate: false })
watch(canPoll, (active) => (active ? resume() : pause()), { immediate: true })

// The interval's own cleanup only fires when this component's owner
// unmounts — but this section is what unmounts on tab switch, so this is
// exactly that owner, and pausing here stops polling reliably.
onUnmounted(pause)
</script>

<template>
  <section class="flex flex-col gap-3">
    <div v-if="isLoading" class="flex flex-col gap-2" data-testid="flow-actions-loading-skeleton">
      <div
        v-for="n in 3"
        :key="n"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
      >
        <Skeleton class="h-4 w-48" />
      </div>
    </div>

    <Alert v-else-if="error" status="danger">
      <p>{{ resolveErrorMessage(error, 'Failed to load runs') }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="refetch()">Retry</Button>
    </Alert>

    <div
      v-else-if="runs.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="activity" size="lg" class="text-text-muted" />
      <Text muted>No runs for this repository yet.</Text>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li v-for="run in runs" :key="run.id">
        <RouterLink
          :to="`/runs/${run.id}`"
          class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5 transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus-ring"
        >
          <div class="flex min-w-0 flex-1 flex-col gap-1">
            <Text class="truncate font-semibold">{{ runTitle(run) }}</Text>
            <div class="flex flex-wrap items-center gap-2">
              <RunStatusChip :status="run.status" />
              <Text muted size="xs">{{ routineKindLabel[run.kind] }}</Text>
              <Text v-if="run.kind === 'release' && flowLabel(run.flow)" muted size="xs">
                {{ flowLabel(run.flow) }}
              </Text>
            </div>
          </div>
          <Text muted size="xs" class="shrink-0">{{ formatDateTime(run.updatedAt) }}</Text>
        </RouterLink>
      </li>
    </ul>
  </section>
</template>
