import type { AgentInfo, Conversation, Message, StreamEvent } from './types'

const base = import.meta.env.VITE_API_BASE ?? '/api/v1'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${base}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
  })
  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || res.statusText)
  }
  return res.json() as Promise<T>
}

export async function healthCheck(): Promise<{ status: string }> {
  return request('/health')
}

export async function listAgents(): Promise<{ agents: AgentInfo[] }> {
  return request('/agents')
}

export async function createConversation(title = ''): Promise<Conversation> {
  return request('/conversations', {
    method: 'POST',
    body: JSON.stringify({ title }),
  })
}

export async function listConversations(): Promise<{ conversations: Conversation[] }> {
  return request('/conversations')
}

export async function listMessages(conversationId: string): Promise<{ messages: Message[] }> {
  return request(`/conversations/${conversationId}/messages`)
}

export interface ChatStreamOptions {
  conversationId?: string
  message: string
  signal?: AbortSignal
  onEvent: (ev: StreamEvent) => void
}

export async function chatStream(opts: ChatStreamOptions): Promise<void> {
  const res = await fetch(`${base}/chat/completions`, {
    method: 'POST',
    signal: opts.signal,
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      conversation_id: opts.conversationId || undefined,
      message: opts.message,
      stream: true,
    }),
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  const reader = res.body?.getReader()
  if (!reader) throw new Error('no response body')

  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const parts = buffer.split('\n\n')
    buffer = parts.pop() ?? ''
    for (const part of parts) {
      for (const line of part.split('\n')) {
        if (!line.startsWith('data: ')) continue
        const payload = line.slice(6).trim()
        if (!payload) continue
        try {
          opts.onEvent(JSON.parse(payload) as StreamEvent)
        } catch {
          /* ignore malformed chunk */
        }
      }
    }
  }
}

export async function cancelRun(runId: string): Promise<void> {
  await request(`/runs/${runId}/cancel`, { method: 'POST' })
}
