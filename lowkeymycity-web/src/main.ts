import { createHead } from '@unhead/vue/client'
import { createWebHistory } from 'vue-router'

import { createApp } from '@/app-factory'

// Deliberately client-render rather than hydrate. The prerendered HTML
// exists for crawlers and first paint; mounting fresh (Vue clears #app and
// re-renders identical markup) avoids hydration mismatches from <Teleport>
// and dev-vs-prod env differences, which would otherwise blank the page.
const { app } = createApp({ hydrate: false, history: createWebHistory() })
app.use(createHead())
app.mount('#app')
