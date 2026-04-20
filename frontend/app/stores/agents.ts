import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

export interface Agent {
  id: number
  email: string
  name: string
  role: number
  createdAt: string
  lastActiveAt?: string | null
}

export const useAgentsStore = defineStore('agents', {
  state: () => ({ items: [] as Agent[], loading: false }),
  actions: {
    async fetch() {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      this.loading = true
      try {
        const res = await api<{ payload?: Agent[] } | Agent[]>(`/accounts/${auth.account.id}/agents`)
        this.items = Array.isArray(res) ? res : (res.payload ?? [])
      } finally {
        this.loading = false
      }
    },
    async invite(email: string, role: string, name?: string) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      return api(`/accounts/${auth.account.id}/agents/invite`, {
        method: 'POST',
        body: { email, role, name }
      })
    },
    async update(userId: number, patch: { role?: string, status?: string }) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/agents/${userId}`, {
        method: 'PATCH', body: patch
      })
      await this.fetch()
    },
    async remove(userId: number) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/agents/${userId}`, { method: 'DELETE' })
      this.items = this.items.filter(a => a.id !== userId)
    }
  }
})
