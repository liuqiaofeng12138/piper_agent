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

  function patchMessage(msgId: string, patch: Partial<Message>) {
    const idx = messages.value.findIndex((m) => m.id === msgId)
    if (idx < 0) return
    const cur = messages.value[idx]
    messages.value[idx] = { ...cur, ...patch }
  }

  function appendStep(msgId: string, step: NonNullable<Message['steps']>[number]) {
    const idx = messages.value.findIndex((m) => m.id === msgId)
    if (idx < 0) return
    const cur = messages.value[idx]
    const steps = [...(cur.steps ?? []), step]
    messages.value[idx] = { ...cur, steps }
  }

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
  async function loadConversation(id: string, opts?: { force?: boolean }): Promise<boolean> {
    if (streaming.value) {
      return true
    }
    if (
      !opts?.force &&
      id === activeConversationId.value &&
      messages.value.length > 0
    ) {
      return true
    }
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

  async function sendMessage(text: string, files?: File[]) {
    const trimmed = text.trim()
    const hasFiles = Boolean(files && files.length > 0)
    if ((!trimmed && !hasFiles) || streaming.value) return

    let display = trimmed
    if (hasFiles && files) {
      const names = files.map((f) => f.name).join(', ')
      display = trimmed
        ? `${trimmed}\n\n[附件: ${names}]`
        : `[附件: ${names}]`
    }

    const userMsg: Message = {
      id: `local_${Date.now()}`,
      role: 'user',
      content: display,
      created_at: new Date().toISOString(),
    }
    messages.value.push(userMsg)

    const assistantId = `local_${Date.now()}_a`
    const assistantMsg: Message = {
      id: assistantId,
      role: 'assistant',
      content: '',
      created_at: new Date().toISOString(),
      streaming: true,
    }
    messages.value.push(assistantMsg)

    streaming.value = true
    abortController = new AbortController()
    currentRunId.value = null

    const applyEvent = (ev: StreamEvent) => {
      const idx = messages.value.findIndex((m) => m.id === assistantId)
      if (idx < 0) return
      const cur = messages.value[idx]

      if (ev.conversation_id && !activeConversationId.value) {
        activeConversationId.value = ev.conversation_id
      }
      if (ev.run_id) currentRunId.value = ev.run_id

      let next: Message = { ...cur }
      if (ev.agent_id) next.agent_id = ev.agent_id
      if (ev.type === 'message.delta' && ev.delta) {
        next.content = (next.content || '') + ev.delta
      }
      if (ev.type === 'message.done' && ev.content) {
        next.content = ev.content
      }
      if (ev.type === 'tool.call') {
        next.steps = [
          ...(next.steps ?? []),
          {
            kind: 'tool',
            tool_name: ev.tool_name,
            text: ev.tool_detail || ev.message || ev.tool_name || 'tool',
          },
        ]
      }
      if (ev.type === 'run.progress') {
        next.steps = [
          ...(next.steps ?? []),
          {
            kind: 'progress',
            tool_name: ev.tool_name,
            text: ev.message || '',
          },
        ]
      }
      if (ev.type === 'run.cancelled') {
        next.content = `${next.content || ''}\n\n[已停止生成]`
      }
      if (ev.type === 'error' && ev.message) {
        next.content = `${next.content || ''}\n\n错误：${ev.message}`
      }
      messages.value[idx] = next
    }

    const runStream = async (conversationId?: string) => {
      await chatStream({
        conversationId,
        message: trimmed,
        files: hasFiles ? files : undefined,
        signal: abortController!.signal,
        onEvent: applyEvent,
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
        patchMessage(assistantId, {
          content: `请求失败：${(e as Error).message}`,
        })
      }
    } finally {
      patchMessage(assistantId, { streaming: false })
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
