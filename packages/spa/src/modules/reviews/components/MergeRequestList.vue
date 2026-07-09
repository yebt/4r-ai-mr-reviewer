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
    <p v-if="loading" class="py-3 text-sm text-muted">Loading merge requests…</p>
    <p v-else-if="error" class="py-3 text-sm text-danger">{{ error }}</p>
    <p v-else-if="items.length === 0" class="py-3 text-sm text-muted">No open merge requests.</p>

    <ul v-else class="border-t border-line/50">
      <li v-for="mr in items" :key="mr.iid" class="row justify-between">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="font-mono text-xs text-muted">!{{ mr.iid }}</span>
            <a :href="mr.webUrl" target="_blank" rel="noreferrer" class="truncate text-sm text-ink hover:text-accent">
              {{ mr.title }}
            </a>
          </div>
          <div class="label-mono mt-0.5">
            {{ mr.sourceBranch }} → {{ mr.targetBranch }}<template v-if="mr.author"> · {{ mr.author }}</template>
          </div>
        </div>

        <div class="flex shrink-0 items-center gap-2">
          <select
            :value="modeFor(mr.iid)"
            class="border-b border-line bg-transparent py-1 pr-1 text-xs text-ink outline-none focus:border-accent"
            :aria-label="`Context mode for !${mr.iid}`"
            @change="modes[mr.iid] = ($event.target as HTMLSelectElement).value"
          >
            <option value="fast">fast</option>
            <option value="deep">deep</option>
          </select>
          <button class="btn-line text-xs" :disabled="busyIid === mr.iid" @click="emit('review', mr.iid, modeFor(mr.iid))">
            <span v-if="busyIid === mr.iid" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
            Review
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
