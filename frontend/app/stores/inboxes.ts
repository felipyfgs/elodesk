import { defineStore } from 'pinia'

export interface InboxAgent {
  id: string
  inboxId: string
  userId: string
  user?: { id: string, name: string, avatarUrl?: string | null }
  createdAt: string
}

export interface Inbox {
  id: string
  accountId: string
  channelId: string
  name: string
  channelType: string
  createdAt: string
  updatedAt?: string
  channelApi?: {
    identifier: string
    webhookUrl: string
    hmacMandatory: boolean
  } | null
  agents?: InboxAgent[]
  openConversationCount?: number
  lastActivityAt?: string
}

export const useInboxesStore = defineStore('inboxes', {
  state: () => ({
    list: [] as Inbox[],
    loading: false
  }),
  actions: {
    setAll(list: Inbox[]) {
      this.list = list
    },
    upsert(inbox: Inbox) {
      const idx = this.list.findIndex(i => i.id === inbox.id)
      if (idx >= 0) this.list[idx] = inbox
      else this.list.push(inbox)
    },
    remove(id: string) {
      this.list = this.list.filter(i => i.id !== id)
    }
  }
})
