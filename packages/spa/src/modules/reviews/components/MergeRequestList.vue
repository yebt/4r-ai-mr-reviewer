<script setup lang="ts">
import type { MergeRequest } from '@shared/api/types'

defineProps<{
  items: MergeRequest[]
  loading?: boolean
  error?: string | null
  busyIid?: number | null
}>()
defineEmits<{ review: [iid: number] }>()
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
        <button class="btn-line shrink-0 text-xs" :disabled="busyIid === mr.iid" @click="$emit('review', mr.iid)">
          <span v-if="busyIid === mr.iid" class="i-lucide-loader-circle animate-spin" aria-hidden="true" />
          Review
        </button>
      </li>
    </ul>
  </div>
</template>
