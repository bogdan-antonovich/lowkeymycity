<script setup lang="ts">
import { onMounted, ref } from 'vue'

import ResultBody from '@/components/result/ResultBody.vue'
import type { QuizResult } from '@/types/quiz'

defineProps<{ result: QuizResult }>()

// The city name starts blurred, then unveils. The pause is the whole point.
const revealed = ref(false)

onMounted(() => {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    revealed.value = true
    return
  }
  setTimeout(() => (revealed.value = true), 900)
})
</script>

<template>
  <div>
    <div class="text-center">
      <p class="text-sm font-semibold tracking-widest text-ink-soft uppercase print:text-xs">
        your city is
      </p>
      <h1
        class="font-display mt-3 bg-gradient-to-r from-lilac-deep to-coral bg-clip-text text-5xl font-extrabold text-transparent transition-[filter,opacity] duration-700 sm:text-6xl print:mt-1 print:text-3xl print:text-lilac-deep"
        :class="revealed ? 'opacity-100 blur-none' : 'opacity-50 blur-xl'"
      >
        {{ result.city }}
      </h1>

      <h2 class="font-display mx-auto mt-8 max-w-md text-2xl font-bold print:mt-3 print:text-xl">
        {{ result.title }}
      </h2>
      <p class="mx-auto mt-4 max-w-md text-lg text-ink-soft print:mt-1 print:text-sm">
        {{ result.summary }}
      </p>
    </div>

    <div class="mt-14 print:mt-5">
      <ResultBody :result="result" alternatives-heading="also would've worked" />
    </div>
  </div>
</template>
