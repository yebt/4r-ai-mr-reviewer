<script setup lang="ts">
/**
 * Notification rules settings section — organism. Mirrors ProvidersSection's
 * shape: an "Add rule" row above a list of existing rules (each with an
 * On/Off Switch wired to `setEnabled` PATCH, and a Delete `ConfirmDialog`),
 * followed by an "Unrouted events" coverage panel flagging events with no
 * enabled rule — their notifications won't be delivered until one is added.
 *
 * `useReposStore` (@modules/repos) backs the optional per-repo Scope select
 * and the rule-row scope label (`scopeLabel`).
 */
import { computed, ref } from 'vue'
import { Alert, Button, ConfirmDialog, Field, Icon, Select, Skeleton, Switch, Text } from '@shared/ui/design-system'
import type { SelectItemOption } from '@shared/ui/design-system'
import { useTelegramStore } from '@modules/telegram/store'
import { useReposStore } from '@modules/repos/store'
import { unroutedEvents, useNotificationsStore } from '../store'
import type { NotificationRule } from '../types'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'

const store = useNotificationsStore()
const telegramStore = useTelegramStore()
const reposStore = useReposStore()

// Friendly labels for known events; unknown events fall back to their raw key.
const EVENT_LABELS: Record<string, string> = {
  'review.finished': 'Review finished',
  'release.finished': 'Release finished',
}
function eventLabel(event: string): string {
  return EVENT_LABELS[event] ?? event
}

function targetName(notifierId: string): string | null {
  return telegramStore.targets.find((target) => target.id === notifierId)?.name ?? null
}

/** '' = global scope ("All repositories"); otherwise the repo's name, falling
 * back to a "(deleted repo)" marker if it no longer exists. */
function scopeLabel(repoId: string): string {
  if (repoId === '') return 'All repositories'
  return reposStore.repos.find((repo) => repo.id === repoId)?.name ?? '(deleted repo)'
}

const hasTargets = computed(() => telegramStore.targets.length > 0)

const eventOptions = computed<SelectItemOption[]>(() =>
  store.events.map((event) => ({ value: event, label: eventLabel(event) })),
)
const targetOptions = computed<SelectItemOption[]>(() =>
  telegramStore.targets.map((target) => ({ value: target.id, label: target.name })),
)
const scopeOptions = computed<SelectItemOption[]>(() => [
  { value: '', label: 'All repositories' },
  ...reposStore.repos.map((repo) => ({ value: repo.id, label: repo.name })),
])

const newEvent = ref('')
const newTarget = ref('')
// '' = global scope ("All repositories"); otherwise a repo id.
const newScope = ref('')
const addError = ref<string | null>(null)
const adding = ref(false)

const canAdd = computed(() => newEvent.value !== '' && newTarget.value !== '')

async function handleAdd() {
  if (!canAdd.value) return
  adding.value = true
  addError.value = null
  try {
    await store.createRule({
      event: newEvent.value,
      notifierId: newTarget.value,
      notifierKind: 'telegram',
      repoId: newScope.value,
    })
    newEvent.value = ''
    newTarget.value = ''
    newScope.value = ''
  } catch (err) {
    addError.value = resolveErrorMessage(err, 'Failed to add rule')
  } finally {
    adding.value = false
  }
}

const togglingId = ref<string | null>(null)
async function handleToggle(rule: NotificationRule, enabled: boolean) {
  togglingId.value = rule.id
  try {
    await store.setEnabled(rule.id, enabled)
  } catch {
    // no-op — the resource's update mutation already rolled back the cache
  } finally {
    togglingId.value = null
  }
}

const deletingId = ref<string | null>(null)
async function handleDelete(rule: NotificationRule) {
  deletingId.value = rule.id
  try {
    await store.removeRule(rule.id)
  } catch {
    // no-op — store already rolled back and toasted the error
  } finally {
    deletingId.value = null
  }
}

const unrouted = computed(() => unroutedEvents(store.events, store.rules))
const showUnrouted = computed(() => store.eventsState.status === 'success' && unrouted.value.length > 0)
</script>

