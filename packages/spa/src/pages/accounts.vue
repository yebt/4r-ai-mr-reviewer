<script setup lang="ts">
definePage({ meta: { title: 'GitLab accounts' } })
import { ref } from 'vue'
import type { Account } from '@shared/api/types'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import Modal from '@shared/components/ui/Modal.vue'
import AccountForm from '@modules/accounts/components/AccountForm.vue'
import AccountList from '@modules/accounts/components/AccountList.vue'

const showForm = ref(false)
// The account being edited (null = the form is in create mode).
const editing = ref<Account | null>(null)

function openAdd() {
  editing.value = null
  showForm.value = true
}

function openEdit(account: Account) {
  editing.value = account
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editing.value = null
}
</script>

<template>
  <div>
    <PageHeader title="GitLab accounts">
      <template #actions>
        <button class="btn-accent text-xs" @click="openAdd">
          <span class="i-lucide-plus text-sm" aria-hidden="true" />
          Add account
        </button>
      </template>
    </PageHeader>

    <AccountList @add="openAdd" @edit="openEdit" />

    <Modal
      :open="showForm"
      :title="editing ? 'Edit account' : 'Add account'"
      @close="closeForm"
    >
      <AccountForm :editing="editing" @done="closeForm" />
    </Modal>
  </div>
</template>
