<script setup lang="ts">
/**
 * Launch-a-review dialog — opened by `MergeRequestListSection`'s per-row
 * "Review" button (`#actions` slot). Owns its own `DialogRoot` (standalone
 * view, not a create/edit form embedded in the parent's dialog), controlled
 * via `open`/`repo`/`mergeRequest` props and `update:open` — mirrors
 * `modules/repos/components/WebhookDialog.vue`'s shape.
 *
 * Launching a review is read-only LLM analysis, not a GitLab write, so this
 * needs no danger confirm — just the form (`Provider`, `Model`, `Mode`) and a
 * primary "Start review" action.
 *
 * Defaults (`../reviewLaunch.ts`): Provider preselects the repo's own
 * provider, else the account-wide default provider. Model preselects the
 * repo's saved model when its own provider is selected, else that provider's
 * own default model — both resolve to `''` ("use the provider's default")
 * when nothing else applies, which is a valid, meaningful value server-side
 * (see `CreateReviewInput`'s doc). Switching provider re-resolves the model
 * default for the new provider instead of clearing it, since "provider
 * default" is itself a filled, useful value (unlike `RepoForm`'s plain
 * reset-to-blank on provider change).
 *
 * Model renders as a `Select` over the chosen provider's declared `models`
 * when non-empty, else a free-text `Input` — same split as `RepoForm`'s
 * Model field. Reka's `Select` reserves `''` for its own placeholder state,
 * so `NO_MODEL_VALUE` stands in for "provider default" in the UI only;
 * `form.model` itself (never the sentinel) is what gets submitted.
 *
 * On success: closes, toasts, invalidates this repo's reviews query
 * (`repoReviewsQueryKey`, the same @pinia/colada cache entry
 * `RepoReviewsSection`/`MergeRequestListSection` read) plus the global
 * reviews list (`useReviewsStore` — a hand-rolled `useInfiniteList`
 * resource, not @pinia/colada, see its module doc), then navigates to the
 * created review's detail page (`/reviews/:id`) — matching the old app's
 * "launch and watch" behavior.
 */
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useQueryCache } from '@pinia/colada'
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { Alert, Button, Field, Input, Select } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { useProvidersStore } from '@modules/providers/store'
import { useReviewsStore } from '@modules/reviews/store'
import { createReview, repoReviewsQueryKey } from '@modules/reviews/api'
import type { ReviewContextMode } from '@modules/reviews/types'
import type { MergeRequest, Repo } from '@modules/repos/types'
import { resolveDefaultModel, resolveDefaultProviderId } from '../reviewLaunch'

