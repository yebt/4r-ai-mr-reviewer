<script setup lang="ts">
/**
 * Status chip for one step in the detail page's step ledger. Thin `Badge`
 * wrapper driven by `stepStatusUi`, mirroring `RunStatusChip.vue` exactly
 * (same icon+label+spin shape, different lookup table) so the two chips read
 * as one visual language.
 */
import { computed } from 'vue'
import { Badge, Icon, Spinner } from '@shared/ui/design-system'
import { stepStatusUi } from '../format'
import type { RoutineStepStatus } from '../types'

const props = defineProps<{
  status: RoutineStepStatus
}>()

const ui = computed(() => stepStatusUi[props.status])
</script>

<template>
  <Badge :status="ui.variant">
    <Spinner v-if="ui.spin" size="xs" aria-hidden="true" />
    <Icon v-else :name="ui.icon" size="xs" />
    {{ ui.label }}
  </Badge>
</template>
