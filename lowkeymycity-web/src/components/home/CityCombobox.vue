<script setup lang="ts">
import { computed, onMounted, ref, shallowRef } from 'vue'

import { loadUsCities } from '@/data/usCities'

const model = defineModel<string>({ required: true })

const emit = defineEmits<{ select: [city: string] }>()

const isOpen = ref(false)
const highlighted = ref(0)

// labels + a pre-lowered copy so 31k toLowerCase() calls don't run per keystroke
const cityLabels = shallowRef<string[]>([])
const cityLowered = shallowRef<string[]>([])

onMounted(async () => {
  try {
    cityLabels.value = await loadUsCities()
    cityLowered.value = cityLabels.value.map((city) => city.toLowerCase())
  } catch {
    // list unavailable — the "can't find that one" hint covers it
  }
})

const matches = computed(() => {
  const query = model.value.trim().toLowerCase()
  if (query.length < 2) return []
  // prefix matches beat substring matches ("port" → Portland before Bridgeport);
  // the list is population-sorted, so big cities surface first either way
  const prefix: string[] = []
  const substring: string[] = []
  const lowered = cityLowered.value
  for (let i = 0; i < lowered.length && prefix.length < 6; i++) {
    if (lowered[i].startsWith(query)) {
      prefix.push(cityLabels.value[i])
    } else if (substring.length < 6 && lowered[i].includes(query)) {
      substring.push(cityLabels.value[i])
    }
  }
  return [...prefix, ...substring].slice(0, 6)
})

const noMatches = computed(
  () => isOpen.value && model.value.trim().length >= 2 && matches.value.length === 0,
)

function onInput() {
  isOpen.value = true
  highlighted.value = 0
}

function select(city: string) {
  model.value = city
  isOpen.value = false
  emit('select', city)
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    isOpen.value = false
    return
  }
  if (!isOpen.value || matches.value.length === 0) return

  if (event.key === 'ArrowDown') {
    event.preventDefault()
    highlighted.value = (highlighted.value + 1) % matches.value.length
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    highlighted.value = (highlighted.value - 1 + matches.value.length) % matches.value.length
  } else if (event.key === 'Enter') {
    event.preventDefault()
    select(matches.value[highlighted.value])
  }
}
</script>

<template>
  <div class="relative">
    <input
      v-model="model"
      type="text"
      placeholder="any US city"
      autocomplete="off"
      role="combobox"
      :aria-expanded="isOpen && matches.length > 0"
      aria-label="City name"
      class="w-full bg-transparent px-4 py-3 font-medium placeholder:text-ink-soft/60 focus:outline-none"
      @input="onInput"
      @keydown="onKeydown"
      @focus="isOpen = true"
      @blur="isOpen = false"
    />

    <ul
      v-if="isOpen && matches.length > 0"
      role="listbox"
      class="absolute z-30 mt-4 w-full overflow-hidden rounded-xl border-2 border-ink bg-white shadow-[4px_4px_0_0_var(--color-ink)]"
    >
      <li
        v-for="(city, index) in matches"
        :key="city"
        role="option"
        :aria-selected="index === highlighted"
        class="cursor-pointer px-4 py-2.5 text-sm font-medium"
        :class="index === highlighted ? 'bg-lilac/30' : ''"
        @mousedown.prevent="select(city)"
        @mousemove="highlighted = index"
      >
        {{ city }}
      </li>
    </ul>

    <p v-if="noMatches" class="absolute mt-4 w-full text-center text-sm text-ink-soft">
      can't find that one — US cities only for now, sorry
    </p>
  </div>
</template>
