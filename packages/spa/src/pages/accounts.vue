<script setup lang="ts">
definePage({ meta: { title: 'GitLab accounts' } })
import { ref } from 'vue'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import Modal from '@shared/components/ui/Modal.vue'
import AccountForm from '@modules/accounts/components/AccountForm.vue'
import AccountList from '@modules/accounts/components/AccountList.vue'

const showForm = ref(false)
</script>

<template>
  <div>
    <PageHeader title="GitLab accounts">
      <template #actions>
        <button class="btn-accent text-xs" @click="showForm = true">
          <span class="i-lucide-plus text-sm" aria-hidden="true" />
          Add account
        </button>
      </template>
    </PageHeader>

    <AccountList @add="showForm = true" />

    <Modal :open="showForm" title="Add account" @close="showForm = false">
      <AccountForm @done="showForm = false" />
    </Modal>
  </div>
</template>
