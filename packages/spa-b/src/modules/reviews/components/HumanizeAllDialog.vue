<script setup lang="ts">
/**
 * "Humanize all" modal (U3) — replaces the always-visible header "Voice
 * profile" `<Select>` on the detail page. Opened either from the header
 * "Humanize all" button, or from a per-card Humanize click when no profile
 * is selected yet (see `[id].vue`'s `ensureProfileSelected`). Reka Dialog,
 * same primitive family/styling as `TelegramSection.vue`'s
 * DialogRoot/Portal/Overlay/Content.
 *
 * Purely presentational + the profile `v-model` — confirming is the
 * caller's job (`@confirm`, expected to call the composable's
 * `humanizeAll()` and close). No-ready-profile mirrors the old inline
 * header hint: disables the action and links to `/settings`.
 */
import { DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { Button, Field, Select, Text } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'

const open = defineModel<boolean>('open', { required: true })
const profileId = defineModel<string>('profileId', { required: true })

defineProps<{
  items: SelectItemOption[]
  hasReadyProfile: boolean
  pending: boolean
}>()

const emit = defineEmits<{
  confirm: []
}>()

function handleCancel() {
  open.value = false
}
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal>
      <DialogOverlay class="overlay z-20" />
      <DialogContent
        class="fixed left-1/2 top-1/2 z-30 w-[min(26rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
      >
        <DialogTitle class="text-md font-semibold text-text">Humanize all</DialogTitle>
        <DialogDescription class="mt-1 text-sm text-text-muted">
          Rewrites the summary and every finding in the selected profile's voice. Each rewrite lands as a new tab
          alongside the original — nothing is published to the MR until you choose to.
        </DialogDescription>

        <Field v-if="hasReadyProfile" label="Voice profile" class="mt-4" v-slot="{ id }">
          <Select :id="id" v-model="profileId" :items="items" />
        </Field>
        <Text v-else muted size="sm" class="mt-4 block">
          <RouterLink to="/settings" class="font-medium underline underline-offset-2">Add a ready profile</RouterLink>
          to humanize findings and the summary.
        </Text>

        <div class="mt-5 flex justify-end gap-2">
          <Button variant="ghost" size="sm" @click="handleCancel">Cancel</Button>
          <Button size="sm" :disabled="!hasReadyProfile || !profileId" :loading="pending" @click="emit('confirm')">
            Humanize all
          </Button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
