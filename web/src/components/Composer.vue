<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  streaming: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  send: [text: string, files?: File[]]
  stop: []
}>()

const text = ref('')
const files = ref<File[]>([])
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

const acceptTypes =
  '.pdf,.doc,.docx,.docm,application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document'

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
  const batch = [...files.value]
  if ((!v && batch.length === 0) || props.disabled || props.streaming) return
  emit('send', v, batch.length ? batch : undefined)
  text.value = ''
  files.value = []
  autoResize()
}

function onStop() {
  emit('stop')
}

function openFilePicker() {
  fileInputRef.value?.click()
}

function onFilesSelected(e: Event) {
  const input = e.target as HTMLInputElement
  if (!input.files?.length) return
  files.value.push(...Array.from(input.files))
  input.value = ''
}

function removeFile(index: number) {
  files.value.splice(index, 1)
}
</script>

<template>
  <div class="composer-outer">
    <div v-if="files.length" class="attachments">
      <span
        v-for="(f, idx) in files"
        :key="`${f.name}-${idx}`"
        class="file-chip"
      >
        {{ f.name }}
        <button type="button" class="file-remove" aria-label="移除" @click="removeFile(idx)">
          ×
        </button>
      </span>
    </div>
    <div class="composer">
      <input
        ref="fileInputRef"
        type="file"
        class="file-input"
        multiple
        :accept="acceptTypes"
        @change="onFilesSelected"
      />
      <button
        type="button"
        class="btn attach"
        title="上传 PDF / Word 文档"
        :disabled="disabled || streaming"
        @click="openFilePicker"
      >
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48" />
        </svg>
      </button>
      <textarea
        ref="textareaRef"
        v-model="text"
        class="input"
        rows="1"
        placeholder="发送消息，或上传 PDF/Word 进行文档问答"
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
          :disabled="(!text.trim() && !files.length) || disabled"
          @click="submit"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M12 3l9 9h-6v9h-6v-9H3l9-9z" />
          </svg>
        </button>
      </div>
    </div>
    <p class="hint">
      Enter 发送 · Shift+Enter 换行 · 上传文档将自动使用文档问答 Agent（跳过意图识别）
    </p>
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
.attachments {
  max-width: 800px;
  margin: 0 auto 8px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  pointer-events: auto;
}
.file-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  font-size: 12px;
  border-radius: 999px;
  background: #eef2ff;
  color: var(--ds-primary);
  border: 1px solid #c7d2fe;
}
.file-remove {
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0 2px;
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
.file-input {
  display: none;
}
.btn.attach {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 10px;
  background: var(--ds-bg);
  color: var(--ds-text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
}
.btn.attach:hover:not(:disabled) {
  background: #f1f5f9;
  color: var(--ds-primary);
}
.btn.attach:disabled {
  opacity: 0.4;
  cursor: not-allowed;
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
