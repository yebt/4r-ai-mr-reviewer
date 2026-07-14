<script setup lang="ts">
import { computed, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { Finding, FindingHumanized } from '@shared/api/types'
import { useReviewsStore } from '@modules/reviews/store'
import { ORIGINAL, buildFindingBody } from '@modules/reviews/humanize-overrides'
import { dimensionLabel, severityClass } from '@modules/reviews/format'

// profileId is the globally selected profile ([id].vue); empty disables Humanize.
// selectable/selected/toggle drive the existing bulk "Publish selected" flow.
const props = defineProps<{
  finding: Finding
  reviewId: string
  profileId: string
  selectable: boolean
  selected: boolean
}>()
defineEmits<{ toggle: [index: number] }>()

const store = useReviewsStore()

const tabs = computed(() => store.findingTabs(props.reviewId, props.finding.index))
const selectedTab = computed(() => store.selectedFindingTab(props.reviewId, props.finding.index))
const humanizing = computed(() => store.findingHumanizing(props.reviewId, props.finding.index))

// The finding parts shown for the active tab: Original is the generated finding;
// a humanize tab is that run's rewritten issue/why/fix.
const shown = computed<Pick<FindingHumanized, 'issue' | 'why' | 'fix'>>(() => {
  if (selectedTab.value === ORIGINAL) {
    return { issue: props.finding.issue, why: props.finding.why, fix: props.finding.fix }
  }
  const tab = tabs.value[selectedTab.value]
  return { issue: tab?.issue ?? '', why: tab?.why ?? '', fix: tab?.fix ?? '' }
})

const publishing = ref(false)

async function humanize() {
  if (!props.profileId || humanizing.value) return
  try {
    await store.humanizeFinding(props.reviewId, props.profileId, props.finding.index)
  } catch (e) {
    toast.error(errorMessage(e))
  }
}

async function publish() {
  if (publishing.value) return
  publishing.value = true
  try {
    const tab = selectedTab.value
    const humanized = tab === ORIGINAL ? null : tabs.value[tab]
    // Original posts the generated body (no override forwarded). A humanize tab
    // forwards the assembled body, with the restored dimension/severity tag.
    await store.publish(props.reviewId, {
      indices: [props.finding.index],
      includeSummary: false,
      ...(humanized
        ? {
            findingOverrides: [
              { index: props.finding.index, text: buildFindingBody(props.finding, humanized) },
            ],
          }
        : {}),
    })
    toast.success('Finding published')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    publishing.value = false
  }
}
</script>

<template>
  <div class="border-line/50 flex gap-3 border-b py-4">
    <input
      v-if="selectable"
      type="checkbox"
      class="accent-accent mt-1"
      :checked="selected"
      :aria-label="`Select finding ${finding.index + 1}`"
      @change="$emit('toggle', finding.index)"
    />
    <span v-else class="mt-1 w-[13px] shrink-0" aria-hidden="true" />

    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-center gap-2">
        <span class="chip text-muted">{{ dimensionLabel[finding.dimension] }}</span>
        <span class="chip" :class="severityClass[finding.severity]">{{ finding.severity }}</span>
        <span v-if="finding.blocking" class="chip text-flame">blocking</span>
        <span v-if="finding.published" class="chip text-ok">published</span>
      </div>

      <!-- Tabs: Original + one per humanize run. -->
      <div v-if="tabs.length" class="mt-2 flex flex-wrap items-center gap-1">
        <button
          class="border px-2 py-0.5 text-xs transition-colors"
          :class="
            selectedTab === ORIGINAL
              ? 'border-accent text-ink bg-accent/10'
              : 'border-line text-muted hover:text-ink'
          "
          :aria-pressed="selectedTab === ORIGINAL"
          @click="store.selectFindingTab(reviewId, finding.index, ORIGINAL)"
        >
          Original
        </button>
        <button
          v-for="(_, i) in tabs"
          :key="i"
          class="border px-2 py-0.5 text-xs transition-colors"
          :class="
            selectedTab === i
              ? 'border-accent text-ink bg-accent/10'
              : 'border-line text-muted hover:text-ink'
          "
          :aria-pressed="selectedTab === i"
          @click="store.selectFindingTab(reviewId, finding.index, i)"
        >
          V{{ i + 1 }}
        </button>
      </div>

      <p class="text-ink mt-2 text-sm whitespace-pre-wrap">{{ shown.issue }}</p>

      <div
        v-if="finding.file && selectedTab === ORIGINAL"
        class="text-muted mt-1 font-mono text-xs"
      >
        {{ finding.file }}<template v-if="finding.line">:{{ finding.line }}</template>
      </div>

      <p v-if="shown.why" class="text-muted mt-2 text-sm whitespace-pre-wrap">
        <span class="label-mono">why</span> {{ shown.why }}
      </p>
      <p v-if="shown.fix" class="text-muted mt-1 text-sm whitespace-pre-wrap">
        <span class="label-mono">fix</span> {{ shown.fix }}
      </p>

      <div class="mt-3 flex flex-wrap items-center gap-2">
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
        <button class="btn-ghost text-xs" :disabled="publishing" @click="publish">
          <span
            v-if="publishing"
            class="i-lucide-loader-circle animate-spin text-sm"
            aria-hidden="true"
          />
          <span v-else class="i-lucide-send text-sm" aria-hidden="true" />
          {{ finding.published ? 'Publish again' : 'Publish' }}
        </button>
      </div>
    </div>
  </div>
</template>
