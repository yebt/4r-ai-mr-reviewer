<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, useId, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useEventListener } from '@vueuse/core'
import { useReposStore } from '@modules/repos/store'
import { useReviewsStore } from '@modules/reviews/store'
import { statusLabel } from '@modules/reviews/format'
import { useCommandPalette } from '@shared/composables/useCommandPalette'
import { useFlowRepo } from '@shared/composables/useFlowRepo'

// Global command palette (Cmd/Ctrl+K). Essential scope: navigate to any page and
// jump straight to a loaded review or a tracked repo by typing. Built on the same
// borderless instrument-panel idioms as the rest of the app; the active row wears
// the sidebar's lime left-rule so "where am I" reads identically everywhere.
//
// A11y: an ARIA combobox — the search input keeps focus, arrow keys move a virtual
// selection via aria-activedescendant over a role="listbox", Enter activates, Esc
// closes and returns focus. Background is inert while open (matches Modal.vue).

interface Command {
  id: string
  group: 'Pages' | 'Reviews' | 'Repos'
  label: string
  hint: string
  icon: string
  to: string
  // When set, activating this command loads the repo into the Flow workspace.
  flowRepoId?: string
}

const PAGES: Array<{ label: string; to: string; icon: string }> = [
  { label: 'Quick access', to: '/', icon: 'i-lucide-zap' },
  { label: 'Flow', to: '/flow', icon: 'i-lucide-layout-panel-left' },
  { label: 'Repositories', to: '/repos', icon: 'i-lucide-folder-git-2' },
  { label: 'Reviews', to: '/reviews', icon: 'i-lucide-list-checks' },
  { label: 'Actions', to: '/actions', icon: 'i-lucide-git-merge' },
  { label: 'Accounts', to: '/accounts', icon: 'i-lucide-users' },
  { label: 'AI providers', to: '/providers', icon: 'i-lucide-cpu' },
  { label: 'Telegram', to: '/telegram', icon: 'i-lucide-send' },
  { label: 'Profiles', to: '/profiles', icon: 'i-lucide-feather' },
  { label: 'Skills', to: '/skills', icon: 'i-lucide-book-open' },
  { label: 'Settings', to: '/settings', icon: 'i-lucide-settings' },
]

const router = useRouter()
const repos = useReposStore()
const reviews = useReviewsStore()

const { open, toggle, hide } = useCommandPalette()
const { repoId: flowRepo } = useFlowRepo()
const query = ref('')
const activeIndex = ref(0)
const input = ref<HTMLInputElement | null>(null)
const listboxId = useId()
let previouslyFocused: HTMLElement | null = null

function setBackgroundInert(inert: boolean) {
  const app = document.getElementById('app')
  if (!app) return
  if (inert) app.setAttribute('inert', '')
  else app.removeAttribute('inert')
}

// The full command set: static pages + whatever reviews/repos are already loaded.
// (Reviews cover the app's loaded set; there is no global review index, so a repo
// never visited this session may not appear until its reviews load.)
const commands = computed<Command[]>(() => {
  const pages: Command[] = PAGES.map((p) => ({
    id: `page:${p.to}`,
    group: 'Pages',
    label: p.label,
    hint: p.to,
    icon: p.icon,
    to: p.to,
  }))
  const revs: Command[] = reviews.allReviews.map((rv) => ({
    id: `review:${rv.id}`,
    group: 'Reviews',
    label: `Review !${rv.mrIid}`,
    hint: `${rv.id.slice(0, 8)} · ${statusLabel[rv.status] ?? rv.status}`,
    icon: 'i-lucide-scan-search',
    to: `/reviews/${rv.id}`,
  }))
  const reps: Command[] = repos.items.map((r) => ({
    id: `repo:${r.id}`,
    group: 'Repos',
    label: r.name,
    hint: 'Open in Flow',
    icon: 'i-lucide-folder-git-2',
    to: '/flow',
    flowRepoId: r.id,
  }))
  return [...pages, ...revs, ...reps]
})

const q = computed(() => query.value.trim().toLowerCase())

// Filter by substring across label + hint. With no query, cap the entity groups so
// the panel stays a glanceable launcher rather than a full dump of every review.
const results = computed<Command[]>(() => {
  const matched = commands.value.filter(
    (c) => !q.value || `${c.label} ${c.hint}`.toLowerCase().includes(q.value),
  )
  if (q.value) return matched.slice(0, 40)
  const take = (group: Command['group'], n: number) =>
    matched.filter((c) => c.group === group).slice(0, n)
  return [...take('Pages', PAGES.length), ...take('Reviews', 5), ...take('Repos', 6)]
})

// Grouped view for rendering; results stays flat for keyboard indexing.
const groups = computed(() =>
  (['Pages', 'Reviews', 'Repos'] as const)
    .map((name) => ({ name, items: results.value.filter((c) => c.group === name) }))
    .filter((g) => g.items.length > 0),
)

const activeId = computed(() => results.value[activeIndex.value]?.id ?? '')

