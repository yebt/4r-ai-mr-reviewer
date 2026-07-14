<script setup lang="ts">
import { computed, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { Review } from '@shared/api/types'
import { useReviewsStore } from '@modules/reviews/store'
import { ORIGINAL } from '@modules/reviews/humanize-overrides'
import { recommendationClass, recommendationLabel } from '@modules/reviews/format'

// profileId is the globally selected humanization profile ([id].vue). Empty
// string means no ready profile is selected, which disables Humanize here.
const props = defineProps<{ review: Review; profileId: string }>()

const store = useReviewsStore()

const reviewId = computed(() => props.review.id)
const tabs = computed(() => store.summaryTabs(reviewId.value))
const selectedTab = computed(() => store.selectedSummaryTab(reviewId.value))
const humanizing = computed(() => store.summaryHumanizing(reviewId.value))

// Text shown for the active tab: Original is the generated summary; a humanize
// tab is that run's rewritten summary.
const shownSummary = computed(() => {
  if (selectedTab.value === ORIGINAL) return props.review.summary
  return tabs.value[selectedTab.value]?.summary ?? ''
})

const publishing = ref(false)

async function humanize() {
  if (!props.profileId || humanizing.value) return
  try {
    await store.humanizeSummary(reviewId.value, props.profileId)
  } catch (e) {
    toast.error(errorMessage(e))
  }
}

async function publish() {
  if (publishing.value) return
  publishing.value = true
  try {
    const tab = selectedTab.value
    // Summary-only publish: no findings, summary note included. On a humanize
    // tab, override with its rewritten text (unless empty → post generated).
    const summary = tab === ORIGINAL ? '' : (tabs.value[tab]?.summary ?? '')
    await store.publish(reviewId.value, {
      indices: [],
      includeSummary: true,
      ...(summary ? { summaryOverride: summary } : {}),
    })
    toast.success('Summary published')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    publishing.value = false
  }
}
</script>

<template>
  <section class="border-line/50 border-b pb-6">
    <div class="flex items-end justify-between gap-6">
      <div>
        <div class="label-mono">Recommendation</div>
        <div
          class="mt-1 text-2xl font-semibold tracking-tight"
          :class="recommendationClass[review.recommendation]"
        >
          {{ recommendationLabel(review.recommendation) }}
        </div>
      </div>
      <div class="text-right">
        <div class="label-mono">Score</div>
        <div class="text-ink mt-1 font-mono text-2xl font-semibold">
          {{ review.score }}<span class="text-muted text-base">/100</span>
        </div>
      </div>
    </div>

    <!-- Tabs: Original + one per humanize run. -->
    <div v-if="tabs.length" class="mt-4 flex flex-wrap items-center gap-1">
      <button
        class="border px-2 py-1 text-xs transition-colors"
        :class="
          selectedTab === ORIGINAL
            ? 'border-accent text-ink bg-accent/10'
            : 'border-line text-muted hover:text-ink'
        "
        :aria-pressed="selectedTab === ORIGINAL"
        @click="store.selectSummaryTab(reviewId, ORIGINAL)"
      >
        Original
      </button>
      <button
        v-for="(_, i) in tabs"
        :key="i"
        class="border px-2 py-1 text-xs transition-colors"
        :class="
          selectedTab === i
            ? 'border-accent text-ink bg-accent/10'
            : 'border-line text-muted hover:text-ink'
        "
        :aria-pressed="selectedTab === i"
        @click="store.selectSummaryTab(reviewId, i)"
      >
        V{{ i + 1 }}
      </button>
    </div>

    <p
      v-if="shownSummary"
      class="text-muted mt-4 max-w-2xl text-sm leading-relaxed whitespace-pre-wrap"
    >
      {{ shownSummary }}
    </p>
    <p v-else class="text-muted/60 mt-4 text-sm italic">No summary.</p>

    <div class="label-mono mt-4 flex flex-wrap gap-x-6 gap-y-1">
      <span>mode {{ review.contextMode }}</span>
      <span>{{ review.findings.length }} findings</span>
      <span>tokens {{ review.inputTokens }} in / {{ review.outputTokens }} out</span>
    </div>

    <div class="mt-4 flex flex-wrap items-center gap-2">
      <button
        class="btn-line text-xs"
        :disabled="!profileId || humanizing"
        :title="profileId ? undefined : 'Select a ready profile first'"
        @click="humanize"
      >
        <span
          v-if="humanizing"
          class="i-lucide-loader-circle animate-spin text-sm"
          aria-hidden="true"
        />
        <span v-else class="i-lucide-feather text-sm" aria-hidden="true" />
        Humanize
      </button>
      <button class="btn-accent text-xs" :disabled="publishing" @click="publish">
        <span
          v-if="publishing"
          class="i-lucide-loader-circle animate-spin text-sm"
          aria-hidden="true"
        />
        <span v-else class="i-lucide-send text-sm" aria-hidden="true" />
        {{ review.summaryPublished ? 'Publish summary again' : 'Publish summary' }}
      </button>
    </div>
  </section>
</template>
