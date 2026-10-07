import { createRouter, createWebHistory } from 'vue-router'
import ChatLayout from '@/layouts/ChatLayout.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: ChatLayout,
      children: [
        { path: '', name: 'chat-new', component: () => import('@/views/ChatView.vue') },
        {
          path: 'c/:conversationId',
          name: 'chat',
          component: () => import('@/views/ChatView.vue'),
          props: true,
        },
      ],
    },
  ],
})

export default router
