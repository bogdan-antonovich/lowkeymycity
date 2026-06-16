<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useHead } from '@unhead/vue'
import { useRoute } from 'vue-router'

import CityCheckResult from '@/components/result/CityCheckResult.vue'
import CityMatchResult from '@/components/result/CityMatchResult.vue'
import { getResult } from '@/services/api'
import type { QuizResult } from '@/types/quiz'

// Ephemeral, per-share results, not for the index. (Rich per-result
// social previews would need server-rendered OG tags, tracked separately.)
useHead({
  title: 'your verdict | lowkeymycity',
  meta: [{ name: 'robots', content: 'noindex, follow' }],
})

const route = useRoute()

const result = ref<QuizResult | null>(null)
const failed = ref(false)

onMounted(async () => {
  try {
    result.value = await getResult(String(route.params.id))
  } catch {
    failed.value = true
  }
})
</script>

<template>
  <div class="mx-auto max-w-5xl px-4 pt-10 pb-20 sm:px-6 print:px-0 print:pt-2 print:pb-0">
    <template v-if="result">
      <CityCheckResult v-if="result.mode === 'city'" :result="result" />
      <CityMatchResult v-else :result="result" />
    </template>

    <div v-else-if="failed" class="pt-14 text-center">
      <div class="mb-6 text-5xl">🗑️</div>
      <h1 class="font-display text-3xl font-bold">that one's gone</h1>
      <p class="mx-auto mt-4 max-w-md text-lg text-ink-soft">
        this result doesn't exist, or it got cleaned up. the quiz takes about 90 seconds, easier to
        get a fresh one than to mourn this one.
      </p>
      <RouterLink
        to="/"
        class="mt-8 inline-block rounded-xl border-2 border-ink bg-white px-5 py-3 font-semibold shadow-[4px_4px_0_0_var(--color-ink)] transition-all motion-safe:hover:-translate-y-0.5 motion-safe:active:translate-y-0.5 motion-safe:active:shadow-none"
      >
        ← back home
      </RouterLink>
    </div>

    <div v-else class="pt-24 text-center" aria-busy="true">
      <div class="mb-6 animate-bounce text-5xl">🧾</div>
      <p class="font-display text-2xl font-bold">pulling up the receipts…</p>
    </div>
  </div>
</template>
