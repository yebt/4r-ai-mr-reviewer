<script setup lang="ts">
definePage({ meta: { title: 'Repository' } })
import { computed, onMounted, ref, watchEffect } from 'vue'
import { useTitle } from '@vueuse/core'
import { useRoute, useRouter } from 'vue-router'
import { api, errorMessage } from '@shared/api/client'
import type { Preflight, Review } from '@shared/api/types'
import { setBreadcrumbs } from '@shared/composables/useBreadcrumbs'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import DependencyAlert from '@shared/components/ui/DependencyAlert.vue'
import Tabs, { type TabItem } from '@shared/components/ui/Tabs.vue'
import PreflightReport from '@modules/repos/components/PreflightReport.vue'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import { useProvidersStore } from '@modules/providers/store'
import { isTerminal } from '@modules/reviews/format'
import MergeRequestList from '@modules/reviews/components/MergeRequestList.vue'
import ReviewList from '@modules/reviews/components/ReviewList.vue'
import RoutinesSection from '@modules/routines/components/RoutinesSection.vue'

const route = useRoute()
const router = useRouter()
const repoId = (route.params as { id: string }).id

const repos = useReposStore()
const reviews = useReviewsStore()
const providers = useProvidersStore()

// Local-only tab selection (no need to persist across visits).
const tabs: TabItem[] = [
  { id: 'reviews', label: 'Reviews', icon: 'i-lucide-scan-search' },
  { id: 'routines', label: 'Routines', icon: 'i-lucide-workflow' },
  { id: 'webhook', label: 'Webhook', icon: 'i-lucide-webhook' },
]
// Local-only tab selection, but deep-linkable via ?tab= so an Action detail page
// can jump straight to this repo's Routines tab. Unknown values fall back safely.
const activeTab = ref(
  tabs.some((t) => t.id === route.query.tab) ? String(route.query.tab) : 'reviews',
)

const creatingIid = ref<number | null>(null)
// Ids with an archive/unarchive request in flight. Tracked as a set (not a
// single ref) so concurrent actions on different rows/lists each keep their own
// button disabled without clearing one another.
const archivingIds = ref<string[]>([])

const repo = computed(() => repos.items.find((r) => r.id === repoId) ?? null)

// Tab title: the repository name (e.g. "my-repo - AI Review"), so a Repository
// tab is identifiable at a glance. Falls back to the generic label from
// definePage while the repo list is still loading.
useTitle(computed(() => (repo.value ? `${repo.value.name} - AI Review` : 'Repository - AI Review')))

// Reviews depend on an AI provider. Guard only after the store settles
// (providers.fetchAll flips `loading` synchronously on mount) so it never flashes.
const noProviders = computed(() => !providers.loading && providers.items.length === 0)

// Preselected provider for launching a review: the repo's assigned provider if
// set, otherwise the global default provider, otherwise none.
const defaultProviderId = computed(
  () => repo.value?.providerId || providers.items.find((p) => p.isDefault)?.id || '',
)
const mrs = computed(() => reviews.mergeRequestsFor(repoId))
const repoReviews = computed(() => reviews.reviewsFor(repoId))
// Latest review per MR iid, so the open-MR list can flag which MRs are already
// reviewed and link straight to the verdict — connecting the two sections rather
// than showing the same MR number twice with no relationship.
const reviewByMr = computed<Record<number, Review>>(() => {
  const m: Record<number, Review> = {}
  for (const rv of repoReviews.value) {
    const cur = m[rv.mrIid]
    if (!cur || rv.createdAt > cur.createdAt) m[rv.mrIid] = rv
  }
  return m
})
const archivedReviews = computed(() => reviews.archivedReviewsFor(repoId))
// Stale-while-revalidate: only show a spinner when nothing is cached yet.
const mrsLoading = computed(() => reviews.mrsLoading && mrs.value.length === 0)
const reviewsLoading = computed(() => reviews.listLoading && repoReviews.value.length === 0)
const archivedLoading = computed(
  () => reviews.archivedLoading && archivedReviews.value.length === 0,
)

const showArchived = ref(false)

// Token-scope + project-permission preflight, run on demand. Kept as local state
// (like a launched review) since it is a one-off action tied to this view.
const preflight = ref<Preflight | null>(null)
const preflightLoading = ref(false)
const preflightError = ref<string | null>(null)

