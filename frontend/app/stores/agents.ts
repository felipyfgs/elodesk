import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

export interface Agent {
  id: number
  userId: number
  email: string
  name: string
  role: number
  createdAt: string
  lastActiveAt?: string | null
}

const ROLE_MAP: Record<string, number> = { agent: 0, admin: 1, owner: 2 }

function roleToInt(role: string): number {
  return ROLE_MAP[role] ?? 0
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
        body: { email, role: roleToInt(role), name }
      })
    },
    async update(userId: number, patch: { role?: string, status?: string }) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      const body: { role?: number, status?: string } = {}
      if (patch.role !== undefined) body.role = roleToInt(patch.role)
      if (patch.status !== undefined) body.status = patch.status
      await api(`/accounts/${auth.account.id}/agents/${userId}`, {
        method: 'PATCH', body
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
