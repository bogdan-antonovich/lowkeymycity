import { createApp as createCSRApp, createSSRApp } from 'vue'
import { createRouter } from 'vue-router'
import type { RouterHistory } from 'vue-router'

import App from '@/App.vue'
import { routes, scrollBehavior } from '@/router'

import '@/assets/main.css'

// Builds the app and its router. The build-time prerender and the browser
// both go through here so they render the exact same tree; only the app
// kind (hydrating vs fresh) and the history backend differ.
export function createApp(options: { hydrate: boolean; history: RouterHistory }) {
  const app = options.hydrate ? createSSRApp(App) : createCSRApp(App)
  const router = createRouter({
    history: options.history,
    routes,
    scrollBehavior,
  })
  app.use(router)
  return { app, router }
}
