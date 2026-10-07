<script setup lang="ts">
import type { Message } from '@/api/types'
import { renderSimpleMarkdown } from '@/utils/format'

defineProps<{
  messages: Message[]
  loading?: boolean
}>()
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
            <div v-if="msg.steps?.length" class="steps">
              <details
                v-for="(step, idx) in msg.steps"
                :key="idx"
                class="step"
                :open="idx === msg.steps!.length - 1 && msg.streaming"
              >
                <summary>
                  <span v-if="step.kind === 'tool'">工具 · {{ step.tool_name || '调用' }}</span>
                  <span v-else>进度 · {{ step.tool_name || '执行' }}</span>
                </summary>
                <pre class="step-body">{{ step.text }}</pre>
              </details>
            </div>
            <div
              v-if="msg.content"
              class="md"
              v-html="renderSimpleMarkdown(msg.content)"
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
  flex: 1;
  overflow-y: auto;
  padding: 24px 0 120px;
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
}
.bubble.assistant .md :deep(strong) {
  font-weight: 600;
}
.steps {
  margin-bottom: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
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
