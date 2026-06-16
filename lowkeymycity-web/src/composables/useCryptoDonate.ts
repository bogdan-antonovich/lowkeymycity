import { ref } from 'vue'

// Shared open state so the header, footer, and anywhere else can pop the same
// donation dialog. One dialog instance lives in App.vue.
const isOpen = ref(false)

export function useCryptoDonate() {
  return {
    isOpen,
    open: () => {
      isOpen.value = true
    },
    close: () => {
      isOpen.value = false
    },
  }
}
