<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '@/stores/chat'

const store = useChatStore()

const primary = computed(() => store.agents.find((a) => a.enabled) ?? store.agents[0])

const statusText = computed(() => {
  if (!primary.value) return '未配置'
  if (!primary.value.enabled) return '未启用'
  if (primary.value.healthy === false) return '离线'
  return '在线'
})

const statusClass = computed(() => {
  if (!primary.value?.enabled) return 'off'
  if (primary.value.healthy === false) return 'bad'
  return 'ok'
})
</script>

<template>
  <div v-if="primary" class="agent-badge" :class="statusClass">
    <span class="dot" />
    <span class="name">{{ primary.display_name }}</span>
    <span class="status">{{ statusText }}</span>
  </div>
</template>

<style scoped>
.agent-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 999px;
  background: #e8eeff;
  color: var(--ds-primary);
  font-weight: 500;
}
.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}
.agent-badge.bad {
  background: #fef2f2;
  color: var(--ds-danger);
}
.agent-badge.off {
  background: var(--ds-hover);
  color: var(--ds-text-muted);
}
.status {
  opacity: 0.85;
  font-weight: 400;
}
</style>
