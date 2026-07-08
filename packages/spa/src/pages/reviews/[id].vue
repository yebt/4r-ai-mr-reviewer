<script setup lang="ts">
import { useRoute } from 'vue-router'
import PageHeader from '@shared/components/ui/PageHeader.vue'

const route = useRoute()
const reviewId = (route.params as { id: string }).id
</script>

<template>
  <div>
    <PageHeader label="Review" title="Findings" />
    <p class="font-mono text-sm text-muted">review: {{ reviewId }}</p>
    <p class="mt-2 text-muted">Coming in phase 3.</p>
  </div>
</template>
