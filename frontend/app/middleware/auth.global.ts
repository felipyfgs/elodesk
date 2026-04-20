import { useAuthStore } from '~/stores/auth'

const publicRoutes = new Set(['/login', '/register', '/forgot-password', '/reset-password'])

let setupChecked = false
let systemHasUsers = true

export function markSystemSetup() {
  systemHasUsers = true
  setupChecked = true
}

export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuthStore()
  if (!auth.user && auth.accessToken === null) auth.hydrate()

  if (!setupChecked) {
    try {
      const runtime = useRuntimeConfig()
      const res = await $fetch<{ hasUsers: boolean }>('/auth/setup', { baseURL: runtime.public.apiUrl })
      systemHasUsers = res.hasUsers
      setupChecked = true
    } catch {
      setupChecked = true
    }
  }

  if (!systemHasUsers && to.path !== '/register') {
    return navigateTo('/register')
  }

  if (publicRoutes.has(to.path)) {
    if (auth.isAuthenticated && to.path !== '/') return navigateTo('/sessions')
    return
  }

  if (!auth.isAuthenticated) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }
})
