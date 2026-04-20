import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

export interface SlaPolicy {
  id: number
  accountId: number
  name: string
  firstResponseMinutes: number
  resolutionMinutes: number
  businessHoursOnly: boolean
  createdAt: string
}

export const useSlaStore = defineStore('sla', {
  state: () => ({ items: [] as SlaPolicy[], loading: false }),
  actions: {
    async fetch() {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      this.loading = true
      try {
        const res = await api<{ payload?: SlaPolicy[] } | SlaPolicy[]>(`/accounts/${auth.account.id}/slas`)
        this.items = Array.isArray(res) ? res : (res.payload ?? [])
      } finally {
        this.loading = false
      }
    },
    async save(sla: Partial<SlaPolicy>) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      if (sla.id) {
        await api(`/accounts/${auth.account.id}/slas/${sla.id}`, { method: 'PATCH', body: sla })
      } else {
        await api(`/accounts/${auth.account.id}/slas`, { method: 'POST', body: sla })
      }
      await this.fetch()
    },
    async remove(id: number) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/slas/${id}`, { method: 'DELETE' })
      this.items = this.items.filter(s => s.id !== id)
    }
  }
})
