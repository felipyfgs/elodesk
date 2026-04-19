import { useAuthStore } from '~/stores/auth'

export const useAuth = () => {
  const auth = useAuthStore()
  const api = useApi()
  const runtime = useRuntimeConfig()

  async function login(email: string, password: string) {
    const res = await api<{ user: { id: string, email: string, name: string }, accessToken: string, refreshToken: string }>(
      '/auth/login',
      { method: 'POST', body: { email, password } }
    )
    auth.setSession({ user: res.user, accessToken: res.accessToken, refreshToken: res.refreshToken })
    await refreshMemberships()
  }

  async function register(payload: { email: string, password: string, name: string, accountName?: string }) {
    const res = await $fetch<{
      user: { id: string, email: string, name: string }
      account: { id: string, name: string, slug: string }
      accessToken: string
      refreshToken: string
    }>('/auth/register', { baseURL: runtime.public.apiUrl, method: 'POST', body: payload })
    auth.setSession({ user: res.user, account: res.account, accessToken: res.accessToken, refreshToken: res.refreshToken })
  }

  async function logout() {
    if (auth.refreshToken) {
      try {
        await api('/auth/logout', { method: 'POST', body: { refreshToken: auth.refreshToken } })
      } catch {
        /* ignore */
      }
    }
    auth.clear()
    await navigateTo('/login')
  }

  async function refreshMemberships() {
    // Placeholder — backend exposes `/me/accounts` (to implement).
  }

  return { login, register, logout, auth }
}
