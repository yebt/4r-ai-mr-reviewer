// Loaded before main.css so the token-based #nprogress override at the end
// of main.css (deliberately unlayered — see that file) wins the cascade
// against nprogress's own unlayered .bar/.peg rules.
import 'nprogress/nprogress.css'
import '@shared/assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from '@core/App.vue'
import router from '@core/router'
import { PiniaColada } from '@pinia/colada'

const app = createApp(App)

app.use(createPinia())
app.use(PiniaColada, {
  queryOptions: {
    // Data younger than 30s is reused as-is: switching back to a tab (window
    // focus) or remounting a view no longer refetches everything. Mutations
    // still invalidate their queries explicitly and active runs keep polling.
    staleTime: 30_000,
  },
  mutationOptions: {
    // add global mutation options here
  },
  plugins: [
    // add Pinia Colada plugins here
  ],
})
app.use(router)

app.mount('#app')
