import { useColorMode } from '@vueuse/core'

/**
 * Light/dark/system theme switch.
 *
 * Backed by @vueuse/core's useColorMode(): persists the choice under
 * `4r-color-scheme` (the same key index.html's inline script reads before
 * paint) and toggles the `light`/`dark` class on <html>, which the token
 * layer in tokens.css and Tailwind's `dark:` variant both key off.
 */
export function useColorScheme() {
  const colorMode = useColorMode({
    storageKey: '4r-color-scheme',
    modes: {
      light: 'light',
      dark: 'dark',
    },
  })

  return { colorMode }
}
