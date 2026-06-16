<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'

import { CRYPTO_WALLETS } from '@/config'
import { useCryptoDonate } from '@/composables/useCryptoDonate'

const { isOpen, close } = useCryptoDonate()

// Tracks which address was just copied so we can flash a "copied!" label.
const copied = ref<string | null>(null)
let copiedTimer: ReturnType<typeof setTimeout> | undefined

async function copy(address: string) {
  try {
    await navigator.clipboard.writeText(address)
  } catch {
    // Clipboard API can be blocked (insecure context, permissions). Fall back
    // to a hidden textarea + execCommand so the button still does something.
    const el = document.createElement('textarea')
    el.value = address
    el.style.position = 'fixed'
    el.style.opacity = '0'
    document.body.appendChild(el)
    el.select()
    document.execCommand('copy')
    document.body.removeChild(el)
  }
  copied.value = address
  clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => (copied.value = null), 1800)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') close()
}

// Lock body scroll and wire Escape only while open.
watch(isOpen, (open) => {
  if (open) {
    document.body.style.overflow = 'hidden'
    window.addEventListener('keydown', onKeydown)
  } else {
    document.body.style.overflow = ''
    window.removeEventListener('keydown', onKeydown)
    copied.value = null
  }
})

onBeforeUnmount(() => {
  document.body.style.overflow = ''
  window.removeEventListener('keydown', onKeydown)
  clearTimeout(copiedTimer)
})
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-150"
      leave-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div
        v-if="isOpen"
        class="fixed inset-0 z-50 flex items-end justify-center bg-ink/50 p-4 backdrop-blur-sm sm:items-center"
        role="dialog"
        aria-modal="true"
        aria-labelledby="donate-title"
        @click.self="close"
      >
        <div
          class="w-full max-w-md rounded-2xl border-2 border-ink bg-cream p-6 shadow-[6px_6px_0_0_var(--color-coral)]"
        >
          <div class="flex items-start justify-between gap-4">
            <div>
              <h2 id="donate-title" class="font-display text-xl font-bold">
                ☕ fuel the vibes
              </h2>
              <p class="mt-1 text-sm text-ink-soft">
                no card needed — just send a little crypto, any amount.
              </p>
            </div>
            <button
              type="button"
              aria-label="close"
              class="-mr-1 -mt-1 rounded-full p-1 text-ink-soft transition-colors hover:bg-ink/5 hover:text-ink"
              @click="close"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                class="h-5 w-5"
              >
                <path d="M18 6 6 18M6 6l12 12" />
              </svg>
            </button>
          </div>

          <ul class="mt-5 flex flex-col gap-3">
            <li
              v-for="wallet in CRYPTO_WALLETS"
              :key="wallet.symbol + wallet.network"
              class="rounded-xl border-2 border-ink/10 bg-white p-3"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-baseline gap-2">
                  <span class="font-display text-sm font-bold">{{ wallet.symbol }}</span>
                  <span class="text-xs text-ink-soft">{{ wallet.name }}</span>
                </div>
                <span
                  class="rounded-full bg-lilac/20 px-2 py-0.5 text-[11px] font-medium text-lilac-deep"
                >
                  {{ wallet.network }}
                </span>
              </div>

              <div class="mt-2 flex items-center gap-2">
                <code
                  class="min-w-0 flex-1 truncate rounded-lg bg-ink/5 px-2 py-1.5 font-mono text-xs"
                  :title="wallet.address"
                >
                  {{ wallet.address }}
                </code>
                <button
                  type="button"
                  class="shrink-0 rounded-lg border-2 border-ink bg-ink px-3 py-1.5 text-xs font-semibold text-cream transition-transform motion-safe:active:scale-95"
                  @click="copy(wallet.address)"
                >
                  {{ copied === wallet.address ? 'copied!' : 'copy' }}
                </button>
              </div>
            </li>
          </ul>

          <p class="mt-4 text-center text-xs text-ink-soft">
            double-check the network before sending — wrong chain, lost coins.
          </p>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
