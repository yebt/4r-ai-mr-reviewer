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
// In-session marker: which tab was published for the summary (null = none).
const publishedTabIdx = computed(() => store.publishedSummaryTab(reviewId.value))

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
    store.markPublished(reviewId.value, 'summary', tab)
    toast.success('Summary published')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    publishing.value = false
  }
}
</script>

<template>
  <!-- Phone: organized left padding, flush right, and the published accent border
       on TOP. Desktop (sm:+) restores the original flush padding and the LEFT
       published border. Both edges of the ok border are colored with literal
       classes so UnoCSS JIT emits them; static widths pick which edge shows. -->
  <section
    class="border-line/50 border-b px-3 pt-2 pb-3 md:pb-6 md:pl-4 md:pr-0 sm:px-0"
    :class="
      review.summaryPublished
        ? 'border-t-ok border-l-ok border-t-2 border-l-0 bg-ok/5 sm:border-t-0 sm:border-l-2 sm:pl-4'
        : ''
    "
  >
    <div class="flex items-end justify-between gap-4 sm:gap-6">
      <div>
        <div class="label-mono hidden md:flex ">Recommendation</div>
        <div
          class="mt-1 text-2xl font-semibold tracking-tight"
          :class="recommendationClass[review.recommendation]"
        >
          {{ recommendationLabel(review.recommendation) }}
        </div>
      </div>
      <div class="text-right">
        <div class="label-mono hidden md:flex ">Score</div>
        <div class="text-ink mt-1 font-mono text-2xl font-semibold">
          {{ review.score }}<span class="text-muted text-base">/100</span>
        </div>
      </div>
    </div>

    <!-- Tabs: Original + one per humanize run. -->
    <div v-if="tabs.length" class="mt-4 flex flex-wrap items-center gap-1">
      <button
        class="inline-flex items-center gap-1 border px-2 py-1 text-xs transition-colors"
        :class="
          selectedTab === ORIGINAL
            ? 'border-accent text-ink bg-accent/10'
            : 'border-line text-muted hover:text-ink'
        "
        :aria-pressed="selectedTab === ORIGINAL"
        @click="store.selectSummaryTab(reviewId, ORIGINAL)"
      >
        Original
        <span
          v-if="publishedTabIdx === ORIGINAL"
          class="i-lucide-check text-ok text-sm"
          role="img"
          aria-label="published"
          title="published"
        />
      </button>
      <button
        v-for="(_, i) in tabs"
        :key="i"
        class="inline-flex items-center gap-1 border px-2 py-1 text-xs transition-colors"
        :class="
          selectedTab === i
            ? 'border-accent text-ink bg-accent/10'
            : 'border-line text-muted hover:text-ink'
        "
        :aria-pressed="selectedTab === i"
        @click="store.selectSummaryTab(reviewId, i)"
      >
        V{{ i + 1 }}
        <span
          v-if="publishedTabIdx === i"
          class="i-lucide-check text-ok text-sm"
          role="img"
          aria-label="published"
          title="published"
        />
      </button>
    </div>

    <p
      v-if="shownSummary"
      class="text-muted mt-4 max-w-2xl text-sm leading-relaxed whitespace-pre-wrap"
    >
      {{ shownSummary }}
    </p>
    <p v-else class="text-muted/60 mt-4 text-sm italic">No summary.</p>

    <div class="label-mono mt-4 hidden flex-wrap gap-x-6 gap-y-1 sm:flex">
      <span>mode {{ review.contextMode }}</span>
      <span>{{ review.findings.length }} findings</span>
      <span>tokens {{ review.inputTokens }} in / {{ review.outputTokens }} out</span>
    </div>

    <div class="mt-4 flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
      <button
        class="btn-line w-full text-xs sm:w-auto"
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
      <button class="btn-accent w-full text-xs sm:w-auto" :disabled="publishing" @click="publish">
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
