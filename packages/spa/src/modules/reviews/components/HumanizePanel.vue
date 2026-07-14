<script setup lang="ts">
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { Finding, HumanizeVariant } from '@shared/api/types'
import { useReviewsStore } from '@modules/reviews/store'
import { useProfilesStore } from '@modules/profiles/store'
import { dimensionLabel } from '@modules/reviews/format'

const props = defineProps<{ reviewId: string; findings: Finding[] }>()

const store = useReviewsStore()
const profiles = useProfilesStore()

const open = ref(false)
const loading = ref(false)
const error = ref<string | null>(null)

const profileId = ref('')
const count = ref(3)
const variants = ref<HumanizeVariant[]>([])
const activeVariant = ref(0)

// Only profiles whose style guide finished distilling can rewrite text.
const readyProfiles = computed(() =>
  profiles.items.filter((p) => p.styleGuideStatus === 'ready'),
)

const selectedProfile = computed(() =>
  readyProfiles.value.find((p) => p.id === profileId.value) ?? null,
)

// Lookup so a variant's finding text can be labelled with the finding it rewrites.
const findingByIndex = computed(() => {
  const map = new Map<number, Finding>()
  for (const f of props.findings) map.set(f.index, f)
  return map
})

const current = computed<HumanizeVariant | null>(() => variants.value[activeVariant.value] ?? null)

async function toggle() {
  open.value = !open.value
  if (open.value && profiles.items.length === 0) await profiles.fetchAll()
}

async function run() {
  if (!profileId.value) return
  loading.value = true
  error.value = null
  try {
    variants.value = await store.humanize(props.reviewId, profileId.value, count.value)
    activeVariant.value = 0
    if (variants.value.length === 0) {
      error.value = 'No variants were returned.'
    }
  } catch (e) {
    variants.value = []
    error.value = humanizeError(e)
    toast.error(error.value)
  } finally {
    loading.value = false
  }
}

// Maps the backend guard/LLM failures to actionable copy.
function humanizeError(e: unknown): string {
  const status = (e as { status?: number })?.status
  if (status === 409) return 'Profile style guide not ready, or the review is not finished.'
  if (status === 502) return 'Humanization failed, try again.'
  return errorMessage(e)
}

function findingLabel(index: number): string {
  const f = findingByIndex.value.get(index)
  if (!f) return `Finding #${index + 1}`
  const loc = f.file ? `${f.file}${f.line ? `:${f.line}` : ''}` : `#${index + 1}`
  return `${dimensionLabel[f.dimension]} · ${loc}`
}

async function copy(text: string) {
  if (!navigator?.clipboard) return
  await navigator.clipboard.writeText(text)
  toast.success('Copied')
}

async function copyAll() {
  const v = current.value
  if (!v) return
  const parts = [v.summary, ...v.findings.map((f) => f.text)].filter(Boolean)
  await copy(parts.join('\n\n'))
}
</script>

<template>
  <section class="mt-8">
    <button
      class="text-muted hover:text-ink flex items-center gap-2 text-sm transition-colors"
      :aria-expanded="open"
      @click="toggle"
    >
      <span :class="open ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'" aria-hidden="true" />
      <span class="i-lucide-feather text-sm" aria-hidden="true" />
      Humanize (preview)
    </button>

    <div v-if="open" class="border-line/50 mt-4 border-t pt-4">
      <!-- No ready profiles: guide the user to create one. -->
      <p v-if="readyProfiles.length === 0" class="text-muted text-sm">
        No ready humanization profiles.
        <RouterLink to="/profiles" class="text-accent hover:underline">Manage profiles</RouterLink>
      </p>

      <template v-else>
        <div class="flex flex-wrap items-end gap-4">
          <label class="block">
            <span class="field-label">Profile</span>
            <select v-model="profileId" class="field-underline min-w-48">
              <option value="" disabled>Select a profile…</option>
              <option v-for="p in readyProfiles" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </label>

          <label class="block">
            <span class="field-label">Variants</span>
            <select v-model.number="count" class="field-underline w-16">
              <option v-for="n in 5" :key="n" :value="n">{{ n }}</option>
            </select>
          </label>

          <button class="btn-accent text-xs" :disabled="loading || !profileId" @click="run">
            <span
              v-if="loading"
              class="i-lucide-loader-circle animate-spin text-sm"
              aria-hidden="true"
            />
            <span v-else class="i-lucide-sparkles text-sm" aria-hidden="true" />
            Humanize
          </button>
        </div>

        <p v-if="loading" class="text-muted mt-3 flex items-center gap-2 text-sm">
          <span class="i-lucide-loader-circle animate-spin text-sm" aria-hidden="true" />
          Rewriting in {{ selectedProfile?.name }}'s voice…
        </p>
        <p v-else-if="error" class="text-danger mt-3 text-sm">{{ error }}</p>

        <!-- Variant tabs + selected preview -->
        <div v-if="!loading && variants.length" class="mt-5">
          <div class="border-line/50 flex flex-wrap gap-1 border-b">
            <button
              v-for="(v, i) in variants"
              :key="i"
              class="-mb-px border-b-2 px-3 py-1.5 text-xs transition-colors"
              :class="
                i === activeVariant
                  ? 'border-accent text-ink'
                  : 'text-muted hover:text-ink border-transparent'
              "
              @click="activeVariant = i"
            >
              Variant {{ i + 1 }}
            </button>
            <div class="ml-auto flex items-center">
              <button class="btn-ghost text-xs" @click="copyAll">
                <span class="i-lucide-copy text-sm" aria-hidden="true" />
                Copy all
              </button>
            </div>
          </div>

          <div v-if="current" class="mt-4">
            <div v-if="current.summary">
              <div class="mb-1.5 flex items-center justify-between gap-2">
                <span class="label-mono">Summary</span>
                <button class="btn-ghost text-xs" @click="copy(current.summary)">
                  <span class="i-lucide-copy text-sm" aria-hidden="true" />
                  Copy
                </button>
              </div>
              <p class="text-ink text-sm leading-relaxed whitespace-pre-wrap">
                {{ current.summary }}
              </p>
            </div>

            <div
              v-for="ft in current.findings"
              :key="ft.index"
              class="border-line/50 mt-4 border-t pt-4"
            >
              <div class="mb-1.5 flex items-center justify-between gap-2">
                <span class="text-muted font-mono text-xs">{{ findingLabel(ft.index) }}</span>
                <button class="btn-ghost text-xs" @click="copy(ft.text)">
                  <span class="i-lucide-copy text-sm" aria-hidden="true" />
                  Copy
                </button>
              </div>
              <p class="text-ink text-sm leading-relaxed whitespace-pre-wrap">{{ ft.text }}</p>
            </div>
          </div>
        </div>
      </template>
    </div>
  </section>
</template>
