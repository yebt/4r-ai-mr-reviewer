<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'
import { formatDateTime, recommendationClass, recommendationLabel, shortId } from '@modules/reviews/format'

const repos = useReposStore()
const reviews = useReviewsStore()
const loading = ref(false)

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
    <PageHeader label="Activity" title="Reviews">
      <template #actions>
        <span class="label-mono">all repositories</span>
      </template>
    </PageHeader>

    <p v-if="loading" class="py-3 text-sm text-muted">Loading…</p>
    <EmptyState
      v-else-if="items.length === 0"
      icon="i-lucide-list-checks"
      title="No reviews yet"
      hint="Open a repository and start a review on a merge request."
    />

    <ul v-else class="border-t border-line/50">
      <li v-for="rv in items" :key="rv.id" class="row justify-between">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
            <RouterLink :to="`/reviews/${rv.id}`" class="text-sm text-ink hover:text-accent">
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
        <div v-if="rv.status === 'done'" class="shrink-0 text-right">
          <div class="text-sm" :class="recommendationClass[rv.recommendation]">{{ recommendationLabel(rv.recommendation) }}</div>
          <div class="label-mono mt-0.5">score {{ rv.score }}</div>
        </div>
      </li>
    </ul>
  </div>
</template>
