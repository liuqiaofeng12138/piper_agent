import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

const STORAGE_KEY = 'piper_auth_token'
const USER_KEY = 'piper_auth_user'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(STORAGE_KEY))
  const username = ref<string | null>(localStorage.getItem(USER_KEY))

  const isLoggedIn = computed(() => Boolean(token.value))

  function setSession(tok: string, user: string) {
    token.value = tok
    username.value = user
    localStorage.setItem(STORAGE_KEY, tok)
    localStorage.setItem(USER_KEY, user)
  }

  function clearSession() {
    token.value = null
    username.value = null
    localStorage.removeItem(STORAGE_KEY)
    localStorage.removeItem(USER_KEY)
  }

  function authHeader(): Record<string, string> {
    if (!token.value) return {}
    return { Authorization: `Bearer ${token.value}` }
  }

  return { token, username, isLoggedIn, setSession, clearSession, authHeader }
})
