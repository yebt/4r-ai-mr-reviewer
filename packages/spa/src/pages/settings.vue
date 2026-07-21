<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import type { NotificationRule } from '@shared/api/types'
import { useNotificationsStore } from '@modules/notifications/store'
import { useTelegramStore } from '@modules/telegram/store'

const store = useNotificationsStore()
const telegram = useTelegramStore()

// Friendly labels for known events; unknown events fall back to their raw key.
const EVENT_LABELS: Record<string, string> = {
  'review.finished': 'Review finished',
}
function eventLabel(event: string): string {
  return EVENT_LABELS[event] ?? event
}

function targetName(notifierId: string): string | null {
  return telegram.items.find((t) => t.id === notifierId)?.name ?? null
}

const newEvent = ref('')
const newTarget = ref('')
const busyId = ref<string | null>(null)

const hasTargets = computed(() => telegram.items.length > 0)
const canAdd = computed(() => newEvent.value !== '' && newTarget.value !== '')

async function addRule() {
  if (!canAdd.value) return
  try {
    await store.add({ event: newEvent.value, notifierId: newTarget.value })
    newEvent.value = ''
    newTarget.value = ''
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
      <p v-if="!hasTargets" class="text-muted mb-6 text-sm">
        <RouterLink to="/telegram" class="text-accent hover:underline">
          Add a Telegram target first
        </RouterLink>
      </p>
      <div v-else class="mb-6 flex flex-wrap items-end gap-3">
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
    </section>
  </div>
</template>
