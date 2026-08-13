<script setup lang="ts">
definePage({ meta: { title: 'Settings' } })
import { computed, onMounted, reactive, ref } from 'vue'
import { api, ApiError, errorMessage } from '@shared/api/client'
import { confirm } from '@shared/composables/useConfirm'
import { toast } from '@shared/composables/useToast'
import PageHeader from '@shared/components/ui/PageHeader.vue'
import EmptyState from '@shared/components/ui/EmptyState.vue'
import DependencyAlert from '@shared/components/ui/DependencyAlert.vue'
import Alert from '@shared/components/ui/Alert.vue'
import type { NotificationRule, VaultStatus } from '@shared/api/types'
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

// --- Vault / Security ---
// Current vault mode. `null` until the first status load resolves.
const vault = ref<VaultStatus | null>(null)
const vaultLoading = ref(false)
// True when the backend reports vault management is unavailable (501). The whole
// section is disabled with a note in that case.
const vaultUnavailable = ref(false)
const vaultLoadError = ref<string | null>(null)

// Change-master-key form. Passwords live only in this reactive state; they are
// never logged, persisted, or put in the URL, and are cleared after submit.
const vaultForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
  // When checked, the new-password fields are ignored and an empty newPassword
  // is submitted — switching the vault to key-file mode (no password).
  useKeyFile: false,
})
const showOld = ref(false)
const showNew = ref(false)
const vaultSubmitting = ref(false)
const vaultFormError = ref<string | null>(null)
// Persistent warning returned by a successful change (e.g. "update AIR_PASSWORD
// before the next restart"). Kept as an inline alert until the next change.
const vaultWarning = ref<string | null>(null)

const vaultReady = computed(() => vault.value !== null && !vaultUnavailable.value)
const passwordProtected = computed(() => vault.value?.passwordProtected ?? false)

async function loadVaultStatus() {
  vaultLoading.value = true
  vaultLoadError.value = null
  try {
    vault.value = await api.vaultStatus()
    vaultUnavailable.value = false
  } catch (e) {
    // 501 = vault management unavailable on this build/instance: hide the form
    // and show a note instead of an error.
    if (e instanceof ApiError && e.status === 501) {
      vaultUnavailable.value = true
      vault.value = null
    } else {
      vaultLoadError.value = errorMessage(e)
    }
  } finally {
    vaultLoading.value = false
  }
}

function clearVaultPasswords() {
  vaultForm.oldPassword = ''
  vaultForm.newPassword = ''
  vaultForm.confirmPassword = ''
}

async function submitVault() {
  if (vaultSubmitting.value || !vaultReady.value) return
  vaultFormError.value = null

  // Key-file mode submits an empty newPassword; otherwise validate the new pair.
  if (!vaultForm.useKeyFile) {
    if (!vaultForm.newPassword) {
      vaultFormError.value = 'Enter a new password, or choose key-file mode below.'
      return
    }
    if (vaultForm.newPassword !== vaultForm.confirmPassword) {
      vaultFormError.value = 'The new password and its confirmation do not match.'
      return
    }
  }
  // The current password authorizes the change while in password mode.
  if (passwordProtected.value && !vaultForm.oldPassword) {
    vaultFormError.value = 'Enter the current password to authorize this change.'
    return
  }

  vaultSubmitting.value = true
  try {
    const result = await api.changeVaultPassword({
      oldPassword: vaultForm.oldPassword,
      newPassword: vaultForm.useKeyFile ? '' : vaultForm.newPassword,
    })
    toast.success(
      result.passwordProtected ? 'Master password updated' : 'Switched to key-file mode',
    )
    // Surface the restart warning as a persistent inline alert, not just a toast.
    vaultWarning.value = result.warning ?? null
    clearVaultPasswords()
    vaultForm.useKeyFile = false
    await loadVaultStatus()
  } catch (e) {
    if (e instanceof ApiError) {
      if (e.status === 401) {
        vaultFormError.value = 'The current password is incorrect.'
      } else if (e.status === 409) {
        vaultFormError.value = 'The vault is not initialized.'
      } else if (e.status === 501) {
        vaultUnavailable.value = true
        vaultFormError.value = null
      } else {
        vaultFormError.value = errorMessage(e)
      }
    } else {
      vaultFormError.value = errorMessage(e)
    }
  } finally {
    vaultSubmitting.value = false
  }
}