async function testApiScope() {
  preflightLoading.value = true
  preflightError.value = null
  try {
    preflight.value = await api.preflightRepo(repoId)
  } catch (e) {
    preflightError.value = errorMessage(e)
    toast.error(errorMessage(e))
  } finally {
    preflightLoading.value = false
  }
}

function toggleArchived() {
  showArchived.value = !showArchived.value
  if (showArchived.value) reviews.fetchArchivedReviews(repoId)
}

watchEffect(() => {
  setBreadcrumbs([
    { label: 'Repositories', to: '/repos' },
    { label: repo.value?.name ?? 'Repository' },
  ])
})

onMounted(async () => {
  if (repos.items.length === 0) await repos.fetchAll()
  if (providers.items.length === 0) providers.fetchAll()
  reviews.fetchMergeRequests(repoId)
  reviews.fetchReviews(repoId)
})

async function startReview(iid: number, mode: string, providerId: string, model: string) {
  // If other reviews are already in progress, let the user choose to watch this
  // one now (it runs in parallel, up to the server's bound) or queue it and keep
  // working. Both paths create the review; the choice only decides navigation.
  let watchNow = true
  if (repoReviews.value.some((r) => !isTerminal(r.status))) {
    watchNow = await confirm({
      title: 'Reviews already running',
      message:
        'Other reviews are still running. Launch this one now and watch it (it runs in parallel), or queue it and keep working — it starts when a slot frees up.',
      confirmText: 'Launch and watch',
      cancelText: 'Queue and stay',
    })
  }

  creatingIid.value = iid
  try {
    const rv = await reviews.create(repoId, iid, mode, providerId, model)
    if (watchNow) router.push(`/reviews/${rv.id}`)
    else toast.success('Review queued — it will run when a slot is free.')
  } catch (e) {
    reviews.mrsError = errorMessage(e)
  } finally {
    creatingIid.value = null
  }
}

async function archiveReview(id: string) {
  archivingIds.value = [...archivingIds.value, id]
  try {
    await reviews.archive(id)
    if (showArchived.value) await reviews.fetchArchivedReviews(repoId)
    toast.success('Review archived')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archivingIds.value = archivingIds.value.filter((x) => x !== id)
  }
}

async function unarchiveReview(id: string) {
  archivingIds.value = [...archivingIds.value, id]
  try {
    await reviews.unarchive(id)
    await reviews.fetchReviews(repoId)
    toast.success('Review unarchived')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archivingIds.value = archivingIds.value.filter((x) => x !== id)
  }
}

async function approveReview(id: string) {
  archivingIds.value = [...archivingIds.value, id]
  try {
    await reviews.approve(id)
    toast.success('Review approved — running now')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archivingIds.value = archivingIds.value.filter((x) => x !== id)
  }
}

async function discardReview(id: string, mrIid: number) {
  const ok = await confirm({
    title: 'Discard review',
    message: `Discard the held review for !${mrIid}? This cannot be undone.`,
    danger: true,
    confirmText: 'Discard',
  })
  if (!ok) return
  archivingIds.value = [...archivingIds.value, id]
  try {
    await reviews.remove(id)
    toast.success('Review discarded')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archivingIds.value = archivingIds.value.filter((x) => x !== id)
  }
}

// --- Webhook settings ---

// Full URL GitLab should POST events to: the browser origin + the server-issued
// webhook path. Empty until the repo has loaded.
const webhookUrl = computed(() =>
  repo.value ? window.location.origin + repo.value.webhookPath : '',
)
const webhookBusy = ref(false)
// The webhook secret is masked by default so it isn't shoulder-surfed; the user
// reveals it only to copy it into GitLab.
const showWebhookSecret = ref(false)

async function rotateWebhook() {
  if (webhookBusy.value) return
  webhookBusy.value = true
  try {
    await repos.rotateWebhookSecret(repoId)
    toast.success('Token rotated — update it in GitLab')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    webhookBusy.value = false
  }
}

async function toggleWebhook(enabled: boolean) {
  if (webhookBusy.value) return
  webhookBusy.value = true
  try {
    // Preserve the current confirmation gate when flipping the enable switch.
    await repos.setWebhook(repoId, enabled, repo.value?.webhookRequireConfirmation ?? false)
    toast.success(enabled ? 'Webhook enabled' : 'Webhook disabled')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    webhookBusy.value = false
  }
}

