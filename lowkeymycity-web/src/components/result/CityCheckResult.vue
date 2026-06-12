<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import ResultBody from '@/components/result/ResultBody.vue'
import type { QuizResult } from '@/types/quiz'

const props = defineProps<{ result: QuizResult }>()

const displayScore = ref(0)

onMounted(() => {
  const target = props.result.score ?? 0
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    displayScore.value = target
    return
  }
  const duration = 1400
  const start = performance.now()
  const tick = (now: number) => {
    const t = Math.min((now - start) / duration, 1)
    displayScore.value = Math.round(target * (1 - Math.pow(1 - t, 3)))
    if (t < 1) requestAnimationFrame(tick)
  }
  requestAnimationFrame(tick)
})

// happy gradient is reserved for cities that earned it
const scoreGradient = computed(() => {
  const score = props.result.score ?? 0
  if (score >= 70) return 'from-lilac-deep to-coral'
  if (score >= 40) return 'from-butter to-coral'
  return 'from-coral to-alarm'
})
</script>

<template>
  <div>
    <div class="text-center">
      <p class="text-sm font-semibold tracking-widest text-ink-soft uppercase print:text-xs">
        the verdict on
      </p>
      <h1 class="font-display mt-1 text-3xl font-bold print:text-2xl">{{ result.city }}</h1>

      <div class="my-8 print:my-2">
        <span
          class="font-display bg-gradient-to-r bg-clip-text text-8xl font-extrabold text-transparent tabular-nums print:text-5xl print:text-coral"
          :class="scoreGradient"
        >
          {{ displayScore }}
        </span>
        <span class="font-display ml-2 text-2xl font-bold text-ink-soft print:text-lg"
          >/100 lowkey</span
        >
      </div>

      <h2 class="font-display mx-auto max-w-md text-2xl font-bold print:text-xl">
        {{ result.title }}
      </h2>
      <p class="mx-auto mt-4 max-w-md text-lg text-ink-soft print:mt-1 print:text-sm">
        {{ result.summary }}
      </p>
    </div>

    <div class="mt-14 print:mt-5">
      <ResultBody :result="result" alternatives-heading="better matches for you" />
    </div>
  </div>
</template>
