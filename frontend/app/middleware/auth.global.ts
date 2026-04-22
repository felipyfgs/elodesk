import { useAuthStore } from '~/stores/auth'

const publicRoutes = new Set(['/login', '/register', '/forgot-password', '/reset-password'])

function useSetupState() {
  return useState('auth-setup', () => ({ checked: false, hasUsers: true }))
}

export function markSystemSetup() {
  if (import.meta.client) {
    const state = useSetupState()
    state.value.hasUsers = true
    state.value.checked = true
  }
}

export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuthStore()
  if (import.meta.client) auth.hydrate()

  const setup = useSetupState()

  if (!setup.value.checked) {
    try {
      const runtime = useRuntimeConfig()
      const res = await $fetch<{ hasUsers: boolean }>('/auth/setup', { baseURL: runtime.public.apiUrl })
      setup.value.hasUsers = res.hasUsers
      setup.value.checked = true
    } catch {
      setup.value.checked = true
    }
  }

  if (!setup.value.hasUsers && to.path !== '/register') {
    return navigateTo('/register')
  }

  if (publicRoutes.has(to.path)) {
    if (auth.isAuthenticated) {
      const primaryId = auth.accounts[0]?.id
      if (primaryId) {
        return navigateTo(`/accounts/${primaryId}`)
      }
    }
    return
  }

  if (!auth.isAuthenticated) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }

  if (to.path === '/') {
    const primaryId = auth.accounts[0]?.id
    if (primaryId) {
      return navigateTo(`/accounts/${primaryId}`, { redirectCode: 302 })
    }
  }
})
