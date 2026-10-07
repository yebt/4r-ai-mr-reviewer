<script setup lang="ts">
/**
 * Security settings card — vault/master-key management. Renders nothing but
 * an info Alert when the backend reports vault management is unavailable
 * (the store's `['vault-status']` query turns a 501 into an
 * `{ unavailable: true }` sentinel rather than a thrown query error).
 * Otherwise shows the current mode (password-protected vs key-file) and a
 * change-master-key form.
 *
 * No optimistic update on submit (see `store.ts`) — this is
 * security-sensitive, so the UI waits for the real server response. A
 * returned `result.warning` (e.g. "update AIR_PASSWORD before the next
 * restart") is kept as a PERSISTENT inline Alert, not just a transient
 * toast — missing that instruction could lock the user out after a restart.
 */
import { computed, reactive, ref } from 'vue'
import { Alert, Badge, Button, Checkbox, Field, Icon, Input, Skeleton, Text } from '@shared/ui/design-system'
import { useToast } from '@shared/composables/useToast'
import { resolveVaultErrorMessage, useVaultStore } from '../store'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'

const store = useVaultStore()
const toast = useToast()

// Passwords live only in this reactive state; never logged, persisted, or
// put in the URL, and cleared right after a successful submit.
const form = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
  // When true, the new-password fields are dimmed/disabled and an empty
  // newPassword is submitted — switching the vault to key-file mode.
  useKeyFile: false,
})
const showOld = ref(false)
const showNew = ref(false)
const submitting = ref(false)
const formError = ref<string | null>(null)
// Persistent warning from a successful change — kept until the next change.
const warning = ref<string | null>(null)

const passwordProtected = computed(() => store.status?.passwordProtected ?? false)

const submitLabel = computed(() => {
  if (form.useKeyFile) return 'Switch to key-file mode'
  return passwordProtected.value ? 'Change password' : 'Set a password'
})

const confirmMismatch = computed(
  () => !form.useKeyFile && form.confirmPassword !== '' && form.newPassword !== form.confirmPassword,
)

const canSubmit = computed(() => {
  if (form.useKeyFile) return true
  if (!form.newPassword || confirmMismatch.value) return false
  if (passwordProtected.value && !form.oldPassword) return false
  return true
})

function resetPasswordFields() {
  form.oldPassword = ''
  form.newPassword = ''
  form.confirmPassword = ''
}

async function handleSubmit() {
  formError.value = null

  if (!form.useKeyFile) {
    if (!form.newPassword) {
      formError.value = 'Enter a new password, or choose key-file mode below.'
      return
    }
    if (form.newPassword !== form.confirmPassword) {
      formError.value = 'The new password and its confirmation do not match.'
      return
    }
  }
  if (passwordProtected.value && !form.oldPassword) {
    formError.value = 'Enter the current password to authorize this change.'
    return
  }

  submitting.value = true
  try {
    const result = await store.changePassword({
      oldPassword: form.oldPassword,
      newPassword: form.useKeyFile ? '' : form.newPassword,
    })
    toast.success(result.passwordProtected ? 'Password changed' : 'Switched to key-file mode')
    warning.value = result.warning ?? null
    resetPasswordFields()
    form.useKeyFile = false
  } catch (err) {
    formError.value = resolveVaultErrorMessage(err)
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section class="flex flex-col gap-4">
    <div class="flex flex-col gap-1">
      <Text as="h2" size="xl" class="font-semibold tracking-tight">Security</Text>
      <Text muted size="sm">Manage this workspace's master key / vault protection.</Text>
    </div>

    <div v-if="store.statusState.status === 'pending'" class="flex flex-col gap-2" data-testid="vault-loading-skeleton">
      <Skeleton class="h-4 w-48" />
      <Skeleton class="h-24 w-full" />
    </div>

    <Alert v-else-if="store.unavailable" status="info">Vault management isn't available on this instance.</Alert>

    <Alert v-else-if="store.statusState.status === 'error'" status="danger">
      <p>{{ resolveErrorMessage(store.error, 'Failed to load vault status') }}</p>
      <Button variant="outline" size="sm" class="mt-2" @click="store.refetch()">Retry</Button>
    </Alert>

    <div v-else class="flex flex-col gap-4 rounded-lg border border-line-subtle bg-bg-panel p-4">
      <div class="flex items-center gap-2">
        <Text size="sm">Current mode:</Text>
        <Badge :status="passwordProtected ? 'info' : 'neutral'">
          {{ passwordProtected ? 'Password-protected' : 'Key-file' }}
        </Badge>
      </div>

      <Alert v-if="warning" status="warning" dismissible @dismiss="warning = null">{{ warning }}</Alert>

      <form class="flex flex-col gap-3" @submit.prevent="handleSubmit">
        <Field v-if="passwordProtected" label="Current password" required v-slot="{ id, describedBy, invalid }">
          <div class="flex items-center gap-2">
            <Input
              :id="id"
              v-model="form.oldPassword"
              :type="showOld ? 'text' : 'password'"
              :aria-describedby="describedBy"
              :aria-invalid="invalid"
              class="flex-1"
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              :aria-label="showOld ? 'Hide password' : 'Show password'"
              :aria-pressed="showOld"
              @click="showOld = !showOld"
            >
              <Icon :name="showOld ? 'eye-off' : 'eye'" size="sm" />
            </Button>
          </div>
        </Field>

        <label class="flex items-center gap-2 text-sm text-text">
          <Checkbox v-model="form.useKeyFile" />
          Use a key file instead of a password
        </label>

        <template v-if="!form.useKeyFile">
          <Field label="New password" required v-slot="{ id, describedBy, invalid }">
            <div class="flex items-center gap-2">
              <Input
                :id="id"
                v-model="form.newPassword"
                :type="showNew ? 'text' : 'password'"
                :aria-describedby="describedBy"
                :aria-invalid="invalid"
                class="flex-1"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                :aria-label="showNew ? 'Hide password' : 'Show password'"
                :aria-pressed="showNew"
                @click="showNew = !showNew"
              >
                <Icon :name="showNew ? 'eye-off' : 'eye'" size="sm" />
              </Button>
            </div>
          </Field>
          <Field
            label="Confirm new password"
            required
            :error="confirmMismatch ? 'Passwords do not match' : undefined"
            v-slot="{ id, describedBy, invalid }"
          >
            <Input
              :id="id"
              v-model="form.confirmPassword"
              type="password"
              :aria-describedby="describedBy"
              :aria-invalid="invalid"
            />
          </Field>
        </template>

        <Alert v-if="formError" status="danger">{{ formError }}</Alert>

        <Button type="submit" :disabled="!canSubmit" :loading="submitting" class="self-start">
          {{ submitLabel }}
        </Button>
      </form>
    </div>
  </section>
</template>
