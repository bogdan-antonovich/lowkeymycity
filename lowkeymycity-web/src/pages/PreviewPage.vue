<script setup lang="ts">
import { ref } from 'vue'

import CityCheckResult from '@/components/result/CityCheckResult.vue'
import CityMatchResult from '@/components/result/CityMatchResult.vue'
import { MOCK_RESULTS } from '@/data/mockResults'
import type { QuizResult } from '@/types/quiz'

interface Scenario {
  id: string
  label: string
  result: QuizResult
}

const scenarios: Scenario[] = [
  { id: 'city-high', label: 'city check — good match', result: MOCK_RESULTS.cityHigh },
  { id: 'city-low', label: 'city check — bad match', result: MOCK_RESULTS.cityLow },
  { id: 'match', label: 'find me a city', result: MOCK_RESULTS.match },
]

const active = ref(scenarios[0])
</script>

<template>
  <div class="mx-auto max-w-5xl px-4 pt-10 pb-20 sm:px-6 print:px-0 print:pt-2 print:pb-0">
    <div
      class="mb-10 rounded-xl border-2 border-dashed border-ink/30 p-4 text-sm text-ink-soft print:hidden"
    >
      <p class="font-semibold text-ink">preview page (not linked from anywhere)</p>
      <p class="mt-1">
        hardcoded results so you can see what users get at the end of each quiz. switch scenarios
        below — animations replay on every switch.
      </p>
      <div class="mt-3 flex flex-wrap gap-2">
        <button
          v-for="scenario in scenarios"
          :key="scenario.id"
          type="button"
          class="rounded-full border-2 border-ink px-4 py-1.5 text-sm font-semibold transition-colors"
          :class="active.id === scenario.id ? 'bg-ink text-cream' : 'bg-white hover:bg-lilac/20'"
          @click="active = scenario"
        >
          {{ scenario.label }}
        </button>
      </div>
      <p class="mt-3">
        same results as stored-result pages (what the pdf endpoint will print):
        <RouterLink class="underline" to="/r/demo-city-high">/r/demo-city-high</RouterLink> ·
        <RouterLink class="underline" to="/r/demo-city-low">/r/demo-city-low</RouterLink> ·
        <RouterLink class="underline" to="/r/demo-match">/r/demo-match</RouterLink>
      </p>
    </div>

    <CityCheckResult
      v-if="active.result.mode === 'city'"
      :key="active.id"
      :result="active.result"
    />
    <CityMatchResult v-else :key="active.id" :result="active.result" />
  </div>
</template>
