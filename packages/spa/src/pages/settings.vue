<script setup lang="ts">
definePage({ meta: { title: 'Settings' } })
import { computed, onMounted, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import DependencyAlert from '@shared/components/ui/DependencyAlert.vue'
import type { NotificationRule } from '@shared/api/types'
import { useNotificationsStore } from '@modules/notifications/store'
import { useTelegramStore } from '@modules/telegram/store'
import { useReposStore } from '@modules/repos/store'
import { isEventRouted } from '@modules/notifications/events'

const store = useNotificationsStore()
const telegram = useTelegramStore()
const repos = useReposStore()

// Friendly labels for known events; unknown events fall back to their raw key.
const EVENT_LABELS: Record<string, string> = {
  'review.finished': 'Review finished',
  'release.finished': 'Release finished',
}
function eventLabel(event: string): string {
  return EVENT_LABELS[event] ?? event
}

function targetName(notifierId: string): string | null {
  return telegram.items.find((t) => t.id === notifierId)?.name ?? null
}

// A rule's scope label: "All repos" when global (empty repoId), otherwise the
// repo's name (falling back to a "(deleted repo)" marker if it no longer exists).
function scopeLabel(repoId: string): string {
  if (repoId === '') return 'All repos'
  return repos.items.find((r) => r.id === repoId)?.name ?? '(deleted repo)'
}

const newEvent = ref('')
const newTarget = ref('')
// '' = global scope ("All repositories"); otherwise a repo id.
const newRepo = ref('')
const busyId = ref<string | null>(null)

const hasTargets = computed(() => telegram.items.length > 0)
// Notification rules depend on a notifier (Telegram target). Guard only after the
// store settles so the warning never flashes during the initial load.
const noTargets = computed(() => !telegram.loading && telegram.items.length === 0)
const canAdd = computed(() => newEvent.value !== '' && newTarget.value !== '')

// Per available event: whether an enabled rule routes it to a notifier. Unrouted
// events are surfaced as warnings — their notifications won't be delivered.
const eventStatuses = computed(() =>
  store.events.map((event) => ({
    event,
    routed: isEventRouted(event, store.rules),
  })),
)

async function addRule() {
  if (!canAdd.value) return
  try {
    await store.add({ event: newEvent.value, notifierId: newTarget.value, repoId: newRepo.value })
    newEvent.value = ''
    newTarget.value = ''
    newRepo.value = ''
    toast.success('Notification rule added')
  } catch (e) {
    toast.error(errorMessage(e))
  }
}

async function toggle(rule: NotificationRule) {
  busyId.value = rule.id
  try {
    const updated = await store.setEnabled(rule.id, !rule.enabled)
    toast.success(updated.enabled ? 'Rule enabled' : 'Rule disabled')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    busyId.value = null
  }
}

async function removeRule(rule: NotificationRule) {
  const ok = await confirm({
    title: 'Delete notification rule',
    message: `Delete the "${eventLabel(rule.event)}" rule?`,
    danger: true,
  })
  if (!ok) return
  busyId.value = rule.id
  try {
    await store.remove(rule.id)
    toast.success('Notification rule deleted')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    busyId.value = null
  }
}

onMounted(() => {
  store.fetchAll()
  // Always refetch targets so a name resolved here can't go stale (e.g. a target
  // deleted in another tab) and silently show a name for a deleted notifier.
  telegram.fetchAll()
  // Repos back the optional per-repo scope selector and its row labels.
  repos.fetchAll()
})
</script>

<template>
  <div>
    <PageHeader title="Settings" label="Configuration" />

    <section>
      <h2 class="section-title mb-1">Notifications</h2>
      <p class="text-muted mb-5 text-sm">
        Assign an event to a Telegram target to get notified when it happens.
      </p>

      <!-- Add rule -->
      <DependencyAlert
        v-if="noTargets"
        class="mb-6"
        message="No notifier configured — add a Telegram target to route notifications."
        cta-label="Add a Telegram target"
        cta-to="/telegram"
      />
      <div v-else-if="hasTargets" class="mb-6 flex flex-wrap items-end gap-3">
        <div class="min-w-40 flex-1">
          <label class="field-label" for="nt-event">Event</label>
          <select id="nt-event" v-model="newEvent" class="field-underline">
            <option value="" disabled>Select an event…</option>
            <option v-for="ev in store.events" :key="ev" :value="ev">{{ eventLabel(ev) }}</option>
          </select>
        </div>
        <div class="min-w-40 flex-1">
          <label class="field-label" for="nt-target">Telegram target</label>
          <select id="nt-target" v-model="newTarget" class="field-underline">
            <option value="" disabled>Select a target…</option>
            <option v-for="t in telegram.items" :key="t.id" :value="t.id">{{ t.name }}</option>
          </select>
        </div>
        <div class="min-w-40 flex-1">
          <label class="field-label" for="nt-repo">Scope</label>
          <select id="nt-repo" v-model="newRepo" class="field-underline">
            <option value="">All repositories</option>
            <option v-for="r in repos.items" :key="r.id" :value="r.id">{{ r.name }}</option>
          </select>
        </div>
        <button class="btn-line shrink-0 text-xs" :disabled="!canAdd" @click="addRule">Add</button>
      </div>

      <!-- Rules list -->
      <p v-if="store.loading" class="text-muted py-3 text-sm">Loading…</p>
      <p v-else-if="store.error" class="text-danger py-3 text-sm">{{ store.error }}</p>
      <EmptyState
        v-else-if="store.rules.length === 0"
        icon="i-lucide-bell"
        title="No notification rules yet"
        hint="Assign an event to a Telegram target."
      />

      <ul v-else class="border-line/50 border-t">
        <li v-for="rule in store.rules" :key="rule.id" class="row justify-between">
          <div class="min-w-0">
            <div class="flex items-center gap-2 text-sm">
              <span class="text-ink truncate">{{ eventLabel(rule.event) }}</span>
              <span class="text-muted" aria-hidden="true">→</span>
              <span v-if="targetName(rule.notifierId)" class="text-ink truncate">
                {{ targetName(rule.notifierId) }}
              </span>
              <span v-else class="text-muted italic">(deleted target)</span>
            </div>
            <div class="text-muted mt-0.5 flex items-center gap-1 text-xs">
              <span
                :class="rule.repoId === '' ? 'i-lucide-globe' : 'i-lucide-folder-git-2'"
                class="shrink-0 text-xs"
                aria-hidden="true"
              />
              <span class="truncate">{{ scopeLabel(rule.repoId) }}</span>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-1">
            <button
              class="btn-ghost text-xs"
              :disabled="busyId === rule.id"
              :aria-pressed="rule.enabled"
              @click="toggle(rule)"
            >
              <span
                :class="rule.enabled ? 'i-lucide-toggle-right text-accent' : 'i-lucide-toggle-left'"
                class="text-base"
                aria-hidden="true"
              />
              {{ rule.enabled ? 'On' : 'Off' }}
            </button>
            <button
              class="btn-ghost hover:text-danger"
              :disabled="busyId === rule.id"
              :aria-label="`Delete ${eventLabel(rule.event)} rule`"
              @click="removeRule(rule)"
            >
              <span class="i-lucide-trash-2 text-sm" aria-hidden="true" />
            </button>
          </div>
        </li>
      </ul>

      <!-- Events status: every available event, flagged when no enabled rule
           routes it to a notifier (its notifications won't be delivered). -->
      <template v-if="!store.loading && !store.error && eventStatuses.length > 0">
        <h3 class="section-title text-muted mt-8 mb-3 flex items-center gap-2">
          <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
          Events
        </h3>
        <ul class="border-line/50 border-t">
          <li
            v-for="status in eventStatuses"
            :key="status.event"
            class="row justify-between"
            :class="status.routed ? '' : 'text-warn'"
          >
            <div class="flex min-w-0 items-center gap-2 text-sm">
              <span
                :class="status.routed ? 'i-lucide-circle-check text-ok' : 'i-lucide-triangle-alert'"
                class="shrink-0 text-sm"
                aria-hidden="true"
              />
              <span :class="status.routed ? 'text-ink' : ''" class="truncate">
                {{ eventLabel(status.event) }}
              </span>
            </div>
            <span v-if="!status.routed" class="shrink-0 text-xs" role="alert">
              no notifier — notifications for this event won't be delivered
            </span>
            <span v-else class="text-muted shrink-0 text-xs">routed</span>
          </li>
        </ul>
      </template>
    </section>
  </div>
</template>
