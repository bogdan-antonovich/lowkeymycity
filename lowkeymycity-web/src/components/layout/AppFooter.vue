<script setup lang="ts">
import { BUY_ME_A_COFFEE_URL } from '@/config'

const year = new Date().getFullYear()
</script>

<template>
  <footer class="bg-ink text-cream/70 print:hidden">
    <div
      class="mx-auto flex max-w-5xl flex-col items-center gap-3 px-4 py-8 text-sm sm:flex-row sm:justify-between sm:px-6"
    >
      <p>made with zero math and pure vibes</p>

      <a
        :href="BUY_ME_A_COFFEE_URL"
        target="_blank"
        rel="noopener noreferrer"
        class="font-medium text-cream underline decoration-coral decoration-2 underline-offset-4 transition-colors hover:text-coral"
      >
        ☕ this site runs on coffee
      </a>

      <p>
        © {{ year }} lowkeymycity.com ·
        <!-- required by the simplemaps free license -->
        <a
          href="https://simplemaps.com/data/us-cities"
          target="_blank"
          rel="noopener noreferrer"
          class="underline decoration-cream/40 underline-offset-2 transition-colors hover:text-cream"
        >
          city data: simplemaps
        </a>
      </p>
    </div>
  </footer>
</template>
