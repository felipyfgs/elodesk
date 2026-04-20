import { $fetch, type FetchOptions } from 'ofetch'
import { useAuthStore } from '~/stores/auth'

let refreshPromise: Promise<void> | null = null

export const useApi = () => {
  const runtime = useRuntimeConfig()
  const auth = useAuthStore()

  const api = $fetch.create({
    baseURL: runtime.public.apiUrl,
    onRequest({ options }) {
      const token = auth.accessToken
      if (token) {
        const headers = new Headers(options.headers as HeadersInit | undefined)
        headers.set('Authorization', `Bearer ${token}`)
        if (auth.account?.id) headers.set('X-Account-Id', auth.account.id)
        options.headers = headers
      }
    },
    onResponse({ response }) {
      if (response._data?.success && response._data?.data !== undefined) {
        response._data = response._data.data
      }
    },
    async onResponseError({ response, request, options }) {
      if (response.status === 401 && auth.refreshToken && !(options as { _retried?: boolean })._retried) {
        try {
          await refreshOnce()
          ;(options as { _retried?: boolean })._retried = true
          await $fetch(request as string, options as FetchOptions)
          return
        } catch {
          auth.clear()
          if (import.meta.client) await navigateTo('/login')
        }
      }
    }
  })

  async function refreshOnce(): Promise<void> {
    if (refreshPromise) return refreshPromise
    refreshPromise = (async () => {
      try {
        const raw = await $fetch<{ success: boolean, data: { accessToken: string, refreshToken: string } }>(
          '/auth/refresh',
          {
            baseURL: runtime.public.apiUrl,
            method: 'POST',
            body: { refreshToken: auth.refreshToken }
          }
        )
        const res = raw.data ?? raw as unknown as { accessToken: string, refreshToken: string }
        auth.setTokens(res.accessToken, res.refreshToken)
      } finally {
        refreshPromise = null
      }
    })()
    return refreshPromise
  }

  return api
}
