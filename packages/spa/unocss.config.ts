import { defineConfig, presetIcons, presetWebFonts, presetWind4, transformerDirectives, transformerVariantGroup } from 'unocss'

export default defineConfig({
  presets: [
    presetWind4(),
    presetIcons(),
    presetWebFonts(),
  ],
  transformers: [
    transformerVariantGroup(),
    transformerDirectives()
  ]
})
