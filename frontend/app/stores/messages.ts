import { defineStore } from 'pinia'

export interface Message {
  id: string
  conversationId: string
  inboxId: string
  accountId: string
  content: string | null
  contentType: string
  messageType: 'INCOMING' | 'OUTGOING' | 'ACTIVITY' | 'TEMPLATE'
  senderType: 'CONTACT' | 'USER' | 'SYSTEM'
  senderId: string | null
  sourceId: string | null
  status: 'PENDING' | 'SENT' | 'DELIVERED' | 'READ' | 'FAILED'
  contentAttributes: Record<string, unknown>
  createdAt: string
  updatedAt: string
}

export const useMessagesStore = defineStore('messages', {
  state: () => ({
    byConversation: {} as Record<string, Message[]>
  }),
  actions: {
    set(conversationId: string, list: Message[]) {
      this.byConversation[conversationId] = list
    },
    upsert(msg: Message) {
      const bucket = (this.byConversation[msg.conversationId] ||= [])
      const idx = bucket.findIndex(m => m.id === msg.id)
      if (idx >= 0) bucket[idx] = msg
      else bucket.push(msg)
      bucket.sort((a, b) => a.createdAt.localeCompare(b.createdAt))
    }
  }
})
