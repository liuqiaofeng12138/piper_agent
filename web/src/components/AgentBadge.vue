<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '@/stores/chat'
import type { AgentInfo } from '@/api/types'

const store = useChatStore()

const agents = computed(() => store.agents)

function statusText(a: AgentInfo): string {
  if (!a.enabled) return '未启用'
  if (a.healthy === false) return '离线'
  return '在线'
}

function statusClass(a: AgentInfo): string {
  if (!a.enabled) return 'off'
  if (a.healthy === false) return 'bad'
  return 'ok'
}
</script>

<template>
  <div class="agent-badges">
    <div
      v-for="a in agents"
      :key="a.id"
      class="agent-badge"
      :class="statusClass(a)"
      :title="a.description"
    >
      <span class="dot" />
      <span class="name">{{ a.display_name }}</span>
      <span class="status">{{ statusText(a) }}</span>
    </div>
  </div>
</template>

<style scoped>
.agent-badges {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
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
