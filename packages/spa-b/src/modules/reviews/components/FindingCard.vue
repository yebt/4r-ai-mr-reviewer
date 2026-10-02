<script setup lang="ts">
/**
 * Presentational card for one review finding — extracted from the review
 * detail page's per-finding `<li>` block (container/presentational split,
 * zero behavior change). Renders the severity/blocking badges, `file:line`,
 * the copy/humanize/publish controls, the `HumanizeTabs` strip, and the
 * active tab's issue/why/fix body.
 *
 * Pure props-in/events-out: all state (the `useReviewHumanize` composable,
 * the page's `pendingPublishTarget` tracker, the copy-icon timer) stays
 * owned by `pages/reviews/[id].vue`. This component only renders what it's
 * given (`finding` plus the page's already-resolved `activeParts`/tab/pending
 * values) and emits user intent — `copy`/`humanize`/`publish`/`select-tab` —
 * for the page to handle with its existing, unchanged handlers.
 */
import { Badge, Button, ConfirmDialog, Icon, Text } from '@shared/ui/design-system'
import { FINDING_SEVERITY_BADGE } from '../findings'
import type { Finding } from '../types'
import HumanizeTabs from './HumanizeTabs.vue'

const props = defineProps<{
  finding: Finding
  /** `review.status === 'done'` — gates the humanize/publish controls. */
  actionsEnabled: boolean
  /** Resolved issue/why/fix to display: the active humanized tab, or the finding's own text. */
  activeParts: { issue: string; why: string; fix: string }
  /** Number of humanized runs for this finding (0 hides the tab strip). */
  tabCount: number
  /** Active tab: `ORIGINAL` (-1) or a 0-based run index. */
  activeTab: number
  isHumanizing: boolean
  isPublishing: boolean
  hasReadyProfile: boolean
  /** Whether the copy button should show its "copied" check icon. */
  copied: boolean
}>()

const emit = defineEmits<{
  copy: []
  humanize: []
  publish: []
  'select-tab': [tab: number]
}>()
</script>

<template>
  <li class="flex min-w-0 flex-col gap-2 rounded-lg border border-line-subtle bg-bg-panel p-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <Badge :status="FINDING_SEVERITY_BADGE[props.finding.severity]">{{ props.finding.severity }}</Badge>
        <Badge v-if="props.finding.blocking" status="danger">Blocking</Badge>
        <Text mono size="sm" muted class="break-words [overflow-wrap:anywhere]"
          >{{ props.finding.file }}:{{ props.finding.line }}</Text
        >
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          :aria-label="props.copied ? 'Finding copied' : 'Copy finding as markdown'"
          @click="emit('copy')"
        >
          <Icon :name="props.copied ? 'check' : 'copy'" size="sm" />
        </Button>
        <Button
          v-if="props.actionsEnabled"
          variant="ghost"
          size="sm"
          :disabled="!props.hasReadyProfile"
          :loading="props.isHumanizing"
          @click="emit('humanize')"
        >
          Humanize
        </Button>
        <Badge v-if="props.actionsEnabled && props.finding.published" status="success">Published</Badge>
        <ConfirmDialog
          v-else-if="props.actionsEnabled"
          title="Publish this finding to the MR?"
          description="Posts a comment to the live merge request. This can't be undone."
          confirm-label="Publish"
          :danger="false"
          :pending="props.isPublishing"
          @confirm="emit('publish')"
        >
          <template #trigger>
            <Button variant="outline" size="sm">Publish</Button>
          </template>
        </ConfirmDialog>
      </div>
    </div>
    <HumanizeTabs
      v-if="props.actionsEnabled && props.tabCount > 0"
      :tab-count="props.tabCount"
      :active="props.activeTab"
      @select="(tab) => emit('select-tab', tab)"
    />
    <Text class="min-w-0 break-words font-medium [overflow-wrap:anywhere]">{{ props.activeParts.issue }}</Text>
    <div v-if="props.activeParts.why" class="min-w-0 border-t border-line-subtle pt-2">
      <Text size="xs" class="font-medium uppercase tracking-wide text-text-muted">Why</Text>
      <Text muted size="sm" class="min-w-0 break-words [overflow-wrap:anywhere]">{{ props.activeParts.why }}</Text>
    </div>
    <div v-if="props.activeParts.fix" class="min-w-0 border-t border-line-subtle pt-2">
      <Text size="xs" class="font-medium uppercase tracking-wide text-text-muted">Suggested fix</Text>
      <Text muted size="sm" class="min-w-0 break-words [overflow-wrap:anywhere]">{{ props.activeParts.fix }}</Text>
    </div>
  </li>
</template>
