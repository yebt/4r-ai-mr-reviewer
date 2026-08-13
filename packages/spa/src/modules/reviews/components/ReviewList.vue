<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Review } from '@shared/api/types'
import ReviewRow from '@modules/reviews/components/ReviewRow.vue'
import FilterBuilder from '@shared/components/ui/FilterBuilder.vue'
import type { ActiveFilter, FilterField } from '@shared/components/ui/filter-builder'
import { recommendationLabel, statusLabel } from '@modules/reviews/format'

const props = defineProps<{
  items: Review[]
  loading?: boolean
  error?: string | null
  // Ids with an action in flight (their buttons are disabled).
  busyIds?: string[]
}>()

defineEmits<{
  archive: [id: string]
  unarchive: [id: string]
  approve: [id: string]
  discard: [id: string, mrIid: number]
  retry: [id: string]
}>()

// --- Progressive filter builder over the reviews in this list ---
const activeFilters = ref<ActiveFilter[]>([])

// Fields are derived from what actually appears in `items`, so the picker never
// offers a Status/Model/Recommendation value that would match nothing.
const fields = computed<FilterField[]>(() => {
  const statuses = [...new Set(props.items.map((r) => r.status))]
  const models = [...new Set(props.items.map((r) => r.model).filter((m): m is string => !!m))]
  const recs = [...new Set(props.items.filter((r) => r.status === 'done').map((r) => r.recommendation))]

  const out: FilterField[] = [
    {
      key: 'status',
      label: 'Status',
      options: [
        { value: 'all', label: 'All' },
        ...statuses.map((s) => ({ value: s, label: statusLabel[s] })),
      ],
    },
  ]
  if (models.length) {
    out.push({ key: 'model', label: 'Model', multi: true, options: models.map((m) => ({ value: m, label: m })) })
  }
  if (recs.length) {
    out.push({
      key: 'recommendation',
      label: 'Recommendation',
      multi: true,
      options: recs.map((r) => ({ value: r, label: recommendationLabel(r) })),
    })
  }
  return out
})

function matchesFilters(rv: Review): boolean {
  for (const f of activeFilters.value) {
    if (f.values.length === 0) continue
    if (f.key === 'status') {
      if (!f.values.includes('all') && !f.values.includes(rv.status)) return false
    } else if (f.key === 'model') {
      if (!rv.model || !f.values.includes(rv.model)) return false
    } else if (f.key === 'recommendation') {
      // Recommendation only exists on a finished review; a rec filter excludes
      // reviews that never produced one.
      if (rv.status !== 'done' || !f.values.includes(rv.recommendation)) return false
    }
  }
  return true
}

const filtered = computed(() => props.items.filter(matchesFilters))

// Group by merge request (iid) so retry/webhook clones collapse under the latest
// attempt. Rule: filters are applied per-review FIRST, then survivors are grouped
// — so a group appears when ANY of its reviews match, and the newest surviving
// review becomes the group's headline (the rest collapse as earlier attempts).
const expandedGroups = ref<Set<string>>(new Set())
function toggleGroup(key: string) {
  const next = new Set(expandedGroups.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  expandedGroups.value = next
}

const groups = computed(() => {
  const map = new Map<string, Review[]>()
  for (const rv of filtered.value) {
    const key = String(rv.mrIid)
    const arr = map.get(key)
    if (arr) arr.push(rv)
    else map.set(key, [rv])
  }
  return [...map.entries()].map(([key, list]) => {
    const byNewest = [...list].sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1))
    const latest = byNewest[0]!
    return { key, mrIid: latest.mrIid, latest, older: byNewest.slice(1) }
  })
})

const isBusy = (id: string) => props.busyIds?.includes(id) ?? false
</script>

<template>
  <div>
    <p v-if="loading" class="text-muted py-3 text-sm">Loading reviews…</p>
    <p v-else-if="error" class="text-danger py-3 text-sm">{{ error }}</p>
    <p v-else-if="items.length === 0" class="text-muted py-3 text-sm">No reviews yet.</p>

    <template v-else>
      <!-- Filter builder + a legible count of what the filters leave visible. -->
      <div class="mb-4 flex flex-col gap-2">
        <FilterBuilder v-model="activeFilters" :fields="fields" />
        <p v-if="activeFilters.length" class="text-muted text-xs">
          Showing <span class="text-ink font-medium">{{ filtered.length }}</span> of {{ items.length }} reviews
        </p>
      </div>

      <p v-if="groups.length === 0" class="text-muted py-3 text-sm">No reviews match the filters.</p>

      <ul v-else class="border-line/50 border-t">
        <template v-for="g in groups" :key="g.key">
          <!-- Group header: the MR the reviews belong to, made prominent so the
               hierarchy reads at a glance. -->
          <li class="flex items-center gap-2 pt-3 pb-1">
            <span class="i-lucide-git-merge text-muted shrink-0 text-sm" aria-hidden="true" />
            <span class="text-ink font-mono text-sm">!{{ g.mrIid }}</span>
            <span
              v-if="g.older.length"
              class="border-line text-muted inline-flex items-center border px-1 font-mono text-[0.6rem]"
            >
              {{ g.older.length + 1 }} attempts
            </span>
          </li>

          <!-- Headline: the latest attempt for this MR. -->
          <ReviewRow
            :review="g.latest"
            :busy="isBusy(g.latest.id)"
            @approve="$emit('approve', $event)"
            @discard="(id, mrIid) => $emit('discard', id, mrIid)"
            @retry="$emit('retry', $event)"
            @archive="$emit('archive', $event)"
            @unarchive="$emit('unarchive', $event)"
          />

          <!-- Collapsed earlier attempts (retry/webhook clones). -->
          <li v-if="g.older.length" class="border-line/40 border-b">
            <button
              type="button"
              class="text-muted hover:text-ink flex w-full items-center gap-1.5 py-2 text-xs"
              :aria-expanded="expandedGroups.has(g.key)"
              @click="toggleGroup(g.key)"
            >
              <span
                :class="expandedGroups.has(g.key) ? 'i-lucide-chevron-down' : 'i-lucide-chevron-right'"
                class="text-sm"
                aria-hidden="true"
              />
              {{ expandedGroups.has(g.key) ? 'Hide' : 'Show' }} {{ g.older.length }} earlier
              {{ g.older.length === 1 ? 'attempt' : 'attempts' }} for !{{ g.mrIid }}
            </button>
          </li>

          <!-- Earlier attempts: subordinate — indented, dimmed, with a rail. -->
          <template v-if="expandedGroups.has(g.key)">
            <ReviewRow
              v-for="rv in g.older"
              :key="rv.id"
              :review="rv"
              :busy="isBusy(rv.id)"
              class="border-line/40 bg-surface/30 border-l-2 pl-4 opacity-75"
              @approve="$emit('approve', $event)"
              @discard="(id, mrIid) => $emit('discard', id, mrIid)"
              @retry="$emit('retry', $event)"
              @archive="$emit('archive', $event)"
              @unarchive="$emit('unarchive', $event)"
            />
          </template>
        </template>
      </ul>
    </template>
  </div>
</template>