const props = defineProps<{
  open: boolean
  repo: Repo
  mergeRequest: MergeRequest | null
  /** A pending/running review already exists for this MR — launching again
   * creates (and bills) a second one; the backend does not dedupe manual launches. */
  activeReview?: { id: string; status: string } | null
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const NO_MODEL_VALUE = '__use_provider_default_model__'

const router = useRouter()
const toast = useToast()
const queryCache = useQueryCache()
const providersStore = useProvidersStore()
const reviewsStore = useReviewsStore()

const hasProviders = computed(() => providersStore.providers.length > 0)
const providerOptions = computed<SelectItemOption[]>(() =>
  providersStore.providers.map((provider) => ({ label: provider.name, value: provider.id })),
)

const form = reactive({
  providerId: '',
  model: '',
  mode: 'fast' as ReviewContextMode,
})

const selectedProviderModels = computed<string[]>(
  () => providersStore.providers.find((provider) => provider.id === form.providerId)?.models ?? [],
)
const hasProviderModels = computed(() => selectedProviderModels.value.length > 0)
const modelOptions = computed<SelectItemOption[]>(() => [
  { label: "Provider default", value: NO_MODEL_VALUE },
  ...selectedProviderModels.value.map((model) => ({ label: model, value: model })),
])
// Proxies form.model (the actual submitted value) through the sentinel
// Reka's Select needs — mirrors RepoForm's `modelSelectValue`.
const modelSelectValue = computed<string>({
  get: () => (form.model ? form.model : NO_MODEL_VALUE),
  set: (value) => {
    form.model = value === NO_MODEL_VALUE ? '' : value
  },
})

const modeHint = computed(() =>
  form.mode === 'deep'
    ? "Deep — full-context reasoning over the whole MR. Slower and more thorough."
    : "Fast — a quicker pass scoped to the diff. Fewer tokens, less context.",
)

function resetForm() {
  const providerId = resolveDefaultProviderId(props.repo, providersStore.providers)
  form.providerId = providerId
  form.model = resolveDefaultModel(props.repo, providerId, providersStore.providers)
  form.mode = 'fast'
}

// Re-resolves the model default for the newly selected provider — switching
// provider never clears the field to blank, since "provider default" (which
// this resolves to when the new provider has none of its own) is already a
// meaningful, filled value.
function onProviderChange(id: string) {
  form.providerId = id
  form.model = resolveDefaultModel(props.repo, id, providersStore.providers)
}

const submitting = ref(false)
const submitError = ref<string | null>(null)

// Resets local, dialog-scoped state whenever the dialog opens (or targets a
// different repo/MR), so reopening never leaks a previous launch's edits or
// error into the next one — mirrors WebhookDialog's reset watcher.
watch(
  () => [props.open, props.repo.id, props.mergeRequest?.iid] as const,
  ([open]) => {
    if (!open) return
    submitError.value = null
    resetForm()
  },
  { immediate: true },
)

async function handleSubmit() {
  if (!props.mergeRequest || submitting.value) return
  submitting.value = true
  submitError.value = null
  try {
    const review = await createReview({
      repoId: props.repo.id,
      mrIid: props.mergeRequest.iid,
      mode: form.mode,
      providerId: form.providerId,
      model: form.model,
    })
    emit('update:open', false)
    toast.success('Review started')
    await queryCache.invalidateQueries({ key: repoReviewsQueryKey(props.repo.id) })
    await reviewsStore.refetch()
    router.push(`/reviews/${review.id}`)
  } catch (err) {
    submitError.value = resolveErrorMessage(err, 'Failed to start review')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <DialogRoot :open="open" @update:open="(value) => emit('update:open', value)">
    <DialogPortal>
      <DialogOverlay class="overlay z-20" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-30 max-h-[85vh] w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
      >
        <DialogTitle class="text-md font-semibold">Start a review</DialogTitle>
        <DialogDescription v-if="mergeRequest" class="mt-1 text-sm text-text-muted">
          <span class="font-mono">!{{ mergeRequest.iid }}</span>
          {{ mergeRequest.title }}
          <span class="mt-0.5 block truncate font-mono text-xs">
            {{ mergeRequest.sourceBranch }} → {{ mergeRequest.targetBranch }}
          </span>
        </DialogDescription>

        <form class="mt-4 flex flex-col gap-4" @submit.prevent="handleSubmit">
          <Alert v-if="!hasProviders" status="info">
            No providers configured yet — the review will use the account's default settings once one is added.
          </Alert>

          <Field v-else label="Provider" v-slot="{ id, describedBy }">
            <Select
              :id="id"
              :model-value="form.providerId"
              :items="providerOptions"
              placeholder="Select a provider…"
              :aria-describedby="describedBy"
              @update:model-value="(value) => onProviderChange(value ?? '')"
            />
          </Field>

          <Field label="Model" description="Optional — leave as provider default to use it." v-slot="{ id, describedBy }">
            <Select
              v-if="hasProviderModels"
              :id="id"
              v-model="modelSelectValue"
              :items="modelOptions"
              :aria-describedby="describedBy"
            />
            <Input
              v-else
              :id="id"
              v-model="form.model"
              placeholder="Provider default"
              autocomplete="off"
              :aria-describedby="describedBy"
            />
          </Field>

          <Field label="Context mode" :description="modeHint" v-slot="{ id, describedBy }">
            <Select
              :id="id"
              v-model="form.mode"
              :items="[
                { label: 'Fast', value: 'fast' },
                { label: 'Deep', value: 'deep' },
              ]"
              :aria-describedby="describedBy"
            />
          </Field>

          <Alert v-if="activeReview" status="warning">
            A review for this MR is already {{ activeReview.status }}. Starting another one runs (and bills) a
            second review.
            <RouterLink :to="`/reviews/${activeReview.id}`" class="font-medium underline underline-offset-2">
              Open the current review
            </RouterLink>
          </Alert>

          <Alert v-if="submitError" status="danger">{{ submitError }}</Alert>

          <div class="flex justify-end gap-2 border-t border-line-subtle pt-4">
            <Button type="button" variant="ghost" @click="emit('update:open', false)">Cancel</Button>
            <Button type="submit" :loading="submitting" :disabled="!mergeRequest">
              {{ activeReview ? 'Start another review' : 'Start review' }}
            </Button>
          </div>
        </form>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
