import { defineStore } from 'pinia'

export interface AuthUser {
  id: string
  email: string
  name: string
}

export interface AuthAccount {
  id: string
  name: string
  slug: string
}

interface AuthState {
  user: AuthUser | null
  account: AuthAccount | null
  accessToken: string | null
  refreshToken: string | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    account: null,
    accessToken: null,
    refreshToken: null
  }),
  getters: {
    isAuthenticated: s => !!s.accessToken && !!s.user
  },
  actions: {
    setSession(payload: { user: AuthUser, account?: AuthAccount, accessToken: string, refreshToken: string }) {
      this.user = payload.user
      if (payload.account) this.account = payload.account
      this.accessToken = payload.accessToken
      this.refreshToken = payload.refreshToken
      this.persist()
    },
    setTokens(accessToken: string, refreshToken: string) {
      this.accessToken = accessToken
      this.refreshToken = refreshToken
      this.persist()
    },
    clear() {
      this.user = null
      this.account = null
      this.accessToken = null
      this.refreshToken = null
      if (import.meta.client) localStorage.removeItem('auth')
    },
    hydrate() {
      if (!import.meta.client) return
      const raw = localStorage.getItem('auth')
      if (!raw) return
      try {
        const data = JSON.parse(raw) as AuthState
        this.$patch(data)
      } catch {
        localStorage.removeItem('auth')
      }
    },
    persist() {
      if (!import.meta.client) return
      localStorage.setItem(
        'auth',
        JSON.stringify({
          user: this.user,
          account: this.account,
          accessToken: this.accessToken,
          refreshToken: this.refreshToken
        })
      )
    }
  }
})
