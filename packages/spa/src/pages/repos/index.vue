<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Repo } from '@shared/api/types'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import Modal from '@shared/components/ui/Modal.vue'
import RepoForm from '@modules/repos/components/RepoForm.vue'
import RepoList from '@modules/repos/components/RepoList.vue'

const editing = ref<Repo | null>(null)
const open = ref(false)

const title = computed(() => (editing.value ? 'Reassign provider' : 'New repository'))

function add() {
  editing.value = null
  open.value = true
}
function edit(r: Repo) {
  editing.value = r
  open.value = true
}
function close() {
  open.value = false
  editing.value = null
}
</script>

<template>
  <div>
    <PageHeader title="Tracked repositories">
      <template #actions>
        <button class="btn-accent text-xs" @click="add">
          <span class="i-lucide-plus text-sm" aria-hidden="true" />
          Track repository
        </button>
      </template>
    </PageHeader>

    <RepoList @edit="edit" />

    <Modal :open="open" :title="title" @close="close">
      <RepoForm :editing="editing" @done="close" />
    </Modal>
  </div>
</template>
