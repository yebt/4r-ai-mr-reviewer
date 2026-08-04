<script setup lang="ts">
import { computed } from 'vue'
import { useLocalStorage } from '@vueuse/core'
import type { ReviewStatus, RoutineRunStatus } from '@shared/api/types'
import { useActivityStore } from '@modules/activity/store'
import ReviewStatusChip from '@modules/reviews/components/ReviewStatusChip.vue'
import RoutineStatusChip from '@modules/routines/components/RoutineStatusChip.vue'

// Google-Drive-style tracker of every in-flight operation. Fixed bottom-right,
// persistent across navigation; hidden entirely when nothing is active or the
// user dismissed it. Collapsed state is a lightweight persisted preference.
const activity = useActivityStore()

const collapsed = useLocalStorage('activity.sheet.collapsed', false)
function toggleCollapsed() {
  collapsed.value = !collapsed.value
}

const visible = computed(() => activity.count > 0 && !activity.dismissed)

// A "running-like" op earns an extra spinner next to its chip so progress reads
// at a glance even where the chip itself carries no animation.
function isBusy(status: string): boolean {
  return status === 'running' || status === 'pending'
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="bg-surface border-line/70 fixed right-4 bottom-20 z-30 w-[20rem] max-w-[calc(100vw-2rem)] border shadow-lg shadow-black/40 md:bottom-4"
      role="region"
      aria-label="In-progress operations"
    >
      <!-- Header: count + collapse/expand + dismiss. -->
      <div class="border-line/50 flex items-center gap-2 border-b px-3 py-2">
        <span class="i-lucide-activity text-accent text-sm shrink-0" aria-hidden="true" />
        <span class="text-ink flex-1 text-sm font-medium">{{ activity.count }} in progress</span>
        <button
          type="button"
          class="btn-ghost"
          :aria-expanded="!collapsed"
          :title="collapsed ? 'Expand' : 'Collapse'"
          aria-label="Toggle in-progress panel"
          @click="toggleCollapsed"
        >
          <span
            :class="collapsed ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
            class="text-sm"
            aria-hidden="true"
          />
        </button>
        <button
          type="button"
          class="btn-ghost"
          title="Dismiss"
          aria-label="Dismiss in-progress panel"
          @click="activity.dismiss()"
        >
          <span class="i-lucide-x text-sm" aria-hidden="true" />
        </button>
      </div>

      <!-- Body: scrollable list of active operations. -->
      <ul v-if="!collapsed" class="max-h-64 overflow-auto">
        <li
          v-for="op in activity.activeOps"
          :key="`${op.kind}:${op.id}`"
          class="border-line/40 border-b last:border-b-0"
        >
          <RouterLink
            :to="op.to"
            class="group hover:bg-muted/10 flex items-center gap-2 px-3 py-2"
          >
            <span
              :class="op.kind === 'action' ? 'i-lucide-git-merge' : 'i-lucide-scan-search'"
              class="text-muted group-hover:text-accent shrink-0 text-sm"
              aria-hidden="true"
            />
            <span class="text-ink min-w-0 flex-1 truncate text-sm">{{ op.title }}</span>
            <span
              v-if="isBusy(op.status)"
              class="i-lucide-loader-circle text-accent shrink-0 animate-spin text-sm"
              aria-hidden="true"
            />
            <RoutineStatusChip
              v-if="op.kind === 'action'"
              :status="(op.status as RoutineRunStatus)"
            />
            <ReviewStatusChip v-else :status="(op.status as ReviewStatus)" />
          </RouterLink>
        </li>
      </ul>
    </div>
  </Teleport>
</template>