<template>
  <section class="flex flex-col gap-4">
    <div class="flex flex-col gap-1">
      <Text as="h2" size="xl" class="font-semibold tracking-tight">Notifications</Text>
      <Text muted size="sm">Route an event to a Telegram target to get notified when it happens.</Text>
    </div>

    <Alert v-if="!hasTargets" status="info">
      No Telegram targets configured. <RouterLink to="/settings/telegram" class="font-medium underline underline-offset-2">Add a Telegram target</RouterLink> to route notifications.
    </Alert>

    <div v-else class="flex flex-col gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3">
      <div class="flex flex-wrap items-start gap-3">
        <Field label="Event" required class="min-w-40 flex-1" v-slot="{ id, describedBy, invalid }">
          <Select
            :id="id"
            v-model="newEvent"
            :items="eventOptions"
            placeholder="Select an event…"
            :aria-describedby="describedBy"
            :aria-invalid="invalid"
          />
        </Field>
        <Field label="Telegram target" required class="min-w-40 flex-1" v-slot="{ id, describedBy, invalid }">
          <Select
            :id="id"
            v-model="newTarget"
            :items="targetOptions"
            placeholder="Select a target…"
            :aria-describedby="describedBy"
            :aria-invalid="invalid"
          />
        </Field>
        <Field label="Scope" description="Optional — defaults to all repositories." class="min-w-40 flex-1" v-slot="{ id, describedBy }">
          <Select :id="id" v-model="newScope" :items="scopeOptions" placeholder="All repositories" :aria-describedby="describedBy" />
        </Field>
      </div>
      <div class="flex justify-end">
        <Button :disabled="!canAdd" :loading="adding" @click="handleAdd">Add rule</Button>
      </div>
      <Alert v-if="addError" status="danger">{{ addError }}</Alert>
    </div>

    <div v-if="store.rulesState.status === 'pending'" class="flex flex-col gap-2" data-testid="rules-loading-skeleton">
      <div
        v-for="n in 3"
        :key="n"
        class="flex items-center justify-between gap-3 rounded-lg border border-line-subtle bg-bg-panel p-3"
      >
        <div class="flex flex-col gap-2">
          <Skeleton class="h-4 w-40" />
          <Skeleton class="h-3 w-28" />
        </div>
        <Skeleton class="h-6 w-16" />
      </div>
    </div>

    <Alert v-else-if="store.rulesState.status === 'error'" status="danger">
      <p>{{ resolveErrorMessage(store.error, 'Failed to load notification rules') }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="store.refetch()">Retry</Button>
    </Alert>

    <div
      v-else-if="store.rules.length === 0"
      class="flex flex-col items-center gap-2 rounded-lg border border-dashed border-line p-8 text-center"
    >
      <Text muted>No notification rules yet.</Text>
    </div>

    <ul v-else class="flex flex-col gap-2">
      <li
        v-for="rule in store.rules"
        :key="rule.id"
        class="flex items-center gap-3 rounded-lg border border-line-subtle bg-bg-panel px-3 py-2.5"
      >
        <div class="flex min-w-0 flex-1 flex-col gap-0.5">
          <div class="flex flex-wrap items-center gap-1.5 text-sm">
            <Text class="font-medium">{{ eventLabel(rule.event) }}</Text>
            <Text muted aria-hidden="true">→</Text>
            <Text v-if="targetName(rule.notifierId)">{{ targetName(rule.notifierId) }}</Text>
            <Text v-else muted class="italic">(deleted target)</Text>
          </div>
          <Text muted size="sm">{{ scopeLabel(rule.repoId) }}</Text>
        </div>

        <div class="flex shrink-0 items-center gap-2">
          <Switch
            :model-value="rule.enabled"
            :disabled="togglingId === rule.id"
            :aria-label="`${rule.enabled ? 'Disable' : 'Enable'} the ${eventLabel(rule.event)} rule`"
            @update:model-value="(value) => handleToggle(rule, value)"
          />
          <ConfirmDialog
            :title='`Delete the "${eventLabel(rule.event)}" rule?`'
            description="This stops notifications for this event/target/scope combination. This cannot be undone."
            confirm-label="Delete"
            danger
            :pending="deletingId === rule.id"
            @confirm="handleDelete(rule)"
          >
            <template #trigger>
              <Button
                variant="ghost"
                size="sm"
                class="text-danger-text hover:bg-danger-bg"
                :aria-label="`Delete the ${eventLabel(rule.event)} rule`"
              >
                <Icon name="trash-2" size="sm" />
              </Button>
            </template>
          </ConfirmDialog>
        </div>
      </li>
    </ul>

    <div v-if="showUnrouted" class="flex flex-col gap-2">
      <Text size="sm" class="font-medium">Unrouted events</Text>
      <Alert status="warning">
        <ul class="flex flex-col gap-1">
          <li v-for="event in unrouted" :key="event">
            {{ eventLabel(event) }} has no enabled rule — its notifications won't be delivered.
          </li>
        </ul>
      </Alert>
    </div>
  </section>
</template>
