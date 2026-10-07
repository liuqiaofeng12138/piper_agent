<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import AppSidebar from '@/components/AppSidebar.vue'
import { useChatStore } from '@/stores/chat'

const store = useChatStore()
const router = useRouter()

onMounted(() => {
  void store.refreshConversations()
})

function onNewChat() {
  store.resetForNewChat()
  void router.push({ name: 'chat-new' })
}
</script>

<template>
  <div class="layout">
    <AppSidebar @new-chat="onNewChat" />
    <main class="main">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}
.main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--ds-bg);
}
</style>
