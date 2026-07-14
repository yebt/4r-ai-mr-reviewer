<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Profile } from '@shared/api/types'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import Modal from '@shared/components/ui/Modal.vue'
import ProfileForm from '@modules/profiles/components/ProfileForm.vue'
import ProfileList from '@modules/profiles/components/ProfileList.vue'

const editing = ref<Profile | null>(null)
const open = ref(false)

const title = computed(() => (editing.value ? 'Edit profile' : 'New profile'))

function add() {
  editing.value = null
  open.value = true
}
function edit(p: Profile) {
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
    <PageHeader title="Humanization profiles">
      <template #actions>
        <button class="btn-accent text-xs" @click="add">
          <span class="i-lucide-plus text-sm" aria-hidden="true" />
          Add profile
        </button>
      </template>
    </PageHeader>

    <ProfileList @edit="edit" />

    <Modal :open="open" :title="title" @close="close">
      <ProfileForm :editing="editing" @done="close" />
    </Modal>
  </div>
</template>
