import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type { ProfileInput } from '@shared/api/client'
import type { Profile } from '@shared/api/types'

export type { ProfileInput }

export const useProfilesStore = defineStore('profiles', () => {
  const items = ref<Profile[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    try {
      items.value = await api.listProfiles()
    } catch (e) {
      error.value = errorMessage(e)
    } finally {
      loading.value = false
    }
  }

  async function create(input: ProfileInput) {
    const created = await api.createProfile(input)
    items.value = [created, ...items.value]
    return created
  }

  async function update(id: string, input: ProfileInput) {
    const updated = await api.updateProfile(id, input)
    items.value = items.value.map((p) => (p.id === id ? updated : p))
    return updated
  }

  async function remove(id: string) {
    await api.deleteProfile(id)
    items.value = items.value.filter((p) => p.id !== id)
  }

  // redistill re-runs style-guide distillation from the stored samples, then
  // refreshes the row so the UI observes the flip to the pending state.
  async function redistill(id: string) {
    await api.redistillProfile(id)
    await refreshOne(id)
  }

  // refreshOne silently updates a single profile in place — used while polling a
  // pending style guide until it reaches a terminal (ready/error) state.
  async function refreshOne(id: string) {
    const fresh = await api.getProfile(id)
    items.value = items.value.map((p) => (p.id === id ? fresh : p))
    return fresh
  }

  return { items, loading, error, fetchAll, create, update, remove, redistill, refreshOne }
})
