<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { login } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    const res = await login(username.value.trim(), password.value)
    auth.setSession(res.token, res.username)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await router.replace(redirect || '/')
  } catch (e) {
    error.value = e instanceof Error ? e.message : '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <form class="card" @submit.prevent="onSubmit">
      <h1>Piper Agent</h1>
      <p class="hint">请登录后使用</p>
      <label>
        <span>用户名</span>
        <input v-model="username" type="text" autocomplete="username" required />
      </label>
      <label>
        <span>密码</span>
        <input v-model="password" type="password" autocomplete="current-password" required />
      </label>
      <p v-if="error" class="error">{{ error }}</p>
      <button type="submit" :disabled="loading">{{ loading ? '登录中…' : '登录' }}</button>
    </form>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--ds-bg, #f4f4f5);
  padding: 24px;
}
.card {
  width: 100%;
  max-width: 360px;
  background: var(--ds-surface, #fff);
  border: 1px solid var(--ds-input-border, #e4e4e7);
  border-radius: var(--ds-radius-lg, 12px);
  padding: 28px 24px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.06);
}
h1 {
  margin: 0;
  font-size: 1.35rem;
  color: var(--ds-text, #18181b);
}
.hint {
  margin: -4px 0 4px;
  color: var(--ds-text-muted, #71717a);
  font-size: 0.9rem;
}
label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 0.85rem;
  color: var(--ds-text-secondary, #3f3f46);
}
input {
  border: 1px solid var(--ds-input-border, #e4e4e7);
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 1rem;
}
button {
  margin-top: 4px;
  border: none;
  border-radius: 8px;
  padding: 10px 14px;
  background: var(--ds-primary, #2563eb);
  color: #fff;
  font-size: 1rem;
  cursor: pointer;
}
button:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}
.error {
  margin: 0;
  color: #dc2626;
  font-size: 0.875rem;
}
</style>
