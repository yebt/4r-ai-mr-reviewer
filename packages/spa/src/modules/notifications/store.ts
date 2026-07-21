import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type { NotificationRule } from '@shared/api/types'

export const useNotificationsStore = defineStore('notifications', () => {
  const rules = ref<NotificationRule[]>([])
  const events = ref<string[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      const [ev, rl] = await Promise.all([
        api.listNotificationEvents(),
        api.listNotificationRules(),
      ])
      events.value = ev.events
      rules.value = rl
    } catch (e) {
      error.value = errorMessage(e)
    } finally {
      loading.value = false
    }
  }

  async function add(input: { event: string; notifierId: string }) {
    const created = await api.createNotificationRule(input)
    rules.value = [...rules.value, created]
    return created
  }

  // Persist the toggle and adopt the returned rule as the source of truth.
  async function setEnabled(id: string, enabled: boolean) {
    const updated = await api.setNotificationRuleEnabled(id, enabled)
    rules.value = rules.value.map((r) => (r.id === id ? updated : r))
    return updated
  }

  async function remove(id: string) {
    await api.deleteNotificationRule(id)
    rules.value = rules.value.filter((r) => r.id !== id)
  }

  return { rules, events, loading, error, fetchAll, add, setEnabled, remove }
})