// toggleRequireConfirmation flips the manual-confirmation gate while keeping the
// webhook enabled: held reviews wait in the Reviews list until the user approves.
async function toggleRequireConfirmation(requireConfirmation: boolean) {
  if (webhookBusy.value) return
  webhookBusy.value = true
  try {
    await repos.setWebhook(repoId, true, requireConfirmation)
    toast.success(
      requireConfirmation ? 'Confirmation required before running' : 'Reviews will run automatically',
    )
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    webhookBusy.value = false
  }
}

// Copy helper for the URL/secret fields. Clipboard access needs a secure context
// (https/localhost), so a failure is surfaced rather than swallowed.
async function copyText(text: string, label: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(`${label} copied`)
  } catch {
    toast.error('Copy failed — clipboard needs a secure context (https/localhost)')
  }
}
</script>

<template>
  <div>
    <PageHeader :title="repo?.name ?? 'Repository'" />

    <Tabs v-model="activeTab" :tabs="tabs" class="mb-8" />

    <!-- Reviews tab: launch reviews on open MRs and browse this repo's reviews. -->
    <div
      v-show="activeTab === 'reviews'"
      id="panel-reviews"
      role="tabpanel"
      aria-labelledby="tab-reviews"
      tabindex="0"
      class="outline-none"
    >
      <section class="mb-10">
        <h2 class="section-title mb-3 flex items-center gap-2">
          <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
          Open merge requests
        </h2>
        <DependencyAlert
          v-if="noProviders"
          class="mb-3"
          message="No AI provider configured — add one to run reviews."
          cta-label="Add a provider"
          cta-to="/providers"
        />
        <MergeRequestList
          :items="mrs"
          :loading="mrsLoading"
          :error="reviews.mrsError"
          :busy-iid="creatingIid"
          :providers="providers.items"
          :default-provider-id="defaultProviderId"
          :review-by-mr="reviewByMr"
          @review="startReview"
        />
      </section>

      <section>
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <h2 class="section-title flex items-center gap-2">
            <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
            Reviews
          </h2>
          <button class="btn-ghost text-xs" @click="toggleArchived">
            <span
              :class="showArchived ? 'i-lucide-eye-off' : 'i-lucide-archive'"
              class="text-sm"
              aria-hidden="true"
            />
            {{ showArchived ? 'Hide archived' : 'Show archived' }}
          </button>
        </div>
        <ReviewList
          :items="repoReviews"
          :loading="reviewsLoading"
          :error="reviews.listError"
          :busy-ids="archivingIds"
          @archive="archiveReview"
          @approve="approveReview"
          @discard="discardReview"
        />

        <template v-if="showArchived">
          <h3 class="section-title text-muted mt-6 mb-3 flex items-center gap-2">
            <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
            Archived
          </h3>
          <ReviewList
            :items="archivedReviews"
            :loading="archivedLoading"
            :error="reviews.archivedError"
            :busy-ids="archivingIds"
            @unarchive="unarchiveReview"
          />
        </template>
      </section>
    </div>

    <!-- Routines tab: preflight the API scope, then start/watch release routines. -->
    <div
      v-show="activeTab === 'routines'"
      id="panel-routines"
      role="tabpanel"
      aria-labelledby="tab-routines"
      tabindex="0"
      class="outline-none"
    >
      <section class="mb-10">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <h2 class="section-title flex items-center gap-2">
            <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
            API scope
          </h2>
          <button
            type="button"
            class="btn-line text-xs"
            :disabled="preflightLoading"
            @click="testApiScope"
          >
            <span
              :class="preflightLoading ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-shield-check'"
              class="text-sm"
              aria-hidden="true"
            />
            {{ preflightLoading ? 'Testing…' : 'Test API scope' }}
          </button>
        </div>

        <p class="text-muted mb-3 text-xs">
          Check which automated actions your token and project access permit before running a
          routine.
        </p>

        <p v-if="preflightError" class="text-danger py-1 text-sm" role="alert">
          {{ preflightError }}
        </p>
        <PreflightReport v-else-if="preflight" :report="preflight" />
      </section>

      <section>
        <RoutinesSection :repo-id="repoId" :merge-requests="mrs" />
      </section>
    </div>

    <!-- Webhook tab: enable auto-review on MR events and reveal the URL + secret. -->
    <div
      v-show="activeTab === 'webhook'"
      id="panel-webhook"
      role="tabpanel"
      aria-labelledby="tab-webhook"
      tabindex="0"
      class="outline-none"
    >
      <section class="mb-10">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <h2 class="section-title flex items-center gap-2">
            <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
            Auto-review webhook
          </h2>
          <button
            type="button"
            class="btn-line text-xs"
            :disabled="webhookBusy || !repo"
            @click="toggleWebhook(!(repo?.webhookEnabled ?? false))"
          >
            <span
              :class="
                webhookBusy
                  ? 'i-lucide-loader-circle animate-spin'
                  : repo?.webhookEnabled
                    ? 'i-lucide-toggle-right'
                    : 'i-lucide-toggle-left'
              "
              class="text-sm"
              aria-hidden="true"
            />
            {{ repo?.webhookEnabled ? 'Disable' : 'Enable' }}
          </button>
        </div>

        <p class="text-muted mb-4 text-xs">
          When enabled, a review is created and run automatically whenever a merge request is opened
          or receives new commits.
        </p>

        <template v-if="repo?.webhookEnabled">
          <div class="mb-4">
            <span class="field-label">Payload URL</span>
            <div class="flex items-center gap-2">
              <code
                class="border-line text-ink block flex-1 overflow-x-auto border-b px-0 py-2 font-mono text-xs"
              >
                {{ webhookUrl }}
              </code>
              <button
                type="button"
                class="btn-ghost text-xs"
                aria-label="Copy webhook URL"
                @click="copyText(webhookUrl, 'URL')"
              >
                <span class="i-lucide-copy text-sm" aria-hidden="true" />
                Copy
              </button>
            </div>
          </div>

          <div class="mb-4">
            <span class="field-label">Secret token</span>
            <div class="flex items-center gap-2">
              <code
                class="border-line text-ink block flex-1 overflow-x-auto border-b px-0 py-2 font-mono text-xs"
              >
                {{ showWebhookSecret ? repo.webhookSecret : '•'.repeat(24) }}
              </code>
              <button
                type="button"
                class="btn-ghost text-xs"
                :aria-label="showWebhookSecret ? 'Hide secret token' : 'Show secret token'"
                :aria-pressed="showWebhookSecret"
                @click="showWebhookSecret = !showWebhookSecret"
              >
                <span
                  :class="showWebhookSecret ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                  class="text-sm"
                  aria-hidden="true"
                />
                {{ showWebhookSecret ? 'Hide' : 'Show' }}
              </button>
              <button
                type="button"
                class="btn-ghost text-xs"
                aria-label="Copy secret token"
                @click="copyText(repo.webhookSecret, 'Secret token')"
              >
                <span class="i-lucide-copy text-sm" aria-hidden="true" />
                Copy
              </button>
              <button
                type="button"
                class="btn-ghost text-xs"
                :disabled="webhookBusy"
                aria-label="Rotate secret token"
                @click="rotateWebhook"
              >
                <span class="i-lucide-refresh-cw text-sm" aria-hidden="true" />
                Rotate
              </button>
            </div>
          </div>

          <p class="text-muted mb-4 text-xs">
            Add this URL and Secret Token as a Merge request events webhook in GitLab → Settings →
            Webhooks. Rotating the token invalidates the old one — update it in GitLab afterwards.
          </p>

          <div class="border-line border-t pt-4">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <span class="field-label mb-0">Require confirmation before running</span>
              <button
                type="button"
                class="btn-line text-xs"
                :disabled="webhookBusy"
                @click="toggleRequireConfirmation(!(repo?.webhookRequireConfirmation ?? false))"
              >
                <span
                  :class="
                    webhookBusy
                      ? 'i-lucide-loader-circle animate-spin'
                      : repo?.webhookRequireConfirmation
                        ? 'i-lucide-toggle-right'
                        : 'i-lucide-toggle-left'
                  "
                  class="text-sm"
                  aria-hidden="true"
                />
                {{ repo?.webhookRequireConfirmation ? 'On' : 'Off' }}
              </button>
            </div>
            <p class="text-muted mt-2 text-xs">
              Held reviews wait in the Reviews list until you approve them.
            </p>
          </div>
        </template>
      </section>
    </div>
  </div>
</template>
