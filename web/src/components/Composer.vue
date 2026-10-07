<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  streaming: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  send: [text: string]
  stop: []
}>()

const text = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)

function autoResize() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 200)}px`
}

watch(text, () => autoResize())

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    submit()
  }
}

function submit() {
  const v = text.value.trim()
  if (!v || props.disabled || props.streaming) return
  emit('send', v)
  text.value = ''
  autoResize()
}

function onStop() {
  emit('stop')
}
</script>

<template>
  <div class="composer-outer">
    <div class="composer">
      <textarea
        ref="textareaRef"
        v-model="text"
        class="input"
        rows="1"
        placeholder="给 Piper Agent 发送消息"
        :disabled="disabled"
        @keydown="onKeydown"
        @input="autoResize"
      />
      <div class="actions">
        <button
          v-if="streaming"
          type="button"
          class="btn stop"
          @click="onStop"
        >
          停止
        </button>
        <button
          v-else
          type="button"
          class="btn send"
          :disabled="!text.trim() || disabled"
          @click="submit"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M12 3l9 9h-6v9h-6v-9H3l9-9z" />
          </svg>
        </button>
      </div>
    </div>
    <p class="hint">Enter 发送 · Shift+Enter 换行 · 由网页采集 Agent 处理</p>
  </div>
</template>

<style scoped>
.composer-outer {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 0 24px 20px;
  background: linear-gradient(180deg, transparent, var(--ds-bg) 24%);
  pointer-events: none;
}
.composer {
  max-width: 800px;
  margin: 0 auto;
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 12px 14px;
  background: var(--ds-input-bg);
  border: 1px solid var(--ds-input-border);
  border-radius: var(--ds-radius-lg);
  box-shadow: 0 4px 24px rgba(15, 23, 42, 0.06);
  pointer-events: auto;
}
.input {
  flex: 1;
  border: none;
  background: transparent;
  resize: none;
  font-size: 15px;
  line-height: 1.5;
  color: var(--ds-text);
  outline: none;
  max-height: 200px;
  font-family: inherit;
}
.input::placeholder {
  color: var(--ds-text-muted);
}
.actions {
  flex-shrink: 0;
}
.btn {
  border: none;
  border-radius: 10px;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, opacity 0.15s;
}
.btn.send {
  background: var(--ds-primary);
  color: #fff;
}
.btn.send:hover:not(:disabled) {
  background: var(--ds-primary-hover);
}
.btn.send:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.btn.stop {
  width: auto;
  padding: 0 14px;
  font-size: 13px;
  background: var(--ds-bg);
  color: var(--ds-danger);
  border: 1px solid #fecaca;
}
.hint {
  max-width: 800px;
  margin: 8px auto 0;
  text-align: center;
  font-size: 12px;
  color: var(--ds-text-muted);
  pointer-events: none;
}
</style>
