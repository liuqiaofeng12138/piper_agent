export type MessageRole = 'user' | 'assistant'

export interface MessageStep {
  kind: 'tool' | 'progress'
  tool_name?: string
  text: string
}

export interface Message {
  id: string
  role: MessageRole
  content: string
  created_at: string
  streaming?: boolean
  steps?: MessageStep[]
}

export interface Conversation {
  id: string
  title: string
  created_at: string
  updated_at: string
}

export interface AgentInfo {
  id: string
  display_name: string
  description: string
  enabled: boolean
}

export type StreamEventType =
  | 'run.started'
  | 'message.delta'
  | 'message.done'
  | 'tool.call'
  | 'run.progress'
  | 'run.cancelled'
  | 'error'

export interface StreamEvent {
  type: StreamEventType
  trace_id?: string
  conversation_id?: string
  run_id?: string
  agent_id?: string
  delta?: string
  content?: string
  message?: string
  tool_name?: string
  tool_detail?: string
}
