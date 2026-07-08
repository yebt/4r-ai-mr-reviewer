<script setup lang="ts">
import PageHeader from '@shared/components/ui/PageHeader.vue'
import AccountForm from '@modules/accounts/components/AccountForm.vue'
import AccountList from '@modules/accounts/components/AccountList.vue'
</script>

<template>
  <div>
    <PageHeader label="Configuration" title="Accounts">
      <template #actions>
        <span class="label-mono">GitLab</span>
      </template>
    </PageHeader>

    <div class="grid gap-12 md:grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)]">
      <AccountForm />
      <AccountList />
    </div>
  </div>
</template>
