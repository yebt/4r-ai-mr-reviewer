<script setup lang="ts">
import { computed, onUnmounted, ref, watch, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useIntervalFn } from '@vueuse/core'
import { errorMessage } from '@shared/api/client'
import { setBreadcrumbs } from '@shared/composables/useBreadcrumbs'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import { useReviewsStore } from '@modules/reviews/store'
import { useReposStore } from '@modules/repos/store'
import { isTerminal } from '@modules/reviews/format'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'
import ReviewSummary from '@modules/reviews/components/ReviewSummary.vue'
import FindingRow from '@modules/reviews/components/FindingRow.vue'

const route = useRoute()
const router = useRouter()
const store = useReviewsStore()
const repos = useReposStore()

const reviewId = computed(() => (route.params as { id: string }).id)
const review = computed(() => store.current)

const crumbs = computed(() => {
  const items: { label: string; to?: string }[] = [{ label: 'Repositories', to: '/repos' }]
  const rv = review.value
  if (rv) {
    const repo = repos.items.find((r) => r.id === rv.repoId)
    items.push({ label: repo?.name ?? 'Repository', to: `/repos/${rv.repoId}` })
    items.push({ label: `Review !${rv.mrIid}` })
  }
  return items
})

watchEffect(() => setBreadcrumbs(crumbs.value))

const selected = ref<number[]>([])
const publishing = ref(false)
const publishError = ref<string | null>(null)

const unpublished = computed(() => review.value?.findings.filter((f) => !f.published) ?? [])

// Multi-pass progress: maps the backend phase to a labelled step (of 4).
const phaseMap: Record<string, { label: string; step: number }> = {
  risk: { label: 'Risk', step: 1 },
  readability: { label: 'Readability', step: 2 },
  reliability: { label: 'Reliability', step: 3 },
  resilience: { label: 'Resilience', step: 4 },
}
const phase = computed(() => (review.value?.phase ? (phaseMap[review.value.phase] ?? null) : null))

const { pause, resume } = useIntervalFn(
  async () => {
    await store.refresh(reviewId.value)
    if (review.value && isTerminal(review.value.status)) pause()
  },
  2500,
  { immediate: false },
)

watch(
  reviewId,
  async (id) => {
    pause()
    selected.value = []
    if (repos.items.length === 0) repos.fetchAll()
    await store.load(id)
    if (review.value && !isTerminal(review.value.status)) resume()
  },
  { immediate: true },
)
onUnmounted(pause)

function toggle(index: number) {
  const at = selected.value.indexOf(index)
  if (at >= 0) selected.value.splice(at, 1)
  else selected.value.push(index)
}

async function publish(payload: { all?: boolean; indices?: number[] }) {
  publishing.value = true
  publishError.value = null
  try {
    await store.publish(reviewId.value, payload)
    selected.value = []
    toast.success('Findings published')
  } catch (e) {
    publishError.value = errorMessage(e)
  } finally {
    publishing.value = false
  }
}

async function retry() {
  try {
    const rv = await store.retry(reviewId.value)
    router.push(`/reviews/${rv.id}`)
  } catch (e) {
    store.currentError = errorMessage(e)
  }
}
</script>

<template>
  <div>
    <PageHeader :title="review ? `Merge request !${review.mrIid}` : 'Review'">
      <template #actions>
        <ReviewStatusChip v-if="review" :status="review.status" />
      </template>
    </PageHeader>

    <p v-if="store.currentLoading && !review" class="text-muted text-sm">Loading…</p>
    <p v-else-if="store.currentError" class="text-danger text-sm">{{ store.currentError }}</p>

    <template v-else-if="review">
      <div v-if="review.status === 'pending' || review.status === 'running'">
        <div class="text-muted flex items-center gap-2 text-sm">
          <span class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
          <template v-if="phase">
            Reviewing {{ phase.label }}
            <span class="text-muted/70 font-mono">({{ phase.step }}/4)</span>…
          </template>
          <template v-else>Review {{ review.status }}… updates automatically.</template>
        </div>
        <div v-if="phase" class="bg-line/40 mt-3 h-1 w-full max-w-xs">
          <div
            class="bg-accent h-full transition-all"
            :style="{ width: `${(phase.step / 4) * 100}%` }"
          />
        </div>
      </div>

      <div v-else-if="review.status === 'error'" class="flex flex-col items-start gap-3">
        <p class="text-danger text-sm">{{ review.error || 'Review failed.' }}</p>
        <button class="btn-line" @click="retry">
          <span class="i-lucide-refresh-cw text-sm" aria-hidden="true" />
          Retry (clones the review)
        </button>
      </div>

      <template v-else>
        <ReviewSummary :review="review" />

        <section class="mt-6">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
            <h2 class="section-title flex items-center gap-2">
              <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
              Findings
            </h2>
            <div v-if="review.findings.length" class="flex items-center gap-2">
              <button
                class="btn-ghost text-xs"
                :disabled="publishing || selected.length === 0"
                @click="publish({ indices: selected })"
              >
                Publish selected ({{ selected.length }})
              </button>
              <button
                class="btn-accent text-xs"
                :disabled="publishing || unpublished.length === 0"
                @click="publish({ all: true })"
              >
                <span
                  v-if="publishing"
                  class="i-lucide-loader-circle animate-spin"
                  aria-hidden="true"
                />
                Comment all
              </button>
            </div>
          </div>

          <p v-if="publishError" class="text-danger mb-3 text-sm">{{ publishError }}</p>

          <p v-if="review.findings.length === 0" class="text-muted text-sm">
            No findings — clean review.
          </p>
          <div v-else class="border-line/50 border-t">
            <FindingRow
              v-for="f in review.findings"
              :key="f.index"
              :finding="f"
              :selectable="!f.published"
              :selected="selected.includes(f.index)"
              @toggle="toggle"
            />
          </div>
        </section>
      </template>
    </template>
  </div>
</template>
