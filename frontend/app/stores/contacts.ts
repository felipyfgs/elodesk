import { defineStore } from 'pinia'
import { useAuthStore } from '~/stores/auth'

export interface Contact {
  id: string
  accountId: string
  name: string | null
  email: string | null
  phoneNumber: string | null
  identifier: string | null
  additionalAttributes: string | null
  lastActivityAt: string | null
  createdAt: string
  updatedAt: string
}

export interface ContactMeta {
  page: number
  pageSize: number
  total: number
}

export interface ContactListResponse {
  meta: ContactMeta
  payload: Contact[]
}

export const useContactsStore = defineStore('contacts', {
  state: () => ({
    list: [] as Contact[],
    meta: { page: 1, pageSize: 25, total: 0 } as ContactMeta,
    loading: false
  }),
  actions: {
    async fetchPage(params: { search?: string, labels?: string, page?: number, pageSize?: number } = {}) {
      const auth = useAuthStore()
      if (!auth.account?.id) return

      this.loading = true
      try {
        const api = useApi()
        const query = new URLSearchParams()
        if (params.search) query.set('search', params.search)
        if (params.labels) query.set('labels', params.labels)
        if (params.page) query.set('page', String(params.page))
        if (params.pageSize) query.set('pageSize', String(params.pageSize))

        const qs = query.toString()
        const url = `/accounts/${auth.account.id}/contacts${qs ? `?${qs}` : ''}`
        const res = await api<ContactListResponse>(url)
        this.list = res.payload
        this.meta = res.meta
      } finally {
        this.loading = false
      }
    },

    setAll(list: Contact[]) {
      this.list = list
    },

    upsert(contact: Contact) {
      const idx = this.list.findIndex(c => c.id === contact.id)
      if (idx >= 0) this.list[idx] = contact
      else this.list.unshift(contact)
    },

    removeMany(ids: string[]) {
      const set = new Set(ids)
      this.list = this.list.filter(c => !set.has(c.id))
    },

    async applyBulkLabel(ids: string[], labelId: string, op: 'add' | 'remove') {
      const auth = useAuthStore()
      if (!auth.account?.id) return

      const api = useApi()
      for (const id of ids) {
        try {
          if (op === 'add') {
            await api(`/accounts/${auth.account.id}/contacts/${id}/labels`, {
              method: 'POST',
              body: { label_id: Number(labelId) }
            })
          } else {
            await api(`/accounts/${auth.account.id}/contacts/${id}/labels/${labelId}`, {
              method: 'DELETE'
            })
          }
        } catch {
          // Continue on individual errors
        }
      }
    }
  }
})
