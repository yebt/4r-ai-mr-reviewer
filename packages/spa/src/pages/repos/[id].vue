<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { errorMessage } from '@shared/api/client'
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

const mode = ref<'fast' | 'deep'>('fast')
const creatingIid = ref<number | null>(null)

const repo = computed(() => repos.items.find((r) => r.id === repoId) ?? null)

onMounted(async () => {
  if (repos.items.length === 0) await repos.fetchAll()
  reviews.fetchMergeRequests(repoId)
  reviews.fetchReviews(repoId)
})

async function startReview(iid: number) {
  creatingIid.value = iid
  try {
    const rv = await reviews.create(repoId, iid, mode.value)
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
    <PageHeader label="Repository" :title="repo?.name ?? 'Repository'">
      <template #actions>
        <span class="label-mono">{{ repo?.url }}</span>
      </template>
    </PageHeader>

    <section class="mb-10">
      <div class="mb-3 flex items-center justify-between gap-4">
        <h2 class="section-title flex items-center gap-2">
          <span class="inline-block h-3.5 w-0.5 bg-accent" aria-hidden="true" />
          Open merge requests
        </h2>
        <label class="flex items-center gap-2 label-mono">
          context
          <select v-model="mode" class="border-b border-line bg-transparent py-1 text-xs text-ink outline-none focus:border-accent px-3">
            <option value="fast">fast</option>
            <option value="deep">deep</option>
          </select>
        </label>
      </div>
      <MergeRequestList
        :items="reviews.mrs"
        :loading="reviews.mrsLoading"
        :error="reviews.mrsError"
        :busy-iid="creatingIid"
        @review="startReview"
      />
    </section>

    <section>
      <h2 class="section-title mb-3 flex items-center gap-2">
        <span class="inline-block h-3.5 w-0.5 bg-accent" aria-hidden="true" />
        Reviews
      </h2>
      <ReviewList :items="reviews.list" :loading="reviews.listLoading" :error="reviews.listError" />
    </section>
  </div>
</template>
