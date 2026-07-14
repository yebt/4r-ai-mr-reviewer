// Guided profile builder: instead of asking users to paste raw writing samples
// (which nobody knows how to fill), we ask them to answer a handful of concrete
// review scenarios in their own voice. Each answer becomes one entry in the
// profile's samples[] — the exact same contract the distillation pipeline
// already consumes. This module is pure and framework-free so the assembly
// logic can be unit-tested in isolation.

export interface Scenario {
  /** Stable key used to index the answers record. */
  key: string
  /** Short label shown above the field. */
  label: string
  /** Muted hint used as the field placeholder. */
  hint: string
}

// The six approved scenarios, in the order their answers are assembled into
// samples[]. Order is significant — answersToSamples preserves it.
export const SCENARIOS: Scenario[] = [
  {
    key: 'minor-critique',
    label: 'Point out a small issue',
    hint: 'How you tell someone a name or function is unclear.',
  },
  {
    key: 'blocking-critique',
    label: 'Flag something serious',
    hint: 'How you call out a hardcoded secret or a security bug.',
  },
  {
    key: 'praise',
    label: 'Acknowledge good work',
    hint: 'How you compliment a solid change or refactor.',
  },
  {
    key: 'request',
    label: 'Ask for a missing test',
    hint: "How you request a test that's missing.",
  },
  {
    key: 'reasoning',
    label: 'Explain the why',
    hint: 'How you explain the reason behind a change you suggest, not just the what.',
  },
  {
    key: 'opener',
    label: 'Your typical opener (optional)',
    hint: 'A phrase you often start a comment with.',
  },
]

// Assemble scenario answers into the profile's samples[]. Walks SCENARIOS in
// order, trims each answer, and keeps only the non-empty ones. Pure: no
// mutation of the input, no side effects.
export function answersToSamples(answers: Record<string, string>): string[] {
  const samples: string[] = []
  for (const scenario of SCENARIOS) {
    const answer = answers[scenario.key]?.trim()
    if (answer) samples.push(answer)
  }
  return samples
}
