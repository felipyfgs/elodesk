import { useAuthStore } from '~/stores/auth'

export function useAccountUrl() {
  const auth = useAuthStore()

  function accountUrl(path: string): string | null {
    if (!auth.account?.id) return null
    return `/accounts/${auth.account.id}${path}`
  }

  return { accountUrl }
}
