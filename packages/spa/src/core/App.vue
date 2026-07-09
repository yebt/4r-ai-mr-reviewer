<script setup lang="ts">
import AppSidebar from '@shared/components/layout/AppSidebar.vue'
import ConfirmDialog from '@shared/components/ui/ConfirmDialog.vue'
</script>

<template>
  <div class="flex h-screen w-screen overflow-hidden">
    <AppSidebar />
    <main class="flex-1 overflow-y-auto">
      <div class="mx-auto max-w-5xl px-8 py-8">
        <RouterView />
      </div>
    </main>
  </div>
  <ConfirmDialog />
</template>
