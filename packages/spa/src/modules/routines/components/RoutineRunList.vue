<script setup lang="ts">
import type { RoutineRun } from '@shared/api/types'
import { formatDateTime, routineKindLabel } from '@modules/routines/format'
import RoutineStatusChip from '@modules/routines/components/RoutineStatusChip.vue'

defineProps<{
  items: RoutineRun[]
  loading?: boolean
  error?: string | null
  // Currently expanded run, so its row reads as selected.
  selectedId?: string | null
}>()

const emit = defineEmits<{ select: [id: string] }>()
</script>

<template>
  <div>
    <p v-if="loading" class="text-muted py-3 text-sm">Loading routines…</p>
    <p v-else-if="error" class="text-danger py-3 text-sm">{{ error }}</p>
    <p v-else-if="items.length === 0" class="text-muted py-3 text-sm">No routine runs yet.</p>

    <ul v-else class="border-line/50 border-t">
      <li v-for="run in items" :key="run.id" class="row justify-between">
        <button
          type="button"
          class="flex min-w-0 flex-1 items-center gap-3 text-left"
          :aria-expanded="selectedId === run.id"
          @click="emit('select', run.id)"
        >
          <span
            class="text-ink font-mono text-sm"
            :class="selectedId === run.id ? 'text-accent' : ''"
          >
            !{{ run.mrIid }}
          </span>
          <RoutineStatusChip :status="run.status" />
          <span class="label-mono truncate">{{ routineKindLabel[run.kind] }}</span>
        </button>
        <div class="label-mono shrink-0">
          <span v-if="run.updatedAt">{{ formatDateTime(run.updatedAt) }}</span>
        </div>
      </li>
    </ul>
  </div>
</template>
