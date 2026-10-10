import { createRouter, createWebHistory } from 'vue-router'
import ChatLayout from '@/layouts/ChatLayout.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: ChatLayout,
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

export default router
