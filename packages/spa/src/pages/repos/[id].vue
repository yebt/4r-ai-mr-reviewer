<script setup lang="ts">
import { computed, onMounted, ref, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { errorMessage } from '@shared/api/client'
import { setBreadcrumbs } from '@shared/composables/useBreadcrumbs'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import MergeRequestList from '@modules/reviews/components/MergeRequestList.vue'
import ReviewList from '@modules/reviews/components/ReviewList.vue'

const route = useRoute()
const router = useRouter()
const repoId = (route.params as { id: string }).id

const repos = useReposStore()
const reviews = useReviewsStore()

const creatingIid = ref<number | null>(null)

const repo = computed(() => repos.items.find((r) => r.id === repoId) ?? null)
const mrs = computed(() => reviews.mergeRequestsFor(repoId))
const repoReviews = computed(() => reviews.reviewsFor(repoId))
// Stale-while-revalidate: only show a spinner when nothing is cached yet.
const mrsLoading = computed(() => reviews.mrsLoading && mrs.value.length === 0)
const reviewsLoading = computed(() => reviews.listLoading && repoReviews.value.length === 0)

watchEffect(() => {
  setBreadcrumbs([
    { label: 'Repositories', to: '/repos' },
    { label: repo.value?.name ?? 'Repository' },
  ])
})

onMounted(async () => {
  if (repos.items.length === 0) await repos.fetchAll()
  reviews.fetchMergeRequests(repoId)
  reviews.fetchReviews(repoId)
})

async function startReview(iid: number, mode: string) {
  creatingIid.value = iid
  try {
    const rv = await reviews.create(repoId, iid, mode)
    router.push(`/reviews/${rv.id}`)
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
        @review="startReview"
      />
    </section>

    <section>
      <h2 class="section-title mb-3 flex items-center gap-2">
        <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
        Reviews
      </h2>
      <ReviewList :items="repoReviews" :loading="reviewsLoading" :error="reviews.listError" />
    </section>
  </div>
</template>
