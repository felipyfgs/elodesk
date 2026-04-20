import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

export interface OutboundWebhook {
  id: number
  accountId: number
  url: string
  subscriptions: string[]
  isActive: boolean
  createdAt: string
}

export const useWebhooksStore = defineStore('webhooks', {
  state: () => ({ items: [] as OutboundWebhook[], loading: false }),
  actions: {
    async fetch() {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      this.loading = true
      try {
        const res = await api<{ payload?: OutboundWebhook[] } | OutboundWebhook[]>(`/accounts/${auth.account.id}/webhooks`)
        this.items = Array.isArray(res) ? res : (res.payload ?? [])
      } finally {
        this.loading = false
      }
    },
    async save(hook: Partial<OutboundWebhook> & { secret?: string }) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      if (hook.id) {
        await api(`/accounts/${auth.account.id}/webhooks/${hook.id}`, { method: 'PATCH', body: hook })
      } else {
        await api(`/accounts/${auth.account.id}/webhooks`, { method: 'POST', body: hook })
      }
      await this.fetch()
    },
    async remove(id: number) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/webhooks/${id}`, { method: 'DELETE' })
      this.items = this.items.filter(w => w.id !== id)
    }
  }
})
