<script setup lang="ts">
/**
 * Create/edit form for a single Telegram target. Used inside a Reka
 * `Dialog` by `TelegramSection`. Mirrors `modules/providers/components/
 * ProviderForm.vue`. Keyed by `target?.id ?? 'create'` in the parent so
 * switching context (create -> edit another row) always remounts with fresh
 * local state, rather than trying to reset a shared instance in place.
 */
import { nextTick, reactive, ref, computed } from 'vue'
import { Alert, Button, Checkbox, Field, Input } from '@shared/ui/design-system'
import { useTelegramStore } from '../store'
import type { TelegramTarget } from '../types'

const props = defineProps<{
  target?: TelegramTarget | null
}>()

const emit = defineEmits<{
  saved: [target: TelegramTarget]
  cancel: []
}>()

const store = useTelegramStore()

const isEditing = computed(() => props.target != null)

const form = reactive({
  name: props.target?.name ?? '',
  chatId: props.target?.chatId ?? '',
  threadId: props.target?.threadId ?? '',
  // Write-only on the server: a saved target never returns its token, so
  // edit mode always starts blank ("leave blank to keep current").
  botToken: '',
  isBot: props.target?.isBot ?? false,
})

type FormErrors = Partial<Record<'name' | 'chatId', string>>
const errors = reactive<FormErrors>({})

const formRef = ref<HTMLFormElement | null>(null)
const saving = ref(false)
const formError = ref<string | null>(null)

async function handleSubmit() {
  errors.name = form.name.trim() ? undefined : 'Name is required'
  errors.chatId = form.chatId.trim() ? undefined : 'Chat ID is required'

  const hasErrors = Object.values(errors).some((value) => value !== undefined)
  if (hasErrors) {
    await nextTick()
    formRef.value?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus()
    return
  }

  saving.value = true
  formError.value = null
  try {
    const result = props.target
      ? await store.updateTarget(props.target.id, {
          name: form.name,
          chatId: form.chatId,
          threadId: form.threadId,
          botToken: form.botToken,
          isBot: form.isBot,
        })
      : await store.createTarget({
          name: form.name,
          chatId: form.chatId,
          threadId: form.threadId,
          botToken: form.botToken,
          isBot: form.isBot,
        })
    emit('saved', result)
  } catch (err) {
    formError.value = err instanceof Error ? err.message : 'Failed to save Telegram target'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <form ref="formRef" class="flex flex-col gap-4" novalidate @submit.prevent="handleSubmit">
    <Field label="Name" required :error="errors.name" v-slot="{ id, describedBy, invalid }">
      <Input :id="id" v-model="form.name" autocomplete="off" :aria-describedby="describedBy" :aria-invalid="invalid" />
    </Field>

    <Field label="Chat ID" required :error="errors.chatId" v-slot="{ id, describedBy, invalid }">
      <Input
        :id="id"
        v-model="form.chatId"
        placeholder="-1001234567890"
        :aria-describedby="describedBy"
        :aria-invalid="invalid"
      />
    </Field>

    <Field label="Thread ID" description="Leave blank for the chat's main thread." v-slot="{ id, describedBy }">
      <Input :id="id" v-model="form.threadId" placeholder="1" :aria-describedby="describedBy" />
    </Field>

    <Field
      label="Bot token"
      :description="isEditing ? 'Leave blank to keep the current token.' : undefined"
      v-slot="{ id, describedBy }"
    >
      <Input
        :id="id"
        v-model="form.botToken"
        type="password"
        autocomplete="off"
        :placeholder="isEditing ? 'leave blank to keep current' : 'bot-token'"
        :aria-describedby="describedBy"
      />
    </Field>

    <div class="flex items-center gap-2">
      <Checkbox
        id="telegram-is-bot"
        :model-value="form.isBot"
        @update:model-value="(value) => (form.isBot = value === true)"
      />
      <label for="telegram-is-bot" class="text-sm text-text">This target is a bot</label>
    </div>

    <Alert v-if="formError" status="danger">{{ formError }}</Alert>

    <div class="flex justify-end gap-2 border-t border-line-subtle pt-4">
      <Button type="button" variant="ghost" @click="emit('cancel')">Cancel</Button>
      <Button type="submit" :loading="saving">{{ isEditing ? 'Save changes' : 'Add target' }}</Button>
    </div>
  </form>
</template>
