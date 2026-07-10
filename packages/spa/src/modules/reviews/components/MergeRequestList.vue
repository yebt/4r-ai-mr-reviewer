<script setup lang="ts">
import { reactive } from 'vue'
import type { MergeRequest } from '@shared/api/types'

defineProps<{
  items: MergeRequest[]
  loading?: boolean
  error?: string | null
  busyIid?: number | null
}>()
const emit = defineEmits<{ review: [iid: number, mode: string] }>()

// Per-MR context mode, chosen at the moment of triggering (default fast).
const modes = reactive<Record<number, string>>({})
function modeFor(iid: number) {
  return modes[iid] ?? 'fast'
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

        <div class="flex w-full items-center justify-end gap-2 sm:w-auto">
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
            @click="emit('review', mr.iid, modeFor(mr.iid))"
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
