<script setup lang="ts">
definePage({ meta: { title: 'Sign in' } })
import { onMounted, ref, useTemplateRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError, errorMessage } from '@shared/api/client'
import { useAuthStore } from '@modules/auth/store'
import { safeRedirect } from '@modules/auth/redirect'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const password = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)
const passwordInput = useTemplateRef<HTMLInputElement>('passwordInput')

onMounted(() => {
  passwordInput.value?.focus()
})

async function onSubmit() {
  if (submitting.value) return
  error.value = null
  submitting.value = true
  try {
    await auth.login(password.value)
    await router.replace(safeRedirect(route.query.redirect))
  } catch (e) {
    if (e instanceof ApiError && e.status === 401) {
      error.value = 'Invalid password'
    } else if (e instanceof ApiError && e.status === 429) {
      error.value = 'Too many attempts, wait a minute'
    } else {
      error.value = errorMessage(e)
    }
    password.value = ''
    passwordInput.value?.focus()
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center px-4">
    <div class="w-full max-w-sm">
      <div class="mb-8 text-center">
        <div class="label-mono">4R · AI MR Reviewer</div>
        <h1 class="mt-1.5 text-2xl font-semibold tracking-tight">Sign in</h1>
        <p class="text-muted mt-2 text-sm">
          This self-hosted instance is password-protected — enter it to continue.
        </p>
      </div>

      <form novalidate @submit.prevent="onSubmit">
        <label class="field-label" for="login-password">Password</label>
        <input
          id="login-password"
          ref="passwordInput"
          v-model="password"
          type="password"
          name="password"
          autocomplete="current-password"
          class="field-underline"
          :aria-invalid="error !== null"
          aria-describedby="login-error"
        />

        <p
          v-if="error"
          id="login-error"
          role="alert"
          class="text-danger mt-3 text-sm"
        >
          {{ error }}
        </p>

        <button type="submit" class="btn-accent mt-6 w-full" :disabled="submitting">
          <span
            v-if="submitting"
            class="i-lucide-loader-circle animate-spin text-base"
            aria-hidden="true"
          />
          {{ submitting ? 'Signing in…' : 'Sign in' }}
        </button>
      </form>
    </div>
  </div>
</template>

<route lang="json">
{ "meta": { "public": true } }
</route>
