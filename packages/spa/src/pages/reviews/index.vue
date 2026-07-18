<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'
import {
  formatDateTime,
  isTerminal,
  recommendationClass,
  recommendationLabel,
  shortId,
} from '@modules/reviews/format'
import { toast } from '@shared/composables/useToast'
import { errorMessage } from '@shared/api/client'

const repos = useReposStore()
const reviews = useReviewsStore()
const loading = ref(false)
// Ids with an archive request in flight, so concurrent clicks on different rows
// each keep their own button disabled independently.
const archivingIds = ref<string[]>([])

async function archiveReview(id: string) {
  archivingIds.value = [...archivingIds.value, id]
  try {
    await reviews.archive(id)
    toast.success('Review archived')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archivingIds.value = archivingIds.value.filter((x) => x !== id)
  }
}

const repoName = (id: string) => repos.items.find((r) => r.id === id)?.name ?? 'repo'

onMounted(async () => {
  loading.value = reviews.allReviews.length === 0
  if (repos.items.length === 0) await repos.fetchAll()
  await Promise.all(repos.items.map((r) => reviews.fetchReviews(r.id)))
  loading.value = false
})

const items = computed(() => reviews.allReviews)
</script>

<template>
  <div>
    <PageHeader title="Reviews" />

    <p v-if="loading" class="text-muted py-3 text-sm">Loading…</p>
    <EmptyState
      v-else-if="items.length === 0"
      icon="i-lucide-list-checks"
      title="No reviews yet"
      hint="Open a repository and start a review on a merge request."
    />

    <ul v-else class="border-line/50 border-t">
      <li v-for="rv in items" :key="rv.id" class="row justify-between">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
            <RouterLink :to="`/reviews/${rv.id}`" class="text-ink hover:text-accent text-sm">
              {{ repoName(rv.repoId) }} · !{{ rv.mrIid }}
            </RouterLink>
            <ReviewStatusChip :status="rv.status" />
          </div>
          <div class="label-mono mt-0.5 flex flex-wrap gap-x-2">
            <span class="text-muted/70">#{{ shortId(rv.id) }}</span>
            <span>{{ rv.contextMode }}</span>
            <span v-if="rv.createdAt">{{ formatDateTime(rv.createdAt) }}</span>
          </div>
        </div>
        <div class="flex shrink-0 items-center gap-3">
          <div v-if="rv.status === 'done'" class="text-right">
            <div class="text-sm" :class="recommendationClass[rv.recommendation]">
              {{ recommendationLabel(rv.recommendation) }}
            </div>
            <div class="label-mono mt-0.5">score {{ rv.score }}</div>
          </div>
          <button
            class="btn-ghost text-xs"
            :disabled="archivingIds.includes(rv.id) || !isTerminal(rv.status)"
            :aria-label="`Archive review !${rv.mrIid}`"
            :title="isTerminal(rv.status) ? 'Archive' : 'Cannot archive a running review'"
            @click="archiveReview(rv.id)"
          >
            <span class="i-lucide-archive text-sm" aria-hidden="true" />
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
