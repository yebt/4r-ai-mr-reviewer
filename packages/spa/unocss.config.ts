import { defineConfig, presetIcons, presetWebFonts, presetWind4, transformerDirectives, transformerVariantGroup } from 'unocss'

export default defineConfig({
  presets: [
    presetWind4(),
    presetIcons(),
    presetWebFonts(),
  ],
  transformers: [
    transformerVariantGroup(),
    transformerDirectives(),
  ],
  // Minimalist dark palette from the reference mockups: near-black canvas,
  // technical mono accents, a single acid-yellow accent.
  theme: {
    colors: {
      canvas: '#0a0a0b',
      surface: '#121214',
      'surface-2': '#1a1a1e',
      line: '#26262b',
      ink: '#e9e9ec',
      muted: '#8a8a92',
      accent: '#e5ff4f',
      'accent-ink': '#0a0a0b',
      danger: '#ff6b6b',
      warn: '#ffb84d',
      ok: '#7ee787',
    },
  },
  shortcuts: {
    'card': 'rounded-lg border border-line bg-surface',
    'btn': 'inline-flex items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none',
    'btn-accent': 'btn bg-accent text-accent-ink hover:opacity-90',
    'btn-ghost': 'btn text-muted hover:(bg-surface-2 text-ink)',
    'btn-danger': 'btn text-danger hover:bg-danger/10',
    'field-input': 'w-full rounded-md border border-line bg-surface-2 px-3 py-2 text-sm text-ink outline-none placeholder:text-muted focus:border-accent/60',
    'label-mono': 'font-mono text-[0.7rem] uppercase tracking-wider text-muted',
  },
})
