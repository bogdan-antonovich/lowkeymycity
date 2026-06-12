<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import { BUY_ME_A_COFFEE_URL } from '@/config'

const route = useRoute()

// Mid-quiz the header goes quiet so nobody taps their way out of question 9.
const quietMode = computed(() => route.name === 'quiz')
</script>

<template>
  <header
    class="sticky top-0 z-40 border-b border-ink/5 bg-cream/80 backdrop-blur print:static print:border-ink/20"
  >
    <div
      class="mx-auto flex h-16 max-w-5xl items-center justify-between gap-4 px-4 sm:px-6 print:h-10 print:px-0"
    >
      <RouterLink
        to="/"
        class="font-display text-xl font-bold tracking-tight transition-transform motion-safe:hover:-rotate-2"
      >
        lowkey<span
          class="bg-gradient-to-r from-lilac-deep to-coral bg-clip-text text-transparent print:text-coral"
          >mycity</span
        >
      </RouterLink>

      <nav class="flex items-center gap-2 sm:gap-5 print:hidden">
        <template v-if="!quietMode">
          <RouterLink
            to="/#check"
            class="hidden text-sm font-medium text-ink-soft transition-colors hover:text-ink sm:block"
          >
            check a city
          </RouterLink>
          <RouterLink
            :to="{ name: 'quiz', query: { mode: 'match' } }"
            class="hidden text-sm font-medium text-ink-soft transition-colors hover:text-ink sm:block"
          >
            find my city
          </RouterLink>
        </template>

        <a
          :href="BUY_ME_A_COFFEE_URL"
          target="_blank"
          rel="noopener noreferrer"
          class="rounded-full bg-gradient-to-r from-lilac-deep to-coral px-4 py-1.5 text-sm font-semibold text-white transition-transform motion-safe:hover:scale-105 motion-safe:active:scale-95"
        >
          ☕ buy me a coffee
        </a>
      </nav>
    </div>
  </header>
</template>
