<script setup lang="ts">
import type { Review } from '@shared/api/types'
import ReviewRow from '@modules/reviews/components/ReviewRow.vue'

defineProps<{
  items: Review[]
  loading?: boolean
  error?: string | null
  // Ids with an action in flight (their buttons are disabled).
  busyIds?: string[]
}>()

defineEmits<{
  archive: [id: string]
  unarchive: [id: string]
  approve: [id: string]
  discard: [id: string, mrIid: number]
  retry: [id: string]
}>()
</script>

<template>
  <div>
    <p v-if="loading" class="text-muted py-3 text-sm">Loading reviews…</p>
    <p v-else-if="error" class="text-danger py-3 text-sm">{{ error }}</p>
    <p v-else-if="items.length === 0" class="text-muted py-3 text-sm">No reviews yet.</p>

    <ul v-else class="border-line/50 border-t">
      <ReviewRow
        v-for="rv in items"
        :key="rv.id"
        :review="rv"
        :busy="busyIds?.includes(rv.id) ?? false"
        @approve="$emit('approve', $event)"
        @discard="(id, mrIid) => $emit('discard', id, mrIid)"
        @retry="$emit('retry', $event)"
        @archive="$emit('archive', $event)"
        @unarchive="$emit('unarchive', $event)"
      />
    </ul>
  </div>
</template>
