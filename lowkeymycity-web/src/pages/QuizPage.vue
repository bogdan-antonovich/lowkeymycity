<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { getQuestions, submitQuiz } from '@/services/api'
import type { QuizAnswer, QuizMode, QuizQuestion } from '@/types/quiz'

const route = useRoute()
const router = useRouter()

const mode: QuizMode = route.query.mode === 'city' ? 'city' : 'match'
const city = typeof route.query.city === 'string' ? route.query.city : undefined
const cityName = city?.split(',')[0].toLowerCase()

type Stage = 'loading' | 'quiz' | 'cooking' | 'error'
const stage = ref<Stage>('loading')

const questions = ref<QuizQuestion[]>([])
const index = ref(0)
const answers = ref<QuizAnswer[]>([])
const picked = ref<number | null>(null)

const current = computed(() => questions.value[index.value])
const progress = computed(() =>
  questions.value.length ? (index.value / questions.value.length) * 100 : 0,
)

const cookingLines =
  mode === 'city'
    ? [
        `walking around ${cityName} real quick…`,
        'checking the brunch density…',
        'counting the quiet streets…',
        'reading what locals mutter about it…',
        'settling on a verdict…',
      ]
    : [
        'reading your answers twice…',
        'arguing with the map…',
        'shortlisting all 50 states…',
        "checking rent prices so you don't have to…",
        "asking locals. they're typing…",
      ]
const cookingLine = ref(cookingLines[0])
let cookingTimer: number | undefined

