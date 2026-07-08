<script setup lang="ts">
import PageHeader from '@shared/components/ui/PageHeader.vue'
</script>

<template>
  <div>
    <PageHeader label="Quick access" title="Recent reviews" />
    <p class="text-muted">
      Add an account and an AI provider, track a repository, then start a review
      of a merge request.
    </p>
    <RouterLink to="/repos" class="btn-accent mt-6">
      Go to repositories
      <span class="i-lucide-arrow-right" aria-hidden="true" />
    </RouterLink>
  </div>
</template>
