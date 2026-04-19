import { defineStore } from 'pinia'

export interface Conversation {
  id: string
  accountId: string
  inboxId: string
  contactInboxId: string
  status: 'OPEN' | 'PENDING' | 'RESOLVED' | 'SNOOZED'
  unreadCount: number
  lastActivityAt: string
  contactInbox?: { contact?: { name?: string | null, phoneNumber?: string | null, waJid?: string | null } }
  inbox?: { name: string }
}

export const useConversationsStore = defineStore('conversations', {
  state: () => ({
    list: [] as Conversation[],
    current: null as Conversation | null,
    loading: false
  }),
  actions: {
    setAll(list: Conversation[]) {
      this.list = list
    },
    setCurrent(conv: Conversation | null) {
      this.current = conv
    },
    upsert(conv: Conversation) {
      const idx = this.list.findIndex(c => c.id === conv.id)
      if (idx >= 0) this.list[idx] = conv
      else this.list.unshift(conv)
    }
  }
})
