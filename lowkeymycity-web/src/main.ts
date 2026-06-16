import { createApp } from 'vue'
import { createHead } from '@unhead/vue/client'

import App from '@/App.vue'
import router from '@/router'

import '@/assets/main.css'

const head = createHead()

createApp(App).use(head).use(router).mount('#app')
