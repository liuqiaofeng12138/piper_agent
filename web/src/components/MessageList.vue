<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '@/stores/chat'
import type { Message } from '@/api/types'
import { renderMarkdown } from '@/utils/markdown'

defineProps<{
  messages: Message[]
  loading?: boolean
}>()

const store = useChatStore()
const agentNames = computed(() => {
  const map: Record<string, string> = {}
  for (const a of store.agents) map[a.id] = a.display_name
  return map
})
</script>

<template>
  <div class="messages">
    <div v-if="loading" class="loading">加载对话中…</div>
    <template v-else>
      <div
        v-for="msg in messages"
        :key="msg.id"
        class="row"
        :class="msg.role"
      >
        <div v-if="msg.role === 'assistant'" class="avatar assistant">P</div>
        <div class="bubble-wrap">
          <div v-if="msg.role === 'user'" class="bubble user">
            {{ msg.content }}
          </div>
          <div
            v-else
            class="bubble assistant"
            :class="{ streaming: msg.streaming }"
          >
            <div v-if="msg.agent_id" class="agent-tag">
              {{ agentNames[msg.agent_id] || msg.agent_id }}
            </div>
            <div v-if="msg.steps?.length" class="steps">
              <div class="steps-title">执行步骤</div>
              <details
                v-for="(step, idx) in msg.steps"
                :key="idx"
                class="step"
                :open="idx === msg.steps!.length - 1 && msg.streaming"
              >
                <summary>
                  <span v-if="step.kind === 'tool'" class="step-kind">工具</span>
                  <span v-else class="step-kind progress">进度</span>
                  <span class="step-name">{{ step.tool_name || '调用' }}</span>
                </summary>
                <pre class="step-body">{{ step.text }}</pre>
              </details>
            </div>
            <div
              v-if="msg.content"
              class="md-body"
              v-html="renderMarkdown(msg.content)"
            />
            <span v-else-if="msg.streaming" class="typing">正在思考</span>
            <span v-if="msg.streaming && msg.content" class="cursor">▍</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.messages {
  height: 100%;
  padding: 24px 0 120px;
  box-sizing: border-box;
}
.loading {
  text-align: center;
  color: var(--ds-text-muted);
  padding: 48px;
}
.row {
  display: flex;
  gap: 12px;
  max-width: 800px;
  margin: 0 auto 24px;
  padding: 0 24px;
}
.row.user {
  flex-direction: row-reverse;
}
.avatar {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
}
.avatar.assistant {
  background: linear-gradient(135deg, #4d6bfe, #6b8cff);
  color: #fff;
}
.bubble-wrap {
  flex: 1;
  min-width: 0;
  display: flex;
}
.row.user .bubble-wrap {
  justify-content: flex-end;
}
.bubble.user {
  background: var(--ds-user-bubble);
  color: var(--ds-text);
  padding: 12px 16px;
  border-radius: var(--ds-radius-lg);
  max-width: 85%;
  font-size: 15px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}
.bubble.assistant {
  font-size: 15px;
  line-height: 1.75;
  color: var(--ds-text);
  word-break: break-word;
  min-width: 0;
}
.agent-tag {
  display: inline-block;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
  background: #e8eeff;
  color: var(--ds-primary);
  font-weight: 500;
  margin-bottom: 6px;
}
.steps {
  margin-bottom: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.steps-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--ds-text-muted);
  margin-bottom: 2px;
}
.step {
  border: 1px solid var(--ds-input-border);
  border-radius: 8px;
  background: #f9fafb;
  font-size: 13px;
}
.step summary {
  cursor: pointer;
  padding: 8px 12px;
  color: var(--ds-text-secondary);
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 8px;
}
.step-kind {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  background: #e8eeff;
  color: var(--ds-primary);
}
.step-kind.progress {
  background: #ecfdf5;
  color: #059669;
}
.step-name {
  overflow: hidden;
  text-overflow: ellipsis;
}
.step-body {
  margin: 0;
  padding: 8px 12px 12px;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  color: var(--ds-text-secondary);
  border-top: 1px solid var(--ds-input-border);
  max-height: 200px;
  overflow: auto;
}
/* Markdown 正文（助手回答） */
.md-body :deep(p) {
  margin: 0 0 0.75em;
}
.md-body :deep(p:last-child) {
  margin-bottom: 0;
}
.md-body :deep(h1),
.md-body :deep(h2),
.md-body :deep(h3),
.md-body :deep(h4) {
  margin: 1em 0 0.5em;
  font-weight: 600;
  line-height: 1.35;
}
.md-body :deep(h1) {
  font-size: 1.35em;
}
.md-body :deep(h2) {
  font-size: 1.2em;
}
.md-body :deep(h3) {
  font-size: 1.05em;
}
.md-body :deep(ul),
.md-body :deep(ol) {
  margin: 0.5em 0 0.75em;
  padding-left: 1.5em;
}
.md-body :deep(li) {
  margin: 0.25em 0;
}
.md-body :deep(li > p) {
  margin: 0.25em 0;
}
.md-body :deep(blockquote) {
  margin: 0.75em 0;
  padding: 0.35em 0 0.35em 1em;
  border-left: 3px solid #c7d2fe;
  color: var(--ds-text-secondary);
  background: #f8faff;
  border-radius: 0 6px 6px 0;
}
.md-body :deep(code) {
  font-family: ui-monospace, 'Cascadia Code', 'Segoe UI Mono', monospace;
  font-size: 0.9em;
  padding: 0.15em 0.4em;
  border-radius: 4px;
  background: #f3f4f6;
  color: #1e293b;
}
.md-body :deep(pre) {
  margin: 0.75em 0;
  padding: 12px 14px;
  border-radius: 8px;
  background: #1e293b;
  color: #e2e8f0;
  overflow-x: auto;
  font-size: 13px;
  line-height: 1.5;
}
.md-body :deep(pre code) {
  padding: 0;
  background: transparent;
  color: inherit;
  font-size: inherit;
}
.md-body :deep(a) {
  color: var(--ds-primary);
  text-decoration: none;
}
.md-body :deep(a:hover) {
  text-decoration: underline;
}
.md-body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 0.75em 0;
  font-size: 14px;
}
.md-body :deep(th),
.md-body :deep(td) {
  border: 1px solid var(--ds-input-border);
  padding: 8px 10px;
  text-align: left;
}
.md-body :deep(th) {
  background: #f9fafb;
  font-weight: 600;
}
.md-body :deep(hr) {
  border: none;
  border-top: 1px solid var(--ds-input-border);
  margin: 1em 0;
}
.md-body :deep(strong) {
  font-weight: 600;
}
.typing {
  color: var(--ds-text-muted);
}
.cursor {
  animation: blink 1s step-end infinite;
  color: var(--ds-primary);
  margin-left: 2px;
}
@keyframes blink {
  50% {
    opacity: 0;
  }
}
</style>
