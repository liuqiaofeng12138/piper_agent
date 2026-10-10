<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'

const emit = defineEmits<{ newChat: [] }>()
const store = useChatStore()
const route = useRoute()
const router = useRouter()

const conversationList = computed(() => store.conversations ?? [])

const activeId = computed(() => {
  const q = route.query.c
  if (typeof q === 'string' && q.length > 0) return q
  return store.activeConversationId
})

function selectConversation(id: string) {
  if (id === store.activeConversationId && route.query.c === id) {
    return
  }
  void store.loadConversation(id, { force: true })
  void router.push({ name: 'chat', query: { c: id } })
}

async function onDelete(id: string) {
  if (!window.confirm('确定删除该对话？此操作不可恢复。')) return
  await store.removeConversation(id)
  if (route.query.c === id) {
    void router.push({ name: 'chat', query: {} })
  }
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
      <div
        v-for="c in conversationList"
        :key="c.id"
        class="history-item"
        :class="{ active: c.id === activeId }"
        role="button"
        tabindex="0"
        @click="selectConversation(c.id)"
        @keydown.enter="selectConversation(c.id)"
      >
        <span class="title">{{ c.title }}</span>
        <button
          type="button"
          class="delete"
          title="删除对话"
          @click.stop="onDelete(c.id)"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path
              d="M3 6h18M8 6V4a2 2 0 012-2h4a2 2 0 012 2v2m3 0v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6h14z"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>
      <p v-if="conversationList.length === 0" class="empty">暂无历史，发送消息开始对话</p>
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
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
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
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.history-item {
  display: flex;
  align-items: center;
  gap: 8px;
  text-align: left;
  border: none;
  background: transparent;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 14px;
  color: var(--ds-text-secondary);
  transition: background 0.15s;
  cursor: pointer;
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
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.delete {
  flex-shrink: 0;
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--ds-text-muted);
  opacity: 0;
  transition: opacity 0.15s, background 0.15s, color 0.15s;
}
.history-item:hover .delete,
.history-item.active .delete {
  opacity: 1;
}
.delete:hover {
  background: #fee2e2;
  color: var(--ds-danger);
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
