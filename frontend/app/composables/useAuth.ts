import { useAuthStore } from '~/stores/auth'

export interface LoginMfaRequired {
  mfaRequired: true
  mfaToken: string
}

export interface LoginSuccess {
  user: { id: number, email: string, name: string }
  account: {
    id: number
    name: string
    slug: string
    locale: string
    status: number
    customAttributes?: Record<string, unknown>
    settings?: Record<string, unknown>
  }
  accessToken: string
  refreshToken: string
}

export const useAuth = () => {
  const auth = useAuthStore()
  const api = useApi()
  const runtime = useRuntimeConfig()

  async function login(email: string, password: string): Promise<LoginMfaRequired | LoginSuccess> {
    const res = await api<LoginMfaRequired | LoginSuccess>(
      '/auth/login',
      { method: 'POST', body: { email, password } }
    )

    if ('mfaRequired' in res && res.mfaRequired) {
      return res as LoginMfaRequired
    }

    const data = res as LoginSuccess
    auth.setSession({
      user: { id: String(data.user.id), email: data.user.email, name: data.user.name },
      account: { ...data.account, id: String(data.account.id) },
      accessToken: data.accessToken,
      refreshToken: data.refreshToken
    })
    await refreshMemberships()
    return data
  }

  async function verifyMfa(mfaToken: string, code: string): Promise<LoginSuccess> {
    const res = await api<LoginSuccess>(
      '/auth/mfa/verify',
      { method: 'POST', body: { mfaToken, code } }
    )
    auth.setSession({
      user: { id: String(res.user.id), email: res.user.email, name: res.user.name },
      account: { ...res.account, id: String(res.account.id) },
      accessToken: res.accessToken,
      refreshToken: res.refreshToken
    })
    await refreshMemberships()
    return res
  }

  async function register(payload: { email: string, password: string, name: string, accountName?: string }) {
    type RegisterData = {
      user: { id: number, email: string, name: string }
      account: {
        id: number
        name: string
        slug: string
        locale: string
        status: number
        customAttributes?: Record<string, unknown>
        settings?: Record<string, unknown>
      }
      accessToken: string
      refreshToken: string
    }
    const raw = await $fetch<{ success: boolean, data: RegisterData }>(
      '/auth/register', { baseURL: runtime.public.apiUrl, method: 'POST', body: payload }
    )
    const res = raw.data ?? raw as unknown as RegisterData
    auth.setSession({
      user: { id: String(res.user.id), email: res.user.email, name: res.user.name },
      account: { ...res.account, id: String(res.account.id) },
      accessToken: res.accessToken,
      refreshToken: res.refreshToken
    })
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

  return { login, verifyMfa, register, logout, auth }
}
