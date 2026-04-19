import { useAuthStore } from '~/stores/auth'

const publicRoutes = new Set(['/login', '/register'])

export default defineNuxtRouteMiddleware((to) => {
  if (import.meta.server) return
  const auth = useAuthStore()
  if (!auth.user && auth.accessToken === null) auth.hydrate()

  if (publicRoutes.has(to.path)) {
    if (auth.isAuthenticated && to.path !== '/') return navigateTo('/sessions')
    return
  }

  if (!auth.isAuthenticated) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }
})
