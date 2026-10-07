<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'

const emit = defineEmits<{ newChat: [] }>()
const store = useChatStore()
const route = useRoute()
const router = useRouter()

const activeId = computed(() => {
  const id = route.params.conversationId
  return typeof id === 'string' ? id : store.activeConversationId
})

function selectConversation(id: string) {
  void store.loadConversation(id)
  void router.push({ name: 'chat', params: { conversationId: id } })
}
</script>

<template>
  <aside class="sidebar">
    <div class="brand">
      <img src="/favicon.svg" alt="" class="logo" width="28" height="28" />
      <span class="brand-name">Piper Agent</span>
    </div>
    <button type="button" class="new-chat" @click="emit('newChat')">
      <span class="icon">+</span>
      开启新对话
    </button>
    <div class="history-label">历史对话</div>
    <nav class="history">
      <button
        v-for="c in store.conversations"
        :key="c.id"
        type="button"
        class="history-item"
        :class="{ active: c.id === activeId }"
        @click="selectConversation(c.id)"
      >
        <span class="title">{{ c.title }}</span>
      </button>
      <p v-if="!store.conversations.length" class="empty">暂无历史，发送消息开始对话</p>
    </nav>
    <footer class="sidebar-foot">
      <span class="phase">Phase W2 · 真实 Agent</span>
    </footer>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 260px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--ds-sidebar);
  border-right: 1px solid var(--ds-sidebar-border);
  padding: 16px 12px;
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 8px 16px;
}
.brand-name {
  font-size: 17px;
  font-weight: 600;
  color: var(--ds-text);
}
.new-chat {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 10px 14px;
  border: 1px solid var(--ds-input-border);
  border-radius: var(--ds-radius);
  background: var(--ds-bg);
  color: var(--ds-text);
  font-size: 14px;
  font-weight: 500;
  transition: background 0.15s, border-color 0.15s;
}
.new-chat:hover {
  background: var(--ds-hover);
  border-color: #d1d5db;
}
.new-chat .icon {
  font-size: 18px;
  line-height: 1;
  color: var(--ds-primary);
}
.history-label {
  margin: 20px 8px 8px;
  font-size: 12px;
  color: var(--ds-text-muted);
  font-weight: 500;
}
.history {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.history-item {
  text-align: left;
  border: none;
  background: transparent;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 14px;
  color: var(--ds-text-secondary);
  transition: background 0.15s;
}
.history-item:hover {
  background: var(--ds-hover);
}
.history-item.active {
  background: #e8eeff;
  color: var(--ds-primary);
  font-weight: 500;
}
.title {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.empty {
  margin: 12px 8px;
  font-size: 13px;
  color: var(--ds-text-muted);
  line-height: 1.5;
}
.sidebar-foot {
  padding: 12px 8px 4px;
  border-top: 1px solid var(--ds-sidebar-border);
  margin-top: 8px;
}
.phase {
  font-size: 11px;
  color: var(--ds-text-muted);
}
</style>
