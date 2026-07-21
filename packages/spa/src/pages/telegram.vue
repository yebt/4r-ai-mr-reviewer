<script setup lang="ts">
import { ref } from 'vue'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import Modal from '@shared/components/ui/Modal.vue'
import TelegramForm from '@modules/telegram/components/TelegramForm.vue'
import TelegramList from '@modules/telegram/components/TelegramList.vue'

const open = ref(false)

function add() {
  open.value = true
}
function close() {
  open.value = false
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

    <TelegramList />

    <Modal :open="open" title="New target" @close="close">
      <TelegramForm @done="close" />
    </Modal>
  </div>
</template>
