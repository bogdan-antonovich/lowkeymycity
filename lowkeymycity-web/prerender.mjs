// Build-time prerender. Runs after `vite build`: spins up Vite in SSR mode,
// renders the indexable routes to static HTML with their per-route <head>,
// and writes them into dist/. Dynamic and internal routes are not rendered;
// they fall through to app.html (a clean SPA shell) via the nginx fallback.
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { createServer } from 'vite'

const rootDir = dirname(fileURLToPath(import.meta.url))
const distDir = resolve(rootDir, 'dist')

// Routes worth shipping as static HTML. /preview is internal and /r/:id is
// dynamic, so both are intentionally left to the app.html fallback.
const ROUTES = ['/', '/quiz']

// The head tags unhead emits per route. The built template carries the
// home-page copies of these (so the app.html fallback still unfurls); strip
// them before injecting the route-specific ones so nothing is duplicated.
const MANAGED_TAGS =
  /<title>[\s\S]*?<\/title>|<meta\s+name="description"[^>]*>|<meta\s+name="theme-color"[^>]*>|<meta\s+name="robots"[^>]*>|<meta\s+property="og:[^"]*"[^>]*>|<meta\s+name="twitter:[^"]*"[^>]*>|<link\s+rel="canonical"[^>]*>/g

const vite = await createServer({
  server: { middlewareMode: true },
  appType: 'custom',
})

try {
  const { render } = await vite.ssrLoadModule('/src/entry-server.ts')
  const template = await readFile(join(distDir, 'index.html'), 'utf-8')

  // app.html: the clean shell nginx serves for non-prerendered routes.
  // Empty #app and the home-page social defaults already baked in.
  await writeFile(join(distDir, 'app.html'), template, 'utf-8')

  for (const url of ROUTES) {
    const { appHtml, headTags } = await render(url)
    const html = template
      .replace(MANAGED_TAGS, '')
      .replace('<div id="app"></div>', `<div id="app">${appHtml}</div>`)
      .replace('</head>', `  ${headTags}\n  </head>`)

    const outFile =
      url === '/' ? join(distDir, 'index.html') : join(distDir, url.slice(1), 'index.html')
    await mkdir(dirname(outFile), { recursive: true })
    await writeFile(outFile, html, 'utf-8')
    console.log(`prerendered ${url} -> ${relative(rootDir, outFile)}`)
  }
} finally {
  await vite.close()
}
