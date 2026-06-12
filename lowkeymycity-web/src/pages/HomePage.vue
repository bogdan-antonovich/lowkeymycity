<script setup lang="ts">
import { computed, onMounted, ref, shallowRef } from 'vue'
import { useRouter } from 'vue-router'

import CityCombobox from '@/components/home/CityCombobox.vue'
import { loadUsCities } from '@/data/usCities'

const router = useRouter()

const cityInput = ref('')
const cities = shallowRef<string[]>([])

onMounted(async () => {
  try {
    cities.value = await loadUsCities()
  } catch {
    // list unavailable — the submit button just stays disabled
  }
})

const selectedCity = computed(() => {
  const query = cityInput.value.trim().toLowerCase()
  return cities.value.find((city) => city.toLowerCase() === query) ?? null
})

function startCityQuiz() {
  if (!selectedCity.value) return
  router.push({ name: 'quiz', query: { mode: 'city', city: selectedCity.value } })
}

const steps = [
  {
    title: 'tap through 12 questions',
    text: 'scenarios, not a survey. about 90 seconds.',
  },
  {
    title: 'we read your answers',
    text: 'judgmental, but in a loving way.',
  },
  {
    title: 'you get the verdict',
    text: 'screenshot it, send it to the group chat.',
  },
]
</script>

<template>
  <div class="mx-auto max-w-5xl px-4 sm:px-6">
    <!-- hero -->
    <section class="pt-20 pb-10 text-center sm:pt-28">
      <h1
        class="font-display mx-auto max-w-2xl text-4xl leading-tight font-extrabold tracking-tight sm:text-6xl"
      >
        how
        <span class="bg-gradient-to-r from-lilac-deep to-coral bg-clip-text text-transparent"
          >lowkey</span
        >
        is your city, actually?
      </h1>
      <p class="mx-auto mt-5 max-w-md text-lg text-ink-soft">
        type it in, answer twelve questions, get an honest verdict.
      </p>
    </section>

    <!-- the check -->
    <section id="check" class="mx-auto max-w-xl pb-24">
      <form @submit.prevent="startCityQuiz">
        <div
          class="flex flex-col gap-2 rounded-2xl border-2 border-ink bg-white p-2 shadow-[5px_5px_0_0_var(--color-ink)] transition-shadow focus-within:ring-4 focus-within:ring-lilac/40 sm:flex-row sm:items-center sm:p-1.5"
        >
          <CityCombobox v-model="cityInput" class="min-w-0 sm:flex-1" />
          <button
            type="submit"
            :disabled="!selectedCity"
            class="shrink-0 rounded-xl bg-ink px-5 py-3 font-semibold text-cream transition-all enabled:motion-safe:hover:-translate-y-0.5 enabled:motion-safe:active:translate-y-0 disabled:opacity-40"
          >
            check it
          </button>
        </div>
      </form>

      <p class="mt-6 text-center text-ink-soft">
        no city in mind?
        <RouterLink
          :to="{ name: 'quiz', query: { mode: 'match' } }"
          class="font-medium text-ink underline decoration-coral decoration-2 underline-offset-4 transition-colors hover:text-coral"
        >
          take the quiz and we'll name one
        </RouterLink>
      </p>
    </section>

    <!-- how it works -->
    <section class="border-t border-ink/10 py-14">
      <h2 class="font-display mb-10 text-center text-2xl font-bold">how this works</h2>
      <ol class="grid gap-8 sm:grid-cols-3">
        <li v-for="(step, index) in steps" :key="step.title" class="text-center">
          <div
            class="font-display mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full border-2 border-ink bg-white text-xl font-bold shadow-[3px_3px_0_0_var(--color-lilac)]"
          >
            {{ index + 1 }}
          </div>
          <h3 class="font-semibold">{{ step.title }}</h3>
          <p class="mt-1 text-sm text-ink-soft">{{ step.text }}</p>
        </li>
      </ol>
    </section>
  </div>
</template>