onMounted(async () => {
  if (mode === 'city' && !city) {
    router.replace('/')
    return
  }
  window.addEventListener('keydown', onKey)
  try {
    questions.value = (await getQuestions(mode, city)).questions
    stage.value = 'quiz'
  } catch {
    stage.value = 'error'
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  if (cookingTimer) clearInterval(cookingTimer)
})

function onKey(event: KeyboardEvent) {
  if (stage.value !== 'quiz' || picked.value !== null || !current.value) return
  const n = Number(event.key)
  if (n >= 1 && n <= current.value.options.length) pick(n - 1)
}

function pick(optionIndex: number) {
  if (picked.value !== null) return
  picked.value = optionIndex
  answers.value[index.value] = {
    questionId: current.value.id,
    question: current.value.text,
    answer: current.value.options[optionIndex],
  }
  // brief beat so the selection registers visually, then move on
  setTimeout(() => {
    picked.value = null
    if (index.value < questions.value.length - 1) {
      index.value += 1
    } else {
      finish()
    }
  }, 280)
}

function back() {
  if (index.value > 0) index.value -= 1
}

async function finish() {
  stage.value = 'cooking'
  let i = 0
  cookingTimer = window.setInterval(() => {
    i = (i + 1) % cookingLines.length
    cookingLine.value = cookingLines[i]
  }, 1500)
  try {
    const result = await submitQuiz({ mode, city, answers: answers.value })
    router.replace(`/r/${result.id}`)
  } catch {
    if (cookingTimer) clearInterval(cookingTimer)
    stage.value = 'error'
  }
}

function retry() {
  if (questions.value.length && answers.value.length === questions.value.length) {
    finish()
  } else {
    window.location.reload()
  }
}
</script>

<template>
  <div class="mx-auto max-w-xl px-4 pt-16 pb-20 sm:px-6">
    <!-- fetching questions -->
    <div v-if="stage === 'loading'" class="pt-10 text-center" aria-busy="true">
      <div class="mb-6 animate-bounce text-5xl">🍳</div>
      <p class="font-display text-2xl font-bold">
        {{ mode === 'city' ? `cooking up questions for ${cityName}…` : 'grabbing the questions…' }}
      </p>
      <p class="mt-3 text-ink-soft">takes a few seconds. stretch or something.</p>
    </div>

    <!-- the quiz -->
    <div v-else-if="stage === 'quiz' && current">
      <div class="mb-8">
        <div class="mb-3 flex items-center justify-between">
          <button
            type="button"
            class="rounded-lg px-2 py-1 font-semibold text-ink-soft transition-colors hover:text-ink"
            :class="index === 0 ? 'invisible' : ''"
            aria-label="previous question"
            @click="back"
          >
            ←
          </button>
          <p class="text-sm font-semibold tracking-widest text-ink-soft uppercase">
            {{ index + 1 }} / {{ questions.length }}
          </p>
        </div>
        <div
          class="h-2 overflow-hidden rounded-full border border-ink/20 bg-white"
          role="progressbar"
          :aria-valuenow="index + 1"
          :aria-valuemin="1"
          :aria-valuemax="questions.length"
        >
          <div
            class="h-full rounded-full bg-gradient-to-r from-lilac-deep to-coral transition-[width] duration-300"
            :style="{ width: `${progress}%` }"
          />
        </div>
      </div>

      <Transition name="q" mode="out-in">
        <div :key="current.id">
          <h1 class="font-display min-h-20 text-2xl font-bold sm:text-3xl">{{ current.text }}</h1>
          <div class="mt-6 space-y-3">
            <button
              v-for="(option, i) in current.options"
              :key="option"
              type="button"
              class="flex w-full items-center gap-3 rounded-xl border-2 border-ink px-4 py-3.5 text-left font-medium shadow-[3px_3px_0_0_var(--color-ink)] transition-all motion-safe:hover:-translate-y-0.5 motion-safe:active:translate-y-0.5 motion-safe:active:shadow-none"
              :class="picked === i ? 'bg-lilac/40' : 'bg-white'"
              @click="pick(i)"
            >
              <span
                class="font-display flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-ink/30 text-sm font-bold text-ink-soft"
              >
                {{ i + 1 }}
              </span>
              {{ option }}
            </button>
          </div>
          <p class="mt-5 hidden text-center text-xs text-ink-soft sm:block">
            tip: keys 1–4 work too
          </p>
        </div>
      </Transition>
    </div>

    <!-- waiting on the verdict -->
    <div v-else-if="stage === 'cooking'" class="pt-10 text-center" aria-busy="true">
      <div class="mb-6 animate-bounce text-5xl">🔮</div>
      <Transition name="q" mode="out-in">
        <p :key="cookingLine" class="font-display text-2xl font-bold">{{ cookingLine }}</p>
      </Transition>
      <p class="mt-3 text-ink-soft">good answers, by the way. give us a moment.</p>
    </div>

    <!-- something broke -->
    <div v-else-if="stage === 'error'" class="pt-10 text-center">
      <div class="mb-6 text-5xl">🧯</div>
      <h1 class="font-display text-3xl font-bold">well. that didn't work</h1>
      <p class="mx-auto mt-4 max-w-md text-lg text-ink-soft">
        something went sideways on our end. your answers are safe — hit retry.
      </p>
      <div class="mt-8 flex justify-center gap-3">
        <button
          type="button"
          class="rounded-xl border-2 border-ink bg-ink px-5 py-3 font-semibold text-cream shadow-[4px_4px_0_0_var(--color-coral)] transition-all motion-safe:hover:-translate-y-0.5 motion-safe:active:translate-y-0.5 motion-safe:active:shadow-none"
          @click="retry"
        >
          retry
        </button>
        <RouterLink
          to="/"
          class="rounded-xl border-2 border-ink bg-white px-5 py-3 font-semibold shadow-[4px_4px_0_0_var(--color-ink)] transition-all motion-safe:hover:-translate-y-0.5 motion-safe:active:translate-y-0.5 motion-safe:active:shadow-none"
        >
          ← home
        </RouterLink>
      </div>
    </div>
  </div>
</template>

<style scoped>
.q-enter-active,
.q-leave-active {
  transition:
    opacity 0.15s ease,
    transform 0.15s ease;
}
.q-enter-from {
  opacity: 0;
  transform: translateX(14px);
}
.q-leave-to {
  opacity: 0;
  transform: translateX(-14px);
}
@media (prefers-reduced-motion: reduce) {
  .q-enter-active,
  .q-leave-active {
    transition: none;
  }
}
</style>
