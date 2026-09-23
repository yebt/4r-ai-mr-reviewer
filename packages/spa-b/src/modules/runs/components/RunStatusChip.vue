<script setup lang="ts">
/**
 * Status chip for a run (or a step, via `status="stepStatusUi"` reuse isn't
 * needed — steps aren't rendered in this first-cut list). Thin `Badge`
 * wrapper driven by `runStatusUi` so the status→variant/icon/label mapping
 * lives in one place (`format.ts`). Spins the icon while `spin` is true
 * (the `running` status only) so the animation itself never needs a
 * `prefers-reduced-motion` opt-out beyond what `Icon`/Tailwind already give
 * other spinners in this app.
 */
import { computed } from 'vue'
import { Badge, Icon } from '@shared/ui/design-system'
import { runStatusUi } from '../format'
import type { RoutineRunStatus } from '../types'

const props = defineProps<{
  status: RoutineRunStatus
}>()

const ui = computed(() => runStatusUi[props.status])
</script>

<template>
  <Badge :status="ui.variant">
    <Icon :name="ui.icon" size="xs" :class="{ 'animate-spin motion-reduce:animate-none': ui.spin }" />
    {{ ui.label }}
  </Badge>
</template>
