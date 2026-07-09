<script setup lang="ts">
import type { ReviewStatus } from '@shared/api/types'
import { statusClass } from '@modules/reviews/format'

defineProps<{ status: ReviewStatus }>()
</script>

<template>
  <span class="chip" :class="statusClass[status]">
    <span class="inline-block h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />
    {{ status }}
  </span>
</template>
