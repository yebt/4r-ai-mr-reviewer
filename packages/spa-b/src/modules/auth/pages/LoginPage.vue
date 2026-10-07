<script setup lang="ts">
/**
 * The /login route UI. Registered manually (not file-based) in
 * src/core/router/index.ts with meta.public = true — this is a bare,
 * centered screen; the app shell does NOT wrap the login route.
 *
 * Auth surface consumed as-is (do not edit): useAuthStore().login() throws
 * on failure, errorMessage() maps the thrown error to copy, safeRedirect()
 * sanitizes the `redirect` query param before navigating.
 */
import { nextTick, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@modules/auth/store'
import { errorMessage } from '@modules/auth/errors'
import { safeRedirect } from '@modules/auth/redirect'
import { Button, Field, Heading, Input, Text } from '@shared/ui/design-system'

const PASSWORD_FIELD_ID = 'login-password'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const password = ref('')
const error = ref<string | null>(null)
const submitting = ref(false)

async function onSubmit(): Promise<void> {
  if (submitting.value) return

  error.value = null
  submitting.value = true

  try {
    await auth.login(password.value)
    const redirectParam = route.query.redirect
    const raw = Array.isArray(redirectParam) ? (redirectParam[0] ?? null) : redirectParam
    await router.replace(safeRedirect(raw))
  } catch (e) {
    error.value = errorMessage(e)
    password.value = ''
    await nextTick()
    document.getElementById(PASSWORD_FIELD_ID)?.focus()
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <main class="flex min-h-screen items-center justify-center bg-bg-app p-6">
    <div class="w-full max-w-sm rounded-xl border border-line bg-bg-panel p-8 shadow-token-md">
      <div class="mb-6 flex flex-col items-center gap-1 text-center">
        <Heading :level="1" size="xl">4R</Heading>
        <Text muted size="sm">Sign in to continue</Text>
      </div>

      <form novalidate class="flex flex-col gap-5" @submit.prevent="onSubmit">
        <Field
          :id="PASSWORD_FIELD_ID"
          label="Password"
          required
          :error="error ?? undefined"
          v-slot="{ id, describedBy, invalid }"
        >
          <Input
            :id="id"
            v-model="password"
            type="password"
            autocomplete="current-password"
            autofocus
            :readonly="submitting"
            :aria-describedby="describedBy"
            :aria-invalid="invalid"
          />
        </Field>

        <Button type="submit" variant="accent" :loading="submitting" class="w-full">
          Sign in
        </Button>
      </form>
    </div>
  </main>
</template>
