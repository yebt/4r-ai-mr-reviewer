<script setup lang="ts">
definePage({ meta: { title: 'Repository' } })
import { computed, onMounted, ref, watchEffect } from 'vue'
import { useTitle } from '@vueuse/core'
import { useRoute, useRouter } from 'vue-router'
import { api, errorMessage } from '@shared/api/client'
import type { Preflight } from '@shared/api/types'
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
          <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
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
          @review="startReview"
        />
      </section>

      <section>
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <h2 class="section-title flex items-center gap-2">
            <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
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
            <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
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
  </div>
</template>
