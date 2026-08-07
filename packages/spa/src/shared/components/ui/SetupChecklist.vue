<script setup lang="ts">
import { computed } from 'vue'

// First-run guide. The four steps a self-hoster must complete to reach the core
// loop (configure → review → publish). Each step's done-state is derived from a
// live count, so the list ticks itself off as the user makes progress; the parent
// hides the whole thing once the first review exists. The single not-yet-done
// step wears the lime "do this next" signal — everything else stays quiet, so the
// One-Signal rule holds.
const props = defineProps<{
  accounts: number
  providers: number
  repos: number
  reviews: number
}>()

interface Step {
  key: string
  label: string
  hint: string
  to: string
  cta: string
  done: boolean
}

const steps = computed<Step[]>(() => [
  {
    key: 'account',
    label: 'Connect a GitLab account',
    hint: 'A token 4R uses to read merge requests and post reviews back.',
    to: '/accounts',
    cta: 'Connect account',
    done: props.accounts > 0,
  },
  {
    key: 'provider',
    label: 'Add an AI provider',
    hint: 'The model account (OpenAI, Groq, Claude…) 4R sends diffs to.',
    to: '/providers',
    cta: 'Add provider',
    done: props.providers > 0,
  },
  {
    key: 'repo',
    label: 'Track a repository',
    hint: 'A GitLab project whose merge requests you want reviewed.',
    to: '/repos',
    cta: 'Track repository',
    done: props.repos > 0,
  },
  {
    key: 'review',
    label: 'Run your first review',
    hint: 'Open a repo, pick an open MR, and launch the 4R engine.',
    to: '/repos',
    cta: 'Start a review',
    done: props.reviews > 0,
  },
])

const doneCount = computed(() => steps.value.filter((s) => s.done).length)
// The first incomplete step is the one we point the user at next.
const nextIndex = computed(() => steps.value.findIndex((s) => !s.done))
</script>

<template>
  <section
    class="border-line bg-surface/40 border p-5"
    aria-labelledby="setup-heading"
  >
    <div class="mb-4 flex items-end justify-between gap-3">
      <div>
        <div class="label-mono">Get started</div>
        <h2 id="setup-heading" class="section-title mt-1">Set up 4R in {{ steps.length }} steps</h2>
      </div>
      <div class="label-mono shrink-0">{{ doneCount }}/{{ steps.length }} done</div>
    </div>

    <!-- Progress: a thin gauge, same idiom as the review score meter. -->
    <div class="bg-line/40 mb-4 h-0.5 w-full" role="img" :aria-label="`${doneCount} of ${steps.length} steps done`">
      <div class="bg-accent h-full transition-all" :style="{ width: `${(doneCount / steps.length) * 100}%` }" />
    </div>

    <ol>
      <li
        v-for="(step, i) in steps"
        :key="step.key"
        class="border-line/50 flex items-center gap-3 border-b py-3 last:border-b-0"
      >
        <span
          class="shrink-0 text-lg"
          :class="
            step.done
              ? 'i-lucide-circle-check-big text-ok'
              : i === nextIndex
                ? 'i-lucide-circle-dashed text-accent'
                : 'i-lucide-circle text-muted/60'
          "
          aria-hidden="true"
        />
        <div class="min-w-0 flex-1">
          <div class="text-sm" :class="step.done ? 'text-muted' : 'text-ink'">{{ step.label }}</div>
          <div v-if="!step.done" class="text-muted mt-0.5 text-xs">{{ step.hint }}</div>
        </div>
        <span v-if="step.done" class="label-mono text-ok shrink-0">Done</span>
        <RouterLink
          v-else
          :to="step.to"
          class="shrink-0 text-xs"
          :class="i === nextIndex ? 'btn-accent' : 'btn-line'"
        >
          {{ step.cta }}
          <span class="i-lucide-arrow-right text-sm" aria-hidden="true" />
        </RouterLink>
      </li>
    </ol>
  </section>
</template>
