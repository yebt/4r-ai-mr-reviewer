<script setup lang="ts">
/**
 * Design-preview page — visual smoke test for the design foundation
 * (tokens, Tailwind v4, Reka UI, Shiki, icons, theme switching). Moved here
 * from `src/pages/index.vue` once the real app shell (task S1) landed; not
 * part of the product IA, kept as a dev-only reference route. See
 * DESIGN.md for the system this page proves.
 */
import { ref, computed } from 'vue'
import { Icon } from '@shared/ui/design-system'
import {
  DialogRoot,
  DialogTrigger,
  DialogPortal,
  DialogOverlay,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogClose,
  TabsRoot,
  TabsList,
  TabsTrigger,
  TabsContent,
  DropdownMenuRoot,
  DropdownMenuTrigger,
  DropdownMenuPortal,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from 'reka-ui'
import { useColorScheme } from '@shared/composables/useColorScheme'
import { useHighlightedCode } from '@shared/composables/useCodeHighlighter'

const { colorMode } = useColorScheme()

const themeOptions = [
  { value: 'light', label: 'Light', icon: 'sun' },
  { value: 'dark', label: 'Dark', icon: 'moon' },
  { value: 'auto', label: 'System', icon: 'monitor' },
] as const

const grayScale = Array.from({ length: 12 }, (_, i) => i + 1)
const accentScale = Array.from({ length: 12 }, (_, i) => i + 1)

const statusTokens = [
  { name: 'success', label: 'Success', icon: 'circle-check' },
  { name: 'warning', label: 'Warning', icon: 'triangle-alert' },
  { name: 'danger', label: 'Danger', icon: 'circle-x' },
  { name: 'info', label: 'Info', icon: 'info' },
] as const

const typeScale = [
  { token: 'text-xs', label: '12px / Caption' },
  { token: 'text-sm', label: '13px / Meta' },
  { token: 'text-base', label: '14px / Body' },
  { token: 'text-md', label: '15px / Body emphasis' },
  { token: 'text-lg', label: '17px / Subheading' },
  { token: 'text-xl', label: '20px / Heading' },
  { token: 'text-2xl', label: '24px / Page title' },
  { token: 'text-3xl', label: '30px / Display' },
] as const

const diffSample = ref(`--- a/src/lib/review-score.ts
+++ b/src/lib/review-score.ts
@@ -12,7 +12,10 @@ export function computeRiskScore(findings: Finding[]): RiskScore {
   const weighted = findings.reduce((total, finding) => {
-    return total + finding.severity
+    const weight = SEVERITY_WEIGHT[finding.severity] ?? 1
+    return total + weight * finding.confidence
   }, 0)

-  return { score: weighted, level: toLevel(weighted) }
+  const normalized = Math.min(weighted / findings.length, 1)
+  return { score: normalized, level: toLevel(normalized) }
 }`)
const diffLang = 'diff' as const
const { html: diffHtml, isLoading: diffLoading } = useHighlightedCode(diffSample, diffLang)

const activeTab = ref('overview')
const dialogOpen = ref(false)

const currentThemeIcon = computed(
  () => themeOptions.find((o) => o.value === colorMode.value)?.icon ?? 'monitor',
)
</script>

<template>
  <div class="min-h-dvh bg-bg-app text-text">
    <header
      class="sticky top-0 z-10 flex items-center justify-between gap-4 border-b border-line bg-bg-app/90 px-4 py-3 backdrop-blur-sm sm:px-8"
    >
      <div class="flex items-center gap-2">
        <div
          class="grid size-7 place-items-center rounded-md bg-accent text-text-on-accent"
          aria-hidden="true"
        >
          <Icon name="layers" class="size-4" />
        </div>
        <p class="text-md font-semibold tracking-tight">4R design foundation</p>
        <span
          class="rounded-full border border-line-subtle bg-bg-panel px-2 py-0.5 text-xs text-text-muted"
        >
          preview — dev-only reference
        </span>
      </div>

      <DropdownMenuRoot>
        <DropdownMenuTrigger
          class="flex items-center gap-2 rounded-md border border-line bg-bg-panel px-3 py-1.5 text-sm text-text transition-colors hover:bg-bg-hover focus-visible:outline-2 focus-visible:outline-focus-ring"
        >
          <Icon :name="currentThemeIcon" class="size-4" />
          <span class="capitalize">{{ colorMode === 'auto' ? 'System' : colorMode }}</span>
          <Icon name="chevron-down" class="size-3.5 text-text-muted" />
        </DropdownMenuTrigger>
        <DropdownMenuPortal>
          <DropdownMenuContent
            :side-offset="6"
            align="end"
            class="min-w-40 rounded-lg border border-line bg-bg-panel-raised p-1 shadow-token-md"
          >
            <DropdownMenuLabel class="px-2 py-1.5 text-xs text-text-muted">Theme</DropdownMenuLabel>
            <DropdownMenuSeparator class="my-1 h-px bg-line-subtle" />
            <DropdownMenuItem
              v-for="option in themeOptions"
              :key="option.value"
              class="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm outline-none data-[highlighted]:bg-accent-subtle-bg data-[highlighted]:text-accent-text-strong"
              @select="colorMode = option.value"
            >
              <Icon :name="option.icon" class="size-4" />
              {{ option.label }}
              <Icon
                v-if="colorMode === option.value"
                name="check"
                class="ml-auto size-3.5 text-accent-text"
              />
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenuPortal>
      </DropdownMenuRoot>
    </header>

    <main class="mx-auto flex max-w-5xl flex-col gap-14 px-4 py-10 sm:px-8 sm:py-14">
      <!-- Type scale -->
      <section aria-labelledby="type-scale-heading" class="flex flex-col gap-4">
        <h2 id="type-scale-heading" class="text-lg font-semibold">Type scale</h2>
        <div class="flex flex-col divide-y divide-line-subtle rounded-lg border border-line">
          <div
            v-for="step in typeScale"
            :key="step.token"
            class="flex flex-wrap items-baseline justify-between gap-x-6 gap-y-1 px-4 py-3"
          >
            <span :class="step.token" class="font-medium">Review the diff before merging</span>
            <span class="font-mono text-xs text-text-muted">{{ step.token }} — {{ step.label }}</span>
          </div>
        </div>
      </section>

      <!-- Gray scale -->
      <section aria-labelledby="gray-scale-heading" class="flex flex-col gap-4">
        <h2 id="gray-scale-heading" class="text-lg font-semibold">Gray scale</h2>
        <div class="grid grid-cols-4 gap-2 sm:grid-cols-6 lg:grid-cols-12">
          <div v-for="step in grayScale" :key="step" class="flex flex-col gap-1.5">
            <div
              class="aspect-square rounded-md border border-line-subtle"
              :style="{ backgroundColor: `var(--gray-${step})` }"
            />
            <span class="text-center font-mono text-xs text-text-muted">{{ step }}</span>
          </div>
        </div>
      </section>

      <!-- Accent scale -->
      <section aria-labelledby="accent-scale-heading" class="flex flex-col gap-4">
        <h2 id="accent-scale-heading" class="text-lg font-semibold">Accent scale (indigo)</h2>
        <div class="grid grid-cols-4 gap-2 sm:grid-cols-6 lg:grid-cols-12">
          <div v-for="step in accentScale" :key="step" class="flex flex-col gap-1.5">
            <div
              class="aspect-square rounded-md border border-line-subtle"
              :style="{ backgroundColor: `var(--accent-${step})` }"
            />
            <span class="text-center font-mono text-xs text-text-muted">{{ step }}</span>
          </div>
        </div>
      </section>

      <!-- Semantic status tokens -->
      <section aria-labelledby="status-heading" class="flex flex-col gap-4">
        <h2 id="status-heading" class="text-lg font-semibold">Semantic status tokens</h2>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="status in statusTokens"
            :key="status.name"
            class="flex flex-col gap-3 rounded-lg border border-line-subtle p-4"
            :style="{ backgroundColor: `var(--${status.name}-bg)` }"
          >
            <div class="flex items-center gap-2">
              <Icon :name="status.icon" class="size-4" :style="{ color: `var(--${status.name}-text)` }" />
              <span class="text-sm font-medium" :style="{ color: `var(--${status.name}-text)` }">
                {{ status.label }}
              </span>
            </div>
            <button
              type="button"
              class="self-start rounded-md px-3 py-1.5 text-xs font-medium text-white transition-opacity hover:opacity-90"
              :style="{ backgroundColor: `var(--${status.name}-solid)` }"
            >
              Solid action
            </button>
          </div>
        </div>
      </section>

      <!-- Reka UI primitives -->
      <section aria-labelledby="primitives-heading" class="flex flex-col gap-4">
        <h2 id="primitives-heading" class="text-lg font-semibold">Reka UI primitives</h2>

        <TabsRoot v-model="activeTab" class="flex flex-col gap-3">
          <TabsList class="flex w-fit gap-1 rounded-lg border border-line-subtle bg-bg-panel p-1">
            <TabsTrigger
              value="overview"
              class="rounded-md px-3 py-1.5 text-sm text-text-muted transition-colors data-[state=active]:bg-bg-panel-raised data-[state=active]:text-text data-[state=active]:shadow-token-sm"
            >
              Overview
            </TabsTrigger>
            <TabsTrigger
              value="diff"
              class="rounded-md px-3 py-1.5 text-sm text-text-muted transition-colors data-[state=active]:bg-bg-panel-raised data-[state=active]:text-text data-[state=active]:shadow-token-sm"
            >
              Diff
            </TabsTrigger>
          </TabsList>

          <TabsContent value="overview" class="rounded-lg border border-line-subtle bg-bg-panel p-4">
            <p class="text-sm text-text-muted">
              Tabs, dialogs, and menus are Reka UI's headless primitives — this page styles them
              entirely with the token layer above. No visual behavior is hand-rolled.
            </p>
            <DialogRoot v-model:open="dialogOpen">
              <DialogTrigger
                class="mt-3 inline-flex items-center gap-2 rounded-md bg-accent px-3 py-1.5 text-sm font-medium text-text-on-accent transition-colors hover:bg-accent-hover focus-visible:outline-2 focus-visible:outline-focus-ring"
              >
                <Icon name="settings-2" class="size-4" />
                Open settings dialog
              </DialogTrigger>
              <DialogPortal>
                <DialogOverlay class="overlay z-20" />
                <DialogContent
                  class="fixed left-1/2 top-1/2 z-30 w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-line bg-bg-panel-raised p-5 shadow-token-lg focus:outline-none"
                >
                  <DialogTitle class="text-md font-semibold">Review threshold</DialogTitle>
                  <DialogDescription class="mt-1 text-sm text-text-muted">
                    Findings below this confidence are shown but never block a merge.
                  </DialogDescription>
                  <div class="mt-4 flex items-center gap-3">
                    <input
                      type="range"
                      min="0"
                      max="100"
                      value="70"
                      class="w-full accent-[var(--accent-9)]"
                    />
                    <span class="font-mono text-sm text-text-muted">70%</span>
                  </div>
                  <div class="mt-5 flex justify-end gap-2">
                    <DialogClose
                      class="rounded-md border border-line px-3 py-1.5 text-sm text-text transition-colors hover:bg-bg-hover"
                    >
                      Cancel
                    </DialogClose>
                    <DialogClose
                      class="rounded-md bg-accent px-3 py-1.5 text-sm font-medium text-text-on-accent transition-colors hover:bg-accent-hover"
                    >
                      Save
                    </DialogClose>
                  </div>
                </DialogContent>
              </DialogPortal>
            </DialogRoot>
          </TabsContent>

          <TabsContent value="diff" class="rounded-lg border border-line-subtle bg-bg-panel p-4">
            <p class="mb-3 text-sm text-text-muted">
              Shiki, dual-theme: highlighted once, follows the app theme via CSS variables — no
              re-highlight on toggle.
            </p>
            <div v-if="diffLoading" class="text-sm text-text-muted">Highlighting…</div>
            <!-- eslint-disable-next-line vue/no-v-html -->
            <div v-else v-html="diffHtml" />
          </TabsContent>
        </TabsRoot>
      </section>
    </main>
  </div>
</template>
