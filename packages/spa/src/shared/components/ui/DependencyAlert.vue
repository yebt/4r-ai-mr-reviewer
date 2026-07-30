<script setup lang="ts">
// Warning banner shown when a screen depends on something that hasn't been
// configured yet. Reuses the shared Alert (warn variant) for styling and adds an
// inline RouterLink call-to-action that sends the user to create the missing
// prerequisite. RouterLink is globally registered, so no import is needed.
import Alert from './Alert.vue'

defineProps<{ message: string; ctaLabel?: string; ctaTo?: string }>()
</script>

<template>
  <Alert variant="warn">
    <span>{{ message }}</span>
    <RouterLink
      v-if="ctaTo && ctaLabel"
      :to="ctaTo"
      class="text-accent ml-1 font-medium whitespace-nowrap hover:underline"
    >
      {{ ctaLabel }}
    </RouterLink>
  </Alert>
</template>
