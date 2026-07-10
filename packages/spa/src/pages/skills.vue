<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import PageHeader from '@shared/components/ui/PageHeader.vue'

interface Skills {
  risk: string
  readability: string
  reliability: string
  resilience: string
}

const skills = ref<Skills | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

onMounted(async () => {
  try {
    skills.value = await api.getSkills()
  } catch (e) {
    error.value = errorMessage(e)
  } finally {
    loading.value = false
  }
})

const sections = computed(() =>
  skills.value
    ? [
        { key: 'R1 · Risk', text: skills.value.risk },
        { key: 'R2 · Readability', text: skills.value.readability },
        { key: 'R3 · Reliability', text: skills.value.reliability },
        { key: 'R4 · Resilience', text: skills.value.resilience },
      ]
    : [],
)
</script>

<template>
  <div>
    <PageHeader title="4R review skills" />

    <p v-if="loading" class="text-muted py-3 text-sm">Loading…</p>
    <p v-else-if="error" class="text-danger py-3 text-sm">{{ error }}</p>

    <div v-else class="flex flex-col gap-10">
      <section v-for="s in sections" :key="s.key">
        <h2 class="section-title mb-3 flex items-center gap-2">
          <span class="bg-accent inline-block h-3.5 w-0.5" aria-hidden="true" />
          {{ s.key }}
        </h2>
        <pre
          class="border-line/50 text-muted overflow-x-auto border-l pl-4 font-mono text-xs leading-relaxed whitespace-pre-wrap"
          >{{ s.text }}</pre
        >
      </section>
    </div>
  </div>
</template>
