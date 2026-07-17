<script setup lang="ts">
import { computed, onMounted, ref, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { errorMessage } from '@shared/api/client'
import { setBreadcrumbs } from '@shared/composables/useBreadcrumbs'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import { useProvidersStore } from '@modules/providers/store'
import { isTerminal } from '@modules/reviews/format'
import MergeRequestList from '@modules/reviews/components/MergeRequestList.vue'
import ReviewList from '@modules/reviews/components/ReviewList.vue'

const route = useRoute()
const router = useRouter()
const repoId = (route.params as { id: string }).id

const repos = useReposStore()
const reviews = useReviewsStore()
const providers = useProvidersStore()

const creatingIid = ref<number | null>(null)

const repo = computed(() => repos.items.find((r) => r.id === repoId) ?? null)

// Preselected provider for launching a review: the repo's assigned provider if
// set, otherwise the global default provider, otherwise none.
const defaultProviderId = computed(
  () => repo.value?.providerId || providers.items.find((p) => p.isDefault)?.id || '',
)
const mrs = computed(() => reviews.mergeRequestsFor(repoId))
const repoReviews = computed(() => reviews.reviewsFor(repoId))
const archivedReviews = computed(() => reviews.archivedReviewsFor(repoId))
// Stale-while-revalidate: only show a spinner when nothing is cached yet.
const mrsLoading = computed(() => reviews.mrsLoading && mrs.value.length === 0)
const reviewsLoading = computed(() => reviews.listLoading && repoReviews.value.length === 0)
const archivedLoading = computed(
  () => reviews.archivedLoading && archivedReviews.value.length === 0,
)

const showArchived = ref(false)

function toggleArchived() {
  showArchived.value = !showArchived.value
  if (showArchived.value) reviews.fetchArchivedReviews(repoId)
}

watchEffect(() => {
  setBreadcrumbs([
    { label: 'Repositories', to: '/repos' },
    { label: repo.value?.name ?? 'Repository' },
  ])
})

onMounted(async () => {
  if (repos.items.length === 0) await repos.fetchAll()
  if (providers.items.length === 0) providers.fetchAll()
  reviews.fetchMergeRequests(repoId)
  reviews.fetchReviews(repoId)
})

async function startReview(iid: number, mode: string, providerId: string, model: string) {
  // If other reviews are already in progress, let the user choose to watch this
  // one now (it runs in parallel, up to the server's bound) or queue it and keep
  // working. Both paths create the review; the choice only decides navigation.
  let watchNow = true
  if (repoReviews.value.some((r) => !isTerminal(r.status))) {
    watchNow = await confirm({
      title: 'Reviews already running',
      message:
        'Other reviews are still running. Launch this one now and watch it (it runs in parallel), or queue it and keep working — it starts when a slot frees up.',
      confirmText: 'Launch and watch',
      cancelText: 'Queue and stay',
    })
  }

  creatingIid.value = iid
  try {
    const rv = await reviews.create(repoId, iid, mode, providerId, model)
    if (watchNow) router.push(`/reviews/${rv.id}`)
    else toast.success('Review queued — it will run when a slot is free.')
  } catch (e) {
    reviews.mrsError = errorMessage(e)
  } finally {
    creatingIid.value = null
  }
}
</script>

<template>
  <div>
    <PageHeader :title="repo?.name ?? 'Repository'" />

    <section class="mb-10">
      <h2 class="section-title mb-3 flex items-center gap-2">
        <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
        Open merge requests
      </h2>
      <MergeRequestList
        :items="mrs"
        :loading="mrsLoading"
        :error="reviews.mrsError"
        :busy-iid="creatingIid"
        :providers="providers.items"
        :default-provider-id="defaultProviderId"
        @review="startReview"
      />
    </section>

    <section>
      <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
        <h2 class="section-title flex items-center gap-2">
          <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
          Reviews
        </h2>
        <button class="btn-ghost text-xs" @click="toggleArchived">
          <span
            :class="showArchived ? 'i-lucide-eye-off' : 'i-lucide-archive'"
            class="text-sm"
            aria-hidden="true"
          />
          {{ showArchived ? 'Hide archived' : 'Show archived' }}
        </button>
      </div>
      <ReviewList :items="repoReviews" :loading="reviewsLoading" :error="reviews.listError" />

      <template v-if="showArchived">
        <h3 class="section-title text-muted mt-6 mb-3 flex items-center gap-2">
          <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
          Archived
        </h3>
        <ReviewList
          :items="archivedReviews"
          :loading="archivedLoading"
          :error="reviews.archivedError"
        />
      </template>
    </section>
  </div>
</template>
