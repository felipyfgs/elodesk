import { $fetch, type FetchOptions } from 'ofetch'
import { useAuthStore } from '~/stores/auth'
import { normalizeApiResponse } from '~/utils/apiAdapter'

let refreshPromise: Promise<void> | null = null

export async function refreshAccessToken(): Promise<void> {
  if (refreshPromise) return refreshPromise
  const runtime = useRuntimeConfig()
  const auth = useAuthStore()
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

export const useApi = () => {
  const runtime = useRuntimeConfig()
  const auth = useAuthStore()

  const baseApi = $fetch.create({
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
      // Normalize Chatwoot-shape (snake_case + epoch seconds) responses to the
      // camelCase + ms-epoch form the stores/components were written against.
      // See utils/apiAdapter.ts for the rules.
      if (response._data) {
        response._data = normalizeApiResponse(response._data)
      }
    }
  })

  const api = async <T = unknown>(request: string, options?: FetchOptions<'json'>): Promise<T> => {
    try {
      return await baseApi<T>(request, options)
    } catch (err: unknown) {
      const e = err as { response?: { status?: number } }
      const status = e?.response?.status
      const retried = (options as { _retried?: boolean } | undefined)?._retried
      if (status === 401 && auth.refreshToken && !retried) {
        try {
          await refreshAccessToken()
          return await baseApi<T>(request, { ...(options ?? {}), _retried: true } as FetchOptions<'json'>)
        } catch {
          auth.clear()
          if (import.meta.client) await navigateTo('/login')
          throw err
        }
      }
      throw err
    }
  }

  return api
}
