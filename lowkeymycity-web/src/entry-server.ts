import { renderToString } from '@vue/server-renderer'
import { createHead, renderSSRHead } from '@unhead/vue/server'
import { createMemoryHistory } from 'vue-router'

import { createApp } from '@/app-factory'

// Renders one route to HTML plus its resolved <head> tags. Called by the
// build-time prerender script (prerender.mjs), never in the browser.
export async function render(url: string) {
  const head = createHead()
  const { app, router } = createApp({ hydrate: true, history: createMemoryHistory() })
  app.use(head)

  await router.push(url)
  await router.isReady()

  const appHtml = await renderToString(app)
  const { headTags } = await renderSSRHead(head)
  return { appHtml, headTags }
}
