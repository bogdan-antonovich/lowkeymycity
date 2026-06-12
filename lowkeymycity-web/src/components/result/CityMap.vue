<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ city: string }>()

// Google's keyless embed. If we ever want custom styling or guarantees,
// switch to the official Maps Embed API (needs a key).
const src = computed(
  () => `https://www.google.com/maps?q=${encodeURIComponent(props.city)}&output=embed&z=11`,
)
</script>

<template>
  <div
    class="overflow-hidden rounded-xl border-2 border-ink bg-white shadow-[4px_4px_0_0_var(--color-ink)]"
  >
    <iframe
      :src="src"
      :title="`map of ${city}`"
      class="block h-full min-h-72 w-full"
      loading="lazy"
      referrerpolicy="no-referrer-when-downgrade"
    />
  </div>
</template>
