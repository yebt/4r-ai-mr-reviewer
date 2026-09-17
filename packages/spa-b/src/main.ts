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
    // change the stale time for all queries to 0ms
    staleTime: 0,
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
