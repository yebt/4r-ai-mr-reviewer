<script setup lang="ts">
import type { RoutineRun } from '@shared/api/types'
import { formatDateTime, routineKindLabel, runTitle } from '@modules/routines/format'
import RoutineStatusChip from '@modules/routines/components/RoutineStatusChip.vue'

defineProps<{
  items: RoutineRun[]
  loading?: boolean
  error?: string | null
}>()
</script>

<template>
  <div>
    <p v-if="loading" class="text-muted py-3 text-sm">Loading routines…</p>
    <p v-else-if="error" class="text-danger py-3 text-sm">{{ error }}</p>
    <p v-else-if="items.length === 0" class="text-muted py-3 text-sm">No routine runs yet.</p>

    <ul v-else class="border-line/50 border-t">
      <li v-for="run in items" :key="run.id" class="row justify-between">
        <RouterLink
          :to="`/actions/${run.id}`"
          class="group flex min-w-0 flex-1 items-center gap-3"
        >
          <!-- Primary label: the MR title when captured, else the !{mrIid}
               fallback. Truncates so a long title never overflows the row. -->
          <span class="text-ink group-hover:text-accent min-w-0 truncate text-sm">
            {{ runTitle(run) }}
          </span>
          <!-- The IID is a secondary hint only once the title already leads. -->
          <span v-if="run.state.mrTitle && run.mrIid" class="text-muted shrink-0 font-mono text-xs">
            !{{ run.mrIid }}
          </span>
          <RoutineStatusChip :status="run.status" />
          <span class="label-mono truncate">{{ routineKindLabel[run.kind] }}</span>
        </RouterLink>
        <div class="label-mono shrink-0">
          <span v-if="run.updatedAt">{{ formatDateTime(run.updatedAt) }}</span>
        </div>
      </li>
    </ul>
  </div>
</template>
