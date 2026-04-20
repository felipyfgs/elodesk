import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

export interface Macro {
  id: number
  accountId: number
  name: string
  visibility: string
  conditions: unknown
  actions: unknown
  createdBy: number
  createdAt: string
}

export const useMacrosStore = defineStore('macros', {
  state: () => ({ items: [] as Macro[], loading: false }),
  actions: {
    async fetch() {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      this.loading = true
      try {
        const res = await api<{ payload?: Macro[] } | Macro[]>(`/accounts/${auth.account.id}/macros`)
        this.items = Array.isArray(res) ? res : (res.payload ?? [])
      } finally {
        this.loading = false
      }
    },
    async save(macro: Partial<Macro>) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      if (macro.id) {
        await api(`/accounts/${auth.account.id}/macros/${macro.id}`, { method: 'PATCH', body: macro })
      } else {
        await api(`/accounts/${auth.account.id}/macros`, { method: 'POST', body: macro })
      }
      await this.fetch()
    },
    async remove(id: number) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/macros/${id}`, { method: 'DELETE' })
      this.items = this.items.filter(m => m.id !== id)
    }
  }
})
