<script setup lang="ts">
/**
 * "Check permissions…" dialog opened from the Flow workspace header's
 * repository settings menu (`src/pages/flow/[repoId].vue`) — a
 * standalone view like `WebhookDialog`, so it owns its own `DialogRoot`
 * directly (controlled via `open`/`repo`/`update:open`), instead of being
 * embedded content inside a parent-owned `Dialog` like `RepoForm`.
 *
 * Runs `GET /repos/{id}/preflight` (a live GitLab read — no writes) as soon
 * as the dialog opens for a repo, and again on every reopen/repo switch, so
 * it never shows a previous repo's stale result. Ported behavior-wise from
 * the old app's `RepoSettingsModal.vue` "API scope" section +
 * `PreflightReport.vue`, using this app's `Badge`/`Alert` atoms instead of
 * bespoke chip classes — the ok/fail/unknown → chip mapping and summary
 * counts live in `../preflight.ts` so they stay unit-testable.
 */
import { computed, ref, watch } from 'vue'
import { DialogClose, DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { Alert, Badge, Button, Icon, Spinner, Text } from '@shared/ui/design-system'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { restoreFocusOnClose } from '@shared/composables/focusAfterRemoval'
import { preflightRepo } from '../api'
import { checkStatusUi, summarizePreflightChecks } from '../preflight'
import type { Preflight, Repo } from '../types'

const props = defineProps<{
  open: boolean
  repo: Repo | null
  /** Where focus goes on close (the menu item that opened this unmounts). */
  restoreFocus?: () => HTMLElement | null | undefined
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

// Initial focus goes to Close: the dialog opens while the check is still
// loading, so no other control exists yet and Reka would otherwise park focus
// on the (non-interactive) dialog container.
const closeButton = ref<{ $el: HTMLElement } | null>(null)
function focusClose(event: Event) {
  event.preventDefault()
  closeButton.value?.$el.focus({ preventScroll: true })
}

const preflight = ref<Preflight | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

const summary = computed(() => (preflight.value ? summarizePreflightChecks(preflight.value.checks) : null))

async function runPreflight() {
  if (!props.repo) return
  loading.value = true
  error.value = null
  try {
    preflight.value = await preflightRepo(props.repo.id)
  } catch (err) {
    error.value = resolveErrorMessage(err, 'Failed to check permissions')
    preflight.value = null
  } finally {
    loading.value = false
  }
}

// Re-run whenever the dialog opens (including reopening on the same repo,
// so "Retry" isn't the only way back after a transient GitLab failure) or
// targets a different repo. Closing clears the previous result so a
// reopen never flashes stale data before the new fetch resolves.
// Watch each value as its own source: a getter returning a fresh array would
// re-fire whenever the repo/MR list refetches (e.g. on window focus) and hands
// down a new object with the same id, wiping the dialog's in-progress state.
watch(
  [() => props.open, () => props.repo?.id],
  ([open]) => {
    preflight.value = null
    error.value = null
    if (open && props.repo) void runPreflight()
  },
  { immediate: true },
)
</script>

<template>
  <DialogRoot :open="open" @update:open="(value) => emit('update:open', value)">
    <DialogPortal>
      <DialogOverlay class="overlay z-20" />
      <DialogContent
        @open-auto-focus="focusClose"
        @close-auto-focus="(event: Event) => restoreFocusOnClose(event, props.restoreFocus)"
        class="fixed left-1/2 top-1/2 z-30 max-h-[85vh] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
      >
        <DialogTitle class="text-md font-semibold">Check permissions</DialogTitle>
        <DialogDescription class="mt-1 text-sm text-text-muted">
          {{
            repo
              ? `Which routine actions "${repo.name}"'s token and access level allow.`
              : 'Check which routine actions this repository allows.'
          }}
        </DialogDescription>

        <div v-if="repo" class="mt-4 flex flex-col gap-4">
          <div v-if="loading" class="flex items-center gap-2 py-4" data-testid="preflight-loading">
            <Spinner size="sm" />
            <Text size="sm" muted>Checking permissions…</Text>
          </div>

          <Alert v-else-if="error" status="danger">
            <p>{{ error }}</p>
            <Button variant="outline" size="sm" class="mt-2" @click="runPreflight">Retry</Button>
          </Alert>

          <template v-else-if="preflight">
            <div class="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-line-subtle pb-3">
              <span class="inline-flex items-center gap-1.5 text-sm text-text">
                <Icon name="shield" size="sm" class="text-text-muted" />
                {{ preflight.accessLevelName }}
              </span>
              <span class="inline-flex items-center gap-1.5 text-sm text-text-muted">
                <Icon name="git-branch" size="sm" />
                {{ preflight.defaultBranch }}
              </span>
              <div class="flex flex-wrap items-center gap-1.5">
                <template v-if="preflight.scopesKnown">
                  <Badge v-for="scope in preflight.tokenScopes" :key="scope" status="info">{{ scope }}</Badge>
                </template>
                <Text v-else size="sm" muted class="italic">scopes could not be read</Text>
              </div>
            </div>

            <Text v-if="summary" size="sm" muted>
              {{ summary.ok }} allowed · {{ summary.fail }} blocked · {{ summary.unknown }} unknown
            </Text>

            <ul class="flex flex-col gap-2.5">
              <li v-for="check in preflight.checks" :key="check.capability" class="flex items-start gap-2.5">
                <Badge :status="checkStatusUi[check.status].badgeStatus" class="mt-0.5 shrink-0">
                  <Icon :name="checkStatusUi[check.status].icon" size="xs" />
                  <span class="sr-only">{{ checkStatusUi[check.status].srLabel }}</span>
                </Badge>
                <div class="min-w-0 flex-1">
                  <Text size="sm">{{ check.label }}</Text>
                  <Text v-if="check.detail" size="sm" muted class="mt-0.5 block">{{ check.detail }}</Text>
                </div>
              </li>
            </ul>

            <Button variant="outline" size="sm" class="self-start" @click="runPreflight">Retry</Button>
          </template>
        </div>

        <div class="mt-5 flex justify-end">
          <DialogClose as-child>
            <Button ref="closeButton" variant="outline" size="sm">Close</Button>
          </DialogClose>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
