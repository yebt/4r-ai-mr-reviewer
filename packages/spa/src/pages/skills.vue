<script setup lang="ts">
definePage({ meta: { title: '4R skills' } })
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

// The rule files are Markdown. Render a safe, minimal subset so the reference
// reads as prose (headings, lists, emphasis, inline code) instead of raw `#`/`*`
// syntax. HTML is escaped BEFORE any markup is added, and only a fixed set of
// tags is emitted, so even a custom skills dir can't inject markup — the content
// is the operator's own trusted files, but this keeps it defensive regardless.
function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}
function inlineMd(t: string): string {
  return t
    .replace(/\*\*([^*]+)\*\*/g, '<strong class="text-ink font-semibold">$1</strong>')
    .replace(/`([^`]+)`/g, '<code class="text-ink font-mono text-[0.85em]">$1</code>')
}
function renderMarkdown(md: string): string {
  const lines = escapeHtml(md).split('\n')
  const out: string[] = []
  let inList = false
  const closeList = () => {
    if (inList) {
      out.push('</ul>')
      inList = false
    }
  }
  for (const raw of lines) {
    const line = raw.replace(/\s+$/, '')
    const heading = /^(#{1,4})\s+(.*)$/.exec(line)
    const item = /^\s*[-*]\s+(.*)$/.exec(line)
    if (heading) {
      closeList()
      const level = heading[1]!.length
      const cls =
        level <= 2
          ? 'text-ink text-[0.95rem] font-semibold tracking-tight mt-5 mb-1.5'
          : 'label-mono mt-4 mb-1'
      out.push(`<p class="${cls}">${inlineMd(heading[2]!)}</p>`)
    } else if (item) {
      if (!inList) {
        out.push('<ul class="my-2 list-disc space-y-1 pl-5 text-sm text-muted">')
        inList = true
      }
      out.push(`<li>${inlineMd(item[1]!)}</li>`)
    } else if (line.trim() === '') {
      closeList()
    } else {
      closeList()
      out.push(`<p class="text-muted my-2 text-sm leading-relaxed">${inlineMd(line)}</p>`)
    }
  }
  closeList()
  return out.join('')
}
</script>

<template>
  <div>
    <PageHeader title="4R review skills" label="Engine reference" />

    <p class="text-muted mb-8 max-w-2xl text-sm">
      The rule sets the engine loads for each 4R lens. These shape what every review looks for; edit
      them via the skills directory to tune your instance.
    </p>

    <p v-if="loading" class="text-muted py-3 text-sm">Loading…</p>
    <p v-else-if="error" class="text-danger py-3 text-sm">{{ error }}</p>

    <div v-else class="flex flex-col gap-10">
      <section v-for="s in sections" :key="s.key">
        <h2 class="section-title mb-3 flex items-center gap-2">
          <span class="bg-line inline-block h-3.5 w-0.5" aria-hidden="true" />
          {{ s.key }}
        </h2>
        <!-- Rendered Markdown (trusted, escaped-first). Prose reads as sans body
             text; only inline code/tokens stay monospace. -->
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div class="border-line/50 max-w-2xl border-l pl-4" v-html="renderMarkdown(s.text)" />
      </section>
    </div>
  </div>
</template>
