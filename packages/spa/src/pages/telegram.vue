<script setup lang="ts">
import { computed, ref } from 'vue'
import type { TelegramTarget } from '@shared/api/types'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import Modal from '@shared/components/ui/Modal.vue'
import TelegramForm from '@modules/telegram/components/TelegramForm.vue'
import TelegramList from '@modules/telegram/components/TelegramList.vue'

const editing = ref<TelegramTarget | null>(null)
const open = ref(false)

const title = computed(() => (editing.value ? 'Edit target' : 'New target'))

function add() {
  editing.value = null
  open.value = true
}
function edit(t: TelegramTarget) {
  editing.value = t
  open.value = true
}
function close() {
  open.value = false
  editing.value = null
}
</script>

<template>
  <div>
    <PageHeader title="Telegram" label="Notifications">
      <template #actions>
        <button class="btn-accent text-xs" @click="add">
          <span class="i-lucide-plus text-sm" aria-hidden="true" />
          Add target
        </button>
      </template>
    </PageHeader>

    <TelegramList @edit="edit" />

    <Modal :open="open" :title="title" @close="close">
      <TelegramForm :editing="editing" @done="close" />
    </Modal>
  </div>
</template>