onMounted(() => {
  store.fetchAll()
  loadVaultStatus()
  // Always refetch targets so a name resolved here can't go stale (e.g. a target
  // deleted in another tab) and silently show a name for a deleted notifier.
  telegram.fetchAll()
  // Repos back the optional per-repo scope selector and its row labels.
  repos.fetchAll()
})
</script>

<template>
  <div>
    <PageHeader title="Notifications" label="Settings" />

    <section>
      <p class="text-muted mb-5 max-w-2xl text-sm">
        Route an event to a Telegram target to get notified when it happens. Provider keys, GitLab
        accounts and other configuration each live on their own page.
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
          <label class="field-label" for="nt-event">
            Event <span class="text-accent" aria-hidden="true">*</span>
          </label>
          <select id="nt-event" v-model="newEvent" class="field-underline">
            <option value="" disabled>Select an event…</option>
            <option v-for="ev in store.events" :key="ev" :value="ev">{{ eventLabel(ev) }}</option>
          </select>
        </div>
        <div class="min-w-40 flex-1">
          <label class="field-label" for="nt-target">
            Telegram target <span class="text-accent" aria-hidden="true">*</span>
          </label>
          <select id="nt-target" v-model="newTarget" class="field-underline">
            <option value="" disabled>Select a target…</option>
            <option v-for="t in telegram.items" :key="t.id" :value="t.id">{{ t.name }}</option>
          </select>
        </div>
        <div class="min-w-40 flex-1">
          <label class="field-label" for="nt-repo">
            Scope <span class="text-muted normal-case">— optional</span>
          </label>
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
                :class="rule.enabled ? 'i-lucide-toggle-right text-ok' : 'i-lucide-toggle-left'"
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

    <!-- Vault / Security: master key mode + change form -->
    <section class="border-line/50 mt-12 border-t pt-8">
      <h2 class="text-ink flex items-center gap-2 text-lg font-semibold">
        <span class="i-lucide-shield-check text-muted text-base" aria-hidden="true" />
        Security
      </h2>
      <p class="text-muted mt-1 mb-5 max-w-2xl text-sm">
        Every stored secret (provider keys, GitLab tokens, bot tokens) is encrypted at rest with the
        vault's master key. Change how that key is derived below.
      </p>

      <p v-if="vaultLoading && !vault" class="text-muted py-3 text-sm">Loading vault status…</p>

      <Alert v-else-if="vaultUnavailable" variant="info" icon="i-lucide-lock">
        Vault management isn't available on this instance. The master key can only be changed from
        the server environment.
      </Alert>

      <p v-else-if="vaultLoadError" class="text-danger py-3 text-sm">{{ vaultLoadError }}</p>

      <template v-else-if="vaultReady">
        <!-- Current mode -->
        <div class="border-line/50 bg-surface/40 mb-6 flex items-start gap-3 border px-4 py-3">
          <span
            :class="passwordProtected ? 'i-lucide-lock-keyhole text-ok' : 'i-lucide-key-round text-muted'"
            class="mt-0.5 shrink-0 text-base"
            aria-hidden="true"
          />
          <div class="min-w-0 text-sm">
            <p class="text-ink font-medium">
              {{ passwordProtected ? 'Protected by a password' : 'Key-file mode (no password)' }}
            </p>
            <p class="text-muted mt-0.5 text-xs">
              <template v-if="passwordProtected">
                The master key is derived from AIR_PASSWORD, which must be supplied at every start.
              </template>
              <template v-else>
                The master key lives in a key file stored beside the database — no password is
                needed to start.
              </template>
            </p>
          </div>
        </div>

        <!-- Persistent post-change warning (must not be missed) -->
        <Alert v-if="vaultWarning" variant="warn" class="mb-6">
          {{ vaultWarning }}
        </Alert>

        <!-- Change master key -->
        <form class="flex max-w-xl flex-col gap-5" @submit.prevent="submitVault">
          <div class="label-mono border-line/50 border-b pb-2">Change the master key</div>

          <!-- Current password: only when the vault is password-protected -->
          <div v-if="passwordProtected">
            <label class="field-label" for="vault-old">
              Current password <span class="text-accent" aria-hidden="true">*</span>
            </label>
            <div class="flex items-center gap-2">
              <input
                id="vault-old"
                v-model="vaultForm.oldPassword"
                :type="showOld ? 'text' : 'password'"
                class="field-underline"
                placeholder="current vault password"
                autocomplete="current-password"
                spellcheck="false"
                aria-required="true"
              />
              <button
                type="button"
                class="btn-ghost shrink-0"
                :aria-label="showOld ? 'Hide current password' : 'Show current password'"
                :aria-pressed="showOld"
                @click="showOld = !showOld"
              >
                <span
                  :class="showOld ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                  class="text-sm"
                  aria-hidden="true"
                />
              </button>
            </div>
          </div>

          <!-- Key-file mode toggle -->
          <label class="text-muted flex cursor-pointer items-start gap-2 text-sm select-none">
            <input
              v-model="vaultForm.useKeyFile"
              type="checkbox"
              class="accent-accent mt-0.5"
            />
            <span>
              Use a key file instead (no password)
              <span class="text-muted/70 mt-0.5 block text-xs">
                Switching to key-file mode needs no restart change. Setting a password requires
                updating AIR_PASSWORD before the next restart, or the vault won't open.
              </span>
            </span>
          </label>

          <!-- New password pair: disabled in key-file mode -->
          <div :class="vaultForm.useKeyFile ? 'pointer-events-none opacity-40' : ''">
            <label class="field-label" for="vault-new">
              New password <span class="text-accent" aria-hidden="true">*</span>
            </label>
            <div class="flex items-center gap-2">
              <input
                id="vault-new"
                v-model="vaultForm.newPassword"
                :type="showNew ? 'text' : 'password'"
                class="field-underline"
                placeholder="new vault password"
                autocomplete="new-password"
                spellcheck="false"
                :disabled="vaultForm.useKeyFile"
              />
              <button
                type="button"
                class="btn-ghost shrink-0"
                :aria-label="showNew ? 'Hide new password' : 'Show new password'"
                :aria-pressed="showNew"
                :disabled="vaultForm.useKeyFile"
                @click="showNew = !showNew"
              >
                <span
                  :class="showNew ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                  class="text-sm"
                  aria-hidden="true"
                />
              </button>
            </div>
          </div>

          <div :class="vaultForm.useKeyFile ? 'pointer-events-none opacity-40' : ''">
            <label class="field-label" for="vault-confirm">
              Confirm new password <span class="text-accent" aria-hidden="true">*</span>
            </label>
            <input
              id="vault-confirm"
              v-model="vaultForm.confirmPassword"
              :type="showNew ? 'text' : 'password'"
              class="field-underline"
              placeholder="repeat the new password"
              autocomplete="new-password"
              spellcheck="false"
              :disabled="vaultForm.useKeyFile"
            />
          </div>

          <p v-if="vaultFormError" class="text-danger text-sm" role="alert">{{ vaultFormError }}</p>

          <div>
            <button type="submit" class="btn-accent" :disabled="vaultSubmitting">
              <span
                v-if="vaultSubmitting"
                class="i-lucide-loader-circle animate-spin"
                aria-hidden="true"
              />
              {{
                vaultSubmitting
                  ? 'Applying'
                  : vaultForm.useKeyFile
                    ? 'Switch to key-file mode'
                    : passwordProtected
                      ? 'Change password'
                      : 'Set a password'
              }}
            </button>
          </div>
        </form>
      </template>
    </section>
  </div>
</template>
