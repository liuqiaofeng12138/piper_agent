import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  cancelRun,
  chatStream,
  deleteConversation,
  healthCheck,
  listAgents,
  listConversations,
  listMessages,
} from '@/api/client'
import type { AgentInfo, Conversation, HealthResponse, Message, StreamEvent } from '@/api/types'

export const useChatStore = defineStore('chat', () => {
  const conversations = ref<Conversation[]>([])
  const messages = ref<Message[]>([])
  const activeConversationId = ref<string | null>(null)
  const loadingHistory = ref(false)
  const streaming = ref(false)
  const currentRunId = ref<string | null>(null)
  const agents = ref<AgentInfo[]>([])
  const health = ref<HealthResponse | null>(null)
  let abortController: AbortController | null = null

  async function refreshAgents() {
    try {
      const { agents: list } = await listAgents()
      agents.value = Array.isArray(list) ? list : []
    } catch {
      agents.value = []
    }
  }

  async function refreshHealth() {
    try {
      health.value = await healthCheck()
    } catch {
      health.value = null
    }
  }

  async function refreshConversations() {
    try {
      const { conversations: list } = await listConversations()
      conversations.value = Array.isArray(list) ? list : []
    } catch {
      conversations.value = []
    }
  }

  /** @returns false 表示会话在后端不存在（如重启后旧链接） */
  async function loadConversation(id: string): Promise<boolean> {
    loadingHistory.value = true
    try {
      const { messages: list } = await listMessages(id)
      activeConversationId.value = id
      messages.value = Array.isArray(list) ? list : []
      return true
    } catch (e) {
      const msg = (e as Error).message ?? ''
      if (msg.includes('conversation not found') || msg.includes('404')) {
        activeConversationId.value = null
        messages.value = []
        return false
      }
      throw e
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

    const runStream = async (conversationId?: string) => {
      await chatStream({
        conversationId,
        message: trimmed,
        signal: abortController!.signal,
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
    }

    try {
      try {
        await runStream(activeConversationId.value ?? undefined)
      } catch (e) {
        const msg = (e as Error).message ?? ''
        if (
          activeConversationId.value &&
          (msg.includes('conversation not found') || msg.includes('404'))
        ) {
          activeConversationId.value = null
          await runStream(undefined)
        } else {
          throw e
        }
      }
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

  async function removeConversation(id: string) {
    try {
      await deleteConversation(id)
    } catch {
      /* 忽略网络错误，仍刷新列表 */
    }
    if (activeConversationId.value === id) {
      activeConversationId.value = null
      messages.value = []
    }
    await refreshConversations()
  }

  return {
    conversations,
    messages,
    activeConversationId,
    loadingHistory,
    streaming,
    agents,
    health,
    refreshConversations,
    refreshAgents,
    refreshHealth,
    loadConversation,
    resetForNewChat,
    sendMessage,
    stopGeneration,
    removeConversation,
  }
})
