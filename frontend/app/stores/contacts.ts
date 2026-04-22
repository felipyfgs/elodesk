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
  avatarUrl: string | null
  blocked: boolean
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

export interface ContactEvent {
  id: string
  action: string
  metadata: Record<string, unknown> | null
  user: { id: string, name: string } | null
  createdAt: string
}

export interface ContactEventListResponse {
  meta: ContactMeta
  payload: ContactEvent[]
}

export const useContactsStore = defineStore('contacts', {
  state: () => ({
    list: [] as Contact[],
    meta: { page: 1, pageSize: 25, total: 0 } as ContactMeta,
    loading: false
  }),
  actions: {
    async fetchPage(params: { search?: string, labels?: string, page?: number, pageSize?: number, append?: boolean } = {}) {
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

        if (params.append) {
          this.list = [...this.list, ...res.payload]
        } else {
          this.list = res.payload
        }

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

    async remove(id: string) {
      const auth = useAuthStore()
      if (!auth.account?.id) return
      const api = useApi()
      await api(`/accounts/${auth.account.id}/contacts/${id}`, { method: 'DELETE' })
      this.removeMany([id])
    },

    async update(id: string, data: Partial<Contact>) {
      const auth = useAuthStore()
      if (!auth.account?.id) return
      const api = useApi()
      const updated = await api<Contact>(`/accounts/${auth.account.id}/contacts/${id}`, {
        method: 'POST',
        body: data
      })
      this.upsert(updated)
      return updated
    },

    async merge(childId: string, primaryId: string) {
      const auth = useAuthStore()
      if (!auth.account?.id) return
      const api = useApi()
      const primary = await api<Contact>(`/accounts/${auth.account.id}/contacts/${childId}/merge`, {
        method: 'POST',
        body: { primary_contact_id: Number(primaryId) }
      })
      this.removeMany([childId])
      this.upsert(primary)
      return primary
    },

    async setBlocked(id: string, blocked: boolean) {
      const auth = useAuthStore()
      if (!auth.account?.id) return
      const api = useApi()
      const updated = await api<Contact>(`/accounts/${auth.account.id}/contacts/${id}/block`, {
        method: 'PATCH',
        body: { blocked }
      })
      this.upsert(updated)
      return updated
    },

    async uploadAvatar(id: string, file: File) {
      const auth = useAuthStore()
      if (!auth.account?.id) throw new Error('No account')
      const api = useApi()

      const ext = file.name.split('.').pop()?.toLowerCase() ?? 'png'
      const objectKey = `${auth.account.id}/contacts/${id}/avatar.${ext}`

      const { upload_url } = await api<{ upload_url: string }>(
        `/accounts/${auth.account.id}/uploads/signed-url?path=${encodeURIComponent(objectKey)}`,
        { method: 'POST' }
      )

      await $fetch(upload_url, {
        method: 'PUT',
        body: file,
        headers: { 'Content-Type': file.type }
      })

      const updated = await api<Contact>(`/accounts/${auth.account.id}/contacts/${id}/avatar`, {
        method: 'POST',
        body: { object_key: objectKey }
      })
      this.upsert(updated)
      return updated
    },

    async deleteAvatar(id: string) {
      const auth = useAuthStore()
      if (!auth.account?.id) return
      const api = useApi()
      await api(`/accounts/${auth.account.id}/contacts/${id}/avatar`, { method: 'DELETE' })
      const idx = this.list.findIndex(c => c.id === id)
      if (idx >= 0 && this.list[idx]) {
        this.list[idx] = { ...this.list[idx]!, avatarUrl: null }
      }
    },

    async listEvents(id: string, page = 1, pageSize = 25): Promise<ContactEventListResponse> {
      const auth = useAuthStore()
      if (!auth.account?.id) {
        return { meta: { page, pageSize, total: 0 }, payload: [] }
      }
      const api = useApi()
      return api<ContactEventListResponse>(
        `/accounts/${auth.account.id}/contacts/${id}/events?page=${page}&pageSize=${pageSize}`
      )
    },

    async applyBulkLabel(ids: string[], labelId: string, op: 'add' | 'remove') {
      const auth = useAuthStore()
      if (!auth.account?.id) return

      const api = useApi()
      const errors: string[] = []

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
        } catch (error) {
          errors.push(id)
          if (import.meta.dev) console.error(`[contacts] bulk label ${op} failed for ${id}`, error)
        }
      }

      if (errors.length > 0) {
        throw new Error(`Failed to ${op} label for ${errors.length} contact(s)`)
      }
    }
  }
})
