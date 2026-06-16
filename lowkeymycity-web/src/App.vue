<script setup lang="ts">
import CryptoDonateDialog from '@/components/support/CryptoDonateDialog.vue'
import AppFooter from '@/components/layout/AppFooter.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
</script>

<template>
  <div class="flex min-h-screen flex-col">
    <AppHeader />
    <main class="flex-1">
      <RouterView />
    </main>
    <AppFooter />
    <CryptoDonateDialog />
  </div>
</template>
