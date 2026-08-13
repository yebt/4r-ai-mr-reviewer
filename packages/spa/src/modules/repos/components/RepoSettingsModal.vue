<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Modal from '@shared/components/ui/Modal.vue'
import { api, errorMessage } from '@shared/api/client'
import { toast } from '@shared/composables/useToast'
import type { Preflight, Repo } from '@shared/api/types'
import PreflightReport from '@modules/repos/components/PreflightReport.vue'
import { useReposStore } from '@modules/repos/store'
import { useProvidersStore } from '@modules/providers/store'

// The per-repo settings that used to live in the Flow cockpit's Settings tab,
// lifted into a modal opened from the workspace header gear: AI provider/model,
// the full webhook config, and the API-scope preflight. All copy and behavior is
// carried over unchanged; the component owns its own `repo` prop instead of a
// shared selection.
const props = defineProps<{ open: boolean; repo: Repo }>()
const emit = defineEmits<{ close: [] }>()

const repos = useReposStore()
const providers = useProvidersStore()

// --- AI provider + model ---
const settingsForm = ref({ providerId: '', model: '' })
function syncSettingsForm() {
  settingsForm.value = { providerId: props.repo.providerId ?? '', model: props.repo.model ?? '' }
}
// Re-sync whenever the modal opens or the repo changes, so a stale edit never
// leaks between repos or across reopens.
watch(() => [props.open, props.repo] as const, syncSettingsForm, { immediate: true })

const providerModels = computed(() => {
  const p = providers.items.find((x) => x.id === settingsForm.value.providerId)
  return [...(p?.models ?? [])].sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()))
})
const providerDirty = computed(
  () =>
    settingsForm.value.providerId !== (props.repo.providerId ?? '') ||
    settingsForm.value.model.trim() !== (props.repo.model ?? ''),
)
const savingProvider = ref(false)
async function saveProvider() {
  if (!providerDirty.value) return
  savingProvider.value = true
  try {
    await repos.assign(props.repo.id, {
      providerId: settingsForm.value.providerId,
      model: settingsForm.value.model.trim(),
    })
    toast.success('Provider updated')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    savingProvider.value = false
  }
}

// --- Webhook config ---
const webhookBusy = ref(false)
const showWebhookSecret = ref(false)
const webhookUrl = computed(() => window.location.origin + props.repo.webhookPath)
async function setWebhook(enabled: boolean, requireConfirmation: boolean, success: string) {
  if (webhookBusy.value) return
  webhookBusy.value = true
  try {
    await repos.setWebhook(props.repo.id, enabled, requireConfirmation)
    toast.success(success)
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    webhookBusy.value = false
  }
}
const toggleWebhook = () =>
  setWebhook(
    !props.repo.webhookEnabled,
    props.repo.webhookRequireConfirmation ?? false,
    props.repo.webhookEnabled ? 'Webhook disabled' : 'Webhook enabled',
  )
const toggleRequireConfirmation = () =>
  setWebhook(
    true,
    !props.repo.webhookRequireConfirmation,
    props.repo.webhookRequireConfirmation ? 'Reviews will run automatically' : 'Confirmation required',
  )
async function rotateWebhook() {
  if (webhookBusy.value) return
  webhookBusy.value = true
  try {
    await repos.rotateWebhookSecret(props.repo.id)
    toast.success('Token rotated — update it in GitLab')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    webhookBusy.value = false
  }
}
async function copyText(text: string, label: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(`${label} copied`)
  } catch {
    toast.error('Copy failed — your browser blocked clipboard access')
  }
}

// --- API scope preflight ---
const preflight = ref<Preflight | null>(null)
const preflightLoading = ref(false)
const preflightError = ref<string | null>(null)
async function runPreflight() {
  preflightLoading.value = true
  preflightError.value = null
  try {
    preflight.value = await api.preflightRepo(props.repo.id)
  } catch (e) {
    preflightError.value = errorMessage(e)
  } finally {
    preflightLoading.value = false
  }
}
// A reopened modal (possibly on a different repo) should not show a previous
// repo's preflight result.
watch(
  () => props.open,
  (open) => {
    if (open) {
      preflight.value = null
      preflightError.value = null
    }
  },
)
</script>

