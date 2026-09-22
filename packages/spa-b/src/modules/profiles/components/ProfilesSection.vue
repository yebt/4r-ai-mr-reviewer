<script setup lang="ts">
/**
 * Profiles settings section — organism. List + row actions (Redistill, Edit,
 * Delete-with-confirm) and an Add-profile Dialog. Mirrors the Providers
 * section: `components/<Section>.vue` (list) + `components/<Entity>Form.vue`
 * (create/edit, opened in a Reka Dialog), backed by a feature-scoped Pinia
 * store.
 *
 * Data fetching, caching, and mutation side effects (optimistic updates,
 * rollback, success/error toasts) all live in `../store.ts` (@pinia/colada).
 * This component only renders the three query states — loading skeleton,
 * error-with-retry, list — and wires row actions to the store's mutations.
 */
import { ref } from 'vue'
import {
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogOverlay,
  AlertDialogPortal,
  AlertDialogRoot,
  AlertDialogTitle,
  AlertDialogTrigger,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
} from 'reka-ui'
import { Badge, Button, Icon, Skeleton, Text } from '@shared/ui/design-system'
import ProfileForm from './ProfileForm.vue'
import { useProfilesStore } from '../store'
import type { Profile, StyleGuideStatus } from '../types'

// `Badge`'s status prop type isn't re-exported from the design-system
// barrel, so this mirrors its literal union locally rather than editing it.
type BadgeStatus = 'neutral' | 'success' | 'warning' | 'danger' | 'info'

const store = useProfilesStore()

const styleGuideStatusLabels: Record<StyleGuideStatus, string> = {
  '': 'No style guide',
  pending: 'Pending',
  ready: 'Ready',
  error: 'Error',
}

const styleGuideStatusBadge: Record<StyleGuideStatus, BadgeStatus> = {
  '': 'neutral',
  pending: 'warning',
  ready: 'success',
  error: 'danger',
}

const dialogOpen = ref(false)
const editingProfile = ref<Profile | null>(null)

function openCreateDialog() {
  editingProfile.value = null
  dialogOpen.value = true
}

function openEditDialog(profile: Profile) {
  editingProfile.value = profile
  dialogOpen.value = true
}

function handleSaved() {
  dialogOpen.value = false
  editingProfile.value = null
}

function handleCancel() {
  dialogOpen.value = false
  editingProfile.value = null
}

const deletingId = ref<string | null>(null)
async function handleDelete(profile: Profile) {
  deletingId.value = profile.id
  try {
    await store.removeProfile(profile.id)
  } catch {
    // no-op — store already rolled back and toasted the error
  } finally {
    deletingId.value = null
  }
}

const redistillingId = ref<string | null>(null)
async function handleRedistill(profile: Profile) {
  redistillingId.value = profile.id
  try {
    // Rollback and the error toast are handled by the store's mutation
    // hooks; this catch only prevents an unhandled rejection here.
    await store.redistillProfile(profile.id)
  } catch {
    // no-op — store already rolled back and toasted the error
  } finally {
    redistillingId.value = null
  }
}
</script>

