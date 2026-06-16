<script setup lang="ts">
import { computed } from 'vue'
import { useHead } from '@unhead/vue'
import { useRoute } from 'vue-router'

import CryptoDonateDialog from '@/components/support/CryptoDonateDialog.vue'
import AppFooter from '@/components/layout/AppFooter.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import { DEFAULT_DESCRIPTION, OG_IMAGE, SITE_NAME, SITE_URL } from '@/config'

const route = useRoute()

// Absolute URL for the current route, kept free of query strings so quiz
// links with mode/city params don't fragment into duplicate canonicals.
const canonicalUrl = computed(() => `${SITE_URL}${route.path}`)

// Site-wide head defaults. Pages override title/description/robots via
// their own useHead(); these fill in everything else and act as the
// fallback for routes that set nothing.
useHead({
  titleTemplate: (title) => title ?? `${SITE_NAME}: how lowkey is your city, actually?`,
  link: [{ rel: 'canonical', href: canonicalUrl }],
  meta: [
    { name: 'description', content: DEFAULT_DESCRIPTION },
    { name: 'theme-color', content: '#faf6ee' },
    { property: 'og:site_name', content: SITE_NAME },
    { property: 'og:type', content: 'website' },
    { property: 'og:image', content: OG_IMAGE },
    { property: 'og:url', content: canonicalUrl },
    { name: 'twitter:card', content: 'summary_large_image' },
    { name: 'twitter:image', content: OG_IMAGE },
  ],
})
</script>

<template>
  <div class="flex min-h-screen flex-col">
    <AppHeader />
    <main class="flex-1">
      <RouterView />
    </main>
    <AppFooter />
    <CryptoDonateDialog />
  </div>
</template>
