import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { useRoutinesStore } from '@modules/routines/store'
import { isRunActive, runTitle } from '@modules/routines/format'
import { useReviewsStore } from '@modules/reviews/store'
import { isTerminal } from '@modules/reviews/format'

// One in-flight operation the status sheet tracks. `kind` drives the row icon;
// `to` is the route the row links to; `status` feeds the status chip/spinner.
export interface ActiveOp {
  kind: 'action' | 'review'
  id: string
  title: string
  status: string
  to: string
}

// The activity store aggregates every in-flight operation (active routine runs +
// non-terminal reviews) into one live list so the StatusBottomSheet can track
// them from anywhere in the app. It owns no data of its own — it reads the
// routines/reviews stores — and drives a paused-when-idle poller that refreshes
// those sources only while something is actually in flight.
export const useActivityStore = defineStore('activity', () => {
  const routines = useRoutinesStore()
  const reviews = useReviewsStore()

  // Active routine runs (pending/running) mapped to sheet rows.
  const activeRuns = computed<ActiveOp[]>(() =>
    routines.recentRuns
      .filter((run) => isRunActive(run.status))
      .map((run) => ({
        kind: 'action',
        id: run.id,
        title: runTitle(run),
        status: run.status,
        to: `/actions/${run.id}`,
      })),
  )

  // Non-terminal reviews (pending/running/awaiting_approval) mapped to sheet
  // rows. Aggregated from whatever reviews the app has already loaded — there is
  // no global active-reviews endpoint, so the poller re-fetches the per-repo
  // review lists we already have cached rather than inventing one.
  const activeReviews = computed<ActiveOp[]>(() =>
    reviews.allReviews
      .filter((rv) => !isTerminal(rv.status))
      .map((rv) => ({
        kind: 'review',
        id: rv.id,
        title: `Review !${rv.mrIid}`,
        status: rv.status,
        to: `/reviews/${rv.id}`,
      })),
  )

  const activeOps = computed<ActiveOp[]>(() => [...activeRuns.value, ...activeReviews.value])
  const count = computed(() => activeOps.value.length)

  // Dismissed hides the sheet until either the user re-opens it or a new op
  // arrives (see the watcher below), matching the Drive-style "come back when
  // something new happens" behaviour.
  const dismissed = ref(false)
  function dismiss() {
    dismissed.value = true
  }
  function reveal() {
    dismissed.value = false
  }

  // Auto-reveal: when the active set grows (a new op appears) clear the dismissed
  // flag so the sheet comes back on its own.
  watch(count, (next, prev) => {
    if (next > prev) dismissed.value = false
  })

  // refresh re-pulls the global sources backing activeOps. Routines expose a
  // real global recent-runs endpoint; reviews do not, so we revalidate each repo
  // that currently has a loaded review list. Best-effort — a failed refresh must
  // not tear the sheet down.
  async function refresh() {
    const jobs: Promise<unknown>[] = [routines.listRecent().catch(() => {})]
    for (const repoId of Object.keys(reviews.reviewsByRepo)) {
      jobs.push(reviews.fetchReviews(repoId).catch(() => {}))
    }
    await Promise.all(jobs)
  }

  // The poller runs on a ~3s cadence but pauses itself the moment no op is
  // active, so it never spins forever: it only stays awake while there is
  // something to watch. `start()`/`stop()` are the lifecycle hooks App.vue calls.
  const { pause, resume: resumePoll, isActive } = useIntervalFn(
    async () => {
      if (count.value === 0) {
        pause()
        return
      }
      await refresh()
    },
    3000,
    { immediate: false },
  )

  function poll() {
    if (!isActive.value && count.value > 0) resumePoll()
  }

  // Any newly-active op (re)starts the poller; it self-pauses when it next sees
  // an empty active set.
  watch(count, () => poll())

  function start() {
    poll()
  }
  function stop() {
    pause()
  }

  return {
    activeOps,
    count,
    dismissed,
    dismiss,
    reveal,
    refresh,
    poll,
    start,
    stop,
  }
})
