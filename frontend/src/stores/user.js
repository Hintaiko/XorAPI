import { defineStore } from 'pinia'
import { get, post } from '../api'

export const useUser = defineStore('user', {
  state: () => ({
    user: JSON.parse(localStorage.getItem('user') || 'null'),
  }),
  getters: {
    loggedIn: (s) => !!localStorage.getItem('token'),
    isAdmin: (s) => s.user?.role === 'admin',
  },
  actions: {
    setAuth(token, user) {
      localStorage.setItem('token', token)
      localStorage.setItem('user', JSON.stringify(user))
      this.user = user
    },
    async login(email, password) {
      const data = await post('/api/auth/login', { email, password })
      this.setAuth(data.token, data.user)
    },
    async refresh() {
      if (!this.loggedIn) return
      try {
        const data = await get('/api/user/profile')
        this.user = data.user
        localStorage.setItem('user', JSON.stringify(data.user))
      } catch {
        /* 401 已由拦截器处理 */
      }
    },
    logout() {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      this.user = null
    },
  },
})
