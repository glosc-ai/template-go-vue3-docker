import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/features/auth/store'
import { notifyWarning } from '@/lib/message'

declare module 'vue-router' {
  interface RouteMeta {
    /** Route requires a signed-in session; see the beforeEach guard below. */
    requiresAuth?: boolean
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/HomeView.vue'),
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/ProfileView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
  scrollBehavior: () => ({ top: 0 }),
})

// Guarded routes wait for the first session probe, then bounce anonymous
// visitors to /login while remembering where they were headed.
router.beforeEach(async (to) => {
  if (!to.meta.requiresAuth) {
    return true
  }

  const auth = useAuthStore()
  await auth.ensureLoaded()
  if (auth.isAuthenticated) {
    return true
  }

  notifyWarning('请先登录后再访问该页面')
  return { name: 'login', query: { redirect_to: to.fullPath } }
})

export default router
