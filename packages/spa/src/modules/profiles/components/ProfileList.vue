<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import type { Profile } from '@shared/api/types'
import { useProfilesStore } from '@modules/profiles/store'

const emit = defineEmits<{ edit: [profile: Profile] }>()

const store = useProfilesStore()
const busyId = ref<string | null>(null)
const expanded = ref<Set<string>>(new Set())

function summary(p: Profile): string {
  const parts = [p.language, p.formality, p.emojis ? 'emojis' : 'no emojis'].filter(Boolean)
  const n = p.samples.length
  parts.push(`${n} sample${n === 1 ? '' : 's'}`)
  return parts.join(' · ')
}

function toggleGuide(id: string) {
  const next = new Set(expanded.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expanded.value = next
}

async function run(id: string, fn: () => Promise<void>): Promise<boolean> {
  busyId.value = id
  try {
    await fn()
    return true
  } catch (e) {
    store.error = errorMessage(e)
    return false
  } finally {
    busyId.value = null
  }
}

async function redistill(p: Profile) {
  if (await run(p.id, () => store.redistill(p.id))) toast.success('Redistilling style guide…')
}

async function removeProfile(p: Profile) {
  const ok = await confirm({
    title: 'Delete profile',
    message: `Delete "${p.name}"?`,
    danger: true,
  })
  if (!ok) return
  if (await run(p.id, () => store.remove(p.id))) toast.success('Profile deleted')
}

// Poll every pending profile until its style guide reaches a terminal state.
// A single poller covers freshly created/updated profiles and manual redistills.
const pendingIds = computed(() =>
  store.items.filter((p) => p.styleGuideStatus === 'pending').map((p) => p.id),
)

const { pause, resume } = useIntervalFn(
  async () => {
    await Promise.all(pendingIds.value.map((id) => store.refreshOne(id).catch(() => {})))
  },
  2500,
  { immediate: false },
)

watch(
  () => pendingIds.value.length,
  (n) => {
    if (n > 0) resume()
    else pause()
  },
)

onMounted(async () => {
  if (store.items.length === 0) await store.fetchAll()
  if (pendingIds.value.length > 0) resume()
})
onUnmounted(pause)
</script>

<template>
  <div>
    <div class="label-mono mb-3">{{ store.items.length }} profile(s)</div>

    <p v-if="store.loading" class="text-muted py-3 text-sm">Loading…</p>
    <p v-else-if="store.error" class="text-danger py-3 text-sm">{{ store.error }}</p>
    <EmptyState
      v-else-if="store.items.length === 0"
      icon="i-lucide-feather"
      title="No profiles yet"
      hint="Capture your writing voice to humanize reviews."
    />

    <ul v-else class="border-line/50 border-t">
      <li v-for="p in store.items" :key="p.id" class="border-line/50 border-b py-3">
        <div class="flex items-center justify-between gap-4">
          <div class="min-w-0">
            <div class="text-ink truncate text-sm">{{ p.name }}</div>
            <div class="text-muted truncate font-mono text-xs">{{ summary(p) }}</div>
          </div>
          <div class="flex shrink-0 items-center gap-1">
            <button
              v-if="p.samples.length"
              class="btn-ghost text-xs"
              :disabled="busyId === p.id || p.styleGuideStatus === 'pending'"
              :aria-label="`Redistill ${p.name}`"
              @click="redistill(p)"
            >
              <span class="i-lucide-refresh-cw text-sm" aria-hidden="true" />
              Redistill
            </button>
            <button
              class="btn-ghost hover:text-ink"
              :aria-label="`Edit ${p.name}`"
              @click="emit('edit', p)"
            >
              <span class="i-lucide-pencil text-sm" aria-hidden="true" />
            </button>
            <button
              class="btn-ghost hover:text-danger"
              :disabled="busyId === p.id"
              :aria-label="`Delete ${p.name}`"
              @click="removeProfile(p)"
            >
              <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
            </button>
          </div>
        </div>

        <!-- Style-guide status indicator -->
        <div class="mt-2">
          <div
            v-if="p.styleGuideStatus === 'pending'"
            class="text-accent flex items-center gap-1.5 font-mono text-xs"
          >
            <span class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
            distilling…
          </div>

          <div v-else-if="p.styleGuideStatus === 'ready'">
            <button
              class="text-ok hover:text-ok/80 flex items-center gap-1.5 font-mono text-xs"
              @click="toggleGuide(p.id)"
            >
              <span class="i-lucide-check text-sm" aria-hidden="true" />
              style guide ready
              <span
                :class="expanded.has(p.id) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
                aria-hidden="true"
              />
            </button>
            <pre
              v-if="expanded.has(p.id)"
              class="border-line/60 text-muted mt-2 max-h-64 overflow-auto border-l pl-3 text-xs whitespace-pre-wrap"
              >{{ p.styleGuide }}</pre
            >
          </div>

          <div v-else-if="p.styleGuideStatus === 'error'" class="text-danger font-mono text-xs">
            <span class="i-lucide-triangle-alert text-sm" aria-hidden="true" />
            {{ p.styleGuideError || 'distillation failed' }}
          </div>

          <div v-else class="text-muted/70 font-mono text-xs">no samples</div>
        </div>
      </li>
    </ul>
  </div>
</template>
