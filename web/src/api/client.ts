import type { AgentInfo, Conversation, HealthResponse, Message, StreamEvent } from './types'

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

export async function healthCheck(): Promise<HealthResponse> {
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

export async function deleteConversation(conversationId: string): Promise<void> {
  const res = await fetch(`${base}/conversations/${conversationId}`, { method: 'DELETE' })
  if (!res.ok) {
    throw new Error(await res.text())
  }
}

export interface ChatStreamOptions {
  conversationId?: string
  message: string
  files?: File[]
  signal?: AbortSignal
  onEvent: (ev: StreamEvent) => void
}

export async function chatStream(opts: ChatStreamOptions): Promise<void> {
  const headers: Record<string, string> = {
    Accept: 'text/event-stream',
    'Cache-Control': 'no-cache',
  }
  let body: BodyInit
  if (opts.files && opts.files.length > 0) {
    const fd = new FormData()
    if (opts.conversationId) fd.append('conversation_id', opts.conversationId)
    fd.append('message', opts.message)
    fd.append('stream', 'true')
    for (const f of opts.files) {
      fd.append('files', f, f.name)
    }
    body = fd
  } else {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify({
      conversation_id: opts.conversationId || undefined,
      message: opts.message,
      stream: true,
    })
  }
  const res = await fetch(`${base}/chat/completions`, {
    method: 'POST',
    signal: opts.signal,
    headers,
    body,
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
