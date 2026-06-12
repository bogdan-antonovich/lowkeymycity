<script setup lang="ts">
import { resultPdfUrl, USE_MOCKS } from '@/services/api'

const props = defineProps<{ resultId?: string }>()

// Stored results get a real pdf from the backend. Anything without an id
// (or while running on mocks) falls back to the browser's print dialog,
// which defaults to save-as-PDF.
const pdfUrl = props.resultId && !USE_MOCKS ? resultPdfUrl(props.resultId) : null

function saveIt() {
  window.print()
}
</script>

<template>
  <div class="flex flex-col gap-3 print:hidden">
    <a
      v-if="pdfUrl"
      :href="pdfUrl"
      download
      class="w-full rounded-xl border-2 border-ink bg-ink px-6 py-3 text-center font-semibold text-cream shadow-[4px_4px_0_0_var(--color-coral)] transition-all motion-safe:hover:-translate-y-0.5 motion-safe:active:translate-y-0.5 motion-safe:active:shadow-none"
    >
      keep the receipts
    </a>
    <button
      v-else
      type="button"
      class="w-full rounded-xl border-2 border-ink bg-ink px-6 py-3 font-semibold text-cream shadow-[4px_4px_0_0_var(--color-coral)] transition-all motion-safe:hover:-translate-y-0.5 motion-safe:active:translate-y-0.5 motion-safe:active:shadow-none"
      @click="saveIt"
    >
      keep the receipts
    </button>
    <RouterLink
      to="/"
      class="w-full rounded-xl border-2 border-ink bg-white px-6 py-3 text-center font-semibold shadow-[4px_4px_0_0_var(--color-ink)] transition-all motion-safe:hover:-translate-y-0.5 motion-safe:active:translate-y-0.5 motion-safe:active:shadow-none"
    >
      run it back
    </RouterLink>
  </div>
</template>
