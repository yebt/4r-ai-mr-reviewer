<script setup lang="ts">
import type { Preflight } from '@shared/api/types'
import { checkStatusUi } from '@modules/repos/preflight'

defineProps<{ report: Preflight }>()
</script>

<template>
  <div>
    <!-- Header: access level, default branch, and token scopes as chips. -->
    <div class="border-line/50 flex flex-wrap items-center gap-x-4 gap-y-2 border-b pb-3">
      <span class="chip text-ink">
        <span class="i-lucide-shield text-muted text-sm" aria-hidden="true" />
        {{ report.accessLevelName }}
      </span>
      <span class="chip text-muted">
        <span class="i-lucide-git-branch text-sm" aria-hidden="true" />
        {{ report.defaultBranch }}
      </span>
      <div class="flex flex-wrap items-center gap-1.5">
        <template v-if="report.scopesKnown">
          <span v-for="scope in report.tokenScopes" :key="scope" class="chip text-accent">
            {{ scope }}
          </span>
        </template>
        <span v-else class="text-muted text-xs italic">scopes could not be read</span>
      </div>
    </div>

    <!-- One row per probed capability. -->
    <ul>
      <li v-for="check in report.checks" :key="check.capability" class="row items-start">
        <span
          :class="[checkStatusUi[check.status].icon, checkStatusUi[check.status].class]"
          class="mt-0.5 shrink-0 text-base"
          aria-hidden="true"
        />
        <div class="min-w-0 flex-1">
          <div class="text-ink text-sm">
            <span class="sr-only">{{ checkStatusUi[check.status].srLabel }}: </span>
            {{ check.label }}
          </div>
          <div v-if="check.detail" class="text-muted mt-0.5 text-xs">{{ check.detail }}</div>
        </div>
      </li>
    </ul>
  </div>
</template>