watch(results, () => {
  activeIndex.value = 0
})

function move(delta: number) {
  const n = results.value.length
  if (n === 0) return
  activeIndex.value = (activeIndex.value + delta + n) % n
  // Keep the active row in view within the scrolling list.
  nextTick(() => {
    document
      .getElementById(`cmd-${activeId.value}`)
      ?.scrollIntoView({ block: 'nearest' })
  })
}

function activate(cmd: Command | undefined) {
  if (!cmd) return
  hide()
  // A repo command loads that repo into the Flow workspace.
  if (cmd.flowRepoId) flowRepo.value = cmd.flowRepoId
  if (router.currentRoute.value.fullPath !== cmd.to) void router.push(cmd.to)
}

// Setup/teardown follows the shared open-state, so the hotkey and any launcher
// button both go through here.
watch(open, (isOpen) => {
  if (isOpen) {
    previouslyFocused = document.activeElement as HTMLElement | null
    query.value = ''
    activeIndex.value = 0
    // Repos back the search; load them once if this session hasn't yet.
    if (repos.items.length === 0) void repos.fetchAll()
    setBackgroundInert(true)
    nextTick(() => input.value?.focus())
  } else {
    setBackgroundInert(false)
    previouslyFocused?.focus?.()
    previouslyFocused = null
  }
})

function onInputKeydown(e: KeyboardEvent) {
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      move(1)
      break
    case 'ArrowUp':
      e.preventDefault()
      move(-1)
      break
    case 'Home':
      e.preventDefault()
      activeIndex.value = 0
      break
    case 'End':
      e.preventDefault()
      activeIndex.value = Math.max(0, results.value.length - 1)
      break
    case 'Enter':
      e.preventDefault()
      activate(results.value[activeIndex.value])
      break
    case 'Escape':
      e.preventDefault()
      hide()
      break
  }
}

// Global hotkey: Cmd/Ctrl+K toggles the palette from anywhere.
useEventListener(window, 'keydown', (e: KeyboardEvent) => {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    toggle()
  }
})

// Safety net: never leave the app inert if this unmounts while open.
onUnmounted(() => setBackgroundInert(false))
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-start justify-center px-4 pt-[12vh]"
      >
        <div class="bg-canvas/80 absolute inset-0 backdrop-blur-sm" @click="hide" />

        <div
          class="border-line bg-surface relative flex max-h-[70vh] w-full max-w-xl flex-col overflow-hidden border shadow-lg shadow-black/40"
          role="dialog"
          aria-modal="true"
          aria-label="Command palette"
        >
          <!-- Search row: borderless, a single hairline under it. -->
          <div class="border-line/60 flex items-center gap-3 border-b px-4">
            <span class="i-lucide-search text-muted shrink-0 text-sm" aria-hidden="true" />
            <input
              ref="input"
              v-model="query"
              type="text"
              role="combobox"
              aria-expanded="true"
              :aria-controls="listboxId"
              :aria-activedescendant="activeId ? `cmd-${activeId}` : undefined"
              aria-autocomplete="list"
              aria-label="Search pages, reviews and repositories"
              placeholder="Search pages, reviews, repos…"
              autocomplete="off"
              spellcheck="false"
              class="text-ink placeholder:text-muted/60 w-full bg-transparent py-3.5 text-sm outline-none"
              @keydown="onInputKeydown"
            />
            <kbd
              class="border-line text-muted hidden shrink-0 border px-1.5 py-0.5 font-mono text-[0.6rem] sm:block"
            >
              ESC
            </kbd>
          </div>

          <!-- Results: grouped listbox. Active row wears the lime left-rule. -->
          <ul
            v-if="results.length"
            :id="listboxId"
            role="listbox"
            aria-label="Results"
            class="min-h-0 flex-1 overflow-y-auto py-1"
          >
            <template v-for="group in groups" :key="group.name">
              <li class="label-mono px-4 pt-3 pb-1.5" role="presentation">{{ group.name }}</li>
              <li
                v-for="cmd in group.items"
                :id="`cmd-${cmd.id}`"
                :key="cmd.id"
                role="option"
                :aria-selected="activeId === cmd.id"
                class="flex cursor-pointer items-center gap-3 border-l-2 px-4 py-2 text-sm"
                :class="
                  activeId === cmd.id
                    ? 'border-accent bg-accent/10 text-ink'
                    : 'text-muted border-transparent'
                "
                @click="activate(cmd)"
                @mousemove="activeIndex = results.indexOf(cmd)"
              >
                <span :class="cmd.icon" class="shrink-0 text-sm opacity-80" aria-hidden="true" />
                <span class="min-w-0 flex-1 truncate">{{ cmd.label }}</span>
                <span class="text-muted shrink-0 truncate font-mono text-[0.68rem]">
                  {{ cmd.hint }}
                </span>
              </li>
            </template>
          </ul>

          <!-- Empty: no matches for the query. -->
          <div v-else class="text-muted px-4 py-10 text-center text-sm">
            No matches for “{{ query }}”.
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
