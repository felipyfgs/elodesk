import { defineStore } from 'pinia'
import { useApi } from '~/composables/useApi'
import { useAuthStore } from '~/stores/auth'

export interface Notification {
  id: number
  accountId: number
  userId: number
  type: string
  payload: Record<string, unknown>
  readAt: string | null
  createdAt: string
}

interface State {
  items: Notification[]
  unreadCount: number
  cursor: number
  loading: boolean
}

export const useNotificationsStore = defineStore('notifications', {
  state: (): State => ({ items: [], unreadCount: 0, cursor: 0, loading: false }),
  actions: {
    async fetchRecent(limit = 25, unreadOnly = false) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      this.loading = true
      try {
        const res = await api<{ items: Notification[], unreadCount: number, nextCursor: number }>(
          `/accounts/${auth.account.id}/notifications`,
          { query: { limit, status: unreadOnly ? 'unread' : 'all' } }
        )
        this.items = res.items || []
        this.unreadCount = res.unreadCount ?? 0
        this.cursor = res.nextCursor ?? 0
      } finally {
        this.loading = false
      }
    },
    async fetchMore() {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id || !this.cursor) return
      const res = await api<{ items: Notification[], nextCursor: number }>(
        `/accounts/${auth.account.id}/notifications`,
        { query: { cursor: this.cursor, status: 'all' } }
      )
      this.items.push(...(res.items || []))
      this.cursor = res.nextCursor ?? 0
    },
    async markRead(id: number) {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/notifications/${id}/read`, { method: 'POST' })
      const item = this.items.find(i => i.id === id)
      if (item && !item.readAt) {
        item.readAt = new Date().toISOString()
        this.unreadCount = Math.max(0, this.unreadCount - 1)
      }
    },
    async markAllRead() {
      const api = useApi()
      const auth = useAuthStore()
      if (!auth.account?.id) return
      await api(`/accounts/${auth.account.id}/notifications/mark_all_read`, { method: 'POST' })
      const now = new Date().toISOString()
      this.items.forEach((i) => {
        if (!i.readAt) i.readAt = now
      })
      this.unreadCount = 0
    },
    handleRealtime(event: { type: string, payload?: Notification | { id: number } }) {
      if (event.type === 'notification.new' && event.payload && 'type' in event.payload) {
        this.items.unshift(event.payload as Notification)
        this.unreadCount += 1
      } else if (event.type === 'notification.read' && event.payload) {
        const id = (event.payload as { id: number }).id
        const item = this.items.find(i => i.id === id)
        if (item && !item.readAt) {
          item.readAt = new Date().toISOString()
          this.unreadCount = Math.max(0, this.unreadCount - 1)
        }
      } else if (event.type === 'notification.read_all') {
        const now = new Date().toISOString()
        this.items.forEach((i) => {
          if (!i.readAt) i.readAt = now
        })
        this.unreadCount = 0
      }
    }
  }
})
