<script setup lang="ts">
/**
 * Status chip for a run (or a step, via `status="stepStatusUi"` reuse isn't
 * needed — steps aren't rendered in this first-cut list). Thin `Badge`
 * wrapper driven by `runStatusUi` so the status→variant/icon/label mapping
 * lives in one place (`format.ts`). Shows the `Spinner` atom while `spin` is
 * true (the `running` status only), which brings its own
 * `prefers-reduced-motion` opt-out.
 */
import { computed } from 'vue'
import { Badge, Icon, Spinner } from '@shared/ui/design-system'
import { runStatusUi } from '../format'
import type { RoutineRunStatus } from '../types'

const props = defineProps<{
  status: RoutineRunStatus
}>()

const ui = computed(() => runStatusUi[props.status])
</script>

<template>
  <Badge :status="ui.variant">
    <Spinner v-if="ui.spin" size="xs" aria-hidden="true" />
    <Icon v-else :name="ui.icon" size="xs" />
    {{ ui.label }}
  </Badge>
</template>
