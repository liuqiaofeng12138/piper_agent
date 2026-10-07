<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AgentBadge from '@/components/AgentBadge.vue'
import Composer from '@/components/Composer.vue'
import MessageList from '@/components/MessageList.vue'
import { useChatStore } from '@/stores/chat'

const props = defineProps<{ conversationId?: string }>()
const store = useChatStore()
const route = useRoute()
const router = useRouter()

const isEmpty = computed(
  () => !store.loadingHistory && store.messages.length === 0 && !store.streaming,
)

const suggestions = [
  '帮我抓取一个 HTTPS API 的 JSON 列表，提取 title 和 url 字段',
  '列出当前可用的采集模版',
  '用 Chrome 模式抓取需要登录的列表页',
]

async function syncRouteConversation(id: string | undefined) {
  if (!id) return
  const ok = await store.loadConversation(id)
  if (!ok) {
    await router.replace({ name: 'chat-new' })
  }
}

onMounted(() => {
  void syncRouteConversation(props.conversationId)
  void store.refreshAgents()
  void store.refreshHealth()
})

watch(
  () => route.params.conversationId,
  (id) => {
    if (typeof id === 'string') {
      void syncRouteConversation(id)
    }
  },
)

async function onSend(text: string) {
  await store.sendMessage(text)
  const cid = store.activeConversationId
  if (cid && route.name === 'chat-new') {
    await router.replace({ name: 'chat', params: { conversationId: cid } })
  }
}

function onSuggestion(text: string) {
  void onSend(text)
}
</script>

<template>
  <div class="chat-view">
    <header class="topbar">
      <h1 class="title">对话</h1>
      <AgentBadge />
      <span
        v-if="store.health"
        class="health"
        :class="store.health.status === 'ok' ? 'ok' : 'warn'"
      >
        {{ store.health.status === 'ok' ? '服务正常' : '部分异常' }}
      </span>
    </header>

    <div class="content">
      <div v-if="isEmpty" class="welcome scroll-pane">
        <div class="welcome-logo">
          <img src="/favicon.svg" alt="" width="48" height="48" />
        </div>
        <h2>我是 Piper Agent，有什么可以帮您？</h2>
        <p class="sub">描述你的采集需求，W2 起将自动执行 Piper 模版与抓取任务。</p>
        <div class="chips">
          <button
            v-for="s in suggestions"
            :key="s"
            type="button"
            class="chip"
            @click="onSuggestion(s)"
          >
            {{ s }}
          </button>
        </div>
      </div>
      <MessageList
        v-else
        class="scroll-pane"
        :messages="store.messages"
        :loading="store.loadingHistory"
      />
    </div>

    <Composer
      :streaming="store.streaming"
      @send="onSend"
      @stop="store.stopGeneration"
    />
  </div>
</template>

<style scoped>
.chat-view {
  position: relative;
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 24px;
  border-bottom: 1px solid var(--ds-sidebar-border);
}
.topbar .title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.health {
  margin-left: auto;
  font-size: 12px;
  color: var(--ds-text-muted);
}
.health.ok {
  color: #059669;
}
.health.warn {
  color: #d97706;
}
.content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}
.scroll-pane {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
}
.welcome {
  max-width: 720px;
  width: 100%;
  margin: 0 auto;
  padding: 80px 24px 140px;
  text-align: center;
  box-sizing: border-box;
}
.welcome-logo {
  margin-bottom: 20px;
}
.welcome h2 {
  margin: 0 0 12px;
  font-size: 22px;
  font-weight: 600;
  color: var(--ds-text);
}
.sub {
  margin: 0 0 32px;
  font-size: 14px;
  color: var(--ds-text-secondary);
  line-height: 1.6;
}
.chips {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: stretch;
}
.chip {
  text-align: left;
  padding: 14px 18px;
  border: 1px solid var(--ds-input-border);
  border-radius: var(--ds-radius);
  background: var(--ds-bg);
  color: var(--ds-text-secondary);
  font-size: 14px;
  line-height: 1.5;
  transition: border-color 0.15s, background 0.15s;
}
.chip:hover {
  border-color: #c7d2fe;
  background: #f8faff;
  color: var(--ds-text);
}
</style>
