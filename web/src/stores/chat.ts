import { defineStore } from 'pinia'
import { ref } from 'vue'
import { cancelRun, chatStream, listConversations, listMessages } from '@/api/client'
import type { Conversation, Message, StreamEvent } from '@/api/types'

export const useChatStore = defineStore('chat', () => {
  const conversations = ref<Conversation[]>([])
  const messages = ref<Message[]>([])
  const activeConversationId = ref<string | null>(null)
  const loadingHistory = ref(false)
  const streaming = ref(false)
  const currentRunId = ref<string | null>(null)
  let abortController: AbortController | null = null

  async function refreshConversations() {
    const { conversations: list } = await listConversations()
    conversations.value = list
  }

  async function loadConversation(id: string) {
    loadingHistory.value = true
    activeConversationId.value = id
    try {
      const { messages: list } = await listMessages(id)
      messages.value = list
    } finally {
      loadingHistory.value = false
    }
  }

  function resetForNewChat() {
    activeConversationId.value = null
    messages.value = []
  }

  async function sendMessage(text: string) {
    const trimmed = text.trim()
    if (!trimmed || streaming.value) return

    const userMsg: Message = {
      id: `local_${Date.now()}`,
      role: 'user',
      content: trimmed,
      created_at: new Date().toISOString(),
    }
    messages.value.push(userMsg)

    const assistantMsg: Message = {
      id: `local_${Date.now()}_a`,
      role: 'assistant',
      content: '',
      created_at: new Date().toISOString(),
      streaming: true,
    }
    messages.value.push(assistantMsg)

    streaming.value = true
    abortController = new AbortController()
    currentRunId.value = null

    try {
      await chatStream({
        conversationId: activeConversationId.value ?? undefined,
        message: trimmed,
        signal: abortController.signal,
        onEvent: (ev: StreamEvent) => {
          if (ev.conversation_id && !activeConversationId.value) {
            activeConversationId.value = ev.conversation_id
          }
          if (ev.run_id) currentRunId.value = ev.run_id
          if (ev.type === 'message.delta' && ev.delta) {
            assistantMsg.content += ev.delta
          }
          if (ev.type === 'message.done' && ev.content) {
            assistantMsg.content = ev.content
          }
          if (ev.type === 'tool.call') {
            if (!assistantMsg.steps) assistantMsg.steps = []
            assistantMsg.steps.push({
              kind: 'tool',
              tool_name: ev.tool_name,
              text: ev.tool_detail || ev.message || ev.tool_name || 'tool',
            })
          }
          if (ev.type === 'run.progress') {
            if (!assistantMsg.steps) assistantMsg.steps = []
            assistantMsg.steps.push({
              kind: 'progress',
              tool_name: ev.tool_name,
              text: ev.message || '',
            })
          }
          if (ev.type === 'run.cancelled') {
            assistantMsg.content += '\n\n[已停止生成]'
          }
          if (ev.type === 'error' && ev.message) {
            assistantMsg.content += `\n\n错误：${ev.message}`
          }
        },
      })
    } catch (e) {
      if ((e as Error).name !== 'AbortError') {
        assistantMsg.content = `请求失败：${(e as Error).message}`
      }
    } finally {
      assistantMsg.streaming = false
      streaming.value = false
      abortController = null
      await refreshConversations()
    }
  }

  async function stopGeneration() {
    if (abortController) abortController.abort()
    if (currentRunId.value) {
      try {
        await cancelRun(currentRunId.value)
      } catch {
        /* gateway may already have closed */
      }
    }
  }

  return {
    conversations,
    messages,
    activeConversationId,
    loadingHistory,
    streaming,
    refreshConversations,
    loadConversation,
    resetForNewChat,
    sendMessage,
    stopGeneration,
  }
})