<template>
  <Modal :open="open" title="Repository settings" @close="emit('close')">
    <div class="flex flex-col gap-10">
      <section>
        <h3 class="section-title mb-3 flex items-center gap-2">
          <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
          AI provider
        </h3>
        <div class="flex flex-col gap-4">
          <div>
            <label class="field-label" for="fl-provider">Provider</label>
            <select id="fl-provider" v-model="settingsForm.providerId" class="field-underline">
              <option value="">Use default provider</option>
              <option v-for="p in providers.items" :key="p.id" :value="p.id">
                {{ p.name }}{{ p.isDefault ? ' (default)' : '' }}
              </option>
            </select>
          </div>
          <div>
            <label class="field-label" for="fl-model">
              Model <span class="text-muted normal-case">— optional, uses the provider's model</span>
            </label>
            <input
              id="fl-model"
              v-model="settingsForm.model"
              class="field-underline"
              list="fl-model-presets"
              placeholder="use provider's model"
              autocomplete="off"
            />
            <datalist id="fl-model-presets">
              <option v-for="m in providerModels" :key="m" :value="m" />
            </datalist>
          </div>
          <div class="flex items-center gap-3">
            <button
              class="btn-accent text-xs"
              :disabled="!providerDirty || savingProvider"
              @click="saveProvider"
            >
              <span
                v-if="savingProvider"
                class="i-lucide-loader-circle animate-spin text-sm"
                aria-hidden="true"
              />
              Save
            </button>
            <button v-if="providerDirty" class="btn-ghost text-xs" @click="syncSettingsForm">
              Reset
            </button>
          </div>
        </div>
      </section>

      <section>
        <h3 class="section-title mb-3 flex items-center gap-2">
          <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
          Webhook
        </h3>
        <div class="flex flex-col gap-4">
          <div class="row justify-between">
            <div class="min-w-0">
              <div class="text-ink text-sm">Auto-review on MR open/update</div>
              <div class="label-mono mt-0.5">
                {{ repo.webhookEnabled ? 'GitLab events trigger a review' : 'Disabled' }}
              </div>
            </div>
            <button class="btn-line text-xs" :disabled="webhookBusy" @click="toggleWebhook">
              <span
                v-if="webhookBusy"
                class="i-lucide-loader-circle animate-spin text-sm"
                aria-hidden="true"
              />
              <span
                v-else
                :class="repo.webhookEnabled ? 'i-lucide-toggle-right text-ok' : 'i-lucide-toggle-left'"
                class="text-base"
                aria-hidden="true"
              />
              {{ repo.webhookEnabled ? 'Enabled' : 'Disabled' }}
            </button>
          </div>

          <!-- When disabled there's no URL/secret yet; say so instead of leaving
               a blank section that reads as "webhook config missing". -->
          <p v-if="!repo.webhookEnabled" class="text-muted text-xs">
            Enable the webhook to get the URL and secret token to paste into GitLab → Settings →
            Webhooks.
          </p>

          <template v-if="repo.webhookEnabled">
            <div class="row justify-between">
              <div class="min-w-0">
                <div class="text-ink text-sm">Require confirmation before running</div>
                <div class="label-mono mt-0.5">
                  Webhook reviews are held for approval instead of running automatically.
                </div>
              </div>
              <button
                class="btn-line text-xs"
                :disabled="webhookBusy"
                @click="toggleRequireConfirmation"
              >
                <span
                  :class="repo.webhookRequireConfirmation ? 'i-lucide-toggle-right text-ok' : 'i-lucide-toggle-left'"
                  class="text-base"
                  aria-hidden="true"
                />
                {{ repo.webhookRequireConfirmation ? 'On' : 'Off' }}
              </button>
            </div>

            <div>
              <span class="field-label">Webhook URL</span>
              <div class="flex items-center gap-2">
                <code
                  class="border-line text-ink block flex-1 overflow-x-auto border-b px-0 py-2 font-mono text-xs"
                >
                  {{ webhookUrl }}
                </code>
                <button
                  class="btn-ghost text-xs"
                  aria-label="Copy webhook URL"
                  @click="copyText(webhookUrl, 'URL')"
                >
                  <span class="i-lucide-copy text-sm" aria-hidden="true" /> Copy
                </button>
              </div>
            </div>

            <div>
              <span class="field-label">Secret token</span>
              <div class="flex items-center gap-2">
                <code
                  class="border-line text-ink block flex-1 overflow-x-auto border-b px-0 py-2 font-mono text-xs"
                >
                  {{ showWebhookSecret ? repo.webhookSecret : '•'.repeat(24) }}
                </code>
                <button
                  class="btn-ghost text-xs"
                  :aria-label="showWebhookSecret ? 'Hide secret token' : 'Show secret token'"
                  @click="showWebhookSecret = !showWebhookSecret"
                >
                  <span
                    :class="showWebhookSecret ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                    class="text-sm"
                    aria-hidden="true"
                  />
                  {{ showWebhookSecret ? 'Hide' : 'Show' }}
                </button>
                <button
                  class="btn-ghost text-xs"
                  aria-label="Copy secret token"
                  @click="copyText(repo.webhookSecret, 'Secret token')"
                >
                  <span class="i-lucide-copy text-sm" aria-hidden="true" /> Copy
                </button>
                <button
                  class="btn-ghost text-xs"
                  :disabled="webhookBusy"
                  aria-label="Rotate secret token"
                  @click="rotateWebhook"
                >
                  <span class="i-lucide-refresh-cw text-sm" aria-hidden="true" /> Rotate
                </button>
              </div>
              <p class="text-muted mt-1.5 text-xs">
                Paste the URL and secret into GitLab → Settings → Webhooks. Rotating invalidates the
                old token until you update it there.
              </p>
            </div>
          </template>
        </div>
      </section>

      <section>
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <h3 class="section-title flex items-center gap-2">
            <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
            API scope
          </h3>
          <button class="btn-line text-xs" :disabled="preflightLoading" @click="runPreflight">
            <span
              :class="preflightLoading ? 'i-lucide-loader-circle animate-spin' : 'i-lucide-shield-check'"
              class="text-sm"
              aria-hidden="true"
            />
            {{ preflightLoading ? 'Testing…' : 'Test API scope' }}
          </button>
        </div>
        <p v-if="preflightError" class="text-danger py-1 text-sm" role="alert">
          {{ preflightError }}
        </p>
        <PreflightReport v-else-if="preflight" :report="preflight" />
        <p v-else class="text-muted text-sm">
          Check which routine and review actions this repo's token and access level permit.
        </p>
      </section>
    </div>
  </Modal>
</template>
