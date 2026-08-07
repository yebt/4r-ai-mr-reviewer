<script setup lang="ts">
import { computed, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import type { RoutineRun } from '@shared/api/types'
import { formatDateTime, routineKindLabel, runTitle } from '@modules/routines/format'
import { useRoutinesStore } from '@modules/routines/store'
import RoutineStatusChip from '@modules/routines/components/RoutineStatusChip.vue'
import RoutineBranchChip from '@modules/routines/components/RoutineBranchChip.vue'

const props = defineProps<{
  items: RoutineRun[]
  loading?: boolean
  error?: string | null
  // When true, terminal runs can be archived / archived runs restored (repo tab
  // + Actions page); left off for glance views like the Home recent list.
  archivable?: boolean
}>()
const emit = defineEmits<{ changed: [] }>()

const isTerminalRun = (s: string) => s === 'done' || s === 'cancelled'

// Runs waiting on the user (a confirm gate or a recoverable block) float to the
// top and wear a warn edge, so "which run needs me?" reads instantly.
const needsAttention = (status: string) =>
  status === 'awaiting_confirmation' || status === 'blocked'
const sortedItems = computed(() =>
  [...props.items].sort(
    (a, b) => Number(needsAttention(b.status)) - Number(needsAttention(a.status)),
  ),
)

const store = useRoutinesStore()
const deletingId = ref<string | null>(null)
const archivingId = ref<string | null>(null)

async function archiveRun(run: RoutineRun) {
  archivingId.value = run.id
  try {
    await store.archive(run.id)
    toast.success('Action archived')
    emit('changed')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archivingId.value = null
  }
}
async function unarchiveRun(run: RoutineRun) {
  archivingId.value = run.id
  try {
    await store.unarchive(run.id)
    toast.success('Action restored')
    emit('changed')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    archivingId.value = null
  }
}

// Delete a run behind the app's shared confirm dialog. Running actions are never
// offered here (the button is hidden), so the backend 409 is only a backstop.
async function remove(run: RoutineRun) {
  const ok = await confirm({
    title: 'Delete action',
    message: `Delete "${runTitle(run)}"? This removes the routine run.`,
    danger: true,
  })
  if (!ok) return
  deletingId.value = run.id
  try {
    await store.remove(run.id)
    toast.success('Action deleted')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    deletingId.value = null
  }
}
</script>

<template>
  <div>
    <p v-if="loading" class="text-muted py-3 text-sm">Loading routines…</p>
    <p v-else-if="error" class="text-danger py-3 text-sm">{{ error }}</p>
    <p v-else-if="items.length === 0" class="text-muted py-3 text-sm">No routine runs yet.</p>

    <ul v-else class="border-line/50 border-t">
      <li
        v-for="run in sortedItems"
        :key="run.id"
        class="row flex-wrap justify-between gap-y-1 border-l-2 pl-3"
        :class="needsAttention(run.status) ? 'border-warn' : 'border-transparent'"
      >
        <RouterLink
          :to="`/actions/${run.id}`"
          class="group flex min-w-0 flex-1 flex-wrap items-center gap-x-3 gap-y-1"
        >
          <!-- Primary label: the MR title when captured, else the !{mrIid}
               fallback. Truncates so a long title never overflows the row. -->
          <span
            class="text-ink group-hover:text-ink min-w-0 truncate text-sm hover:underline"
            :style="{ viewTransitionName: `action-${run.id}` }"
          >
            {{ runTitle(run) }}
          </span>
          <!-- The IID is a secondary hint only once the title already leads. -->
          <span v-if="run.state.mrTitle && run.mrIid" class="text-muted shrink-0 font-mono text-xs">
            !{{ run.mrIid }}
          </span>
          <RoutineStatusChip :status="run.status" />
          <span class="label-mono truncate">{{ routineKindLabel[run.kind] }}</span>
          <!-- Secondary branch/flow indicator; renders only when known. -->
          <RoutineBranchChip :run="run" />
        </RouterLink>
        <div class="flex shrink-0 items-center gap-1">
          <span v-if="run.updatedAt" class="label-mono">{{ formatDateTime(run.updatedAt) }}</span>
          <button
            v-if="archivable && run.archived"
            type="button"
            class="btn-ghost text-xs"
            :disabled="archivingId === run.id"
            :aria-label="`Restore ${runTitle(run)}`"
            title="Restore"
            @click="unarchiveRun(run)"
          >
            <span
              :class="archivingId === run.id ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-archive-restore'"
              class="text-sm"
              aria-hidden="true"
            />
          </button>
          <button
            v-else-if="archivable && isTerminalRun(run.status)"
            type="button"
            class="btn-ghost text-xs"
            :disabled="archivingId === run.id"
            :aria-label="`Archive ${runTitle(run)}`"
            title="Archive"
            @click="archiveRun(run)"
          >
            <span
              :class="archivingId === run.id ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-archive'"
              class="text-sm"
              aria-hidden="true"
            />
          </button>
          <button
            v-if="run.status !== 'running'"
            type="button"
            class="btn-ghost hover:text-danger"
            :disabled="deletingId === run.id"
            :aria-label="`Delete ${runTitle(run)}`"
            @click="remove(run)"
          >
            <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>
