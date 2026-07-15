<script setup lang="ts">
import { computed, ref } from 'vue'
import { useMediaQuery } from '@vueuse/core'
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

// The severity accent border is ALWAYS present. Blocking overrides the color
// (flame); then high/medium/low map to danger/warn/muted. Published state uses a
// separate channel (chip, tab check, faint tint).
//
// The accent edge moves to the TOP on phone and back to the LEFT on desktop. Both
// edges are colored with literal classes so UnoCSS JIT emits them; the static
// width classes on the card pick which edge is thick. On desktop the top color is
// reset to `line` so the box keeps its hairline top border exactly as before.
const borderClass = computed<string>(() => {
  if (props.finding.blocking) return 'border-t-flame sm:border-t-line sm:border-l-flame'
  switch (props.finding.severity) {
    case 'high':
      return 'border-t-danger sm:border-t-line sm:border-l-danger'
    case 'medium':
      return 'border-t-warn sm:border-t-line sm:border-l-warn'
    default:
      return 'border-t-muted sm:border-t-line sm:border-l-muted'
  }
})

// Phone-only progressive disclosure: on phone a card shows only its header, file
// badge and issue headline until expanded; on desktop (`!isPhone`) everything is
// always rendered so the desktop layout is unchanged.
const isPhone = useMediaQuery('(max-width: 639px)')
const expanded = ref(false)
const showDeep = computed(() => expanded.value || !isPhone.value)

// Shared humanize-tab + publish logic (identical to the classic card).
const { store, tabs, selectedTab, humanizing, publishedTabIdx, shown, publishing, humanize, publish } =
  useFindingCard(props)
</script>

<template>
  <!-- Triage card: a boxed, well-separated variant of FindingCard. There is no
       `card` shortcut in this borderless theme, so the box is built inline from
       the palette idioms (border-line + bg-surface). Published state uses a
       faint bg-ok/5 tint (never opacity) so the actions stay fully legible. -->
  <div
    class="border-line flex flex-col gap-4 border border-t-2 p-4 sm:border-t sm:border-l-2"
    :class="[borderClass, finding.published ? 'bg-ok/5' : 'bg-surface']"
  >
    <!-- 1. Header: dimension/severity/blocking/published chips + selection box. -->
    <div class="flex flex-wrap items-start justify-between gap-2">
      <div class="flex flex-wrap items-center gap-2">
        <span class="chip text-muted">{{ dimensionLabel[finding.dimension] }}</span>
        <span class="chip" :class="severityClass[finding.severity]">{{ finding.severity }}</span>
        <span v-if="finding.blocking" class="chip text-flame">
          <span class="i-lucide-flame text-sm" aria-hidden="true" />
          blocking
        </span>
        <span v-if="finding.published" class="chip text-ok">
          <span class="i-lucide-check text-sm" aria-hidden="true" />
          published
        </span>
      </div>
      <!-- Selection checkbox stays visible in the collapsed header; the chevron
           (phone only) expands/collapses the deep content. -->
      <div class="flex shrink-0 items-start gap-2">
        <input
          v-if="selectable"
          type="checkbox"
          class="accent-accent mt-0.5 shrink-0"
          :checked="selected"
          :aria-label="`Select finding ${finding.index + 1}`"
          @change="$emit('toggle', finding.index)"
        />
        <button
          type="button"
          class="btn-ghost -mr-1 p-1 sm:hidden"
          :aria-expanded="expanded"
          :aria-label="expanded ? 'Collapse finding' : 'Expand finding'"
          @click="expanded = !expanded"
        >
          <span
            class="i-lucide-chevron-down text-base transition-transform"
            :class="expanded ? 'rotate-180' : ''"
            aria-hidden="true"
          />
        </button>
      </div>
    </div>

    <!-- 2. File badge on its own line — the location stands out as a mono pill.
         Shown on every tab: the file/line is the finding's location and does not
         change when humanized. Renders even without a line; the line is a
         separate accent pill. -->
    <div v-if="finding.file" class="flex min-w-0 flex-wrap items-center gap-2">
      <span
        class="border-line bg-surface text-muted inline-flex min-w-0 items-center gap-1.5 border px-2 py-1 font-mono text-xs"
      >
        <span class="i-lucide-file-code shrink-0 text-sm" aria-hidden="true" />
        <span class="truncate">{{ finding.file }}</span>
      </span>
      <span
        v-if="finding.line > 0"
        class="border-accent/40 bg-accent/10 text-accent inline-flex shrink-0 items-center border px-1.5 py-0.5 font-mono text-xs"
      >
        :{{ finding.line }}
      </span>
    </div>

    <!-- 3. Issue headline — larger/heavier than the body prose. -->
    <p class="text-ink text-base font-medium whitespace-pre-wrap">{{ shown.issue }}</p>

    <!-- Phone-only collapsed hint: signals there is more to read below. -->
    <button
      v-if="isPhone && !expanded && (shown.why || shown.fix)"
      type="button"
      class="text-muted/70 hover:text-ink -mt-1 flex items-center gap-1 text-xs"
      @click="expanded = true"
    >
      <span class="i-lucide-chevron-down text-sm" aria-hidden="true" />
      Show details
    </button>

    <!-- 4. Why — labelled block with a subtle left rule for separation. -->
    <div v-if="showDeep && shown.why" class="border-line/60 border-l pl-3">
      <div class="label-mono mb-1">Why</div>
      <p class="text-muted text-sm whitespace-pre-wrap">{{ shown.why }}</p>
    </div>

    <!-- 5. Suggested fix — same labelled-block treatment. -->
    <div v-if="showDeep && shown.fix" class="border-line/60 border-l pl-3">
      <div class="label-mono mb-1">Suggested fix</div>
      <p class="text-muted text-sm whitespace-pre-wrap">{{ shown.fix }}</p>
    </div>

    <!-- Controls footer: tabs + actions, separated from content by a hairline. -->
    <div v-if="showDeep" class="border-line/50 flex flex-col gap-3 border-t pt-3">
      <!-- 6. Tabs: Original + one per humanize run. -->
      <div v-if="tabs.length" class="flex flex-wrap items-center gap-1">
        <button
          class="inline-flex items-center gap-1 border px-2 py-1.5 text-xs transition-colors sm:py-0.5"
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
          class="inline-flex items-center gap-1 border px-2 py-1.5 text-xs transition-colors sm:py-0.5"
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

      <!-- 7. Actions: Humanize + Publish. Stacked and full-width on phone; the
           original inline row is restored at sm:+. -->
      <div class="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
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
        <button class="btn-ghost w-full text-xs sm:w-auto" :disabled="publishing" @click="publish">
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
