import { createRouter, createWebHistory } from 'vue-router'

import HomePage from '@/pages/HomePage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: HomePage },
    { path: '/quiz', name: 'quiz', component: () => import('@/pages/QuizPage.vue') },
    { path: '/preview', name: 'preview', component: () => import('@/pages/PreviewPage.vue') },
    { path: '/r/:id', name: 'result', component: () => import('@/pages/ResultPage.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
  scrollBehavior(to) {
    if (to.hash) {
      return { el: to.hash, behavior: 'smooth' }
    }
    return { top: 0 }
  },
})

export default router
