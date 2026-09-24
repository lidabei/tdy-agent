import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import http from '../api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('tdy_token') || '')
  const user = ref(JSON.parse(localStorage.getItem('tdy_user') || 'null'))

  const isAdmin = computed(() => user.value?.role === 'admin')
  const isLoggedIn = computed(() => !!token.value)
  const permissions = computed(() => user.value?.permissions || [])

  function setSession(t, u) {
    token.value = t
    user.value = u
    localStorage.setItem('tdy_token', t)
    localStorage.setItem('tdy_user', JSON.stringify(u))
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('tdy_token')
    localStorage.removeItem('tdy_user')
  }

  function hasPerm(...codes) {
    if (!isAdmin.value) return false
    const set = new Set(permissions.value || [])
    return codes.some((c) => set.has(c))
  }

  async function login(username, password) {
    const { data } = await http.post('/auth/login', { username, password })
    setSession(data.token, data.user)
    return data.user
  }

  async function fetchMe() {
    if (!token.value) return null
    const { data } = await http.get('/auth/me')
    user.value = data
    localStorage.setItem('tdy_user', JSON.stringify(data))
    return data
  }

  return { token, user, isAdmin, isLoggedIn, permissions, hasPerm, login, logout, fetchMe, setSession }
})
