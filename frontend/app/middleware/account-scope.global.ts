import { useAuthStore } from '~/stores/auth'

const publicPrefixes = ['/login', '/register', '/forgot-password', '/reset-password']

export default defineNuxtRouteMiddleware((to) => {
  if (publicPrefixes.some(p => to.path.startsWith(p))) return

  const accountId = to.params.accountId as string | undefined
  if (!accountId) return

  const auth = useAuthStore()

  if (!auth.isAuthenticated) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }

  const found = auth.accounts.find(a => String(a.id) === accountId)
  if (!found) {
    const primary = auth.accounts[0]
    if (primary) {
      return navigateTo(`/accounts/${primary.id}`)
    }
    return navigateTo('/login')
  }

  if (String(auth.account?.id) !== accountId) {
    auth.setActiveAccount(accountId)
  }
})
