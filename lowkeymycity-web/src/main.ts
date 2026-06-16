import { createHead } from '@unhead/vue/client'
import { createWebHistory } from 'vue-router'

import { createApp } from '@/app-factory'

// Prerendered routes ship real markup inside #app, so hydrate those; the
// SPA fallback (app.html) ships an empty shell, so mount fresh there.
const root = document.getElementById('app')!
const { app } = createApp({ hydrate: root.hasChildNodes(), history: createWebHistory() })
app.use(createHead())
app.mount('#app')
