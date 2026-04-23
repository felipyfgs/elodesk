import { defineStore } from 'pinia'

export interface AuthUser {
  id: string
  email: string
  name: string
}

export interface AuthAccountUser {
  userId: string
  accountId: string
  role: number
}

export interface AuthAccount {
  id: string
  name: string
  slug: string
  locale: string
  status: number
  customAttributes?: Record<string, unknown>
  settings?: Record<string, unknown>
}

interface AuthState {
  user: AuthUser | null
  account: AuthAccount | null
  accounts: AuthAccount[]
  accountUser: AuthAccountUser | null
  accessToken: string | null
  refreshToken: string | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    account: null,
    accounts: [],
    accountUser: null,
    accessToken: null,
    refreshToken: null
  }),
  getters: {
    isAuthenticated: s => !!s.accessToken && !!s.user
  },
  actions: {
    setSession(payload: { user: AuthUser, account?: AuthAccount, accounts?: AuthAccount[], accountUser?: AuthAccountUser, accessToken: string, refreshToken: string }) {
      this.user = { ...payload.user, id: String(payload.user.id) }
      if (payload.account) {
        const account = { ...payload.account, id: String(payload.account.id) }
        this.account = account
        if (!this.accounts.some(a => a.id === account.id)) {
          this.accounts.unshift(account)
        }
      }
      if (payload.accounts) this.accounts = payload.accounts.map(a => ({ ...a, id: String(a.id) }))
      if (payload.accountUser) this.accountUser = payload.accountUser
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
      this.accounts = []
      this.accountUser = null
      this.accessToken = null
      this.refreshToken = null
      if (import.meta.client) sessionStorage.removeItem('auth')
    },
    hydrate() {
      if (!import.meta.client) return
      const raw = sessionStorage.getItem('auth')
      if (!raw) {
        this.$reset()
        return
      }
      try {
        const data = JSON.parse(raw) as AuthState
        this.user = data.user ?? null
        this.account = data.account ?? null
        this.accounts = data.accounts ?? []
        this.accountUser = data.accountUser ?? null
        this.accessToken = data.accessToken ?? null
        this.refreshToken = data.refreshToken ?? null
      } catch (error) {
        if (import.meta.dev) console.error('[auth] hydrate failed', error)
        sessionStorage.removeItem('auth')
        this.$reset()
      }
    },
    setActiveAccount(id: string) {
      const found = this.accounts.find(a => String(a.id) === id)
      if (found) {
        this.account = found
        this.persist()
      }
    },
    persist() {
      if (!import.meta.client) return
      sessionStorage.setItem(
        'auth',
        JSON.stringify({
          user: this.user,
          account: this.account,
          accounts: this.accounts,
          accountUser: this.accountUser,
          accessToken: this.accessToken,
          refreshToken: this.refreshToken
        })
      )
    }
  }
})
