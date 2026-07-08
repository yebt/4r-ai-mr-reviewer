<script setup lang="ts">
import PageHeader from '@shared/components/ui/PageHeader.vue'
import ProviderForm from '@modules/providers/components/ProviderForm.vue'
import ProviderList from '@modules/providers/components/ProviderList.vue'
</script>

<template>
  <div>
    <PageHeader label="Configuration" title="AI providers">
      <template #actions>
        <span class="label-mono">4R engine</span>
      </template>
    </PageHeader>

    <div class="grid gap-12 md:grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)]">
      <ProviderForm />
      <ProviderList />
    </div>
  </div>
</template>
