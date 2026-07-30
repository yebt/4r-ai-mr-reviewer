<script setup lang="ts">
import { reactive } from 'vue'
import type { MergeRequest, Provider } from '@shared/api/types'

const props = defineProps<{
  items: MergeRequest[]
  loading?: boolean
  error?: string | null
  busyIid?: number | null
  providers: Provider[]
  // Preselected provider id (repo's provider, else the global default).
  defaultProviderId: string
}>()
const emit = defineEmits<{
  review: [iid: number, mode: string, providerId: string, model: string]
}>()

// Per-MR context mode, chosen at the moment of triggering (default fast).
const modes = reactive<Record<number, string>>({})
function modeFor(iid: number) {
  return modes[iid] ?? 'fast'
}

// Per-MR provider override; falls back to the preselected default until touched.
const providerIds = reactive<Record<number, string>>({})
function providerFor(iid: number) {
  return providerIds[iid] ?? props.defaultProviderId
}

// Per-MR model override; empty means "use the provider's default model".
const models = reactive<Record<number, string>>({})
function modelFor(iid: number) {
  return models[iid] ?? ''
}

// Models declared by the MR's currently-selected provider (empty if none).
function providerModelsFor(iid: number) {
  return props.providers.find((p) => p.id === providerFor(iid))?.models ?? []
}
</script>

<template>
  <div>
    <p v-if="loading" class="text-muted py-3 text-sm">Loading merge requests…</p>
    <p v-else-if="error" class="text-danger py-3 text-sm">{{ error }}</p>
    <p v-else-if="items.length === 0" class="text-muted py-3 text-sm">No open merge requests.</p>

    <ul v-else class="border-line/50 border-t">
      <li v-for="mr in items" :key="mr.iid" class="row flex-wrap justify-between gap-y-2">
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <span class="text-muted font-mono text-xs">!{{ mr.iid }}</span>
            <a
              :href="mr.webUrl"
              target="_blank"
              rel="noreferrer"
              class="text-ink hover:text-accent truncate text-sm"
            >
              {{ mr.title }}
            </a>
          </div>
          <div class="label-mono mt-0.5">
            {{ mr.sourceBranch }} → {{ mr.targetBranch
            }}<template v-if="mr.author"> · {{ mr.author }}</template>
          </div>
        </div>

        <div class="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto">
          <select
            v-if="providers.length"
            :value="providerFor(mr.iid)"
            class="border-line text-ink focus:border-accent border-b bg-transparent py-1 pr-1 text-xs outline-none"
            :aria-label="`Provider for !${mr.iid}`"
            @change="
              providerIds[mr.iid] = ($event.target as HTMLSelectElement).value;
              models[mr.iid] = ''
            "
          >
            <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
          </select>
          <select
            v-if="providerModelsFor(mr.iid).length"
            :value="modelFor(mr.iid)"
            class="border-line text-ink focus:border-accent border-b bg-transparent py-1 pr-1 text-xs outline-none"
            :aria-label="`Model for !${mr.iid}`"
            @change="models[mr.iid] = ($event.target as HTMLSelectElement).value"
          >
            <option value="">default model</option>
            <option v-for="m in providerModelsFor(mr.iid)" :key="m" :value="m">{{ m }}</option>
          </select>
          <input
            v-else
            :value="modelFor(mr.iid)"
            type="text"
            placeholder="default model"
            class="border-line text-ink placeholder:text-muted/50 focus:border-accent w-28 border-b bg-transparent py-1 text-xs outline-none"
            :aria-label="`Model for !${mr.iid}`"
            @input="models[mr.iid] = ($event.target as HTMLInputElement).value"
          />
          <select
            :value="modeFor(mr.iid)"
            class="border-line text-ink focus:border-accent border-b bg-transparent py-1 pr-1 text-xs outline-none"
            :aria-label="`Context mode for !${mr.iid}`"
            @change="modes[mr.iid] = ($event.target as HTMLSelectElement).value"
          >
            <option value="fast">fast</option>
            <option value="deep">deep</option>
          </select>
          <button
            class="btn-line text-xs"
            :disabled="busyIid === mr.iid"
            @click="emit('review', mr.iid, modeFor(mr.iid), providerFor(mr.iid), modelFor(mr.iid))"
          >
            <span
              v-if="busyIid === mr.iid"
              class="i-lucide-loader-circle animate-spin"
              aria-hidden="true"
            />
            Review
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
