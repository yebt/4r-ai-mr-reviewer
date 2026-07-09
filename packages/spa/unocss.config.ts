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
  // Borderless technical-minimalist palette (see reference mockups): near-black
  // canvas, hairline dividers, mono labels, a single lime accent + a flame
  // accent for highlights/danger emphasis.
  theme: {
    colors: {
      canvas: '#0a0a0b',
      surface: '#111113',
      line: '#2a2a30',
      ink: '#ededf0',
      muted: '#7c7c85',
      accent: '#d6ff3f',
      'accent-ink': '#0a0a0b',
      flame: '#ff5a1f',
      danger: '#ff5a5a',
      ok: '#8bef8b',
    },
  },
  shortcuts: {
    // Borderless building blocks: hairlines and whitespace, no boxed cards.
    'hair': 'border-line/70',
    'row': 'flex items-center gap-4 border-b border-line/50 py-3',
    'divider': 'h-px w-full bg-line/50',

    'label-mono': 'font-mono text-[0.68rem] uppercase tracking-[0.14em] text-muted',
    'field-label': 'label-mono mb-1.5 block',
    // Form/section subtitle — clearly dominant over the muted field labels.
    'section-title': 'text-[0.95rem] font-semibold tracking-tight text-ink',
    'field-underline': 'w-full border-0 border-b border-line bg-transparent px-0 py-2 text-sm text-ink outline-none transition-colors placeholder:text-muted/50 focus:border-accent',

    'btn': 'inline-flex cursor-pointer items-center justify-center gap-2 rounded-none text-sm font-medium transition-colors disabled:opacity-40 disabled:pointer-events-none',
    'btn-accent': 'btn bg-accent px-4 py-2 text-accent-ink hover:opacity-90',
    'btn-line': 'btn border border-line px-4 py-2 text-ink hover:border-ink',
    'btn-ghost': 'btn text-muted px-2 py-1 hover:text-ink hover:bg-muted/20',
    'btn-danger-solid': 'btn bg-danger px-4 py-2 text-canvas hover:opacity-90',

    'chip': 'inline-flex items-center gap-1 font-mono text-[0.66rem] uppercase tracking-wider',
  },
})
