<script setup lang="ts">
import { computed } from 'vue'
import type { Finding } from '@shared/api/types'
import { useFindingCard } from '@modules/reviews/useFindingCard'
import { ORIGINAL } from '@modules/reviews/humanize-overrides'
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

// Left border encodes severity as visual weight and is ALWAYS present. Blocking
// overrides the color (flame); high/medium/low map to danger/warn/muted.
// Published state uses a separate channel (chip, tab check, tint).
const borderClass = computed<string>(() => {
  if (props.finding.blocking) return 'border-l-flame'
  switch (props.finding.severity) {
    case 'high':
      return 'border-l-danger'
    case 'medium':
      return 'border-l-warn'
    default:
      return 'border-l-muted'
  }
})

// Shared humanize-tab + publish logic (identical to the triage card).
const { store, tabs, selectedTab, humanizing, publishedTabIdx, shown, publishing, humanize, publish } =
  useFindingCard(props)
</script>

<template>
  <div
    class="border-line/50 flex gap-3 border-b border-l-2 py-4 pl-3"
    :class="[borderClass, finding.published ? 'bg-ok/5' : '']"
  >
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
          class="inline-flex items-center gap-1 border px-2 py-0.5 text-xs transition-colors"
          :class="
            selectedTab === ORIGINAL
              ? 'border-accent text-ink bg-accent/10'
              : 'border-line text-muted hover:text-ink'
          "
          :aria-pressed="selectedTab === ORIGINAL"
          @click="store.selectFindingTab(reviewId, finding.index, ORIGINAL)"
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
          class="inline-flex items-center gap-1 border px-2 py-0.5 text-xs transition-colors"
          :class="
            selectedTab === i
              ? 'border-accent text-ink bg-accent/10'
              : 'border-line text-muted hover:text-ink'
          "
          :aria-pressed="selectedTab === i"
          @click="store.selectFindingTab(reviewId, finding.index, i)"
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
