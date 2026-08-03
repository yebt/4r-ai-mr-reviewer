<script setup lang="ts">
definePage({ meta: { title: 'More' } })
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import { errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import { useAuthStore } from '@modules/auth/store'

const auth = useAuthStore()
const router = useRouter()
const loggingOut = ref(false)

async function logout() {
  loggingOut.value = true
  try {
    await auth.logout()
    await router.replace({ path: '/login' })
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    loggingOut.value = false
  }
}

const items = [
  {
    to: '/actions',
    label: 'Actions',
    icon: 'i-lucide-git-merge',
    hint: 'Recent routine runs across repos',
  },
  {
    to: '/accounts',
    label: 'GitLab accounts',
    icon: 'i-lucide-users',
    hint: 'Manage GitLab connections',
  },
  {
    to: '/providers',
    label: 'AI providers',
    icon: 'i-lucide-cpu',
    hint: 'Configure models and keys',
  },
  {
    to: '/telegram',
    label: 'Telegram',
    icon: 'i-lucide-send',
    hint: 'Notification targets and bot',
  },
  {
    to: '/profiles',
    label: 'Humanization profiles',
    icon: 'i-lucide-feather',
    hint: 'Capture your writing voice',
  },
  {
    to: '/skills',
    label: '4R review skills',
    icon: 'i-lucide-book-open',
    hint: 'Rule sets used by the engine',
  },
  {
    to: '/settings',
    label: 'Settings',
    icon: 'i-lucide-settings',
    hint: 'App configuration & notifications',
  },
]
</script>

<template>
  <div>
    <PageHeader title="More" />

    <ul class="border-line/50 border-t">
      <li v-for="it in items" :key="it.to">
        <RouterLink
          :to="it.to"
          class="group border-line/50 flex items-center justify-between gap-4 border-b py-4"
        >
          <div class="flex items-center gap-3">
            <span
              :class="it.icon"
              class="text-muted group-hover:text-ink text-lg"
              aria-hidden="true"
            />
            <div>
              <div class="text-ink text-sm">{{ it.label }}</div>
              <div class="label-mono mt-0.5">{{ it.hint }}</div>
            </div>
          </div>
          <span
            class="i-lucide-chevron-right text-muted group-hover:text-accent"
            aria-hidden="true"
          />
        </RouterLink>
      </li>
    </ul>

    <!-- Session controls only exist when password auth is enabled server-side. -->
    <div v-if="auth.enabled" class="mt-8">
      <button type="button" class="btn-line text-xs" :disabled="loggingOut" @click="logout">
        <span class="i-lucide-log-out text-sm" aria-hidden="true" />
        {{ loggingOut ? 'Logging out…' : 'Log out' }}
      </button>
    </div>
  </div>
</template>
