import type { RouteRecordRaw, RouterScrollBehavior } from 'vue-router'

import HomePage from '@/pages/HomePage.vue'

export const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: HomePage },
  { path: '/quiz', name: 'quiz', component: () => import('@/pages/QuizPage.vue') },
  { path: '/preview', name: 'preview', component: () => import('@/pages/PreviewPage.vue') },
  { path: '/r/:id', name: 'result', component: () => import('@/pages/ResultPage.vue') },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

export const scrollBehavior: RouterScrollBehavior = (to) => {
  if (to.hash) {
    return { el: to.hash, behavior: 'smooth' }
  }
  return { top: 0 }
}
