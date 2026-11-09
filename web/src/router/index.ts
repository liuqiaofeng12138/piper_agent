import { createRouter, createWebHistory } from 'vue-router'
import ChatLayout from '@/layouts/ChatLayout.vue'
import { useAuthStore } from '@/stores/auth'
import { fetchCurrentUser } from '@/api/client'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: ChatLayout,
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'chat',
          component: () => import('@/views/ChatView.vue'),
        },
        {
          path: 'c/:conversationId',
          redirect: (to) => ({
            name: 'chat',
            query: { c: to.params.conversationId as string },
          }),
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.meta.public) {
    if (to.name === 'login' && auth.isLoggedIn) {
      return { path: '/' }
    }
    return true
  }
  if (!auth.isLoggedIn) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  try {
    await fetchCurrentUser()
  } catch {
    auth.clearSession()
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  return true
})

export default router