<template>
  <section class="flex flex-col gap-4">
    <div class="flex items-center justify-between gap-3">
      <div class="flex flex-col gap-1">
        <Text as="h2" size="xl" class="font-semibold tracking-tight">Profiles</Text>
        <Text muted size="sm">Configure the writing-voice profiles used to humanize reviews.</Text>
      </div>
      <Button @click="openCreateDialog">
        <template #leading><Icon name="plus" size="sm" /></template>
        Add profile
      </Button>
    </div>

    <div
      v-if="store.profilesState.status === 'pending'"
      class="flex flex-col gap-2"
      data-testid="profiles-loading-skeleton"
    >
      <div
        v-for="n in 3"
        :key="n"
        class="flex items-center justify-between gap-3 rounded-lg border border-line bg-bg-panel p-3"
      >
        <div class="flex flex-col gap-2">
          <Skeleton class="h-4 w-40" />
          <Skeleton class="h-3 w-28" />
        </div>
        <Skeleton class="h-8 w-20" />
      </div>
    </div>

    <div
      v-else-if="store.profilesState.status === 'error'"
      role="alert"
      class="flex flex-col items-start gap-2 rounded-md border border-danger-solid/30 bg-danger-bg p-3 text-sm text-danger-text"
    >
      <p>{{ store.error?.message ?? 'Failed to load profiles' }}</p>
      <Button variant="outline" size="sm" @click="store.refetch()">Retry</Button>
    </div>

    <div
      v-else-if="store.profiles.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Icon name="users" size="lg" class="text-text-muted" />
      <Text muted>No profiles configured yet.</Text>
      <Button variant="outline" size="sm" @click="openCreateDialog">Add your first profile</Button>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="profile in store.profiles"
        :key="profile.id"
        class="flex flex-col gap-3 rounded-lg border border-line bg-bg-panel p-3 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="flex flex-col gap-1">
          <div class="flex flex-wrap items-center gap-2">
            <Text class="font-medium">{{ profile.name }}</Text>
            <Badge :status="styleGuideStatusBadge[profile.styleGuideStatus]">
              {{ styleGuideStatusLabels[profile.styleGuideStatus] }}
            </Badge>
          </div>
          <Text muted size="sm">{{ profile.language }}</Text>
          <Text v-if="profile.styleGuideStatus === 'error' && profile.styleGuideError" size="xs" class="text-danger-text">
            {{ profile.styleGuideError }}
          </Text>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            :loading="redistillingId === profile.id"
            @click="handleRedistill(profile)"
          >
            Redistill
          </Button>
          <Button variant="ghost" size="sm" @click="openEditDialog(profile)">Edit</Button>

          <AlertDialogRoot>
            <AlertDialogTrigger as-child>
              <Button variant="ghost" size="sm" class="text-danger-text hover:bg-danger-bg">Delete</Button>
            </AlertDialogTrigger>
            <AlertDialogPortal>
              <AlertDialogOverlay class="overlay z-20" />
              <AlertDialogContent
                class="fixed left-1/2 top-1/2 z-30 w-[min(24rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
              >
                <AlertDialogTitle class="text-md font-semibold">Delete "{{ profile.name }}"?</AlertDialogTitle>
                <AlertDialogDescription class="mt-1 text-sm text-text-muted">
                  This removes the profile and its style guide. This cannot be undone.
                </AlertDialogDescription>
                <div class="mt-5 flex justify-end gap-2">
                  <AlertDialogCancel
                    class="rounded-md border border-line px-3 py-1.5 text-sm text-text transition-colors hover:bg-bg-hover"
                  >
                    Cancel
                  </AlertDialogCancel>
                  <AlertDialogAction
                    class="rounded-md bg-danger-solid px-3 py-1.5 text-sm font-medium text-white transition-colors hover:opacity-90"
                    :disabled="deletingId === profile.id"
                    @click="handleDelete(profile)"
                  >
                    Delete
                  </AlertDialogAction>
                </div>
              </AlertDialogContent>
            </AlertDialogPortal>
          </AlertDialogRoot>
        </div>
      </li>
    </ul>

    <DialogRoot v-model:open="dialogOpen">
      <DialogPortal>
        <DialogOverlay class="overlay z-20" />
        <DialogContent
          class="fixed left-1/2 top-1/2 z-30 max-h-[85vh] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
        >
          <DialogTitle class="text-md font-semibold">
            {{ editingProfile ? 'Edit profile' : 'Add profile' }}
          </DialogTitle>
          <DialogDescription class="mt-1 text-sm text-text-muted">
            {{
              editingProfile
                ? "Update this profile's writing-voice settings."
                : 'Create a new writing-voice profile for this workspace.'
            }}
          </DialogDescription>
          <ProfileForm
            :key="editingProfile?.id ?? 'create'"
            class="mt-4"
            :profile="editingProfile"
            @saved="handleSaved"
            @cancel="handleCancel"
          />
        </DialogContent>
      </DialogPortal>
    </DialogRoot>
  </section>
</template>
