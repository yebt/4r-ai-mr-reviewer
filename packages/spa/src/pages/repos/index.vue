<script setup lang="ts">
import { ref } from 'vue'
import type { Repo } from '@shared/api/types'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import RepoForm from '@modules/repos/components/RepoForm.vue'
import RepoList from '@modules/repos/components/RepoList.vue'

const editing = ref<Repo | null>(null)
</script>

<template>
  <div>
    <PageHeader label="Workspace" title="Repositories">
      <template #actions>
        <span class="label-mono">GitLab</span>
      </template>
    </PageHeader>

    <div class="flex flex-col md:grid gap-8 md:grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)] md:gap-12">
      <RepoForm :editing="editing" @done="editing = null" />
      <RepoList @edit="editing = $event" />
    </div>
  </div>
</template>
