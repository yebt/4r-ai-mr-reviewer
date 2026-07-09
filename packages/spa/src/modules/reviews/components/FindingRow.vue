<script setup lang="ts">
import type { Finding } from '@shared/api/types'
import { dimensionLabel, severityClass } from '@modules/reviews/format'

defineProps<{ finding: Finding; selected: boolean; selectable: boolean }>()
defineEmits<{ toggle: [index: number] }>()
</script>

<template>
  <div class="flex gap-3 border-b border-line/50 py-4">
    <input
      v-if="selectable"
      type="checkbox"
      class="mt-1 accent-accent"
      :checked="selected"
      :aria-label="`Select finding ${finding.index + 1}`"
      @change="$emit('toggle', finding.index)"
    />
    <span v-else class="mt-1 w-[13px] shrink-0" aria-hidden="true" />

    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-center gap-2">
        <span class="chip text-muted">{{ dimensionLabel[finding.dimension] }}</span>
        <span class="chip" :class="severityClass[finding.severity]">{{ finding.severity }}</span>
        <span v-if="finding.blocking" class="chip text-flame">blocking</span>
        <span v-if="finding.published" class="chip text-ok">published</span>
      </div>

      <p class="mt-2 text-sm text-ink">{{ finding.issue }}</p>

      <div v-if="finding.file" class="mt-1 font-mono text-xs text-muted">
        {{ finding.file }}<template v-if="finding.line">:{{ finding.line }}</template>
      </div>

      <p v-if="finding.why" class="mt-2 text-sm text-muted"><span class="label-mono">why</span> {{ finding.why }}</p>
      <p v-if="finding.fix" class="mt-1 text-sm text-muted"><span class="label-mono">fix</span> {{ finding.fix }}</p>
    </div>
  </div>
</template>
