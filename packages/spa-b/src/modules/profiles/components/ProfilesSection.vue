<script setup lang="ts">
/**
 * Profiles settings section — organism. Dense list rows (title — muted
 * metadata — actions) with the secondary/destructive row actions (Edit,
 * Delete) collapsed into a Reka `DropdownMenu` triggered by a trailing `⋯`
 * icon-button, keeping only the single standout primary action (Redistill)
 * inline. Mirrors the Providers section: `components/<Section>.vue` (list)
 * + `components/<Entity>Form.vue` (create/edit, opened in a Reka Dialog),
 * backed by a feature-scoped Pinia store.
 *
 * Data fetching, caching, and mutation side effects (optimistic updates,
 * rollback, success/error toasts) all live in `../store.ts` (@pinia/colada).
 * This component only renders the three query states — loading skeleton,
 * error-with-retry, list — and wires row actions to the store's mutations.
 */
import { ref } from 'vue'
import {
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from 'reka-ui'
import { Alert, Badge, Button, ConfirmDialog, Fab, Icon, Skeleton, Text } from '@shared/ui/design-system'
import ProfileForm from './ProfileForm.vue'
import { useProfilesStore } from '../store'
import type { Profile, StyleGuideStatus } from '../types'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'

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
      <div class="flex min-w-0 flex-col gap-1">
        <Text as="h2" size="xl" class="font-semibold tracking-tight">Profiles</Text>
        <Text muted size="sm" class="truncate">Configure the writing-voice profiles used to humanize reviews.</Text>
      </div>
      <Button class="max-md:hidden whitespace-nowrap" @click="openCreateDialog">
        <template #leading><Icon name="plus" size="sm" /></template>
        Add profile
      </Button>
    </div>

    <Fab label="Add profile" @click="openCreateDialog" />

    <div
      v-if="store.profilesState.status === 'pending'"
      class="flex flex-col gap-2"
      data-testid="profiles-loading-skeleton"
    >
      <div
        v-for="n in 3"
        :key="n"
        class="flex items-center justify-between gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
      >
        <div class="flex flex-col gap-2">
          <Skeleton class="h-4 w-40" />
          <Skeleton class="h-3 w-28" />
        </div>
        <Skeleton class="h-8 w-20" />
      </div>
    </div>

    <Alert v-else-if="store.profilesState.status === 'error'" status="danger">
      <p>{{ resolveErrorMessage(store.error, 'Failed to load profiles') }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="store.refetch()">Retry</Button>
    </Alert>

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
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <div class="flex flex-wrap items-center gap-2">
            <Text class="truncate font-medium">{{ profile.name }}</Text>
            <Badge :status="styleGuideStatusBadge[profile.styleGuideStatus]">
              {{ styleGuideStatusLabels[profile.styleGuideStatus] }}
            </Badge>
          </div>
          <Text muted size="sm" class="truncate">{{ profile.language }}</Text>
          <Text v-if="profile.styleGuideStatus === 'error' && profile.styleGuideError" size="xs" class="truncate text-danger-text">
            {{ profile.styleGuideError }}
          </Text>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <Button
            variant="outline"
            size="sm"
            :loading="redistillingId === profile.id"
            @click="handleRedistill(profile)"
          >
            Redistill
          </Button>

          <DropdownMenuRoot>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="sm" :aria-label="`More actions for ${profile.name}`">
                <Icon name="ellipsis" size="sm" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuPortal>
              <DropdownMenuContent
                align="end"
                :side-offset="4"
                class="z-30 min-w-40 rounded-md border border-line bg-bg-panel-raised p-1 shadow-token-lg"
              >
                <DropdownMenuItem
                  class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-text outline-none data-[highlighted]:bg-bg-hover"
                  @select="openEditDialog(profile)"
                >
                  Edit
                </DropdownMenuItem>
                <DropdownMenuSeparator class="my-1 h-px bg-line-subtle" />
                <ConfirmDialog
                  :title='`Delete "${profile.name}"?`'
                  description="This removes the profile and its style guide. This cannot be undone."
                  confirm-label="Delete"
                  danger
                  :pending="deletingId === profile.id"
                  @confirm="handleDelete(profile)"
                >
                  <template #trigger>
                    <DropdownMenuItem
                      class="flex min-h-8 cursor-pointer items-center rounded-sm px-2 text-sm text-danger-text outline-none data-[highlighted]:bg-danger-bg"
                      @select.prevent
                    >
                      Delete
                    </DropdownMenuItem>
                  </template>
                </ConfirmDialog>
              </DropdownMenuContent>
            </DropdownMenuPortal>
          </DropdownMenuRoot>
        </div>
      </li>
    </ul>

    <DialogRoot v-model:open="dialogOpen">
      <DialogPortal>
        <DialogOverlay class="overlay z-20" />
        <DialogContent
          class="fixed left-1/2 top-1/2 z-30 max-h-[85svh] w-[min(32rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
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
