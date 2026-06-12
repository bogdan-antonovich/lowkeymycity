<script setup lang="ts">
import CityMap from '@/components/result/CityMap.vue'
import ResultActions from '@/components/result/ResultActions.vue'
import type { QuizResult } from '@/types/quiz'

defineProps<{
  result: QuizResult
  alternativesHeading: string
}>()
</script>

<template>
  <div class="grid gap-10 lg:grid-cols-[minmax(0,1fr)_340px] print:block">
    <!-- the read -->
    <div class="space-y-10 text-left print:space-y-3">
      <section>
        <h3 class="font-display text-2xl font-bold print:text-base">green flags</h3>
        <div class="mt-3 space-y-4 print:mt-1 print:space-y-2">
          <p
            v-for="paragraph in result.greenFlags"
            :key="paragraph"
            class="leading-relaxed print:text-[11px] print:leading-snug"
          >
            {{ paragraph }}
          </p>
        </div>
      </section>

      <section>
        <h3 class="font-display text-2xl font-bold print:text-base">red flags</h3>
        <div class="mt-3 space-y-4 print:mt-1 print:space-y-2">
          <p
            v-for="paragraph in result.redFlags"
            :key="paragraph"
            class="leading-relaxed print:text-[11px] print:leading-snug"
          >
            {{ paragraph }}
          </p>
        </div>
      </section>

      <section v-if="result.alternatives.length">
        <h3 class="font-display text-2xl font-bold print:text-base">{{ alternativesHeading }}</h3>
        <ol class="mt-3 space-y-6 print:mt-1 print:space-y-2">
          <li v-for="(alt, index) in result.alternatives" :key="alt.city">
            <p class="font-display text-lg font-bold print:text-xs">
              {{ index + 1 }}. {{ alt.city }}
            </p>
            <p class="mt-1 leading-relaxed print:mt-0 print:text-[11px] print:leading-snug">
              {{ alt.blurb }}
            </p>
          </li>
        </ol>
      </section>

      <p
        class="border-t border-ink/10 pt-6 leading-relaxed print:pt-2 print:text-[11px] print:leading-snug"
      >
        {{ result.closing }}
      </p>

      <!-- only shows up in the saved pdf -->
      <p class="hidden text-sm text-ink-soft print:block print:text-[10px]">
        made at lowkeymycity.com
      </p>
    </div>

    <!-- the card: map stretches to match the text height; sinks below it on mobile -->
    <aside class="flex flex-col gap-4 print:hidden">
      <CityMap :city="result.city" class="flex-1" />
      <ResultActions :result-id="result.id" />
    </aside>
  </div>
</template>
