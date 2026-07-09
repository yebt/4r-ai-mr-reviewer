<script setup lang="ts">
import { ref } from 'vue'
import type { Provider } from '@shared/api/types'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import ProviderForm from '@modules/providers/components/ProviderForm.vue'
import ProviderList from '@modules/providers/components/ProviderList.vue'

const editing = ref<Provider | null>(null)
</script>

<template>
  <div>
    <PageHeader title="AI providers" />

    <div class="flex flex-col md:grid gap-8 md:grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)] md:gap-12">
      <ProviderForm :editing="editing" @done="editing = null" />
      <ProviderList @edit="editing = $event" />
    </div>
  </div>
</template>
