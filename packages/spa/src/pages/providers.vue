<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Provider } from '@shared/api/types'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import Modal from '@shared/components/ui/Modal.vue'
import ProviderForm from '@modules/providers/components/ProviderForm.vue'
import ProviderList from '@modules/providers/components/ProviderList.vue'

const editing = ref<Provider | null>(null)
const open = ref(false)

const title = computed(() => (editing.value ? 'Edit provider' : 'New provider'))

function add() {
  editing.value = null
  open.value = true
}
function edit(p: Provider) {
  editing.value = p
  open.value = true
}
function close() {
  open.value = false
  editing.value = null
}
</script>

<template>
  <div>
    <PageHeader title="AI providers">
      <template #actions>
        <button class="btn-accent text-xs" @click="add">
          <span class="i-lucide-plus text-sm" aria-hidden="true" />
          Add provider
        </button>
      </template>
    </PageHeader>

    <ProviderList @edit="edit" />

    <Modal :open="open" :title="title" @close="close">
      <ProviderForm :editing="editing" @done="close" />
    </Modal>
  </div>
</template>
