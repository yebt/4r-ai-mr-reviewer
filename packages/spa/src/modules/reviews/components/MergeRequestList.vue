<script setup lang="ts">
import type { MergeRequest, Review } from '@shared/api/types'
import ReviewedBadge from '@modules/reviews/components/ReviewedBadge.vue'

defineProps<{
  items: MergeRequest[]
  loading?: boolean
  error?: string | null
  busyIid?: number | null
  // Latest review per MR iid, so a row can show it's already reviewed and link
  // to the verdict instead of the same MR silently appearing in two sections.
  reviewByMr?: Record<number, Review>
  // When true, development-targeting MRs also get a "Release" action so the Flow
  // workspace can launch a dev-flow release straight from the open-MR list.
  showRelease?: boolean
}>()
// The "Review" button just names the MR; the provider/model/mode are chosen in
// the launch modal the consumer opens in response to this event.
const emit = defineEmits<{
  review: [iid: number]
  release: [iid: number]
}>()
</script>

<template>
  <div>
    <p v-if="loading" class="text-muted py-3 text-sm">Loading merge requests…</p>
    <p v-else-if="error" class="text-danger py-3 text-sm">{{ error }}</p>
    <p v-else-if="items.length === 0" class="text-muted py-3 text-sm">No open merge requests.</p>

    <ul v-else class="border-line/50 border-t">
      <li v-for="mr in items" :key="mr.iid" class="row flex-wrap justify-between gap-y-2">
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <span class="text-muted font-mono text-xs">!{{ mr.iid }}</span>
            <a
              :href="mr.webUrl"
              target="_blank"
              rel="noreferrer"
              class="text-ink hover:text-accent truncate text-sm"
            >
              {{ mr.title }}
            </a>
          </div>
          <div class="label-mono mt-0.5">
            {{ mr.sourceBranch }} → {{ mr.targetBranch
            }}<template v-if="mr.author"> · {{ mr.author }}</template>
          </div>
          <ReviewedBadge :review="reviewByMr?.[mr.iid]" />
        </div>

        <div class="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto">
          <button
            class="btn-line text-xs"
            :disabled="busyIid === mr.iid"
            :aria-label="`Review !${mr.iid}`"
            @click="emit('review', mr.iid)"
          >
            <span
              v-if="busyIid === mr.iid"
              class="i-lucide-loader-circle animate-spin"
              aria-hidden="true"
            />
            <span v-else class="i-lucide-scan-search text-sm" aria-hidden="true" />
            Review
          </button>
          <button
            v-if="showRelease && mr.targetBranch === 'development'"
            class="btn-line text-xs"
            :aria-label="`Release !${mr.iid}`"
            @click="emit('release', mr.iid)"
          >
            <span class="i-lucide-rocket text-sm" aria-hidden="true" />
            Release
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
